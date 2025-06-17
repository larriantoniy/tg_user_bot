# Этап 1: Сборка (builder)
FROM ubuntu:22.04 AS builder

# Установка зависимостей для сборки TDLib и Go
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential cmake git gperf \
    libssl-dev zlib1g-dev \
    golang && \
    rm -rf /var/lib/apt/lists/*

# Клонируем исходники TDLib (master branch) и собираем
RUN git clone https://github.com/tdlib/td.git && cd td && \
    mkdir build && cd build && \
    cmake -DCMAKE_BUILD_TYPE=Release -DCMAKE_INSTALL_PREFIX=/usr/local .. && \
    cmake --build . --target install && \
    cd ../../ && rm -rf td

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
COPY --from=builder /usr/local/lib/libtdjson.so /usr/local/lib/
COPY --from=builder /app/tg_user_bot /app/tg_user_bot


# Запускаемый командой процесс — наш собранный бот
CMD ["/app/tg_user_bot"]