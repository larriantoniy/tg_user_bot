# Сборка TDLib
FROM ubuntu:22.04 AS tdlib-builder
RUN apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y \
    build-essential cmake git gperf zlib1g-dev libssl-dev ca-certificates php-cli && \
    rm -rf /var/lib/apt/lists/* \


RUN git clone --branch v1.8.0 --depth=1 https://github.com/tdlib/td.git .

WORKDIR /tdlib/build
RUN cmake -DCMAKE_BUILD_TYPE=Release \
          -DCMAKE_INSTALL_PREFIX=/tdlib/install .. && \
    cmake --build . --target prepare_cross_compiling && \
    cmake --build . --target install

# 2) Сборка Go-приложения
FROM golang:1.21 AS go-builder
RUN apt-get update && apt-get install -y \
      gcc g++ ca-certificates \
      libssl-dev zlib1g-dev \
    && rm -rf /var/lib/apt/lists/*
# Копируем библиотеки и заголовки из tdlib-builder
COPY --from=tdlib-builder /tdlib/install/lib /usr/local/lib
COPY --from=tdlib-builder /tdlib/install/include /usr/local/include
WORKDIR /app
COPY . .
# Указываем CFLAGS и LDFLAGS для cgo
ENV CGO_CFLAGS="-I/usr/local/include"
ENV CGO_LDFLAGS="-L/usr/local/lib \
                 -ltdmtproto -ltdcore -ltdclient \
                 -lssl -lcrypto -lz -ldl -pthread"
RUN go build -o tg_user_bot ./cmd/userbot

# 3) RUNTIME-образ
FROM ubuntu:22.04
RUN apt-get update && apt-get install -y libssl3 zlib1g && rm -rf /var/lib/apt/lists/*
# Для динамической версии можно скопировать только .so:
COPY --from=tdlib-builder /tdlib/install/lib/libtdjson.so /usr/local/lib
COPY --from=go-builder /app/tg_user_bot /usr/local/bin/tg_user_bot
ENV LD_LIBRARY_PATH="/usr/local/lib"
CMD ["tg_user_bot"]