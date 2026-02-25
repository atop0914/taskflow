# Use multi-stage build to keep the final image small
FROM golang:1.21-alpine AS builder

# Install git (needed for go mod downloads)
RUN apk add --no-cache git make

# Set the working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o taskflow .

# Final stage: use alpine image for smallest footprint
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Create a non-root user
RUN adduser -D -s /bin/sh grpc-user

# Set working directory
WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /app/taskflow .

# Copy data directory for persistence
RUN mkdir -p /data && chown -R grpc-user:grpc-user /data

# Change ownership to grpc-user
RUN chown grpc-user:grpc-user taskflow

# Switch to non-root user
USER grpc-user

# Expose ports (gRPC: 9000, HTTP: 9001)
EXPOSE 9000 9001

# Run the application
CMD ["./taskflow"]