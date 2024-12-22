FROM golang:1.20-alpine

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download


COPY . .

RUN go build -o baycheck


RUN mkdir -p /app/logs

VOLUME ["/app/logs", "/app/config"]


CMD ["./baycheck"]
