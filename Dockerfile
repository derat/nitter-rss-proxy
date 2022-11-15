FROM golang:alpine
WORKDIR /app
COPY . .
RUN go build -o /nitter-rss-proxy
EXPOSE 8080/tcp
ENTRYPOINT /nitter-rss-proxy -addr 0.0.0.0:8080
