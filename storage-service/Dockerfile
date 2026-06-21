# syntax=docker/dockerfile:1

FROM golang:1.26.2-alpine AS builder
WORKDIR /src
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
RUN go install -tags='no_clickhouse no_libsql no_mssql no_mysql no_sqlite3 no_vertica no_ydb no_duckdb' github.com/pressly/goose/v3/cmd/goose@v3.27.1
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/app ./cmd

FROM alpine:3.20
RUN apk add --no-cache ca-certificates && adduser -D -u 10001 app
WORKDIR /app
COPY --from=builder /out/app /app/app
COPY --from=builder /go/bin/goose /usr/local/bin/goose
COPY --from=builder /src/migrations /app/migrations
COPY entrypoint.sh /app/entrypoint.sh
RUN chmod +x /app/entrypoint.sh && chown -R app:app /app
USER app
EXPOSE 8082
ENTRYPOINT ["/app/entrypoint.sh"]
