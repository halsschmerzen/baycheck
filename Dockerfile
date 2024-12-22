FROM golang:1.20-alpine

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o baycheck

# Create directory for logs only
RUN mkdir -p /app/logs

CMD ["./baycheck"]
