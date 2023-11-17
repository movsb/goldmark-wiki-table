package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"gopkg.in/yaml.v2"
)

type Test struct {
	Wiki string `yaml:"wiki"`
	Html string `yaml:"html"`
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
	r ByteReader
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

// "{|"
func (p *Parser) parseStart() {
	p.expect('{', '|')
}

func (p *Parser) parseCaptionStart() {
	p.expect('|', '+')
}

func (p *Parser) parseRowStart() {
	p.expect('|', '-')
}

func (p *Parser) parseHeaderStart() {
	p.expect('!')
}

func (p *Parser) parseDataStart() {
	p.expect('|')
}

func (p *Parser) parseRowData() string {
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
			if err == nil && x != '|' && x != '!' {
				p.r.PutBackOneByte()
				b = append(b, ' ')
				continue
			}
			p.r.PutBackOneByte()
			p.r.PutBackOneByte()
			break
		}
		b = append(b, c)
	}
	return strings.TrimSpace(string(b))
}

func (p *Parser) convert(w string) string {
	p.r = *NewByteReader(w)
	buf := &bytes.Buffer{}
	p.parseStart()
	buf.WriteString("<table>\n")
	p.expect('\n')

	trPut := false
	for {
		if p.skip('|') {
			if p.skip('}') {
				// end
				break
			} else if p.skip('+') {

			} else if p.skip('-') {
				p.expect('\n')
				buf.WriteString("</tr>\n")
				trPut = false
				continue
			} else {
				s := p.parseRowData()
				if !trPut {
					buf.WriteString(`<tr>`)
					trPut = true
				}
				buf.WriteString(fmt.Sprintf(`<td>%s</td>`, s))
				if p.skip('\n') {
					continue
				} else if p.skip('|') {
					continue
				}
			}
		} else if p.skip('!') {
			s := p.parseRowData()
			if !trPut {
				buf.WriteString(`<tr>`)
				trPut = true
			}
			buf.WriteString(fmt.Sprintf(`<th>%s</th>`, s))
			if p.skip('\n') {
				continue
			}
		} else {
			b, err := p.r.ReadByte()
			p.fatal(fmt.Errorf(`unexpected: %c, err: %v`, b, err))
		}
	}

	if trPut {
		trPut = false
		buf.WriteString("</tr>\n")
	}
	buf.WriteString("</table>\n")
	return buf.String()
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

	fmt.Println(`<style> table,th,td {border-collapse: collapse;} th,td { border: 1px solid red; } table { margin: 1em; }</style>`)

	for i, test := range file.Tests {
		log.Println(`test:`, i)
		p := &Parser{}
		h := p.convert(test.Wiki)
		if h != test.Html {
			log.Println("+++not equal")
			log.Println(h, "\n---\n", test.Html)
			p.fatal(fmt.Errorf("---not equal"))
		}
		fmt.Println(h)
	}
}
