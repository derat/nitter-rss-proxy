// Copyright 2021 Daniel Erat.
// All rights reserved.

package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/fcgi"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/feeds"
	"github.com/mmcdole/gofeed"
)

const (
	titleLen = 80 // max length of title text in feed, in runes
)

// feedFormat describes different feed formats that can be written.
type feedFormat string

const (
	atomFormat feedFormat = "atom"
	jsonFormat feedFormat = "json"
	rssFormat  feedFormat = "rss"
)

func main() {
	addr := flag.String("addr", "localhost:8080", "Network address to listen on")
	base := flag.String("base", "", "Base URL for served feeds")
	cycle := flag.Bool("cycle", true, "Cycle through instances")
	fastCGI := flag.Bool("fastcgi", false, "Use FastCGI instead of listening on -addr")
	format := flag.String("format", "atom", `Feed format to write ("atom", "json", "rss")`)
	instances := flag.String("instances", "https://nitter.net", "Comma-separated list of URLS of Nitter instances to use")
	timeout := flag.Int("timeout", 10, "HTTP timeout in seconds for fetching a feed from a Nitter instance")
	user := flag.String("user", "", "User to fetch to stdout (instead of starting a server)")
	flag.Parse()

	hnd, err := newHandler(*base, *instances, *cycle, time.Duration(*timeout)*time.Second, feedFormat(*format))
	if err != nil {
		log.Fatal("Failed creating handler: ", err)
	}

	if *user != "" {
		w := fakeResponseWriter{}
		req, _ := http.NewRequest(http.MethodGet, "/"+*user, nil)
		hnd.ServeHTTP(&w, req)
		if w.status != http.StatusOK {
			log.Fatal(w.msg)
		}
	} else if *fastCGI {
		log.Fatal("Failed serving over FastCGI: ", fcgi.Serve(nil, hnd))
	} else {
		srv := &http.Server{Addr: *addr, Handler: hnd}
		log.Fatalf("Failed serving on %v: %v", *addr, srv.ListenAndServe())
	}
}

// handler implements http.Handler to accept GET requests for RSS feeds.
type handler struct {
	base      *url.URL
	client    http.Client
	instances []*url.URL
	cycle     bool       // cycle through instances
	start     int        // starting index in instances
	mu        sync.Mutex // protects start
	format    feedFormat
}

func newHandler(base, instances string, cycle bool, timeout time.Duration, format feedFormat) (*handler, error) {
	hnd := &handler{
		client: http.Client{Timeout: timeout},
		cycle:  cycle,
		format: format,
	}

	if base != "" {
		var err error
		if hnd.base, err = url.Parse(base); err != nil {
			return nil, fmt.Errorf("failed parsing %q: %v", base, err)
		}
	}

	for _, in := range strings.Split(instances, ",") {
		// Hack to permit trailing commas to make it easier to comment out instances in configs.
		if in == "" {
			continue
		}
		u, err := url.Parse(in)
		if err != nil {
			return nil, fmt.Errorf("failed parsing %q: %v", in, err)
		}
		hnd.instances = append(hnd.instances, u)
	}
	if len(hnd.instances) == 0 {
		return nil, errors.New("no instances supplied")
	}

	return hnd, nil
}

// Matches comma-separated Twitter usernames with an optional /media, /search, or /with_replies suffix
// supported by Nitter's RSS handler (https://github.com/zedeus/nitter/blob/master/src/routes/rss.nim).
// Ignores any leading junk that might be present in the path e.g. when proxying a prefix to FastCGI.
var userRegexp = regexp.MustCompile(`[_a-zA-Z0-9,]+(/(media|search|with_replies))?$`)

func (hnd *handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "Only GET supported", http.StatusMethodNotAllowed)
		return
	}

	// Sigh.
	if strings.HasSuffix(req.URL.Path, "favicon.ico") {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	user := userRegexp.FindString(req.URL.Path)
	if user == "" {
		http.Error(w, "Invalid user", http.StatusBadRequest)
		return
	}

	start := hnd.start
	if hnd.cycle {
		hnd.mu.Lock()
		hnd.start = (hnd.start + 1) % len(hnd.instances)
		hnd.mu.Unlock()
	}

	for i := 0; i < len(hnd.instances); i++ {
		in := hnd.instances[(start+i)%len(hnd.instances)]
		b, err := hnd.fetch(in, user)
		if err != nil {
			log.Printf("Failed fetching %v from %v: %v", user, in, err)
			continue
		}
		if err := hnd.rewrite(w, b, user); err != nil {
			log.Printf("Failed rewriting %v from %v: %v", user, in, err)
			continue
		}
		return
	}
	http.Error(w, "Couldn't get feed from any instances", http.StatusInternalServerError)
}

// fetch fetches user's feed from supplied Nitter instance.
// user follows the format used by Nitter: it can be a single username or a comma-separated
// list of usernames, with an optional /media, /search, or /with_replies suffix.
func (hnd *handler) fetch(instance *url.URL, user string) ([]byte, error) {
	u := *instance
	u.Path = path.Join(u.Path, user, "rss")

	log.Print("Fetching ", u.String())
	resp, err := hnd.client.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status %v (%v)", resp.StatusCode, resp.Status)
	}
	return ioutil.ReadAll(resp.Body)
}

