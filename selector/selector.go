package selector

import (
	"errors"
	"log"
	"net"
	"regexp"
	"strings"

	"github.com/valyala/fasthttp"
)

type Selector struct {
	RequestAttr string
	SubAttr     string
	Regexp      *regexp.Regexp
	Negate      bool
}

const NSLOOKUP_PREFIX = "nslookup("
const NSLOOKUP_SUFFIX = ")"

func ParseLookup(hostname string) (string, error) {
	addrs, err := net.LookupHost(hostname)
	if err != nil {
		return "", err
	}
	return strings.Join(addrs, "|"), nil
}

func ParseExpr(str string) (string, error) {
	if len(str) > len(NSLOOKUP_PREFIX)+len(NSLOOKUP_SUFFIX) &&
		str[0:len(NSLOOKUP_PREFIX)] == NSLOOKUP_PREFIX &&
		str[len(str)-len(NSLOOKUP_SUFFIX):] == NSLOOKUP_SUFFIX {
		return ParseLookup(str[len(NSLOOKUP_PREFIX) : len(str)-len(NSLOOKUP_SUFFIX)])
	}
	return str, nil
}

func Parse(str string) (*Selector, error) {
	// TODO proper parsing
	var reqAttr string
	var subAttr string
	var re *regexp.Regexp
	startPos := 0
	endPos := len(str)
	negate := false
	if str[0] == '!' {
		negate = true
		startPos = 1
	}
	if idx := strings.IndexRune(str, '='); idx != -1 {
		val, err := ParseExpr(str[idx+1:])
		if err != nil {
			return nil, errors.New("invalid expression")
		}
		re, err = regexp.Compile(val)
		if err != nil {
			return nil, errors.New("invalid regexp")
		}
		endPos = idx
	}

	if idx := strings.IndexRune(str, ':'); idx >= startPos {
		reqAttr = str[startPos:idx]
		subAttr = str[idx+1 : endPos]
	} else {
		reqAttr = str[startPos:endPos]
	}
	if reqAttr == "" {
		return nil, errors.New("missing request attribute")
	}
	return &Selector{reqAttr, subAttr, re, negate}, nil
}

func (s *Selector) Match(ctx *fasthttp.RequestCtx) (string, bool) {
	var matchSlice []byte
	found := false
	switch s.RequestAttr {
	case "IP":
		matchSlice = []byte(ctx.RemoteIP().String())
	case "Method":
		matchSlice = ctx.Method()
	case "Path":
		matchSlice = ctx.Path()
	case "Host":
		matchSlice = ctx.Host()
	case "POST":
		matchSlice = ctx.PostArgs().Peek(s.SubAttr)
	case "GET":
		matchSlice = ctx.QueryArgs().Peek(s.SubAttr)
	case "Param":
		matchSlice = ctx.PostArgs().Peek(s.SubAttr)
		if matchSlice == nil {
			matchSlice = ctx.QueryArgs().Peek(s.SubAttr)
		}
	case "Header":
		matchSlice = ctx.Request.Header.Peek(s.SubAttr)
	default:
		log.Println("unknown request attribute:", s.RequestAttr)
	}
	if matchSlice != nil && (s.Regexp == nil || s.Regexp.Match(matchSlice)) {
		found = true
	}
	if s.Negate {
		found = !found
	}
	return string(matchSlice), found
}
