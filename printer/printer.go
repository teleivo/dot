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
	p.layoutNode(doc, g)
	// TODO add error handling in case fmt.Print fails?
	doc.Render(p.w)

	return nil
}

func (p *Printer) layoutNode(doc *layout.Doc, node ast.Node) {
	switch n := node.(type) {
	case ast.Graph:
		p.layoutGraph(doc, n)
	}
}

func (p *Printer) layoutGraph(doc *layout.Doc, graph ast.Graph) {
	// TODO create strict graph id in a group? so ideally on one line but if not break each onto
	// their own line? or at least the id?
	if graph.IsStrict() {
		doc.Text(token.Strict.String()).
			Space()
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
	doc.Group(func(f *layout.Doc) {
		// TODO wrap in indent block
		p.layoutStmts(doc, graph.Stmts)

		if len(graph.Stmts) > 0 {
			doc.SpaceIf(layout.Flat).
				BreakIf(1, layout.Broken)
		}
		doc.Text(token.RightBrace.String())
	})
}

func (p *Printer) layoutStmts(doc *layout.Doc, stmts []ast.Stmt) {
	for _, stmt := range stmts {
		p.layoutStmt(doc, stmt)
	}
}

// layoutID prints a DOT [identifier]. newlines without preceding '\' are not mentioned as legal but
// are supported by the DOT tooling. Such newlines are normalized to line continuations.
//
// [identifier:] https://graphviz.org/doc/info/lang.html#ids
func (p *Printer) layoutID(doc *layout.Doc, id ast.ID) {
	doc.Text(id.Literal)
}

func (p *Printer) layoutStmt(doc *layout.Doc, stmt ast.Stmt) {
	// TODO indent here I think
	switch st := stmt.(type) {
	case *ast.NodeStmt:
		p.layoutNodeStmt(doc, st)
	case *ast.EdgeStmt:
		p.layoutEdgeStmt(doc, st)
	case *ast.AttrStmt:
		p.layoutAttrStmt(doc, st)
	case ast.Attribute:
		doc.Break(1)
		p.layoutAttribute(doc, st)
	case ast.Subgraph:
		doc.Break(1)
		p.layoutSubgraph(doc, st)
	}
}

func (p *Printer) layoutNodeStmt(doc *layout.Doc, nodeStmt *ast.NodeStmt) {
	doc.Break(1).
		Group(func(d *layout.Doc) {
			p.layoutNodeID(doc, nodeStmt.NodeID)
			doc.Space()
			p.layoutAttrList(doc, nodeStmt.AttrList)
		})
}

func (p *Printer) layoutNodeID(doc *layout.Doc, nodeID ast.NodeID) {
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
	// don't print empty []
	if attrList == nil {
		return
	}

	doc.Group(func(d *layout.Doc) {
		doc.Text(token.LeftBracket.String()).
			BreakIf(1, layout.Broken)
		// TODO indent block
		// p.increaseIndentation()
		for cur := attrList; cur != nil; cur = cur.Next {
			p.layoutAList(doc, cur.AList)
		}
		// p.decreaseIndentation()
		doc.BreakIf(1, layout.Broken).
			Text(token.RightBracket.String())
	})
}

func (p *Printer) layoutAList(doc *layout.Doc, aList *ast.AList) {
	for cur := aList; cur != nil; cur = cur.Next {
		p.layoutAttribute(doc, cur.Attribute)
		// TODO implement delayed printing in Render to prevent trailing whitespace
		if cur.Next != nil {
			doc.SpaceIf(layout.Flat)
			doc.BreakIf(1, layout.Broken)
		}
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
	doc.Break(1)
	p.layoutEdgeOperand(doc, edgeStmt.Left)
	doc.Space()

	if edgeStmt.Right.Directed {
		doc.Text(token.DirectedEgde.String())
	} else {
		doc.Text(token.UndirectedEgde.String())
	}
	doc.Space()

	p.layoutEdgeOperand(doc, edgeStmt.Right.Right)

	for cur := edgeStmt.Right.Next; cur != nil; cur = cur.Next {
		doc.Space()
		if edgeStmt.Right.Directed {
			doc.Text(token.DirectedEgde.String())
		} else {
			doc.Text(token.UndirectedEgde.String())
		}
		doc.Space()
		p.layoutEdgeOperand(doc, cur.Right)
	}

	p.layoutAttrList(doc, edgeStmt.AttrList)
}

func (p *Printer) layoutEdgeOperand(doc *layout.Doc, edgeOperand ast.EdgeOperand) {
	switch op := edgeOperand.(type) {
	case ast.NodeID:
		p.layoutNodeID(doc, op)
	case ast.Subgraph:
		p.layoutSubgraph(doc, op)
	}
}

func (p *Printer) layoutAttrStmt(doc *layout.Doc, attrStmt *ast.AttrStmt) {
	doc.Break(1).
		Group(func(d *layout.Doc) {
			p.layoutID(doc, attrStmt.ID)
			doc.Space()
			p.layoutAttrList(doc, &attrStmt.AttrList)
		})
}

func (p *Printer) layoutAttribute(doc *layout.Doc, attribute ast.Attribute) {
	p.layoutID(doc, attribute.Name)
	doc.Text(token.Equal.String())
	p.layoutID(doc, attribute.Value)
}

func (p *Printer) layoutSubgraph(doc *layout.Doc, subraph ast.Subgraph) {
	doc.Text(token.Subgraph.String()).
		Space()
	if subraph.ID != nil {
		p.layoutID(doc, *subraph.ID)
		doc.Space()
	}

	doc.Text(token.LeftBrace.String())
	p.layoutStmts(doc, subraph.Stmts)
	// TODO who closes this brace?
}
