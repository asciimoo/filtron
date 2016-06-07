package action

import (
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/valyala/fasthttp"

	"github.com/asciimoo/filtron/types"
)

type Action interface {
	Act(string, *fasthttp.RequestCtx) error
	SetParams(map[string]string) error
	GetResponseState() types.ResponseState
}

type ActionJSON struct {
	Name   string            `json:"name"`
	Params map[string]string `json:"params"`
}

func FromJSON(j ActionJSON) (Action, error) {
	return Create(j.Name, j.Params)
}

func Create(name string, params map[string]string) (Action, error) {
	var a Action
	var e error
	switch name {
	case "log":
		a = &logAction{}
	case "block":
		a = &blockAction{}
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

func (l *logAction) SetParams(params map[string]string) error {
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

func (b *blockAction) SetParams(params map[string]string) error {
	if message, found := params["message"]; found {
		b.message = []byte(message)
	} else {
		b.message = []byte("Blocked")
	}
	return nil
}
