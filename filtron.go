package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/asciimoo/filtron/api"
	"github.com/asciimoo/filtron/proxy"
	"github.com/asciimoo/filtron/rule"
)

const VERSION string = "0.1.0"

func main() {
	target := flag.String("target", "127.0.0.1:8888", "Target address for reverse proxy")
	listen := flag.String("listen", "127.0.0.1:4004", "Proxy listen address")
	apiAddr := flag.String("api", "127.0.0.1:4005", "API listen address")
	ruleFile := flag.String("rules", "rules.json", "JSON rule list")
	readBufferSize := flag.Int("read-buffer-size", 16*1024, "Read buffer size")
	printVersionInfo := flag.Bool("version", false, "Version information")
	flag.Parse()

	if *printVersionInfo {
		fmt.Printf("Filtron v%s\n", VERSION)
		return
	}

	rules, err := rule.ParseJSONFile(*ruleFile)
	if err != nil {
		log.Fatal("Cannot parse rules: ", err)
		return
	}
	log.Println(rule.RulesLength(rules), "rules loaded from", *ruleFile)
	p := proxy.Listen(*listen, *target, *readBufferSize, &rules)
	api.Listen(*apiAddr, *ruleFile, p)
}
