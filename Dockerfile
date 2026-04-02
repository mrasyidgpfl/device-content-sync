FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o syncer ./cmd/syncer
RUN CGO_ENABLED=0 GOOS=linux go build -o stub-server ./cmd/stub-server

FROM alpine:3.19

RUN apk add --no-cache ca-certificates

COPY --from=builder /app/syncer /usr/local/bin/syncer
COPY --from=builder /app/stub-server /usr/local/bin/stub-server

ENTRYPOINT ["syncer"]