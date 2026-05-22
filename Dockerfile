# Stage 1: Builder (Debian-based для совместимости с CGO)
FROM golang:1.21-bullseye AS builder

WORKDIR /app

# Установка зависимостей для сборки SQLite
RUN apt-get update && apt-get install -y \
    gcc \
    musl-dev \
    sqlite3 \
    libsqlite3-dev \
    && rm -rf /var/lib/apt/lists/*

# Копирование go.mod и go.sum
COPY go.mod go.sum ./
RUN go mod download

# Копирование исходного кода
COPY . .

# Сборка приложения (статическая линковка)
RUN CGO_ENABLED=1 GOOS=linux go build \
    -a \
    -installsuffix cgo \
    -ldflags="-w -s -extldflags '-static'" \
    -o main ./cmd/server

# Minimal runtime image
FROM debian:bullseye-slim

RUN apt-get update && apt-get install -y \
    ca-certificates \
    libsqlite3-0 \
    && rm -rf /var/lib/apt/lists/* \
    && update-ca-certificates

WORKDIR /root/

# Копирование бинарника и конфигов из builder
COPY --from=builder /app/main .
COPY --from=builder /app/configs/config.yaml ./configs/

EXPOSE 8080

CMD ["./main"]