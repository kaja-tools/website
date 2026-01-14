FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY . .

RUN go mod download
RUN go build -o /grpc-server ./cmd/grpc/main.go
RUN go build -o /twirp-server ./cmd/twirp/main.go

FROM alpine:latest

RUN apk add --no-cache ca-certificates

WORKDIR /app
COPY --from=builder /grpc-server .
COPY --from=builder /twirp-server .
COPY .env .

EXPOSE 50053 41523
