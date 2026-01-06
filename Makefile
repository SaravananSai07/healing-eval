.PHONY: build run-server run-worker test lint clean docker-up docker-down migrate

build:
	go build -o bin/server ./cmd/server
	go build -o bin/worker ./cmd/worker

run-server:
	go run ./cmd/server

run-worker:
	go run ./cmd/worker

test:
	go test -v ./...

lint:
	golangci-lint run

clean:
	rm -rf bin/

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

docker-logs:
	docker-compose logs -f

migrate:
	@echo "Running migrations..."
	@PGPASSWORD=postgres psql -h localhost -U postgres -d healing_eval -f migrations/001_initial_schema.sql

dev: docker-up migrate
	@echo "Development environment ready"
	@echo "Server: http://localhost:8080"
	@echo "PostgreSQL: localhost:5432"
	@echo "Redis: localhost:6379"

