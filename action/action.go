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
	Act(*http.Request, http.ResponseWriter) (types.ResponseState, error)
	SetParams(map[string]string) error
}

type ActionJSON struct {
	Name   string            `json:"name"`
	Params map[string]string `json:"params"`
}

func Create(j ActionJSON) (Action, error) {
	switch j.Name {
	case "log":
		var a Action
		a = &logAction{}
		if err := a.SetParams(j.Params); err != nil {
			return nil, err
		}
		return a, nil
	}
	return nil, errors.New("Unknown action: " + j.Name)
}

type logAction struct {
	destination io.Writer
}

func (l *logAction) SetParams(params map[string]string) error {
	if _, found := params["destination"]; found {
		// TODO support destinations
		l.destination = os.Stderr
		return nil
	}
	return errors.New("Missing destination parameter")
}

func (l *logAction) Act(r *http.Request, w http.ResponseWriter) (types.ResponseState, error) {
	_, err := fmt.Fprintf(l.destination, "[%v] %v %v\n", r.Method, r.URL.String(), r.PostForm.Encode())
	return types.UNTOUCHED, err
}
