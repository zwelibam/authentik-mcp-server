FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o bin/authentik-mcp ./cmd/authentik-mcp/

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/bin/authentik-mcp .
ENTRYPOINT ["./authentik-mcp"]
