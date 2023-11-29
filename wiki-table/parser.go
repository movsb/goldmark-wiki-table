package wikitable

/* TODO:
- Pipe character as content. (needs escape <nowiki>)
- tbody? (not rendered)
- '' for italic, ''' for bold (markdown already has)
- Blank spaces at the beginning of a line are ignored. (not ignored)
- scope=col/row support (can use styles instead)
- {{sticky header}} (wiki-specific)
*/

import (
	"errors"
	"fmt"
	"html"
	"io"
	"strings"
)

type Parser struct {
	b []byte
	i int
	j int
	m int
}

// maxProcessingSize: a simple naive buffer size to enabling seeking rewind.
func NewParserSize(maxProcessingSize int) *Parser {
	return &Parser{
		j: maxProcessingSize,
		m: maxProcessingSize,
	}
}

func Parse(r io.Reader) (*Table, error) {
	return NewParserSize(1 << 20).Parse(r)
}

func (p *Parser) mark() func() {
	i := p.i
	return func() { p.i = i }
}

func (p *Parser) fatal(err string) {
	panic(err)
}

func (p *Parser) remains() int {
	return p.j - p.i
}

func (p *Parser) read() byte {
	if p.remains() < 1 {
		panic("read eof")
	}
	b := p.b[p.i]
	p.i++
	return b
}

func (p *Parser) expect(rs ...byte) {
	if p.remains() < len(rs) {
		panic("eof expecting")
	}

	for _, r := range rs {
		if c := p.read(); c != r {
			p.fatal(fmt.Sprintf(`unexpected %c: expected %c`, c, r))
		}
	}
}

func (p *Parser) skip(rs ...byte) bool {
	if p.remains() < len(rs) {
		return false
	}

	restoreFn := p.mark()
	restore := true

	defer func() {
		if restore {
			restoreFn()
		}
	}()

	for _, r := range rs {
		if c := p.read(); c != r {
			return false
		}
	}

	restore = false
	return true
}

func (p *Parser) peek() byte {
	return p.peekAt(0)
}

func (p *Parser) peekAt(n int) byte {
	if p.remains() < n+1 {
		return 0xFF
	}
	return p.b[p.i+n]
}

func (p *Parser) advance(n int) {
	p.i += n
}

func (p *Parser) parseCellData() []any {
	parseText := func() (string, bool) {
		b := []byte{}
		for {
			c := p.peek()
			if c == '|' || c == '!' {
				break
			} else if c == '\n' {
				c2 := p.peekAt(1)
				if c2 != 0xFF && c2 != '|' && c2 != '!' && c2 != '{' {
					p.advance(1)
					b = append(b, ' ')
					continue
				}
				break
			} else if c == 0xFF {
				p.fatal(`unexpected eof`)
			} else {
				p.advance(1)
				b = append(b, c)
			}
		}
		return string(b), len(b) > 0
	}
	parseTable := func() (*Table, bool) {
		if p.peekAt(0) == '\n' && p.peekAt(1) == '{' && p.peekAt(2) == '|' {
			p.advance(1)
			return p.parseTable(), true
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
	restore := p.mark()

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
					restore()
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
			restore()
			return attrs
		}
		p.skipSpaces()
		if p.peek() != '=' {
			restore()
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
			restore()
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
	name             string
	attributes       *Attributes
	openWithNewline  bool
	closeWithNewline bool
}

func (e *_Base) Open(w io.Writer) {
	fmt.Fprint(w, `<`, e.name)
	if e.attributes != nil && e.attributes.Count() > 0 {
		fmt.Fprint(w, ` `, e.attributes)
	}
	fmt.Fprint(w, `>`)
	if e.openWithNewline {
		fmt.Fprintln(w)
	}
}

func (e *_Base) Close(w io.Writer) {
	fmt.Fprint(w, `</`, e.name, `>`)
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
			name:             `table`,
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
			name:             `caption`,
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
	return &Row{_Base: _Base{name: `tr`, closeWithNewline: true}}
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
}

func NewCell(header bool, items ...any) *Cell {
	name := `th`
	if !header {
		name = `td`
	}
	return &Cell{_Base: _Base{name: name}, Items: items}
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
	cell.attributes = attrs
	return cell
}

func (p *Parser) Parse(r io.Reader) (table *Table, err error) {
	b, err := io.ReadAll(io.LimitReader(r, int64(p.m)))
	if err != nil {
		return nil, err
	}
	p.i, p.j, p.b = 0, len(b), b

	t := [1]byte{}
	if _, err := r.Read(t[:]); err != io.EOF {
		return nil, errors.New(`data too large`)
	}

	defer func() {
		if err2 := recover(); err2 != nil {
			err = fmt.Errorf("error parsing wiki-table: %v", err2)
			// panic(err2) // for debug
		}
	}()

	return p.parseTable(), nil
}

func (p *Parser) parseTable() *Table {
	p.expect('{', '|')
	m := p.parseAttributes()
	table := NewTable()
	table.attributes = m
	p.skipSpaces()
	p.expect('\n')

	tr := (*Row)(nil)

	parseCell := func(header bool) {
		if tr == nil {
			tr = NewRow()
			table.AddRow(tr)
		}
		cell := p.parseCell(header)
		tr.AddCell(cell)
		if p.peek() == '\n' {
			p.advance(1)
		} else if p.peekAt(0) == '|' && p.peekAt(1) == '|' {
			p.advance(1)
		} else if p.peekAt(0) == '!' && p.peekAt(1) == '!' {
			p.advance(1)
		}
	}

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
						caption.attributes = attrs
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
				parseCell(false)
			}
		} else if p.skip('!') {
			parseCell(true)
		} else {
			b := p.read()
			p.fatal(fmt.Sprintf(`unexpected: %c`, b))
		}
	}

	return table
}
