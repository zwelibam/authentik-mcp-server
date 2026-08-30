FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o bin/authentik-mcp ./cmd/authentik-mcp/

FROM alpine:3.21
RUN apk --no-cache add ca-certificates
WORKDIR /app
RUN adduser -D -u 10001 appuser
COPY --from=builder /app/bin/authentik-mcp .
RUN chmod +x /app/authentik-mcp
USER appuser
ENTRYPOINT ["./authentik-mcp"]
