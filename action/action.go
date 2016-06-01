package action

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/asciimoo/filtron/types"
)

type Action interface {
	Act(*http.Request, http.ResponseWriter) error
	SetParams(map[string]string) error
	GetResponseState() types.ResponseState
}

type ActionJSON struct {
	Name   string            `json:"name"`
	Params map[string]string `json:"params"`
}

func Create(j ActionJSON) (Action, error) {
	var a Action
	var e error
	switch j.Name {
	case "log":
		a = &logAction{}
	case "block":
		a = &blockAction{}
	}
	if a != nil {
		e = a.SetParams(j.Params)
	} else {
		e = errors.New(fmt.Sprintf("Unknown action: %v", j.Name))
	}
	return a, e
}

type logAction struct {
	destination io.Writer
}

func (l *logAction) Act(r *http.Request, w http.ResponseWriter) error {
	_, err := fmt.Fprintf(l.destination, "[%v] %v %v\n", r.Method, r.URL.String(), r.PostForm.Encode())
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

func (b *blockAction) Act(r *http.Request, w http.ResponseWriter) error {
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
