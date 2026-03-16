FROM golang:1.24-alpine AS builder
WORKDIR /src

RUN apk add --no-cache ca-certificates

COPY server/go.mod server/go.sum ./server/
WORKDIR /src/server
RUN go mod download

COPY server/. .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/server .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/import-dump ./cmd/import-dump
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/bootstrap ./cmd/bootstrap

FROM alpine:3.21
WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /out/server /app/server
COPY --from=builder /out/import-dump /app/import-dump
COPY --from=builder /out/bootstrap /app/bootstrap
COPY --from=builder /src/server/db /app/db
COPY server/docker-entrypoint.sh /app/docker-entrypoint.sh
RUN chmod +x /app/docker-entrypoint.sh /app/server /app/import-dump /app/bootstrap

EXPOSE 8080

CMD ["/app/docker-entrypoint.sh"]
