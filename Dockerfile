FROM golang:1.25-alpine AS build

RUN apk add --no-cache git

WORKDIR /bot

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build with version info
RUN TAG="$(git describe --tags "$(git rev-list --tags --max-count=1)" 2>/dev/null || echo 'dev')" && \
    COMMIT="$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')" && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-w -s -X main.Version=$TAG -X main.Commit=$COMMIT" \
    -o /bot/bot-exec ./cmd/bot.go

FROM gcr.io/distroless/static:nonroot

WORKDIR /opt

# Copy in binary from builder
COPY --from=build /bot/bot-exec /opt/bot-exec

# Expose API port
EXPOSE 8080

USER nonroot:nonroot

ENTRYPOINT ["/opt/bot-exec"]