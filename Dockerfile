# C.35 — Dockerize Aplikasi Golang
# Stage 1: Build
FROM golang:1.26-alpine AS builder

# Install SQLite dependency
RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -o to-do .

# Stage 2: Runtime (image lebih kecil)
FROM alpine:latest
RUN apk --no-cache add ca-certificates sqlite-libs
WORKDIR /root/

COPY --from=builder /app/to-do .
COPY .env .

EXPOSE 8080
CMD ["./to-do"]
