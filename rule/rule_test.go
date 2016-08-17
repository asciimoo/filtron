package rule

import (
	"testing"

	"github.com/valyala/fasthttp"

	"github.com/asciimoo/filtron/types"
)

func TestNew(t *testing.T) {
	r, err := New("test rule", 10, 2, []string{"GET:a=b", "IP"})
	if err != nil {
		t.Error("Cannot create rule:", err)
		return
	}
	if len(r.Filters) != 2 {
		t.Error("Invalid length of filters:", len(r.Filters))
	}
}

func dummyRequestCtx() *fasthttp.RequestCtx {
	var ctx fasthttp.RequestCtx
	var req fasthttp.Request
	req.SetRequestURI("http://127.0.0.1/")
	ctx.Init(&req, nil, nil)
	return &ctx
}

func TestRuleStop(t *testing.T) {
	r1, _ := New("test rule", 10, 0, []string{"Path"})
	r2, _ := New("test rule 2", 10, 0, []string{"Path"})
	r3, _ := New("test rule 2", 10, 0, []string{"Path", "Path=nonmatching"})
	r3.Stop = true
	rules := []*Rule{r3, r1, r2}
	respState := types.UNTOUCHED
	validateRuleList(&rules, &respState, dummyRequestCtx())
	if r1.MatchCount != 1 || r2.MatchCount != 1 {
		t.Error("Expected MatchCount is 1, 1 - got:", r1.MatchCount, ",", r2.MatchCount)
	}
	r1.Stop = true
	validateRuleList(&rules, &respState, dummyRequestCtx())
	if r1.MatchCount != 2 || r2.MatchCount != 1 {
		t.Error("Expected MatchCount is 2, 1 - got:", r1.MatchCount, ",", r2.MatchCount)
	}
	if r1.MatchCount != 2 || r2.MatchCount != 1 {
		t.Error("Expected MatchCount is 2, 1 - got:", r1.MatchCount, ",", r2.MatchCount)
	}
	if r3.MatchCount != 0 {
		t.Error("Nonmatching rule unexpectedly matched")
	}
}
