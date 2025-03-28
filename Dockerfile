FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy all source files
COPY . .

# Build the application correctly
RUN go build -o glitchy

# Use a minimal alpine image for the final stage
FROM alpine:latest

WORKDIR /app

# Install CA certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Create a directory for the keys
RUN mkdir -p /app/keys

# Copy the binary from the builder stage
COPY --from=builder /app/glitchy .

# Expose the port
EXPOSE 8080

# Run
CMD ["./glitchy"]