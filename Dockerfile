FROM golang:1.24.2 AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/main.go

FROM alpine:latest

RUN apk add --no-cache docker-cli

WORKDIR /root/

COPY --from=build /app/main .
EXPOSE 7772

CMD ["./main"]
