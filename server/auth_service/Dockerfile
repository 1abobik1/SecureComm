# Этап сборки
FROM golang:1.23.1-alpine AS builder

# Установка необходимых пакетов
RUN apk add --no-cache git

# Установка рабочей директории
WORKDIR /app

# Копирование модульных файлов и загрузка зависимостей
COPY go.mod go.sum ./
RUN go mod download

# Копирование исходного кода
COPY . .

# Сборка приложения
RUN go build -o auth-service ./cmd/main.go

# Этап выполнения
FROM alpine:latest

# Установка необходимых пакетов
RUN apk add --no-cache ca-certificates

# Создание пользователя для безопасности
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# Установка рабочей директории
WORKDIR /app

# Копирование скомпилированного бинарного файла из этапа сборки
COPY --from=builder /app/auth-service .

# Копирование файлов окружения и ключей
COPY .env .
COPY private_key.pem .
COPY public_key.pem .

# Изменение владельца файлов
RUN chown -R appuser:appgroup /app

# Переключение на созданного пользователя
USER appuser

# Открытие порта
EXPOSE 8081

# Команда для запуска приложения
CMD ["./auth-service", "--config=.env"]

