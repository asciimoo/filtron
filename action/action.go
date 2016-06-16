package action

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/valyala/fasthttp"

	"github.com/asciimoo/filtron/selector"
	"github.com/asciimoo/filtron/types"
)

type ActionParams map[string]interface{}

type Action interface {
	Act(string, *fasthttp.RequestCtx) error
	SetParams(ActionParams) error
	GetResponseState() types.ResponseState
}

type ActionJSON struct {
	Name   string       `json:"name"`
	Params ActionParams `json:"params"`
}

func FromJSON(j ActionJSON) (Action, error) {
	return Create(j.Name, j.Params)
}

func Create(name string, params ActionParams) (Action, error) {
	var a Action
	var e error
	switch name {
	case "log":
		a = &logAction{}
	case "block":
		a = &blockAction{}
	case "shell":
		a = &shellAction{}
	}
	if a != nil {
		e = a.SetParams(params)
	} else {
		e = errors.New(fmt.Sprintf("Unknown action: %v", name))
	}
	return a, e
}

type logAction struct {
	destination io.Writer
}

func (l *logAction) Act(ruleName string, ctx *fasthttp.RequestCtx) error {
	_, err := fmt.Fprintf(
		l.destination,
		"[%v] %v %s %s %s%s \"%s\" \"%s\"\n",
		ruleName,
		time.Now().Format("2006-01-02 15:04:05.000"),
		ctx.Request.Header.Peek("X-Forwarded-For"),
		ctx.Method(),
		ctx.Host(),
		ctx.RequestURI(),
		ctx.PostBody(),
		ctx.Request.Header.UserAgent(),
	)
	return err
}

func (_ *logAction) GetResponseState() types.ResponseState {
	return types.UNTOUCHED
}

func (l *logAction) SetParams(params ActionParams) error {
	if _, found := params["destination"]; found {
		// TODO support destinations
		l.destination = os.Stderr
	} else {
		l.destination = os.Stderr
	}
	return nil
}

type blockAction struct {
	message []byte
}

func (b *blockAction) Act(_ string, ctx *fasthttp.RequestCtx) error {
	ctx.SetStatusCode(429)
	ctx.Write(b.message)
	return nil
}

func (_ *blockAction) GetResponseState() types.ResponseState {
	return types.SERVED
}

func (b *blockAction) SetParams(params ActionParams) error {
	if val, found := params["message"]; found {
		message, found := val.(string)
		if !found {
			return errors.New("String type expected as block action message param")
		}
		b.message = []byte(message)
	} else {
		b.message = []byte("Blocked")
	}
	return nil
}

type shellAction struct {
	cmd  string
	args []*selector.Selector
}

func (s *shellAction) Act(_ string, ctx *fasthttp.RequestCtx) error {
	args := make([]interface{}, 0, len(s.args))
	for _, sel := range s.args {
		m, found := sel.Match(ctx)
		if found {
			args = append(args, m)
		}
	}
	rawCmd := fmt.Sprintf(s.cmd, args...)
	log.Println("[shell action] running command:", rawCmd)
	parts := strings.Fields(rawCmd)
	cmd := exec.Command(parts[0], parts[1:len(parts)]...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd.Run()
}

func (_ *shellAction) GetResponseState() types.ResponseState {
	return types.UNTOUCHED
}

func (s *shellAction) SetParams(params ActionParams) error {
	if val, found := params["cmd"]; found {
		cmd, found := val.(string)
		if !found {
			return errors.New("String type expected as shell action cmd param")
		}
		s.cmd = cmd
	} else {
		return errors.New("Missing \"cmd\" argument in shell action")
	}
	if val, found := params["args"]; found {
		args, found := val.([]interface{})
		if !found {
			return errors.New("Array of selector strings expected as shell action args")
		}
		s.args = make([]*selector.Selector, 0, len(args))
		for _, val := range args {
			arg, found := val.(string)
			if !found {
				return errors.New("Selector string expected as shell argument")
			}
			sel, err := selector.Parse(arg)
			if err != nil {
				return errors.New("Invalid selector in shell argument")
			}
			s.args = append(s.args, sel)
		}
	}
	return nil
}
