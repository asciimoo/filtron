package action

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/asciimoo/filtron/types"
)

type Action interface {
	Act(string, *http.Request, http.ResponseWriter) error
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

func (l *logAction) Act(ruleName string, r *http.Request, w http.ResponseWriter) error {
	_, err := fmt.Fprintf(
		l.destination,
		"%v [%v] %v %v%v \"%v\" \"%v\"\n",
		time.Now(),
		ruleName,
		r.Method,
		r.Host,
		r.URL.String(),
		r.PostForm.Encode(),
		r.Header.Get("User-Agent"),
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
		return nil
	}
	return errors.New("Missing destination parameter")
}

type blockAction struct {
	message []byte
}

func (b *blockAction) Act(_ string, r *http.Request, w http.ResponseWriter) error {
	w.WriteHeader(429)
	w.Write(b.message)
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
