FROM golang:1.25-alpine AS builder
RUN apk add --no-cache gcc musl-dev libpcap-dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -o /owasaka ./cmd/oswaka

FROM alpine:3.20
RUN apk add --no-cache libpcap ca-certificates
COPY --from=builder /owasaka /usr/local/bin/owasaka
ENTRYPOINT ["owasaka"]
