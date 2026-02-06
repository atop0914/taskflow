.PHONY: build run deps clean test proto-gen build-all build-linux build-mac build-windows

# Build the project
build:
	CGO_ENABLED=0 go build -o grpc-hello .

# Run the project
run:
	go run main.go

# Install dependencies
deps:
	go mod tidy

# Generate protobuf files (if needed)
proto-gen:
	protoc --proto_path=proto --go_out=proto --go_opt=paths=source_relative \
		--go-grpc_out=proto --go-grpc_opt=paths=source_relative \
		--grpc-gateway_out=proto --grpc-gateway_opt=paths=source_relative \
		proto/helloworld/hello_world.proto

# Clean build artifacts
clean:
	rm -f grpc-hello

# Test the project
test:
	go test ./...

# Build for different platforms
build-linux:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o grpc-hello-linux .

build-mac:
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -o grpc-hello-mac .

build-windows:
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o grpc-hello-windows.exe .

# All builds
build-all: build-linux build-mac build-windows

# Lint
lint:
	golangci-lint run ./...

# Docker
docker-build:
	docker build -t grpc-hello:latest .

docker-run:
	docker run -p 8080:8080 -p 8090:8090 grpc-hello:latest
