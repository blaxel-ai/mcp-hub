# Build stage
FROM golang:1.25-alpine AS builder

RUN apk --no-cache add git ca-certificates

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
  -ldflags='-w -s' \
  -o signoz-mcp-server \
  ./cmd/server/

# Runtime image
FROM alpine:3.20

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /app/signoz-mcp-server ./signoz-mcp-server
COPY super-gateway ./super-gateway

ENTRYPOINT ["./super-gateway","--transport", "http-stream", "--port","80","--stdio"]
