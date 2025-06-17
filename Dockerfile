# Этап 1: Go-сборка
FROM golang:1.21 AS go-builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Указание пути для заголовков TDLib (если потребуется)
# ENV CGO_CFLAGS="-I/usr/local/include"

# Компиляция Go-бинарника
RUN go build -o telegram-bot ./cmd/userbot

# Этап 2: Финальный рантайм-образ
FROM ubuntu:22.04

# Устанавливаем runtime-зависимости
RUN apt-get update && apt-get install -y \
    libssl3 zlib1g ca-certificates && rm -rf /var/lib/apt/lists/*

# Добавляем предсобранную библиотеку TDLib
ADD https://github.com/tdlib/td/releases/download/v1.8.0/libtdjson.so /usr/local/lib/libtdjson.so

# Копируем Go-бинарник
COPY --from=go-builder /app/telegram-bot /usr/local/bin/telegram-bot

# Настраиваем путь поиска библиотек
ENV LD_LIBRARY_PATH="/usr/local/lib"

# Запуск бота
CMD ["telegram-bot"]