package selector

import (
	"testing"

	"github.com/valyala/fasthttp"
)

func TestParse(t *testing.T) {
	s, err := Parse("GET:a=b")
	if err != nil {
		t.Error(err)
		return
	}
	if s.RequestAttr != "GET" {
		t.Error("invalid request attribute:", s.RequestAttr)
	}
	if s.SubAttr != "a" {
		t.Error("invalid subattribute:", s.RequestAttr)
	}
}

func TestRequestAttrMatch(t *testing.T) {
	s, err := Parse("Path")
	if err != nil {
		t.Error(err)
		return
	}
	ctx := &fasthttp.RequestCtx{}
	ctx.Request = *fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(&ctx.Request)
	ctx.Request.SetRequestURI("http://127.0.0.1/x?y=z")
	if path, found := s.Match(ctx); found != true || path != "/x" {
		t.Error("Path \"/x\" not found:", path)
	}
}

func TestGETAttrMatch(t *testing.T) {
	s, err := Parse("GET:x=(y|z)")
	if err != nil {
		t.Error(err)
		return
	}
	ctx := &fasthttp.RequestCtx{}
	ctx.Request = *fasthttp.AcquireRequest()
	ctx.Request.SetRequestURI("http://127.0.0.1/?x=y")
	defer fasthttp.ReleaseRequest(&ctx.Request)
	if attr, found := s.Match(ctx); found != true || attr != "y" {
		t.Error("GET attribute not found")
	}
	ctx.Request.SetRequestURI("http://127.0.0.1/?x=a")
	if _, found := s.Match(ctx); found == true {
		t.Error("Found non existent attribute")
	}
}
