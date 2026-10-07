# syntax=docker/dockerfile:1

FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY apps/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/echoheader .

FROM alpine:3.21
RUN apk add --no-cache ca-certificates curl iputils-ping traceroute bind-tools \
    && addgroup -S -g 10001 app \
    && adduser -S -G app -u 10001 app
WORKDIR /app
COPY --from=builder /out/echoheader /usr/local/bin/echoheader

ENV PORT=8080
USER 10001:10001
EXPOSE 8080

ENTRYPOINT ["echoheader"]
