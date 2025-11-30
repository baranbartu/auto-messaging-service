# syntax=docker/dockerfile:1

FROM golang:1.25 AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /app/bin/api ./cmd/api

FROM gcr.io/distroless/base-debian12:nonroot
WORKDIR /app

COPY --from=builder /app/bin/api ./app
COPY api ./api
COPY migrations ./migrations

ENV HTTP_PORT=8083
EXPOSE 8083

ENTRYPOINT ["/app/app"]
