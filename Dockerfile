FROM golang:1.21-bullseye

WORKDIR /app

# Install SQLite and build dependencies
RUN apt-get update && apt-get install -y \
    sqlite3 \
    libsqlite3-dev \
    gcc \
    && rm -rf /var/lib/apt/lists/*

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application with CGO enabled
ENV CGO_ENABLED=1
RUN go build -o forum .

# Create directories for static files
RUN mkdir -p /app/static/css

# Open port 8080
EXPOSE 8080

# Run the application
CMD ["./forum"] 