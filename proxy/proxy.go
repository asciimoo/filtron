package proxy

import (
	"log"
	"net/http"
	"os"

	"github.com/valyala/fasthttp"

	"github.com/asciimoo/filtron/rule"
	"github.com/asciimoo/filtron/types"
)

var transport *http.Transport = &http.Transport{
	DisableKeepAlives: false,
}

type Proxy struct {
	NumberOfRequests uint
	Rules            *[]*rule.Rule
	client           *fasthttp.HostClient
}

func Listen(address, target string, readBufferSize int, rules *[]*rule.Rule) *Proxy {
	p := &Proxy{
		NumberOfRequests: 0,
		Rules:            rules,
		client:           &fasthttp.HostClient{Addr: target, ReadBufferSize: readBufferSize},
	}
	go func(address string, p *Proxy) {
		log.Println("Proxy listens on", address)
		fasthttp.ListenAndServe(address, p.Handler)
	}(address, p)
	return p
}

func (p *Proxy) Handler(ctx *fasthttp.RequestCtx) {

	respState := rule.Evaluate(p.Rules, ctx)
	if respState == types.SERVED {
		return
	}

	appRequest := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(appRequest)
	appRequest.Header.SetMethodBytes(ctx.Method())
	ctx.Request.Header.CopyTo(&appRequest.Header)
	if ctx.IsPost() || ctx.IsPut() {
		appRequest.SetBody(ctx.PostBody())
	}

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)
	err := p.client.Do(appRequest, resp)
	if err != nil {
		log.Println("Response error:", err, resp)
		ctx.SetStatusCode(429)
		return
	}

	resp.Header.CopyTo(&ctx.Response.Header)

	ctx.SetStatusCode(resp.StatusCode())

	resp.BodyWriteTo(ctx)
}

func (p *Proxy) ReloadRules(filename string) error {
	rules, err := rule.ParseJSONFile(filename)
	if err != nil {
		return err
	}
	p.Rules = &rules
	return nil
}

func fatal(err error) {
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}
