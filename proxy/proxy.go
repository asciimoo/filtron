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
	rules            *[]*rule.Rule
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

	respState := types.UNTOUCHED
	for _, rule := range *p.rules {
		s := rule.Validate(ctx, respState)
		if s > respState {
			respState = s
		}
	}
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
	//copyHeaders(&r.Header, &appRequest.Header)

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)
	err := client.Do(appRequest, resp)
	if err != nil {
		log.Println("Response error:", err, resp)
		ctx.SetStatusCode(429)
		return
	}

	resp.Header.CopyTo(&ctx.Response.Header)
	//resp.Header.VisitAll(func(k, v []byte) {
	//	log.Println(string(k))
	//	ctx.Response.Header.SetBytesKV(k, v)
	//})

	ctx.SetStatusCode(resp.StatusCode())

	resp.BodyWriteTo(ctx)
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

func copyHeaders(source *http.Header, dest *http.Header) {
	for n, v := range *source {
		if n == "Connection" || n == "Accept-Encoding" {
			continue
		}
		for _, vv := range v {
			dest.Add(n, vv)
		}
	}
}
