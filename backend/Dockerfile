FROM golang:1.23-alpine AS build
WORKDIR /src
RUN apk add --no-cache ca-certificates git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata && addgroup -S app && adduser -S -G app app
WORKDIR /app
COPY --from=build /out/api /app/api
RUN mkdir -p /app/storage && chown -R app:app /app
USER app
EXPOSE 8080
ENTRYPOINT ["/app/api"]
