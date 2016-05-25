package selector

import (
	"errors"
	"log"
	"net/http"
	"regexp"
	"strings"
)

type Selector struct {
	RequestAttr string
	SubAttr     string
	Regexp      *regexp.Regexp
	Negate      bool
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
		var err error
		re, err = regexp.Compile(str[idx+1:])
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

func (s *Selector) Match(r *http.Request) (string, bool) {
	var matchingStr *string
	found := false
	switch s.RequestAttr {
	case "IP":
		matchingStr = &r.RemoteAddr
	case "Path":
		matchingStr = &r.URL.Path
	case "Host":
		matchingStr = &r.Host
	case "POST":
		if data := r.PostForm.Get(s.SubAttr); data != "" {
			matchingStr = &data
		}
	case "GET":
		if data := r.URL.Query().Get(s.SubAttr); data != "" {
			matchingStr = &data
		}
	case "Param":
		if data := r.Form.Get(s.SubAttr); data != "" {
			matchingStr = &data
		}
	case "Header":
		h := r.Header.Get(s.SubAttr)
		if h != "" {
			matchingStr = &h
		}
	default:
		log.Println("unknown request attribute:", s.RequestAttr)
	}
	if matchingStr != nil && (s.Regexp == nil || s.Regexp.MatchString(*matchingStr)) {
		found = true
	}
	if s.Negate {
		found = !found
	}
	if matchingStr == nil {
		return "", found
	}
	return *matchingStr, found
}
