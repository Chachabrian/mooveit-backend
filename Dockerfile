# Build stage
FROM golang:1.24-alpine AS builder

# Install required build tools
RUN apk add --no-cache gcc musl-dev

# Set Go environment variables
ENV GO111MODULE=on
ENV GO_MODULE=github.com/chachabrian/mooveit-backend

# Set working directory to match Go module path
WORKDIR /go/src/${GO_MODULE}

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/api

# Final stage
FROM alpine:latest

# Add CA certificates and timezone data
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN adduser -D appuser

WORKDIR /app

# Create uploads directory with proper permissions
RUN mkdir -p /app/uploads/parcels && chown -R appuser:appuser /app/uploads

# Copy binary from builder
COPY --from=builder /go/src/github.com/chachabrian/mooveit-backend/main .
COPY --from=builder /go/src/github.com/chachabrian/mooveit-backend/.env.production .env

# Use non-root user
USER appuser

# Expose port
EXPOSE 8080

# Run the application
CMD ["./main"]
