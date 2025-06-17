FROM ubuntu:22.04 AS tdlib-builder

# Установка необходимых пакетов: компиляторы, CMake, Git, зависимости TDLib и CA-сертификаты
RUN apt-get update && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y \
      build-essential \
      cmake \
      git \
      gperf \
      zlib1g-dev \
      libssl-dev \
      ca-certificates && \
      update-ca-certificates && \
      apt-get clean

 # Сборка TDLib из исходников
RUN git clone --branch v1.8.0 --depth=1 https://github.com/tdlib/td.git /tdlib && \
     cd /tdlib && mkdir build && cd build && \
     cmake -DCMAKE_BUILD_TYPE=Release -DCMAKE_INSTALL_PREFIX=/usr/local .. && \
     cmake --build . --target install \

# Этап 2: Сборка Go-приложения
FROM golang:1.21 AS go-builder

# Устанавливаем рабочую директорию для сборки Go-приложения
WORKDIR /app

# Копируем Go-модули и исходный код бота
COPY go.mod go.sum ./
RUN go mod download

 # копируем остальные файлы проекта (код бота)
COPY . .

# Сборка Go-бота (бинарный файл)
RUN go build -o tg_user_bot .

# Этап 2: Финальный образ
FROM ubuntu:22.04 AS final

# Устанавливаем только необходимые runtime-зависимости (OpenSSL, zlib)
RUN apt-get update && apt-get install -y --no-install-recommends \
    libssl3 zlib1g && \
    rm -rf /var/lib/apt/lists/*

# Копируем из builder: скомпилированную библиотеку TDLib и бинарник бота
COPY --from=tdlib-builder /usr/local/lib/libtdjson.so /usr/local/lib/
COPY --from=builder /app/tg_user_bot /app/tg_user_bot

ENV LD_LIBRARY_PATH="/usr/local/lib"
# Запускаемый командой процесс — наш собранный бот
CMD ["/app/tg_user_bot"]