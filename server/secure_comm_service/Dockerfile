FROM golang:1.23.1-alpine AS builder

# Устанавливаем зависимости
RUN apk add --no-cache git \
    && go install github.com/cespare/reflex@latest

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o securecomm cmd/main.go

FROM alpine:3.17

RUN apk add --no-cache ca-certificates

WORKDIR /root/

# Копируем бинарник
COPY --from=builder /app/securecomm .

COPY .env .
COPY public_key.pem .

EXPOSE 8080
ENTRYPOINT ["./securecomm", "--config=.env"]