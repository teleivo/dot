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

// New creates a new printer that reads DOT code from r, formats it, and writes the
// formatted output to w. The format parameter controls the output representation.
func New(r io.Reader, w io.Writer, format layout.Format) *Printer {
	return &Printer{
		r:      r,
		w:      w,
		format: format,
	}
}

// Print parses the DOT code from the reader and writes the formatted output to the writer.
// Returns an error if parsing or formatting fails.
func (p *Printer) Print() error {
	ps, err := dot.NewParser(p.r)
	if err != nil {
		return err
	}

	tree, err := ps.Parse()
	if err != nil {
		return err
	}

	if errs := ps.Errors(); len(errs) > 0 {
		return errs[0]
	}

	gs := ast.NewGraph(tree)
	for i, g := range gs {
		if i > 0 {
			_, err = p.w.Write([]byte("\n"))
			if err != nil {
				return err
			}
		}
		doc := layout.NewDoc(maxColumn)
		p.layoutGraph(doc, g)
		err = doc.Render(p.w, p.format)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *Printer) layoutGraph(doc *layout.Doc, graph *ast.Graph) {
	if graph.IsStrict() {
		doc.Text(token.Strict.String()).
			Space()
	}

	if graph.Directed() {
		doc.Text(token.Digraph.String())
	} else {
		doc.Text(token.Graph.String())
	}
	doc.Space()

	if graph.ID() != nil {
		p.layoutID(doc, *graph.ID())
		doc.Space()
	}

	doc.Text(token.LeftBrace.String())
	doc.Group(func(f *layout.Doc) {
		doc.Indent(1, func(d *layout.Doc) {
			p.layoutStmts(doc, graph.Stmts())
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
	doc.Text(id.Literal())
}

func (p *Printer) layoutStmt(doc *layout.Doc, stmt ast.Stmt) {
	switch st := stmt.(type) {
	case ast.NodeStmt:
		p.layoutNodeStmt(doc, st)
	case ast.EdgeStmt:
		p.layoutEdgeStmt(doc, st)
	case ast.AttrStmt:
		p.layoutAttrStmt(doc, st)
	case ast.Attribute:
		doc.Break(1)
		p.layoutAttribute(doc, st)
	case ast.Subgraph:
		doc.Break(1)
		p.layoutSubgraph(doc, st)
	}
}

func (p *Printer) layoutNodeStmt(doc *layout.Doc, nodeStmt ast.NodeStmt) {
	doc.Break(1).
		Group(func(d *layout.Doc) {
			p.layoutNodeID(doc, nodeStmt.NodeID())
			p.layoutAttrList(doc, nodeStmt.AttrList())
		})
}

func (p *Printer) layoutNodeID(doc *layout.Doc, nodeID ast.NodeID) {
	p.layoutID(doc, nodeID.ID())

	if nodeID.Port() == nil {
		return
	}

	if nodeID.Port().Name() != nil {
		doc.Text(token.Colon.String())
		p.layoutID(doc, *nodeID.Port().Name())
	}
	if cp := nodeID.Port().CompassPoint(); cp != nil && cp.Type() != ast.CompassPointUnderscore {
		doc.Text(token.Colon.String())
		doc.Text(cp.String())
	}
}

func (p *Printer) layoutAttrList(doc *layout.Doc, attrList ast.AttrList) {
	lists := attrList.Lists()
	if len(lists) == 0 {
		return
	}

	doc.Space()
	doc.Group(func(d *layout.Doc) {
		for i, attrs := range lists {
			doc.Group(func(d *layout.Doc) {
				doc.Text(token.LeftBracket.String()).
					BreakIf(1, layout.Broken).
					Indent(1, func(d *layout.Doc) {
						for j, attr := range attrs {
							p.layoutAttribute(doc, attr)
							if j < len(attrs)-1 {
								doc.TextIf(token.Comma.String(), layout.Flat)
								doc.BreakIf(1, layout.Broken)
							}
						}
					})
				doc.BreakIf(1, layout.Broken).
					Text(token.RightBracket.String())
			})
			if i < len(lists)-1 {
				doc.Space()
			}
		}
	})
}

func (p *Printer) layoutEdgeStmt(doc *layout.Doc, edgeStmt ast.EdgeStmt) {
	doc.Break(1)

	doc.Group(func(d *layout.Doc) {
		doc.Group(func(d *layout.Doc) {
			operands := edgeStmt.Operands()
			for i, op := range operands {
				p.layoutEdgeOperand(doc, op)
				if i < len(operands)-1 {
					doc.Space()
					if edgeStmt.Directed() {
						doc.Text(token.DirectedEdge.String())
					} else {
						doc.Text(token.UndirectedEdge.String())
					}
					doc.Space()
				}
			}
		})
		p.layoutAttrList(doc, edgeStmt.AttrList())
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

func (p *Printer) layoutAttrStmt(doc *layout.Doc, attrStmt ast.AttrStmt) {
	doc.Break(1).
		Group(func(d *layout.Doc) {
			p.layoutID(doc, attrStmt.Target())
			doc.Space()
			p.layoutAttrList(doc, attrStmt.AttrList())
		})
}

func (p *Printer) layoutAttribute(doc *layout.Doc, attribute ast.Attribute) {
	p.layoutID(doc, attribute.Name())
	doc.Text(token.Equal.String())
	p.layoutID(doc, attribute.Value())
}

func (p *Printer) layoutSubgraph(doc *layout.Doc, subgraph ast.Subgraph) {
	doc.Group(func(f *layout.Doc) {
		if subgraph.HasKeyword() {
			doc.Text(token.Subgraph.String()).
				Space()
		}
		if subgraph.ID() != nil {
			p.layoutID(doc, *subgraph.ID())
			doc.Space()
		}

		doc.Text(token.LeftBrace.String())
		doc.Group(func(f *layout.Doc) {
			doc.Indent(1, func(d *layout.Doc) {
				p.layoutStmts(doc, subgraph.Stmts())
			})

			doc.Break(1).
				Text(token.RightBrace.String())
		})
	})
}
