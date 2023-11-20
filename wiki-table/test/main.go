package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	wikitable "github.com/movsb/goldmark-wiki-table/wiki-table"
	"gopkg.in/yaml.v2"
)

type Test struct {
	Wiki        string `yaml:"wiki"`
	Html        string `yaml:"html"`
	Description string `yaml:"description"`
}

func main() {
	fp, err := os.Open(`test.yaml`)
	if err != nil {
		fp, err = os.Open(`test/test.yaml`)
		if err != nil {
			panic(err)
		}
	}
	defer fp.Close()

	var file = struct {
		Tests []Test `yaml:"tests"`
	}{}
	yd := yaml.NewDecoder(fp)
	if err := yd.Decode(&file); err != nil {
		panic(err)
	}

	fmt.Println(`<style> table,th,td {border-collapse: collapse;} th,td { border: 1px solid red; padding: 8px; } table { margin: 1em; }</style>`)

	for i, test := range file.Tests {
		log.Println(`test:`, i, test.Description)
		if strings.TrimSpace(test.Wiki) == "" || strings.TrimSpace(test.Html) == "" {
			log.Fatalln("empty wiki or html")
		}
		if i == 1 {
			log.Println("stop here")
		}
		table, err := wikitable.Parse(strings.NewReader(test.Wiki))
		if err != nil {
			log.Fatalln(err)
		}
		buf := strings.Builder{}
		table.Html(&buf)
		html := buf.String()
		if html != test.Html {
			log.Println("+++not equal")
			log.Println(html)
			log.Println("===")
			log.Println(test.Html)
			log.Println("---not equal")
			log.Fatalln()
		}
		fmt.Println(html)
	}
}
