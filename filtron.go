package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/asciimoo/filtron/api"
	"github.com/asciimoo/filtron/proxy"
	"github.com/asciimoo/filtron/rule"
)

var target *string
var transport http.Transport

func main() {
	target = flag.String("target", "http://127.0.0.1:8888/", "Target URL for reverse proxy")
	listen := flag.String("listen", "127.0.0.1:4004", "Proxy listen address")
	apiAddr := flag.String("api", "127.0.0.1:4005", "API listen address")
	ruleFile := flag.String("rules", "rules.json", "JSON rule list")
	flag.Parse()
	rules, err := rule.ParseJSON(*ruleFile)
	if err != nil {
		log.Fatal("Cannot parse rules:", err)
		return
	}
	log.Println(len(rules), "rules loaded from", *ruleFile)
	p := proxy.Listen(*listen, *target, &rules)
	api.Listen(*apiAddr, *ruleFile, p)
}
