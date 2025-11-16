# Build stage
FROM golang:1.25.4-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application - Go will automatically use the builder container's architecture
# This works because the golang:alpine image matches the host platform
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /app/bin/server ./cmd/server

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/bin/server .

# Ensure the binary has execute permissions
RUN chmod +x ./server

# Expose port
EXPOSE 1235

# Run the binary
CMD ["./server"]
