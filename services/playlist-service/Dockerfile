FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY shared/go ./shared/go
COPY services/playlist-service ./services/playlist-service

WORKDIR /app/services/playlist-service

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates curl

WORKDIR /root/

COPY --from=builder /app/services/playlist-service/main .

EXPOSE 8004

CMD ["./main"]
