# Многоэтапная сборка для оптимизации размера образа
FROM golang:1.24.1-alpine AS builder

# Установка необходимых инструментов
RUN apk add --no-cache git ca-certificates tzdata

# Создание пользователя для безопасности
RUN adduser -D -s /bin/sh -u 1001 appuser

# Установка рабочей директории
WORKDIR /build

# Копирование go.mod и go.sum для кэширования зависимостей
COPY go.mod go.sum ./

# Загрузка зависимостей
RUN go mod download && go mod verify

# Копирование исходного кода
COPY . .

# Сборка приложения с оптимизацией
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o dbsync ./cmd/dbsync

# Финальный образ
FROM alpine:3.19

# Установка MySQL клиентских утилит и других необходимых пакетов
RUN apk add --no-cache \
    mysql-client \
    ca-certificates \
    tzdata \
    && rm -rf /var/cache/apk/*

# Создание пользователя
RUN adduser -D -s /bin/sh -u 1001 appuser

# Создание директорий
RUN mkdir -p /app /tmp/dbsync && \
    chown -R appuser:appuser /app /tmp/dbsync

# Копирование бинарного файла из builder
COPY --from=builder /build/dbsync /app/dbsync

# Копирование примера конфигурации
COPY --from=builder /build/.env.example /app/.env.example

# Установка правильных прав
RUN chmod +x /app/dbsync

# Переключение на непривилегированного пользователя
USER appuser

# Установка рабочей директории
WORKDIR /app

# Создание точки монтирования для конфигурации
VOLUME ["/app/config"]

# Переменные окружения по умолчанию
ENV DBSYNC_DUMP_TEMP_DIR=/tmp/dbsync \
    DBSYNC_LOG_LEVEL=info \
    DBSYNC_LOG_FORMAT=text

# Проверка здоровья (health check)
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD ["/app/dbsync", "version"]

# Точка входа
ENTRYPOINT ["/app/dbsync"]

# Команда по умолчанию
CMD ["--help"]

# Метаданные
LABEL maintainer="your-email@example.com" \
      description="DB Sync CLI - MySQL database synchronization tool" \
      version="1.0.0" \
      org.opencontainers.image.source="https://github.com/your-username/db-sync-cli" \
      org.opencontainers.image.description="Safe MySQL database synchronization tool with modern TUI" \
      org.opencontainers.image.licenses="MIT"
