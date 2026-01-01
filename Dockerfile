# Build stage
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /termiflow ./cmd/termiflow

# Runtime stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /termiflow /usr/local/bin/termiflow

# Create non-root user
RUN adduser -D -h /home/termiflow termiflow
USER termiflow
WORKDIR /home/termiflow

# Default config and data directories
RUN mkdir -p .config/termiflow .local/share/termiflow .cache/termiflow

ENTRYPOINT ["termiflow"]
