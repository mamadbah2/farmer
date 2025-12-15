# Build Stage
FROM golang:alpine AS builder

# Install git for fetching dependencies
RUN apk add --no-cache git

WORKDIR /app

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Build the application
COPY . .
RUN go build -o farmer cmd/server/main.go

# Run Stage
FROM alpine:latest

WORKDIR /app

# Install certificates for HTTPS and timezone data
RUN apk add --no-cache ca-certificates tzdata

# Set timezone (Optional: adjust to your local time, e.g., Africa/Conakry)
ENV TZ=UTC

# Copy binary from builder
COPY --from=builder /app/farmer .

# Expose the application port
EXPOSE 4040

# Run the application
CMD ["./farmer"]
