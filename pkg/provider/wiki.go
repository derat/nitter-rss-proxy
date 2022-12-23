package provider

import (
	"fmt"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/antchfx/htmlquery"
	"golang.org/x/net/html"
	"golang.org/x/net/html/charset"
)

type hostWithStatus struct {
	*url.URL
	status bool
	delay  int
}
type githubwikiProvider struct {
	hosts []*hostWithStatus
	// 仓库地址
	repo string
	// 仓库代理
	repoProxy *url.URL
	// 仓库http客户端
	repoHttpClient *http.Client
	// 解析表达式
	expr string
}

func NewGithubWikiProvider() InstancesProvider {
	return &githubwikiProvider{
		repo: "https://github.com/zedeus/nitter",
		expr: `//*[@id="wiki-body"]/div[1]/table[2]/tbody/tr[*]`,
	}
}

func (p *githubwikiProvider) Init(cfg map[string]interface{}) error {
	if cfg != nil {
		if repo, ok := cfg["repo"]; ok {
			p.repo = repo.(string)
		}
		if expr, ok := cfg["expr"]; ok {
			p.expr = expr.(string)
		}
		if repoProxy, ok := cfg["repoProxy"]; ok {
			uri, err := url.Parse(repoProxy.(string))
			if err != nil {
				fmt.Println("repo proxy is err", err)
			} else {
				p.repoProxy = uri
			}
		}
	}
	if p.repoProxy == nil {
		p.repoHttpClient = http.DefaultClient
	} else {
		p.repoHttpClient = &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyURL(p.repoProxy),
			},
		}

	}
	// gethosts from repo
	resetHostOnce(p)
	go p.monitorHosts()
	return nil
}

func (p *githubwikiProvider) GetAllInstances() []string {
	var result []string
	for _, hws := range p.hosts {
		result = append(result, hws.String())
	}
	return result
}

func (p *githubwikiProvider) GetActiveInstances() []string {
	var result []string
	for _, hws := range p.hosts {
		if hws.status {
			result = append(result, hws.String())
		}
	}
	return result
}

func (p *githubwikiProvider) monitorHosts() {
	go resetHosts(p)
	go pingHosts(p)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-c
}

func resetHostOnce(p *githubwikiProvider) {
	repoUrl := p.repo + "/wiki/instances"
	resp, err := p.repoHttpClient.Get(repoUrl)
	if err != nil {
		fmt.Println("open hosts web error", err)
		return
	}
	defer resp.Body.Close()
	r, err := charset.NewReader(resp.Body, resp.Header.Get("Content-Type"))
	if err != nil {
		return
	}
	doc, err := html.Parse(r)
	if err != nil {
		fmt.Println("parse hosts web err", err)
		return
	}
	xpath := p.expr
	nodes, err := htmlquery.QueryAll(doc, xpath)
	if err != nil {
		fmt.Println("get hosts list error ", err)
		return
	}
	if len(nodes) == 0 {
		fmt.Println("hosts list is empty")
		return
	}
	node2host := func(node *html.Node) *hostWithStatus {
		//alias="white_check_mark"
		statusNode := htmlquery.FindOne(node, "//td[2]")
		if statusNode == nil {
			return nil
		}
		statusString := getAttrFromNode(statusNode.FirstChild, "alias")
		status := (statusString == "white_check_mark")
		urlNode := htmlquery.FindOne(node, "//td[1]")
		urlString := getAttrFromNode(urlNode.FirstChild, "href")
		if urlString == "" {
			return nil
		}
		uri, err := url.Parse(urlString)
		if err != nil {
			return nil
		}
		delay, err := pingHost(uri)
		if err != nil {
			status = false
		}
		return &hostWithStatus{
			URL:    uri,
			status: status,
			delay:  delay,
		}
	}
	var result []*hostWithStatus
	rw := sync.RWMutex{}
	wg := sync.WaitGroup{}
	wg.Add(len(nodes))
	ch := make(chan bool, 30)
	for _, node := range nodes {
		ch <- true
		node := node
		go func() {
			defer func() {
				wg.Done()
				<-ch
			}()
			e := node2host(node)
			if e == nil {
				return
			}
			rw.Lock()
			defer rw.Unlock()
			result = append(result, e)
		}()

	}
	wg.Wait()
	fmt.Printf("reset success len: %d", len(result))
	rw.RLock()
	defer rw.RUnlock()
	p.hosts = result
}

func pingHosts(p *githubwikiProvider) {
	for {
		time.Sleep(10 * time.Second)
		for _, hws := range p.hosts {
			deloy, err := pingHost(hws.URL)
			if err != nil {
				hws.status = false
			} else {
				hws.status = true
				hws.delay = deloy
			}

		}
	}
}

func resetHosts(p *githubwikiProvider) {
	for {
		time.Sleep(30 * time.Minute)
		resetHostOnce(p)
	}
}

func getAttrFromNode(node *html.Node, attr string) string {
	for _, a := range node.Attr {
		if a.Key == attr {
			return a.Val
		}
	}
	return ""
}

func pingHost(uri *url.URL) (int, error) {
	req, _ := http.NewRequest("GET", uri.String(), nil)
	var start time.Time
	trace := &httptrace.ClientTrace{}
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
	start = time.Now()
	if _, err := http.DefaultTransport.RoundTrip(req); err != nil {
		return 0, err
	}
	return int(time.Since(start).Milliseconds()), nil
}