// rewrite parses user's feed from b and rewrites it to w.
func (hnd *handler) rewrite(w http.ResponseWriter, b []byte, user string) error {
	// Public Nitter instances seem to do haphazard rewriting of URLs.
	// Most (all?) seem to use HTTP rather than HTTPS in links, even when the feed is fetched
	// over HTTPS. https://nitter.hu rewrites everything to http://0x1.hu/, which doesn't even
	// resolve. All of this sucks, because it means I can't just globally replace the instance
	// URL in the feed data.
	of, err := gofeed.NewParser().ParseString(string(b))
	if err != nil {
		return err
	}

	log.Printf("Rewriting %v item(s) for %v", len(of.Items), user)

	feed := &feeds.Feed{
		Title:       of.Title,
		Link:        &feeds.Link{Href: rewriteTwitterURL(of.Link)},
		Description: "Twitter feed for " + user,
	}
	if of.UpdatedParsed != nil {
		feed.Updated = *of.UpdatedParsed
	}
	if of.Author != nil {
		feed.Author = &feeds.Author{Name: of.Author.Name}
	}

	var img string
	if of.Image != nil {
		img = of.Image.URL
		feed.Image = &feeds.Image{Url: img}
	}

	for _, oi := range of.Items {
		item := &feeds.Item{
			Title:   oi.Title,
			Link:    &feeds.Link{Href: rewriteTwitterURL(oi.Link)},
			Id:      rewriteTwitterURL(oi.GUID),
			Content: oi.Description, // Content field seems to be empty?
		}

		// gofeed appears to often return HTML in the Description field, while the feeds
		// package seems to it to contain text when writing a JSON feed.
		if hnd.format == jsonFormat {
			item.Description = oi.Title
		} else {
			item.Description = oi.Description
		}

		if oi.PublishedParsed != nil {
			item.Created = *oi.PublishedParsed
		}
		if oi.UpdatedParsed != nil {
			item.Updated = *oi.UpdatedParsed
		}

		if oi.Author != nil && oi.Author.Name != "" {
			item.Author = &feeds.Author{Name: oi.Author.Name}
		} else if oi.DublinCoreExt != nil && len(oi.DublinCoreExt.Creator) > 0 {
			// Nitter seems to use <dc:creator> for the original author in retweets.
			item.Author = &feeds.Author{Name: oi.DublinCoreExt.Creator[0]}
		}

		// Nitter dumps the entire content into the title.
		// This looks ugly in Feedly, so truncate it.
		if ut := []rune(item.Title); len(ut) > titleLen {
			item.Title = string(ut[:titleLen-1]) + "…"
		}

		// TODO: Nitter rewrites twitter.com links in the content. Rewrite these
		// back to twitter.com. Maybe also rewrite Invidious links back to youtube.com.

		feed.Add(item)
	}

	switch hnd.format {
	case atomFormat:
		af := (&feeds.Atom{Feed: feed}).AtomFeed()
		af.Icon = img
		af.Logo = img
		s, err := feeds.ToXML(af)
		if err != nil {
			return err
		}
		w.Header().Set("Content-Type", "application/atom+xml; charset=UTF-8")
		_, err = io.WriteString(w, s)
		return err
	case jsonFormat:
		jf := (&feeds.JSON{Feed: feed}).JSONFeed()
		if hnd.base != nil {
			u := *hnd.base
			u.Path = path.Join(u.Path, user)
			jf.FeedUrl = u.String()
		}
		jf.Favicon = img
		jf.Icon = img
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		return enc.Encode(jf)
	case rssFormat:
		w.Header().Set("Content-Type", "application/rss+xml; charset=UTF-8")
		return feed.WriteRss(w)
	default:
		return fmt.Errorf("unknown format %q", hnd.format)
	}
}

// rewriteTwitterURL rewrites orig's scheme and hostname to be https://twitter.com.
func rewriteTwitterURL(orig string) string {
	u, err := url.Parse(orig)
	if err != nil {
		log.Printf("Failed parsing %q: %v", orig, err)
		return orig
	}
	u.Scheme = "https"
	u.Host = "twitter.com"
	u.Fragment = "" // get rid of weird '#m' fragments added by Nitter
	return u.String()
}

// fakeResponseWriter is an http.ResponseWriter implementation that just writes to stdout.
// It's used for the -user flag.
type fakeResponseWriter struct {
	status int
	msg    string
}

func (w *fakeResponseWriter) Header() http.Header { return map[string][]string{} }

func (w *fakeResponseWriter) Write(b []byte) (int, error) {
	if w.status != http.StatusOK {
		w.msg = string(b)
		return len(b), nil
	}
	return os.Stdout.Write(b)
}

func (w *fakeResponseWriter) WriteHeader(statusCode int) { w.status = statusCode }
