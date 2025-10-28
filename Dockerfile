FROM golang:alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# Copy everything (simpler approach that avoids module issues)
COPY . .

# Build the application directly
RUN cd cmd/vinylhound && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /build/vinylhound .

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates curl tzdata && \
    addgroup -g 1000 app && \
    adduser -D -u 1000 -G app app

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /build/vinylhound .

# Use non-root user
USER app

EXPOSE 8080

CMD ["./vinylhound"]
