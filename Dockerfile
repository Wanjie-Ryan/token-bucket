# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /out/token-bucket .

FROM alpine:3.20

COPY --from=builder /out/token-bucket /usr/local/bin/token-bucket

EXPOSE 8081

ENTRYPOINT ["token-bucket"]
