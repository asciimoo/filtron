package api

import (
	"log"
	"net/http"

	"github.com/asciimoo/filtron/proxy"
)

type API struct {
	Proxy    *proxy.Proxy
	RuleFile string
}

func Listen(address, ruleFile string, p *proxy.Proxy) {
	log.Println("API listens on", address)
	a := &API{p, ruleFile}
	s := http.NewServeMux()
	s.HandleFunc("/", a.Handler)
	http.ListenAndServe(address, s)
}

func (a *API) Handler(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/reload_rules":
		if err := a.Proxy.ReloadRules(a.RuleFile); err != nil {
			w.Write([]byte(err.Error()))
		} else {
			log.Println("Rule file reloaded")
			w.Write([]byte("ok"))
		}
	default:
		http.NotFound(w, r)
	}
}
