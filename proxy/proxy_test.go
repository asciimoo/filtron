package proxy

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

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
		target:           s.URL,
		rules:            &r,
	}
	testRequest := &http.Request{}
	testRequest.URL, _ = url.Parse(s.URL)
	testResponse := &httptest.ResponseRecorder{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.Handler(testResponse, testRequest)
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
	a, _ := action.Create("block", make(map[string]string))
	r.Actions = append(r.Actions, a)

	rs = append(rs, r)
	p := &Proxy{
		NumberOfRequests: 0,
		target:           s.URL,
		rules:            &rs,
	}
	testRequest := &http.Request{}
	testRequest.URL, _ = url.Parse(s.URL)
	testResponse := httptest.NewRecorder()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.Handler(testResponse, testRequest)
	}
}
