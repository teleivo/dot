// Package printer prints dot ASTs formatted in the spirit of https://github.com/mvdan/gofumpt.
package printer

import (
	"io"

	"github.com/teleivo/dot"
	"github.com/teleivo/dot/ast"
	"github.com/teleivo/dot/layout"
	"github.com/teleivo/dot/token"
)

const (
	// maxColumn is the max number of runes after which lines are broken up into multiple lines. Not
	// every dot construct can be broken up though.
	maxColumn = 20
	// tabWidth represents the number of columns a tab takes up
	tabWidth = 2
)

// Printer formats DOT code.
type Printer struct {
	r       io.Reader // r reader to parse dot code from
	w       io.Writer // w writer to output formatted DOT code to
	row     int       // row is the current one-indexed row the printer is at i.e. how many newlines it has printed. 0 means nothing has been printed
	column  int       // column is the current one-indexed column in terms of runes the printer is at. A tab counts as [tabWidth] columns. 0 means no rune has been printed on the current row
	newline bool      // newline indicates a buffered newline that should be printed
}

func NewPrinter(r io.Reader, w io.Writer) *Printer {
	return &Printer{
		r: r,
		w: w,
	}
}

func (p *Printer) Print() error {
	ps, err := dot.NewParser(p.r)
	if err != nil {
		return err
	}

	g, err := ps.Parse()
	if err != nil {
		return err
	}

	doc := layout.NewDoc(maxColumn)
	err = p.layoutNode(doc, g)
	if err != nil {
		return err
	}
	// TODO add error handling in case fmt.Print fails?
	doc.Render(p.w)

	return nil
}

func (p *Printer) layoutNode(doc *layout.Doc, node ast.Node) error {
	switch n := node.(type) {
	case ast.Graph:
		return p.layoutGraph(doc, n)
	}
	return nil
}

func (p *Printer) layoutGraph(doc *layout.Doc, graph ast.Graph) error {
	// TODO create strict graph id in a group? so ideally on one line but if not break each onto
	// their own line? or at least the id?
	if graph.IsStrict() {
		doc.Text(token.Strict.String())
		doc.Space()
	}

	if graph.Directed {
		doc.Text(token.Digraph.String())
	} else {
		doc.Text(token.Graph.String())
	}
	doc.Space()

	if graph.ID != nil {
		p.layoutID(doc, *graph.ID)
		doc.Space()
	}

	doc.Text(token.LeftBrace.String())
	var err error
	doc.Group(func(f *layout.Doc) {
		// TODO wrap in indent block

		err = p.layoutStmts(doc, graph.Stmts)
		if err != nil {
			return
		}

		if len(graph.Stmts) > 0 {
			doc.SpaceIf(layout.Flat)
			doc.BreakIf(1, layout.Broken)
		}
		doc.Text(token.RightBrace.String())
	})

	return err
}

func (p *Printer) layoutStmts(doc *layout.Doc, stmts []ast.Stmt) error {
	for _, stmt := range stmts {
		err := p.layoutStmt(doc, stmt)
		if err != nil {
			return err
		}
	}
	return nil
}

// layoutID prints a DOT [identifier]. newlines without preceding '\' are not mentioned as legal but
// are supported by the DOT tooling. Such newlines are normalized to line continuations.
//
// [identifier:] https://graphviz.org/doc/info/lang.html#ids
func (p *Printer) layoutID(doc *layout.Doc, id ast.ID) {
	doc.Text(id.Literal)
}

func (p *Printer) layoutStmt(doc *layout.Doc, stmt ast.Stmt) error {
	var err error
	switch st := stmt.(type) {
	case *ast.NodeStmt:
		p.layoutNodeStmt(doc, st)
	case *ast.EdgeStmt:
		p.layoutEdgeStmt(doc, st)
	case *ast.AttrStmt:
		p.layoutAttrStmt(doc, st)
	case ast.Attribute:
		// p.printNewline()
		p.layoutAttribute(doc, st)
	case ast.Subgraph:
		// p.printNewline()
		err = p.layoutSubgraph(doc, st)
	}
	return err
}

func (p *Printer) layoutNodeStmt(doc *layout.Doc, nodeStmt *ast.NodeStmt) {
	// TODO why does this not create a newline?
	doc.Break(1)
	doc.Group(func(d *layout.Doc) {
		p.printNodeID(doc, nodeStmt.NodeID)
		p.layoutAttrList(doc, nodeStmt.AttrList)
	})
}

func (p *Printer) printNodeID(doc *layout.Doc, nodeID ast.NodeID) {
	p.layoutID(doc, nodeID.ID)

	if nodeID.Port == nil {
		return
	}

	if nodeID.Port.Name != nil {
		// p.printToken(token.Colon, withColumnOffset(nodeID.Port.Name.StartPos, -1))
		p.layoutID(doc, *nodeID.Port.Name)
	}
	if nodeID.Port.CompassPoint != nil && nodeID.Port.CompassPoint.Type != ast.CompassPointUnderscore {
		// p.printToken(token.Colon, withColumnOffset(nodeID.Port.CompassPoint.StartPos, -1))
		// p.print(nodeID.Port.CompassPoint)
	}
}

