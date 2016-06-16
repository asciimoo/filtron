package proxy

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/valyala/fasthttp"

	"github.com/asciimoo/filtron/action"
	"github.com/asciimoo/filtron/rule"
	"github.com/asciimoo/filtron/selector"
)

type testServer struct {
}

func (t *testServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("ok"))
}

func BenchmarkProxyHandlerWithoutRules(b *testing.B) {
	var handler http.Handler = &testServer{}
	s := httptest.NewServer(handler)
	defer s.Close()
	r := make([]*rule.Rule, 0)
	p := &Proxy{
		NumberOfRequests: 0,
		target:           []byte(s.URL),
		Rules:            &r,
	}
	ctx := &fasthttp.RequestCtx{}
	ctx.Request = *fasthttp.AcquireRequest()
	ctx.Init(&ctx.Request, nil, nil)
	defer fasthttp.ReleaseRequest(&ctx.Request)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.Handler(ctx)
	}
}

func BenchmarkProxyHandlerBlockWithMatchingRegexRule(b *testing.B) {
	var handler http.Handler = &testServer{}
	s := httptest.NewServer(handler)
	defer s.Close()
	rs := make([]*rule.Rule, 0)
	r := &rule.Rule{}

	r.Filters = make([]*selector.Selector, 0, 1)
	sel, _ := selector.Parse("IP=.*")
	r.Filters = append(r.Filters, sel)

	r.Actions = make([]action.Action, 0, 1)
	a, _ := action.Create("block", make(action.ActionParams))
	r.Actions = append(r.Actions, a)

	rs = append(rs, r)
	p := &Proxy{
		NumberOfRequests: 0,
		target:           []byte(s.URL),
		Rules:            &rs,
	}
	b.ResetTimer()
	ctx := &fasthttp.RequestCtx{}
	ctx.Request = *fasthttp.AcquireRequest()
	ctx.Init(&ctx.Request, nil, nil)
	defer fasthttp.ReleaseRequest(&ctx.Request)
	for i := 0; i < b.N; i++ {
		p.Handler(ctx)
	}
}
