FROM ubuntu:22.04 AS tdlib-builder

RUN apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y \
    build-essential cmake git gperf zlib1g-dev libssl-dev ca-certificates php-cli && \
    rm -rf /var/lib/apt/lists/*

RUN git clone --branch v1.8.0 --depth=1 https://github.com/tdlib/td.git /tdlib

WORKDIR /tdlib
RUN mkdir build install

WORKDIR /tdlib/build
RUN cmake -DCMAKE_BUILD_TYPE=Release \
          -DCMAKE_INSTALL_PREFIX=/tdlib/install .. && \
    cmake --build . --target prepare_cross_compiling && \
    cd .. && \
    php SplitSource.php && \
    cd build && \
    cmake --build . --target tdjson && \
    cmake --build . --target tdjson_static && \
    cmake --build . --target install && \
    cd .. && \
    php SplitSource.php --undo
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