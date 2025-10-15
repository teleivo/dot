// Package printer prints DOT ASTs formatted in the spirit of [gofumpt].
//
// [gofumpt]: https://github.com/mvdan/gofumpt
package printer

import (
	"io"

	"github.com/teleivo/dot"
	"github.com/teleivo/dot/ast"
	"github.com/teleivo/dot/internal/layout"
	"github.com/teleivo/dot/token"
)

const (
	// maxColumn is the max number of runes after which lines are broken up into multiple lines. Not
	// every dot construct can be broken up though.
	maxColumn = 80
)

// Printer formats DOT code.
type Printer struct {
	r      io.Reader     // r reader to parse dot code from
	w      io.Writer     // w writer to output formatted DOT code to
	format layout.Format // format in which to print the DOT code
}

// NewPrinter creates a new printer that reads DOT code from r, formats it, and writes the
// formatted output to w. The format parameter controls the output representation.
func NewPrinter(r io.Reader, w io.Writer, format layout.Format) *Printer {
	return &Printer{
		r:      r,
		w:      w,
		format: format,
	}
}

// Print parses the DOT code from the reader and writes the formatted output to the writer.
// Returns an error if parsing or formatting fails.
func (p *Printer) Print() error {
	// TODO wrap errors in here to give some context?
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
	err = doc.Render(p.w, p.format)

	return err
}

func (p *Printer) layoutNode(doc *layout.Doc, node ast.Node) {
	switch n := node.(type) {
	case *ast.Graph:
		p.layoutGraph(doc, n)
	}
}

func (p *Printer) layoutGraph(doc *layout.Doc, graph *ast.Graph) {
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
		doc.Indent(1, func(d *layout.Doc) {
			p.layoutStmts(doc, graph.Stmts)
		})

		doc.Break(1).
			Text(token.RightBrace.String())
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
			p.layoutAttrList(doc, nodeStmt.AttrList)
		})
}

func (p *Printer) layoutNodeID(doc *layout.Doc, nodeID ast.NodeID) {
	p.layoutID(doc, nodeID.ID)

	if nodeID.Port == nil {
		return
	}

	if nodeID.Port.Name != nil {
		doc.Text(token.Colon.String())
		p.layoutID(doc, *nodeID.Port.Name)
	}
	if nodeID.Port.CompassPoint != nil && nodeID.Port.CompassPoint.Type != ast.CompassPointUnderscore {
		doc.Text(token.Colon.String())
		doc.Text(nodeID.Port.CompassPoint.String())
	}
}

func (p *Printer) layoutAttrList(doc *layout.Doc, attrList *ast.AttrList) {
	// don't print empty [] in node_stmt or edge_stmt where attr_list is optional
	if attrList == nil {
		return
	}

	doc.Space()
	doc.Group(func(d *layout.Doc) {
		for cur := attrList; cur != nil; cur = cur.Next {
			doc.Group(func(d *layout.Doc) {
				doc.Text(token.LeftBracket.String()).
					BreakIf(1, layout.Broken).
					Indent(1, func(d *layout.Doc) {
						p.layoutAList(doc, cur.AList)
					})
				doc.BreakIf(1, layout.Broken).
					Text(token.RightBracket.String())
			})
			if cur.Next != nil {
				doc.Space()
			}
		}
	})
}

func (p *Printer) layoutAList(doc *layout.Doc, aList *ast.AList) {
	for cur := aList; cur != nil; cur = cur.Next {
		p.layoutAttribute(doc, cur.Attribute)
		if cur.Next != nil {
			doc.TextIf(token.Comma.String(), layout.Flat)
			doc.BreakIf(1, layout.Broken)
		}
	}
}

func (p *Printer) layoutEdgeStmt(doc *layout.Doc, edgeStmt *ast.EdgeStmt) {
	doc.Break(1)

	doc.Group(func(d *layout.Doc) {
		doc.Group(func(d *layout.Doc) {
			p.layoutEdgeOperand(doc, edgeStmt.Left)
			doc.Space()

			if edgeStmt.Right.Directed {
				doc.Text(token.DirectedEdge.String())
			} else {
				doc.Text(token.UndirectedEdge.String())
			}
			doc.Space()

			p.layoutEdgeOperand(doc, edgeStmt.Right.Right)
			for cur := edgeStmt.Right.Next; cur != nil; cur = cur.Next {
				doc.Space()
				if edgeStmt.Right.Directed {
					doc.Text(token.DirectedEdge.String())
				} else {
					doc.Text(token.UndirectedEdge.String())
				}
				doc.Space()

				p.layoutEdgeOperand(doc, cur.Right)
			}
		})
		p.layoutAttrList(doc, edgeStmt.AttrList)
	})
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
	doc.Group(func(f *layout.Doc) {
		if subraph.SubgraphStart != nil {
			doc.Text(token.Subgraph.String()).
				Space()
		}
		if subraph.ID != nil {
			p.layoutID(doc, *subraph.ID)
			doc.Space()
		}

		doc.Text(token.LeftBrace.String())
		doc.Group(func(f *layout.Doc) {
			doc.Indent(1, func(d *layout.Doc) {
				p.layoutStmts(doc, subraph.Stmts)
			})

			doc.Break(1).
				Text(token.RightBrace.String())
		})
	})
}
