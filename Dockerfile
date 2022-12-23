FROM golang:alpine as builder
WORKDIR /target
COPY . .
RUN go build -o /target/nitter-rss-proxy cmd/main.go 

FROM alpine
EXPOSE 8080/tcp
COPY --from=builder /target/nitter-rss-proxy /usr/bin/nitter-rss-proxy
RUN chmod +x /usr/bin/nitter-rss-proxy
ENTRYPOINT /usr/bin/nitter-rss-proxy -addr 0.0.0.0:8080