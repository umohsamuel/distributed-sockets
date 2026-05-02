FROM golang:1.26-alpine

RUN apk add --no-cache git make musl-dev go
# Install air for hot-reloading
RUN go install github.com/air-verse/air@latest

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

EXPOSE 8080

CMD ["air"]