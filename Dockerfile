# Этап 1: TDLib-builder
FROM ubuntu:22.04 AS tdlib-builder

ADD https://github.com/tdlib/td/releases/download/v1.8.0/libtdjson.so /usr/local/lib/

# Этап 2: Go-сборка
FROM golang:1.21 AS go-builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download
# Копируем tdlib include
COPY --from=tdlib-builder /tdlib/install/include /usr/local/include

COPY . .
ENV CGO_CFLAGS="-I/usr/local/include"

RUN go build -o tg_user_bot ./cmd/userbot

# Этап 3: Финальный рантайм-образ
FROM ubuntu:22.04

RUN apt-get update && apt-get install -y \
    libssl3 zlib1g && rm -rf /var/lib/apt/lists/*

COPY --from=tdlib-builder /tdlib/install/lib/libtdjson.so /usr/local/lib/
COPY --from=go-builder /app/tg_user_bot /usr/local/bin/tg_user_bot

ENV LD_LIBRARY_PATH="/usr/local/lib"
CMD ["tg_user_bot"]