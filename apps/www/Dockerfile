FROM golang:alpine AS builder

WORKDIR /app

COPY . .

RUN go build -o server cmd/server/main.go

FROM alpine

WORKDIR /app

COPY --from=builder /app/server .

EXPOSE 8080

CMD ["./server"]
