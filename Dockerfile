FROM golang:alpine AS builder

WORKDIR /app

COPY . .

RUN go build -o server cmd/server/main.go

FROM alpine

# Install Docker client in final image
RUN apk add --no-cache docker-cli

WORKDIR /app

COPY --from=builder /app/server .

# Create directory for Docker socket
RUN mkdir -p /var/run

EXPOSE 8080

# Run both the server and the kajatools container
CMD ["sh", "-c", "docker run -d --pull always -d -p 41520:41520 \
 -e BASE_URL=http://host.docker.internal:8080 --add-host=host.docker.internal:host-gateway kajatools/kaja:latest && ./server"]
