package main

import (
	"bytes"
	"fmt"

	wikitable "github.com/movsb/goldmark-wiki-table"
	"github.com/yuin/goldmark"
)

func main() {
	md := goldmark.New(
		goldmark.WithExtensions(wikitable.New()),
	)
	buf := &bytes.Buffer{}
	if err := md.Convert([]byte(`
# abc

{|
|+ An example table
|-
! First header
! colspan="2" | Second header
|-
| Upper left
| Upper middle
| rowspan="2" | Right side
|-
| Lower left
| Lower middle
|-
| colspan="3"| Text before a nested table...
{|
|+ A table in a table
| AAA
| BBB
|}
|}
`), buf); err != nil {
		panic(err)
	}
	fmt.Println(buf.String())
}
