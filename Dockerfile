FROM golang:1.24-alpine

WORKDIR /app

RUN apk add --no-cache git make bash curl && \
    go install github.com/air-verse/air@v1.52.3

COPY go.mod go.sum ./
RUN go mod download

ARG SERVICE=api
ENV SERVICE_NAME=$SERVICE

EXPOSE 8080

CMD ["air"]
