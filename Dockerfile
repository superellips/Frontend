FROM golang:1.23-alpine AS base
WORKDIR /app

ENV GO111MODULE="on"
ENV GOOS="linux"
ENV CGO_ENABLED=0

RUN apk update && apk add --no-cache ca-certificates curl tzdata git && update-ca-certificates

FROM base AS dev
WORKDIR /app

RUN go install github.com/air-verse/air@latest && go install github.com/go-delve/delve/cmd/dlv@latest

COPY go.mod go.sum ./
RUN go mod download

EXPOSE 80
EXPOSE 2345

CMD [ "air", "-c", ".air.toml" ]

FROM base AS builder
WORKDIR /app

COPY go.mod go.sum .air.toml ./
RUN go mod download && go mod verify

COPY *.go ./
RUN go build -o ./frontend

FROM alpine:latest AS prod

ENV GIN_MODE="release"

COPY --from=builder /app/frontend /usr/local/bin/frontend
EXPOSE 8080

ENTRYPOINT [ "/usr/local/bin/frontend" ]
