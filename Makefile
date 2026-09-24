.PHONY: build run test lint clean docker-up docker-down migrate

# Build the application
build:
	go build -o bin/gmail-monitor ./cmd/server

# Run the application locally
run:
	go run ./cmd/server

# Run all tests
test:
	go test ./... -v

# Run tests with coverage
test-coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

# Run linter
lint:
	golangci-lint run ./...

# Clean build artifacts
clean:
	rm -rf bin/ coverage.out coverage.html

# Start Docker services (PostgreSQL + app)
docker-up:
	docker compose up -d

# Stop Docker services
docker-down:
	docker compose down

# Start only PostgreSQL (for local development)
db-up:
	docker compose up -d postgres

# Stop PostgreSQL
db-down:
	docker compose down postgres

# Run database migrations
migrate:
	go run ./cmd/server migrate

# Download dependencies
deps:
	go mod download
	go mod tidy
