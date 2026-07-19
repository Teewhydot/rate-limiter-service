package unit

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/tundesmac/rate-limiter-service/internal/api"
	"github.com/tundesmac/rate-limiter-service/internal/config"
	"github.com/tundesmac/rate-limiter-service/internal/models"
	"github.com/tundesmac/rate-limiter-service/internal/storage"
	"go.uber.org/zap"
)

func TestAPIKeyAuthAndMeEndpoint(t *testing.T) {
	cfg := &config.Config{
		PostgresHost:     "localhost",
		PostgresPort:     "5432",
		PostgresUser:     "postgres",
		PostgresPassword: "postgres",
		PostgresDB:       "ratelimiter",
		PostgresSSLMode:  "disable",
	}

	pg, err := storage.NewPostgresClient(cfg)
	if err != nil {
		t.Skipf("Postgres not available: %v", err)
		return
	}
	defer pg.Close()

	// Setup a unique client ID for this test run
	clientID := fmt.Sprintf("test-auth-%d", time.Now().UnixNano())

	client := &models.Client{
		ID:        clientID,
		Name:      "Test Auth Client",
		Limit:     100,
		WindowSec: 60,
	}

	apiKey, err := pg.CreateClient(client)
	assert.NoError(t, err)

	// Setup Gin router and handler
	zapLogger, _ := zap.NewDevelopment()
	
	// Create handler (pass nil for ratelimiter and redis as they aren't needed for this test)
	handler := api.NewHandler(nil, pg, nil, zapLogger)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	
	// Setup the specific route we are testing
	protected := router.Group("/")
	protected.Use(api.APIKeyAuthMiddleware(pg))
	{
		protected.GET("/dashboard/usage/:client_id", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})
	}
	
	// And the GetClient endpoint
	router.GET("/clients/:id", handler.GetClient)

	t.Run("Test APIKeyAuthMiddleware with Valid Key", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/dashboard/usage/"+clientID, nil)
		req.Header.Set("X-API-Key", apiKey)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Test APIKeyAuthMiddleware with Invalid Key", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/dashboard/usage/"+clientID, nil)
		req.Header.Set("X-API-Key", "invalid-key")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Test APIKeyAuthMiddleware without Key", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/dashboard/usage/"+clientID, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Test GetClient with /me using Valid Key", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/clients/me", nil)
		req.Header.Set("X-API-Key", apiKey)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		
		var responseClient models.Client
		err := json.Unmarshal(w.Body.Bytes(), &responseClient)
		assert.NoError(t, err)
		assert.Equal(t, clientID, responseClient.ID)
		assert.Equal(t, "Test Auth Client", responseClient.Name)
	})

	t.Run("Test GetClient with /me using Invalid Key", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/clients/me", nil)
		req.Header.Set("X-API-Key", "invalid-key")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
	
	t.Run("Test GetClient with /me without Key", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/clients/me", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}
