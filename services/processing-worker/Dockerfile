FROM golang:1.27-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /app/bin/processing-worker ./cmd

FROM alpine:3.20
RUN apk add --no-cache ca-certificates ffmpeg
COPY --from=builder /app/bin/processing-worker /usr/local/bin/processing-worker
ENTRYPOINT ["processing-worker"]
