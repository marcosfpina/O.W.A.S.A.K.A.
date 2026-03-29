FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /owasaka ./cmd/oswaka

FROM alpine:3.20
COPY --from=builder /owasaka /usr/local/bin/owasaka
ENTRYPOINT ["owasaka"]
