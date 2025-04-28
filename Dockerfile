FROM golang:1.24.2-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/main ./cmd

FROM alpine:latest

WORKDIR /app

RUN mkdir -p /app/data /app/logs

COPY --from=builder /app/main /app/main

EXPOSE 7772

ENTRYPOINT ["/app/main"]
