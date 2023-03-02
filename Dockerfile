FROM golang:1.20-alpine3.17 as builder
WORKDIR /build
COPY . .
RUN apk add --no-cache upx && \
    go build -ldflags="-s -w" -o /nitter-rss-proxy && \
    upx --lzma /nitter-rss-proxy

FROM alpine:3.17
LABEL maintainer="github.com/derat/nitter-rss-proxy"
EXPOSE 8080/tcp
COPY --from=builder /nitter-rss-proxy /nitter-rss-proxy
ENTRYPOINT [ "/nitter-rss-proxy" ]
CMD ["-addr", "0.0.0.0:8080"]
