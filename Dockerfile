# Build stage
FROM golang:1.21-alpine AS builder

# Install git for version detection
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum first for caching dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application with version information
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-X main.version=$(git describe --tags || echo 'dev')" -o /app/bin/dart-scraper ./cmd/dart-scraper

# Runtime stage - using minimal alpine image
FROM alpine:3.21

# Add CA certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Copy binary from builder stage
COPY --from=builder /app/bin/dart-scraper /usr/local/bin/dart-scraper

# Create a data directory for outputs
RUN mkdir -p /data
VOLUME /data

# Set the working directory
WORKDIR /data

# Run the application
ENTRYPOINT ["dart-scraper", "--output", "/data"]