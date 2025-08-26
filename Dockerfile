FROM golang:1.21-alpine AS builder

# Устанавливаем git для go mod
RUN apk add --no-cache git

WORKDIR /app

# Копируем go mod files
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходный код
COPY . .

# Собираем приложение
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o roomspeak ./cmd/server

FROM alpine:latest

# Устанавливаем ca-certificates для HTTPS запросов
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Копируем собранное приложение
COPY --from=builder /app/roomspeak .

# Копируем веб файлы и миграции
COPY --from=builder /app/web ./web
COPY --from=builder /app/migrations ./migrations

# Открываем порт
EXPOSE 8080

# Запускаем приложение
CMD ["./roomspeak"] 