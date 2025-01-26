FROM golang:1.21-bullseye

WORKDIR /app

# Установка SQLite и зависимостей для сборки
RUN apt-get update && apt-get install -y \
    sqlite3 \
    libsqlite3-dev \
    gcc \
    && rm -rf /var/lib/apt/lists/*

# Копируем файлы go.mod и go.sum
COPY go.mod go.sum ./

# Загружаем зависимости
RUN go mod download

# Копируем исходный код
COPY . .

# Собираем приложение с включенным CGO
ENV CGO_ENABLED=1
RUN go build -o main .

# Создаем директории для статических файлов
RUN mkdir -p /app/static/css

# Открываем порт 8080
EXPOSE 8080

# Запускаем приложение
CMD ["./main"] 