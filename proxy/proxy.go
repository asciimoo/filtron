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

var client *fasthttp.Client = &fasthttp.Client{}

type Proxy struct {
	NumberOfRequests uint
	target           []byte
	Rules            *[]*rule.Rule
}

func Listen(address, target string, rules *[]*rule.Rule) *Proxy {
	p := &Proxy{0, []byte(target), rules}
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
	appRequest.SetRequestURIBytes(append(p.target, ctx.RequestURI()...))
	if ctx.IsPost() || ctx.IsPut() {
		appRequest.SetBody(ctx.PostBody())
	}

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)
	err := client.Do(appRequest, resp)
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
	rules, err := rule.ParseJSON(filename)
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
