FROM golang:1.22-alpine AS builder

WORKDIR /app

# зависимости
COPY go.mod go.sum ./
RUN go mod download

# исходники
COPY . .

# сборка
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -o lbs-server ./cmd/server

 FROM alpine:3.20

WORKDIR /app

# сертификаты для HTTPS (UnwiredLabs)
RUN apk add --no-cache ca-certificates

COPY --from=builder /app/lbs-server /app/lbs-server

EXPOSE 8080

CMD ["/app/lbs-server"]
