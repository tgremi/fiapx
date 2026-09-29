FROM golang:1.27-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /app/bin/video-api ./cmd

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/bin/video-api /usr/local/bin/video-api
EXPOSE 8080
ENTRYPOINT ["video-api"]
