package wikitable

import (
	"bytes"

	wikitable "github.com/movsb/goldmark-wiki-table/wiki-table"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

type WikiTable struct{}

func (t *WikiTable) Extend(md goldmark.Markdown) {
	md.Parser().AddOptions(
		parser.WithParagraphTransformers(
			util.Prioritized(_NewParser(), 0),
		),
	)
	md.Renderer().AddOptions(
		renderer.WithNodeRenderers(
			util.Prioritized(_NewRenderer(), 0),
		),
	)
}

var _wikiTable = &WikiTable{}

func New() *WikiTable {
	return _wikiTable
}

type _WikiTableParser struct{}

var _ parser.ParagraphTransformer = (*_WikiTableParser)(nil)

func _NewParser() *_WikiTableParser {
	return &_WikiTableParser{}
}

func (t *_WikiTableParser) Transform(node *ast.Paragraph, reader text.Reader, pc parser.Context) {
	source := t.getSource(node, reader, pc)
	if source == nil {
		return
	}
	table, err := wikitable.Parse(bytes.NewReader(source))
	if err != nil {
		return
	}
	t.transform(node, table)
}

func (t *_WikiTableParser) getSource(node *ast.Paragraph, reader text.Reader, pc parser.Context) []byte {
	values := []byte{}
	lines := node.Lines()
	line0 := lines.At(0)
	text := reader.Value(line0)
	if !bytes.Contains(text, []byte{'{', '|'}) {
		return nil
	}

	values = append(values, text...)

	// check if it has matching start and end.
	numBrackets := 1
	for i := 1; i < lines.Len(); i++ {
		line := lines.At(i)
		text := reader.Value(line)
		if bytes.Contains(text, []byte{'{', '|'}) {
			numBrackets++
		} else if bytes.Contains(text, []byte{'|', '}'}) {
			numBrackets--
		}
		values = append(values, text...)
	}

	if numBrackets != 0 {
		return nil
	}

	// extraneous char
	lastLine := reader.Value(lines.At(lines.Len() - 1))
	if !bytes.Contains(lastLine, []byte{'|', '}'}) {
		return nil
	}

	return values
}

func (t *_WikiTableParser) transform(node ast.Node, table *wikitable.Table) {
	tableNode := &_TableNode{
		table: table,
	}
	node.Parent().InsertAfter(node.Parent(), node, tableNode)
	node.Parent().RemoveChild(node.Parent(), node)
}

var (
	nodeKindTable = ast.NewNodeKind(`WikiTable`)
)

type _TableNode struct {
	ast.BaseBlock
	table *wikitable.Table
}

func (t *_TableNode) Kind() ast.NodeKind {
	return nodeKindTable
}

func (t *_TableNode) Dump(source []byte, level int) {
	ast.DumpHelper(t, source, level, nil, nil)
}

type _WikiTableRender struct{}

func _NewRenderer() *_WikiTableRender {
	return &_WikiTableRender{}
}

func (t *_WikiTableRender) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(nodeKindTable, t.renderTable)
}

func (t *_WikiTableRender) renderTable(w util.BufWriter, source []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	table := n.(*_TableNode).table
	if entering {
		table.Html(w)
	}
	return ast.WalkContinue, nil
}
