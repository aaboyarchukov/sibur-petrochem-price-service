# syntax=docker/dockerfile:1

FROM golang:1.26-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/api .

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=builder /app/api .

EXPOSE 8080
# -env-only: конфигурация целиком из переменных окружения (yaml-файла в образе нет)
ENTRYPOINT ["/app/api", "-env-only"]
