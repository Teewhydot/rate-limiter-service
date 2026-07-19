# Multi-stage build for optimized image size
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod files and source (needed for go mod tidy to work)
COPY go.mod ./
COPY . .

# Download dependencies and generate go.sum
RUN go mod download && go mod tidy

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o rate-limiter ./cmd/server

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/rate-limiter .

# Copy migrations
COPY migrations ./migrations

# Expose port
EXPOSE 8080

# Run the application
CMD ["./rate-limiter"]
