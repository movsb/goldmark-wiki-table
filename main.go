package main

/* TODO:
- Pipe character as content. (needs escape <nowiki>)
- tbody? (not rendered)
- '' for italic, ''' for bold (markdown already has)
- Blank spaces at the beginning of a line are ignored. (not ignored)
- scope=col/row support (can use styles instead)
- {{sticky header}}
*/

import (
	"fmt"
	"html"
	"io"
	"log"
	"os"
	"strings"

	"gopkg.in/yaml.v2"
)

type Test struct {
	Wiki        string `yaml:"wiki"`
	Html        string `yaml:"html"`
	Description string `yaml:"description"`
}

type ByteReader struct {
	b []byte
	p int
}

func NewByteReader(s string) *ByteReader {
	return &ByteReader{
		b: []byte(s),
		p: 0,
	}
}

func (r *ByteReader) ReadByte() (byte, error) {
	if r.p >= len(r.b) {
		return 0, io.EOF
	}
	b := r.b[r.p]
	r.p++
	return b, nil
}

func (r *ByteReader) peekAt(n int) (byte, error) {
	if r.p+n >= len(r.b) {
		return 0, io.EOF
	}
	return r.b[r.p+n], nil
}

func (r *ByteReader) GetPos() int {
	return r.p
}

func (r *ByteReader) SetPos(pos int) {
	r.p = pos
}

func (r *ByteReader) PutBackOneByte() {
	r.p--
}

type Parser struct {
	r *ByteReader
}

func (p *Parser) fatal(err error) {
	panic(err)
}

func (p *Parser) expect(rs ...byte) {
	for _, r := range rs {
		c, err := p.r.ReadByte()
		if err != nil {
			p.fatal(err)
		}
		if c != r {
			p.fatal(fmt.Errorf(`unexpected %c: expected %c`, c, r))
		}
	}
}

func (p *Parser) skip(rs ...byte) bool {
	pp := p.r.GetPos()
	restore := true
	defer func() {
		if restore {
			p.r.SetPos(pp)
		}
	}()
	for _, r := range rs {
		c, err := p.r.ReadByte()
		if err != nil {
			return false
		}
		if c != r {
			return false
		}
	}
	restore = false
	return true
}

func (p *Parser) parseCellData() []any {
	parseText := func() (string, bool) {
		b := []byte{}
		for {
			c, err := p.r.ReadByte()
			if err != nil {
				p.fatal(err)
			}
			if c == '|' {
				p.r.PutBackOneByte()
				break
			} else if c == '\n' {
				x, err := p.r.ReadByte()
				if err == nil && x != '|' && x != '!' && x != '{' {
					p.r.PutBackOneByte()
					b = append(b, ' ')
					continue
				}
				p.r.PutBackOneByte()
				p.r.PutBackOneByte()
				break
			} else {
				b = append(b, c)
			}
		}
		return string(b), len(b) > 0
	}
	parseTable := func() (*Table, bool) {
		if p.peekAt(0) == '\n' && p.peekAt(1) == '{' && p.peekAt(2) == '|' {
			p.advance(1)
			return p.parseTable(""), true
		}
		return nil, false
	}

	data := make([]any, 0)

	for {
		if t, ok := parseText(); ok {
			data = append(data, t)
			continue
		}
		if t, ok := parseTable(); ok {
			data = append(data, t)
			continue
		}
		break
	}

	return data
}

func (p *Parser) peek() byte {
	return p.peekAt(0)
}

func (p *Parser) peekAt(n int) byte {
	b, err := p.r.peekAt(n)
	if err != nil {
		return 0xFF
	}
	return b
}

func (p *Parser) advance(n int) {
	p.r.p++
}

func (p *Parser) skipSpaces() {
	for p.peek() == ' ' {
		p.advance(1)
	}
}

type Attributes struct {
	pairs []string
}

func (a *Attributes) Count() int {
	return len(a.pairs)
}

func (a *Attributes) Add(name, value string) {
	a.pairs = append(a.pairs, name, value)
}

func (a *Attributes) String() string {
	if len(a.pairs) == 0 {
		return ""
	}
	buf := strings.Builder{}
	for i := 0; i < len(a.pairs)/2; i++ {
		if i > 0 {
			fmt.Fprint(&buf, " ")
		}
		// TODO: key is not escaped (no need?)
		fmt.Fprintf(
			&buf, `%s="%s"`,
			a.pairs[i*2+0],
			html.EscapeString(a.pairs[i*2+1]),
		)
	}
	return buf.String()
}

func (p *Parser) parseAttributes() *Attributes {
	pp := p.r.GetPos()

	attrs := &Attributes{}

	for {
		p.skipSpaces()
		name := []byte{}
		c := byte(0)
		for {
			c = p.peek()
			if c == 0xFF || c == '|' || c == '\n' {
				// no attrs, may be data.
				if attrs.Count() == 0 {
					p.r.SetPos(pp)
				}
				return attrs
			} else if c == '=' || c == ' ' {
				break
			} else {
				name = append(name, c)
				p.advance(1)
			}
		}
		if string(name) == "" {
			p.r.SetPos(pp)
			return attrs
		}
		p.skipSpaces()
		if p.peek() != '=' {
			p.r.SetPos(pp)
			return attrs
		}

		p.advance(1)

		quoted := false
		if p.peek() == '"' {
			quoted = true
			p.advance(1)
		}

		value := []byte{}
		done := false
		for {
			c = p.peek()
			if c == '"' && quoted {
				p.advance(1)
				break
			}
			if c == ' ' && !quoted {
				p.advance(1)
				break
			}
			if c == 0xFF || c == '|' || c == '\n' {
				done = true
				break
			}
			value = append(value, c)
			p.advance(1)
		}

		if string(value) == "" && !quoted {
			p.r.SetPos(pp)
			return attrs
		}

		attrs.Add(string(name), string(value))

		if done {
			break
		}
	}

	return attrs
}

