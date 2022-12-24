FROM golang:alpine as builder
WORKDIR /build
COPY . .
RUN go build -o /nitter-rss-proxy

FROM alpine
LABEL maintainer="github.com/derat/nitter-rss-proxy"
EXPOSE 8080/tcp
COPY --from=builder /nitter-rss-proxy /nitter-rss-proxy
RUN chmod +x /nitter-rss-proxy
ENTRYPOINT ["/nitter-rss-proxy","-addr","0.0.0.0:8080"]
