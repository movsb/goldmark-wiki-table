# goldmark-wiki-table

An experimental Wikipedia/MediaWiki table parser and renderer written in golang.

[Try it in your browser](https://movsb.github.io/goldmark-wiki-table/)

## Why?

The GitHub Flavored Markdown Table syntax sometimes cannot express complex tables, or tables with styles, or tables with colspans and rowspans.
This parser/goldmark extension enables you depicting tables using Wiki Tables syntax.

## How to use it in Goldmark?

```go
import wikitable "github.com/movsb/goldmark-wiki-table"

md := goldmark.New(
	goldmark.WithExtensions(wikitable.New()),
)
```

See [test/main.go](test/main.go) for more details.

## Examples

Write them directly in Markdown as paragraph, not in code blocks.

Note: Some styles may not be shown by GitHub.

### minimal syntax

```wiki-table
{|
|Orange
|Apple
|-
|Bread
|Pie
|-
|Butter
|Ice cream
|}
```

<table>
<tr><td>Orange</td><td>Apple</td></tr>
<tr><td>Bread</td><td>Pie</td></tr>
<tr><td>Butter</td><td>Ice cream</td></tr>
</table>

### With styles

```
{| class="wikitable" style="color:green; background-color:#ffffcc;" cellpadding="10"
|Orange
|Apple
|-
|Bread
|Pie
|-
|Butter
|Ice cream
|}
```

<table class="wikitable" style="color:green; background-color:#ffffcc;" cellpadding="10">
<tr><td>Orange</td><td>Apple</td></tr>
<tr><td>Bread</td><td>Pie</td></tr>
<tr><td>Butter</td><td>Ice cream</td></tr>
</table>

### with spans

```
{| class="wikitable"
!colspan="6"|Shopping List
|-
|rowspan="2"|Bread & Butter
|Pie
|Buns
|Danish
|colspan="2"|Croissant
|-
|Cheese
|colspan="2"|Ice cream
|Butter
|Yogurt
|}
```

<table class="wikitable">
<tr><th colspan="6">Shopping List</th></tr>
<tr><td rowspan="2">Bread &amp; Butter</td><td>Pie</td><td>Buns</td><td>Danish</td><td colspan="2">Croissant</td></tr>
<tr><td>Cheese</td><td colspan="2">Ice cream</td><td>Butter</td><td>Yogurt</td></tr>
</table>

### with caption

```
{| class="wikitable"
|+ style="caption-side:bottom; color:#e76700;"|Food complements
|-
! style="color:green" | Fruits
! style="color:red" | Fats
|-
|Orange
|Butter
|-
|Pear
|Pie
|-
|Apple
|Ice cream
|}
```

<table class="wikitable">
<caption style="caption-side:bottom; color:#e76700;">Food complements</caption>
<tr><th style="color:green">Fruits</th><th style="color:red">Fats</th></tr>
<tr><td>Orange</td><td>Butter</td></tr>
<tr><td>Pear</td><td>Pie</td></tr>
<tr><td>Apple</td><td>Ice cream</td></tr>
</table>

### Table in a table

```
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
```

<table>
<caption>An example table</caption>
<tr><th>First header</th><th colspan="2">Second header</th></tr>
<tr><td>Upper left</td><td>Upper middle</td><td rowspan="2">Right side</td></tr>
<tr><td>Lower left</td><td>Lower middle</td></tr>
<tr><td colspan="3">Text before a nested table...<table>
<caption>A table in a table</caption>
<tr><td>AAA</td><td>BBB</td></tr>
</table>
</td></tr>
</table>

## References

- [Help:Tables - MediaWiki](https://www.mediawiki.org/wiki/Help:Tables)
- [Help:Table - Wikipedia](https://en.wikipedia.org/wiki/Help:Table#Minimalist_table)

## License

MIT
