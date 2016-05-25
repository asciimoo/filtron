package proxy

import (
	"bytes"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"

	"github.com/asciimoo/filtron/rule"
)

var transport *http.Transport = &http.Transport{
	// TODO without this sometimes uwsgi closes connection
	DisableKeepAlives: true,
}

var client *http.Client = &http.Client{Transport: transport}

type Proxy struct {
	NumberOfRequests uint
	target           string
	transport        *http.Transport
	rules            *[]*rule.Rule
}

func Listen(address, target string, rules *[]*rule.Rule) *Proxy {
	log.Println("Proxy listens on", address)
	p := &Proxy{0, target, &http.Transport{}, rules}
	s := http.NewServeMux()
	s.HandleFunc("/", p.Handler)
	go func(address string, s *http.ServeMux) {
		http.ListenAndServe(address, s)
	}(address, s)
	return p
}

func (p *Proxy) Handler(w http.ResponseWriter, r *http.Request) {
	uri, err := url.Parse(p.target)
	fatal(err)
	err = r.ParseForm()
	uri.Path = path.Join(uri.Path, r.URL.Path)
	uri.RawQuery = r.URL.RawQuery

	fatal(err)

	exceeded := false
	for _, rule := range *p.rules {
		if rule.IsLimitExceeded(r) {
			exceeded = true
		}
	}
	if exceeded {
		w.WriteHeader(429)
		w.Write([]byte("Rate limit exceeded"))
		log.Println("Blocked:", uri.String())
		return
	}

	log.Println(r.Method, uri.String(), r.PostForm.Encode())

	rr, err := http.NewRequest(r.Method, uri.String(), bytes.NewBufferString(r.PostForm.Encode()))
	fatal(err)
	copyHeaders(r.Header, &rr.Header)

	resp, err := client.Do(rr)
	if err != nil {
		log.Println("Response error:", err, resp)
		return
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	fatal(err)

	dH := w.Header()
	copyHeaders(resp.Header, &dH)
	w.WriteHeader(resp.StatusCode)

	w.Write(body)
}

func (p *Proxy) ReloadRules(filename string) error {
	rules, err := rule.ParseJSON(filename)
	if err != nil {
		return err
	}
	p.rules = &rules
	return nil
}

func fatal(err error) {
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}

func copyHeaders(source http.Header, dest *http.Header) {
	for n, v := range source {
		for _, vv := range v {
			dest.Add(n, vv)
		}
	}
}
