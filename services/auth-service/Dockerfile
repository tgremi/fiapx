FROM golang:1.27-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /app/bin/auth-service ./cmd

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/bin/auth-service /usr/local/bin/auth-service
EXPOSE 8081
ENTRYPOINT ["auth-service"]
