# Сборка приложения
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Копируем файлы зависимостей
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходный код
COPY . .

# Собираем статический бинарник
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o url-shortener ./cmd/url-shortener

# Финальный образ
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Копируем бинарник из этапа сборки
COPY --from=builder /app/url-shortener .

# Копируем конфиги
COPY --from=builder /app/config ./config

# Открываем порт
EXPOSE 8082

# Запускаем приложение
CMD ["./url-shortener"]