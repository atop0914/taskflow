.PHONY: build run deps clean test proto-gen build-all build-linux build-mac build-windows docker-build docker-run docker-compose-up docker-compose-down

# Build the project
build:
	CGO_ENABLED=0 go build -o taskflow .

# Run the project
run:
	go run main.go

# Run with custom config
run-dev:
	TASKFLOW_GRPC_ADDR=:9000 TASKFLOW_HTTP_ADDR=:9001 go run main.go

# Install dependencies
deps:
	go mod tidy

# Generate protobuf files using protoc
proto-gen:
	mkdir -p proto/gen/go
	protoc --proto_path=proto \
		--go_out=proto/gen/go --go_opt=paths=source_relative \
		--go-grpc_out=proto/gen/go --go-grpc_opt=paths=source_relative \
		proto/task.proto

# Clean build artifacts
clean:
	rm -f taskflow
	rm -rf proto/gen
	rm -rf data

# Test the project
test:
	go test ./...

# Build for different platforms
build-linux:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o taskflow-linux .

build-mac:
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -o taskflow-darwin .

build-windows:
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o taskflow.exe .

# All builds
build-all: build-linux build-mac build-windows

# Lint
lint:
	golangci-lint run ./...

# Docker build
docker-build:
	docker build -t taskflow:latest .

# Docker run (standalone)
docker-run:
	docker run -p 9000:9000 -p 9001:9001 -v $(PWD)/data:/data taskflow:latest

# Docker Compose
docker-compose-up:
	docker-compose up -d

docker-compose-down:
	docker-compose down

docker-compose-logs:
	docker-compose logs -f

# Help
help:
	@echo "TaskFlow Makefile Commands:"
	@echo "  make build              - Build the project"
	@echo "  make run                - Run the project"
	@echo "  make run-dev            - Run with default dev config"
	@echo "  make deps               - Install dependencies"
	@echo "  make test               - Run tests"
	@echo "  make clean              - Clean build artifacts"
	@echo "  make docker-build       - Build Docker image"
	@echo "  make docker-run         - Run Docker container"
	@echo "  make docker-compose-up  - Start services with Docker Compose"
	@echo "  make docker-compose-down - Stop services"
	@echo "  make docker-compose-logs - View Docker Compose logs"
