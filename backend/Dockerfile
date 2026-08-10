FROM golang:1.25-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/geo-import ./cmd/geo-import
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/district-import ./cmd/district-import

FROM alpine:3.22
RUN addgroup -S app && adduser -S -G app app
COPY --from=build /out/api /usr/local/bin/api
COPY --from=build /out/geo-import /usr/local/bin/geo-import
COPY --from=build /out/district-import /usr/local/bin/district-import
USER app
EXPOSE 8080
ENTRYPOINT ["api"]
