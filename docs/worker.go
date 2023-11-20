//go:build wasm

package main

import (
	"bytes"
	"strings"
	"syscall/js"

	wikitable "github.com/movsb/goldmark-wiki-table/wiki-table"
)

func parse(this js.Value, args []js.Value) interface{} {
	text := args[0].String()
	table, err := wikitable.Parse(strings.NewReader(text))
	if err != nil {
		return js.ValueOf(err.Error())
	}
	buf := new(bytes.Buffer)
	table.Html(buf)
	return js.ValueOf(buf.String())
}

func registerCallbacks() {
	js.Global().Set("parse", js.FuncOf(parse))
}

func main() {
	registerCallbacks()
	select {}
}
