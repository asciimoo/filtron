package selector

import (
	"net/http"
	"testing"
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
	s, err := Parse("IP")
	if err != nil {
		t.Error(err)
		return
	}
	r := &http.Request{}
	addr := "127.0.0.1:42424"
	r.RemoteAddr = addr
	if ip, found := s.Match(r); found != true || ip != addr {
		t.Error("Client IP not found:", ip)
	}
}

func TestGETAttrMatch(t *testing.T) {
	s, err := Parse("GET:x=(y|z)")
	if err != nil {
		t.Error(err)
		return
	}
	r, _ := http.NewRequest("GET", "http://127.0.0.1/?x=y", nil)
	if attr, found := s.Match(r); found != true || attr != "y" {
		t.Error("GET attribute not found")
	}
	r, _ = http.NewRequest("GET", "http://127.0.0.1/?x=a", nil)
	if _, found := s.Match(r); found == true {
		t.Error("Found non existent attribute")
	}
}
