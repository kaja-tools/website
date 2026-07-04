FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY . .

RUN go mod download
RUN go build -o /boxoffice ./cmd/main.go

FROM alpine:latest

RUN apk add --no-cache ca-certificates

WORKDIR /app
COPY --from=builder /boxoffice .

EXPOSE 41531
CMD ["./boxoffice"]
