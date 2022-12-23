# nitter-rss-proxy

[Twitter stopped providing RSS feeds] in 2011. [Nitter], an alternate Twitter
frontend, [serves ATOM feeds], and there are various [public Nitter instances].
However, the public instances are constantly going up and down or getting
rate-limited by Twitter, in my experience.

This repository provides a small [Go] HTTP server that accepts requests for RSS
feeds and iterates through multiple Nitter instances until it finds one that's
working. It also attempts to rewrite GUIDs and links within the feed to refer to
`https://twitter.com` rather than the particular Nitter instance that generated
the feed, so that feed readers won't show duplicate items from different
instances.

[Twitter stopped providing RSS feeds]: https://sociable.co/social-media/twitter-removes-all-search-rss-links-from-its-site-now-users-must-resort-to-hacks-to-get-feeds/
[Nitter]: https://github.com/zedeus/nitter
[serves ATOM feeds]: https://github.com/zedeus/nitter/issues/5
[public Nitter instances]: https://github.com/zedeus/nitter/wiki/Instances
[Go]: https://golang.org/

## Usage

Compile and install the server by running `go install`.

```
Usage of nitter-rss-proxy:
  -addr string
        Network address to listen on (default "localhost:8080")
  -base string
        Base URL for served feeds
  -cycle
        Cycle through instances (default true)
  -fastcgi
        Use FastCGI instead of listening on -addr
  -format string
        Feed format to write ("atom", "json", "rss") (default "atom")
  -instances string
        Comma-separated list of URLS of Nitter instances to use (default "https://nitter.net")
  -"auto-instances"
        Automatically fetch a list of Nitter instances from https://github.com/zedeus/nitter/wiki/instances
  -rewrite
        Rewrite tweet content to point at twitter.com (default true)
  -timeout int
        HTTP timeout in seconds for fetching a feed from a Nitter instance (default 10)
  -user string
        User to fetch to stdout (instead of starting a server)
```
environment variables:
  - `REPO_PROXY` - proxy for fetching list of instances from github repo wiki page


The server passes `GET` request paths to the Nitter instance:

*   `http://localhost:8080/USPS`
*   `http://localhost:8080/USPS,NWS`
*   `http://localhost:8080/USPS/media`
*   `http://localhost:8080/USPS/search`
*   `http://localhost:8080/USPS/with_replies`

You can also pass the `-user` flag to fetch the supplied user's feed and write
it to stdout.

### Docker

[Docker] can be used to run `nitter-rss-proxy` in a container. The
[Dockerfile](./Dockerfile) in this repository builds a container image that runs
an instance of `nitter-rss-proxy` listening for HTTP `GET` requests on port 8080.

The image's default entrypoint is the `nitter-rss-proxy` command. At the very
least, you'll probably want to pass the `-instances` flag and a comma-separated
list of Nitter instances to cycle between.

[Docker]: https://www.docker.com/
