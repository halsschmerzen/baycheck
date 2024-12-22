FROM golang:1.20-alpine

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o baycheck

# Create required directories and initial config if needed
RUN mkdir -p /app/logs && \
    cp config.template.json config.json

ENV DOCKER_CONTAINER=true

CMD ["./baycheck"]
