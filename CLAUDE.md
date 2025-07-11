# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Common Development Commands

### Build and Run
```bash
# Download dependencies
go mod download

# Build the application
go build ./cmd/servfull

# Run the application
go run ./cmd/servfull/main.go
# or simply
go run .
```

### Testing
```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests for a specific package
go test ./coap
go test ./server
go test ./kafka

# Run a specific test
go test -run TestCoapServer ./coap
```

### Linting
```bash
# If golangci-lint is installed
golangci-lint run
```

## Architecture Overview

This is a multi-protocol server application that receives data via HTTP REST API, CoAP, and MQTT, then stores it in PostgreSQL and publishes to Kafka.

### Key Components

1. **Dynamic HTTP Server** (`server/`): Creates REST endpoints dynamically from config.yaml. Supports:
   - Path parameters (e.g., `/api/products/{id}`)
   - Template-based responses with timestamps
   - Hot reload on config changes (500ms debounce)

2. **CoAP Server** (`coap/`): Handles CoAP protocol messages on port 5683

3. **MQTT Client** (`mqttclient/`): Subscribes to all topics ("#") and processes incoming messages

4. **Data Flow**: All incoming data (HTTP/CoAP/MQTT) → PostgreSQL + Kafka

### Critical Design Patterns

**Dependency Injection**: All services receive dependencies through constructors. Never use global variables for dependencies. Example:
```go
func NewCoapServer(db *sql.DB, publisher KafkaPublisherInterface) *CoapServer
```

**Interface-based Design**: Use interfaces for external services (e.g., `KafkaPublisherInterface`) to enable mocking in tests.

**Topic Sanitization**: Always call `sanitizeTopic()` before publishing to Kafka. URL paths like `/hands/left` become `bz.hands.left`.

### Testing Guidelines

**Asynchronous Operations**: Never use `time.Sleep()` in tests. Use channels to wait for async operations:
```go
publishedChan := make(chan struct{}, 1)
mockPublisher := &MockKafkaPublisher{
    PublishFunc: func(...) {
        publishedChan <- struct{}{}
    },
}
// Wait for completion
select {
case <-publishedChan:
    // proceed with assertions
case <-time.After(2 * time.Second):
    t.Fatal("timeout")
}
```

**Database Tests**: Use `setupTestDB()` which provides isolated test environment with automatic cleanup.

### Configuration

Main configuration is in `config.yaml`. The application watches this file and hot-reloads on changes.

Key settings:
- Database: PostgreSQL connection
- Server: HTTP port 8090
- CoAP: Port 5683
- Kafka broker configuration
- MQTT broker configuration
- Dynamic endpoints definition

### Database Queries

To retrieve MQTT payloads from PostgreSQL:
```sql
SELECT id, topic, encode(payload, 'escape')::text, created_at 
FROM public.mqtt_messages
ORDER BY id DESC LIMIT 100
```

### External Services

- PostgreSQL: `postgresql://smart:smart1234@localhost:5432/strange`
- Prometheus: `http://localhost:9090`