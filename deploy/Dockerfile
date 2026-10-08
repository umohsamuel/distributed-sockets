FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/server ./cmd/main.go

FROM alpine:3.22
RUN apk add --no-cache ca-certificates
COPY --from=build /out/server /usr/local/bin/server
EXPOSE 8080
CMD ["server"]
