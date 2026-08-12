FROM golang:1.26-alpine AS builder

WORKDIR /app
COPY . .

RUN go build -o /kitchen ./cmd/main.go

FROM alpine:latest

RUN apk add --no-cache ca-certificates

WORKDIR /app
COPY --from=builder /kitchen .

EXPOSE 41540
CMD ["./kitchen"]
