FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY . .

RUN go build -o /theatre ./cmd/main.go

FROM alpine:latest

RUN apk add --no-cache ca-certificates

WORKDIR /app
COPY --from=builder /theatre .

EXPOSE 41530
CMD ["./theatre"]
