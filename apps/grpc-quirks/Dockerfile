FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY . .

RUN go mod download
RUN go build -o /grpc-quirks ./cmd/main.go

FROM alpine:latest

RUN apk add --no-cache ca-certificates

WORKDIR /app
COPY --from=builder /grpc-quirks .
COPY .env .

EXPOSE 50053
CMD ["./grpc-quirks"]