func (p *Printer) layoutAttrList(doc *layout.Doc, attrList *ast.AttrList) {
	if attrList == nil {
		return
	}

	doc.Text(token.LeftBracket.String())
	doc.Space()
	// TODO indent block
	// p.increaseIndentation()

	for cur := attrList; cur != nil; cur = cur.Next {
		p.layoutAList(doc, cur.AList)
	}

	// p.decreaseIndentation()
	doc.BreakIf(1, layout.Broken)

	// TODO if I remember correctly I am merging A [color=blue] [style=filled] into A [color=blue,
	// style=filled]. How does me taking out '[]' affect printing of comments? Add to the test case.
	doc.Text(token.RightBracket.String())
}

func (p *Printer) layoutAList(doc *layout.Doc, aList *ast.AList) {
	for cur := aList; cur != nil; cur = cur.Next {
		doc.BreakIf(1, layout.Broken)
		p.layoutAttribute(doc, cur.Attribute)
		// TODO implement delayed printing in Render to prevent trailing whitespace
		doc.SpaceIf(layout.Flat)
	}
}

// hasMultipleAttributes traverses the AttrLists and ALists counting up to two ALists. This can be
// used to omit empty brackets or split attributes onto multiple lines.
func hasMultipleAttributes(attrList *ast.AttrList) (int, bool) {
	if attrList == nil {
		return 0, false
	}

	var cnt int
	for cur := attrList; cur != nil; cur = cur.Next {
		for curAList := cur.AList; curAList != nil; curAList = curAList.Next {
			cnt++
			if cnt > 1 {
				return cnt, true
			}
		}
	}

	return cnt, false
}

func (p *Printer) layoutEdgeStmt(doc *layout.Doc, edgeStmt *ast.EdgeStmt) {
	// // p.printNewline()

	err := p.printEdgeOperand(doc, edgeStmt.Left)
	if err != nil {
		return
	}

	// // p.printSpace()
	if edgeStmt.Right.Directed {
		// // p.printToken(token.DirectedEgde, edgeStmt.Right.StartPos)
	} else {
		// // p.printToken(token.UndirectedEgde, edgeStmt.Right.StartPos)
	}

	// // p.printSpace()
	err = p.printEdgeOperand(doc, edgeStmt.Right.Right)
	if err != nil {
		return
	}

	for cur := edgeStmt.Right.Next; cur != nil; cur = cur.Next {
		// // p.printSpace()
		if edgeStmt.Right.Directed {
			// // p.printToken(token.DirectedEgde, cur.StartPos)
		} else {
			// // p.printToken(token.UndirectedEgde, cur.StartPos)
		}
		// // p.printSpace()
		p.printEdgeOperand(doc, cur.Right)
	}

	p.layoutAttrList(doc, edgeStmt.AttrList)
}

func (p *Printer) printEdgeOperand(doc *layout.Doc, edgeOperand ast.EdgeOperand) error {
	var err error
	switch op := edgeOperand.(type) {
	case ast.NodeID:
		p.printNodeID(doc, op)
	case ast.Subgraph:
		err = p.layoutSubgraph(doc, op)
	}
	return err
}

func (p *Printer) layoutAttrStmt(doc *layout.Doc, attrStmt *ast.AttrStmt) {
	cnt, _ := hasMultipleAttributes(&attrStmt.AttrList)
	if cnt == 0 {
		return
	}

	// // p.printNewline()
	p.layoutID(doc, attrStmt.ID)
	p.layoutAttrList(doc, &attrStmt.AttrList)
}

func (p *Printer) layoutAttribute(doc *layout.Doc, attribute ast.Attribute) {
	p.layoutID(doc, attribute.Name)
	// TODO fix this using the correct position of the '=' which I need to know the position of equal
	// to support a comment before it. Add the position info to the ast
	// // p.printToken(token.Equal, attribute.Name.EndPos)
	doc.Text(token.Equal.String())
	p.layoutID(doc, attribute.Value)
}

func (p *Printer) layoutSubgraph(doc *layout.Doc, subraph ast.Subgraph) error {
	// // p.printToken(token.Subgraph, subraph.Start())
	// // p.printSpace()
	if subraph.ID != nil {
		p.layoutID(doc, *subraph.ID)
		// // p.printSpace()
	}

	// // p.printToken(token.LeftBrace, subraph.LeftBrace)

	err := p.layoutStmts(doc, subraph.Stmts)
	if err != nil {
		return err
	}

	return nil
}
