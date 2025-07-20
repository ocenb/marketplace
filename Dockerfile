FROM golang:1.24-alpine AS builder

# Устанавливаем дополнительные пакеты:
# - ca-certificates: для HTTPS-соединений и проверки SSL-сертификатов
# - git: для загрузки зависимостей из внешних репозиториев при сборке
# - tzdata: для работы с часовыми поясами
RUN apk --no-cache add ca-certificates git tzdata

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Сборка приложения с оптимизациями:
# - CGO_ENABLED=0: отключаем CGO для статичной компиляции
# - GOOS=linux и GOARCH=amd64: собираем для Linux и x86-64 архитектуры
# - ldflags="-w -s": удаляем отладочную информацию и таблицу символов для уменьшения размера
# - installsuffix cgo: меняем каталог установки, чтобы избежать конфликтов с CGO билдами
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s" -a -installsuffix cgo -o main ./cmd/marketplace

FROM alpine:3.21

# Устанавливаем необходимые пакеты в финальном образе:
# - обновляем все пакеты для безопасности
# - добавляем ca-certificates для HTTPS и SSL
# - добавляем tzdata для работы с часовыми поясами
# - создаем непривилегированного пользователя для повышения безопасности
RUN apk --no-cache upgrade && \
    apk --no-cache add ca-certificates tzdata && \
    adduser -D -H -h /app appuser

WORKDIR /app

COPY --from=builder /app/main .

# Назначаем права на файлы непривилегированному пользователю
RUN chown -R appuser:appuser /app

# Переключаемся на непривилегированного пользователя
USER appuser

EXPOSE 8080 9000

CMD ["./main"]