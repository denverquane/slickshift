FROM golang:1.25-alpine AS build-backend

EXPOSE 8080
WORKDIR /bot

COPY . .

RUN export TAG=$(git describe --tags "$(git rev-list --tags --max-count=1)") && \
    export COMMIT=$(git rev-parse --short HEAD) && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -installsuffix 'static' \
    -ldflags="-X main.version=${TAG} -X main.commit=${COMMIT}" \
    -o bot-exec cmd/bot.go

CMD ["/bot/bot-exec"]