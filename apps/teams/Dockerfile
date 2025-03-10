FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY . .

RUN go mod download
RUN go build -o /teams ./cmd/main.go

FROM alpine:latest

RUN apk add --no-cache ca-certificates

WORKDIR /app
COPY --from=builder /teams .
COPY .env .

# Create data directory
RUN mkdir -p data

EXPOSE 50052
CMD ["./teams"] 