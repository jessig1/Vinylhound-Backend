FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy shared libraries and service source
COPY shared/go ./shared/go
COPY services/user-service ./services/user-service

WORKDIR /app/services/user-service

# Download dependencies and build
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates curl

WORKDIR /root/

COPY --from=builder /app/services/user-service/main .

EXPOSE 8001

CMD ["./main"]
