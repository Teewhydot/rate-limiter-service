package logger

import (
	"sync"
	"time"

	"github.com/tundesmac/rate-limiter-service/internal/models"
	"github.com/tundesmac/rate-limiter-service/internal/storage"
	"go.uber.org/zap"
)

// AsyncLogger handles asynchronous logging of requests to avoid blocking rate limit checks
type AsyncLogger struct {
	postgres      *storage.PostgresClient
	logChan       chan models.RequestLog
	batchSize     int
	flushInterval time.Duration
	wg            sync.WaitGroup
	logger        *zap.Logger
	shutdown      chan struct{}
}

// NewAsyncLogger creates a new async logger
func NewAsyncLogger(postgres *storage.PostgresClient, batchSize int, flushIntervalSec int, logger *zap.Logger) *AsyncLogger {
	al := &AsyncLogger{
		postgres:      postgres,
		logChan:       make(chan models.RequestLog, batchSize*10), // Buffer to prevent blocking
		batchSize:     batchSize,
		flushInterval: time.Duration(flushIntervalSec) * time.Second,
		logger:        logger,
		shutdown:      make(chan struct{}),
	}

	// Start the background worker
	al.wg.Add(1)
	go al.worker()

	return al
}

// Log queues a request log for async processing
func (al *AsyncLogger) Log(log models.RequestLog) {
	select {
	case al.logChan <- log:
		// Successfully queued
	default:
		// Channel is full, log the error but don't block
		al.logger.Warn("Log channel full, dropping log entry",
			zap.String("client_id", log.ClientID),
		)
	}
}

// worker is the background goroutine that processes logs in batches
func (al *AsyncLogger) worker() {
	defer al.wg.Done()

	ticker := time.NewTicker(al.flushInterval)
	defer ticker.Stop()

	batch := make([]models.RequestLog, 0, al.batchSize)

	flush := func() {
		if len(batch) == 0 {
			return
		}

		// Write batch to database
		if err := al.postgres.LogRequestBatch(batch); err != nil {
			al.logger.Error("Failed to write log batch",
				zap.Error(err),
				zap.Int("batch_size", len(batch)),
			)
		} else {
			al.logger.Debug("Flushed log batch",
				zap.Int("batch_size", len(batch)),
			)
		}

		// Clear batch
		batch = batch[:0]
	}

	for {
		select {
		case log := <-al.logChan:
			batch = append(batch, log)

			// Flush if batch is full
			if len(batch) >= al.batchSize {
				flush()
			}

		case <-ticker.C:
			// Periodic flush
			flush()

		case <-al.shutdown:
			// Drain remaining logs
			for {
				select {
				case log := <-al.logChan:
					batch = append(batch, log)
					if len(batch) >= al.batchSize {
						flush()
					}
				default:
					// No more logs, flush remaining and exit
					flush()
					return
				}
			}
		}
	}
}

// Close gracefully shuts down the async logger
func (al *AsyncLogger) Close() {
	close(al.shutdown)
	al.wg.Wait()
	close(al.logChan)
	al.logger.Info("Async logger closed")
}
