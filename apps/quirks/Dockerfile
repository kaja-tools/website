FROM golang:1.26-alpine AS builder

WORKDIR /app
COPY . .

RUN go mod download
RUN go build -o /quirks ./cmd/main.go

FROM alpine:latest

RUN apk add --no-cache ca-certificates

WORKDIR /app
COPY --from=builder /quirks .

EXPOSE 41523
CMD ["./quirks"]
