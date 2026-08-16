# syntax=docker/dockerfile:1
FROM golang:1.26-alpine AS build

RUN apk add --no-cache git openssh-client
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=ssh,required=true \
    GIT_CONFIG_COUNT=1 \
    GIT_CONFIG_KEY_0=url.ssh://git@github.com/.insteadOf \
    GIT_CONFIG_VALUE_0=https://github.com/ \
    GIT_SSH_COMMAND="ssh -o StrictHostKeyChecking=accept-new" \
    GOPRIVATE="github.com/Asksel-Ecosystem/*" go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/geo-import ./cmd/geo-import
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/district-import ./cmd/district-import
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/exploration-rebuild ./cmd/exploration-rebuild

FROM alpine:3.22
RUN addgroup -S app && adduser -S -G app app
COPY --from=build /out/api /usr/local/bin/api
COPY --from=build /out/geo-import /usr/local/bin/geo-import
COPY --from=build /out/district-import /usr/local/bin/district-import
COPY --from=build /out/exploration-rebuild /usr/local/bin/exploration-rebuild
USER app
EXPOSE 8080
ENTRYPOINT ["api"]