type _Base struct {
	Name             string
	Attributes       *Attributes
	openWithNewline  bool
	closeWithNewline bool
}

func (e *_Base) Open(w io.Writer) {
	fmt.Fprint(w, `<`, e.Name)
	if e.Attributes != nil && e.Attributes.Count() > 0 {
		fmt.Fprint(w, ` `, e.Attributes)
	}
	fmt.Fprint(w, `>`)
	if e.openWithNewline {
		fmt.Fprintln(w)
	}
}

func (e *_Base) Close(w io.Writer) {
	fmt.Fprint(w, `</`, e.Name, `>`)
	if e.closeWithNewline {
		fmt.Fprintln(w)
	}
}

type Table struct {
	_Base
	Caption *Caption
	Rows    []*Row
}

func NewTable() *Table {
	return &Table{
		_Base: _Base{
			Name:             `table`,
			openWithNewline:  true,
			closeWithNewline: true,
		},
	}
}

func (t *Table) AddRow(row *Row) {
	t.Rows = append(t.Rows, row)
}

func (t *Table) Html(w io.Writer) {
	t.Open(w)
	if t.Caption != nil {
		t.Caption.Open(w)
		t.Caption.Data(w)
		t.Caption.Close(w)
	}
	for _, tr := range t.Rows {
		tr.Open(w)
		tr.Data(w)
		tr.Close(w)
	}
	t.Close(w)
}

type Caption struct {
	_Base
	Text string
}

func NewCaption(caption string) *Caption {
	return &Caption{
		_Base: _Base{
			Name:             `caption`,
			openWithNewline:  false,
			closeWithNewline: true,
		},
		Text: caption,
	}
}

func (c *Caption) Data(w io.Writer) {
	t := strings.TrimSpace(keepTags.Replace(c.Text))
	fmt.Fprint(w, t)
}

type Row struct {
	_Base
	Cells []*Cell
}

func NewRow() *Row {
	return &Row{_Base: _Base{Name: `tr`, closeWithNewline: true}}
}

func (r *Row) AddCell(cell *Cell) {
	r.Cells = append(r.Cells, cell)
}

func (r *Row) Data(w io.Writer) {
	for _, cell := range r.Cells {
		cell.Open(w)
		cell.Data(w)
		cell.Close(w)
	}
}

type Cell struct {
	_Base
	IsHeader bool
	Items    []any // Texts and Tables
	// Text     string
	// Table    *Table // exclusive with Text
}

func NewCell(header bool, items ...any) *Cell {
	name := `th`
	if !header {
		name = `td`
	}
	return &Cell{_Base: _Base{Name: name}, Items: items}
}

var keepTags = strings.NewReplacer("&", "&amp;")

func (c *Cell) Data(w io.Writer) {
	for _, item := range c.Items {
		switch item := item.(type) {
		case string:
			item = strings.TrimSpace(keepTags.Replace(item))
			fmt.Fprint(w, item)
		case *Table:
			item.Html(w)
		default:
			panic("unknown cell type")
		}
	}
}

func (p *Parser) parseCell(header bool) *Cell {
	attrs := p.parseAttributes()
	if attrs.Count() > 0 {
		p.skip('|')
	}
	data := p.parseCellData()
	cell := NewCell(header, data...)
	cell.Attributes = attrs
	return cell
}

func (p *Parser) parseTable(w string) *Table {
	if w != "" {
		p.r = NewByteReader(w)
	}
	p.expect('{', '|')
	m := p.parseAttributes()
	table := NewTable()
	table.Attributes = m
	p.skipSpaces()
	p.expect('\n')

	tr := (*Row)(nil)
	for {
		if p.skip('|') {
			if p.skip('}') {
				p.skip('\n')
				break
			} else if p.skip('+') {
				attrs := p.parseAttributes()
				if attrs.Count() > 0 {
					p.skip('|')
				}
				data := p.parseCellData()
				if len(data) == 1 {
					if text, ok := data[0].(string); ok {
						caption := NewCaption(text)
						caption.Attributes = attrs
						table.Caption = caption
						p.expect('\n')
						continue
					}
				}
				panic("wrong caption")
			} else if p.skip('-') {
				p.expect('\n')
				tr = nil
				continue
			} else {
				if tr == nil {
					tr = NewRow()
					table.AddRow(tr)
				}
				cell := p.parseCell(false)
				tr.AddCell(cell)
				if p.peek() == '\n' {
					p.advance(1)
				} else if p.peekAt(0) == '|' && p.peekAt(1) == '|' {
					p.advance(1)
				}
			}
		} else if p.skip('!') {
			if tr == nil {
				tr = NewRow()
				table.AddRow(tr)
			}
			cell := p.parseCell(true)
			tr.AddCell(cell)
			if p.peek() == '\n' {
				p.advance(1)
			} else if p.peekAt(0) == '|' && p.peekAt(1) == '|' {
				p.advance(1)
			}
		} else {
			b, err := p.r.ReadByte()
			p.fatal(fmt.Errorf(`unexpected: %c, err: %v`, b, err))
		}
	}

	return table
}

func main() {
	fp, err := os.Open(`test.yaml`)
	if err != nil {
		panic(err)
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
		p := &Parser{}
		if i == 8 {
			log.Println("stop here")
		}
		table := p.parseTable(test.Wiki)
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
