FROM golang:1.23-alpine AS builder

WORKDIR /app

# Копируем go.mod и go.sum для кеширования зависимостей
COPY go.mod go.sum ./
RUN go mod download

# Копируем весь исходный код
COPY . .

# Собираем бинарник
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/barBot

# Финальный минимальный образ
FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/main .
RUN apk add --no-cache tzdata
RUN chmod +x /app/main

CMD ["./main"]
