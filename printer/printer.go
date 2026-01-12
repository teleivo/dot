// Package printer prints DOT syntax trees formatted in the spirit of [gofumpt].
//
// [gofumpt]: https://github.com/mvdan/gofumpt
package printer

import (
	"io"

	"github.com/teleivo/dot"
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
	src    []byte        // src is the DOT source code to format
	w      io.Writer     // w writer to output formatted DOT code to
	format layout.Format // format in which to print the DOT code
}

// New creates a new printer that formats DOT source code and writes the formatted output to w.
// The format parameter controls the output representation.
func New(src []byte, w io.Writer, format layout.Format) *Printer {
	return &Printer{
		src:    src,
		w:      w,
		format: format,
	}
}

// Print parses the DOT code and writes the formatted output to the writer.
// Returns an error if parsing or formatting fails.
func (p *Printer) Print() error {
	ps := dot.NewParser(p.src)
	file := ps.Parse()

	if errs := ps.Errors(); len(errs) > 0 {
		return errs[0]
	}

	first := true
	for _, child := range file.Children {
		if tc, ok := child.(dot.TreeChild); ok && tc.Kind == dot.KindGraph {
			if !first {
				if _, err := p.w.Write([]byte("\n")); err != nil {
					return err
				}
			}
			first = false
			doc := layout.NewDoc(maxColumn)
			p.layoutGraph(doc, tc.Tree)
			if err := doc.Render(p.w, p.format); err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *Printer) layoutGraph(doc *layout.Doc, tree *dot.Tree) {
	if _, ok := dot.TokenAt(tree, token.Strict, 0); ok {
		doc.Text(token.Strict.String()).Space()
	}

	if _, ok := dot.TokenFirst(tree, token.Digraph); ok {
		doc.Text(token.Digraph.String())
	} else {
		doc.Text(token.Graph.String())
	}
	doc.Space()

	// graph : [ strict ] (graph | digraph) [ ID ] '{' stmt_list '}'
	// ID appears after the graph/digraph keyword
	_, idx, _ := dot.TokenFirstWithin(tree, token.Graph|token.Digraph, 1)
	if id, ok := dot.TreeAt(tree, dot.KindID, idx+1); ok {
		p.layoutID(doc, id)
		doc.Space()
	}

	doc.Text(token.LeftBrace.String())
	doc.Group(func(f *layout.Doc) {
		doc.Indent(1, func(d *layout.Doc) {
			if stmtList, ok := dot.TreeFirst(tree, dot.KindStmtList); ok {
				p.layoutStmtList(doc, stmtList)
			}
		})

		doc.Break(1).Text(token.RightBrace.String())
	})
}

func (p *Printer) layoutStmtList(doc *layout.Doc, tree *dot.Tree) {
	for _, child := range tree.Children {
		if tc, ok := child.(dot.TreeChild); ok {
			p.layoutStmt(doc, tc.Tree)
		}
	}
}

// layoutID prints a DOT [identifier]. newlines without preceding '\' are not mentioned as legal but
// are supported by the DOT tooling. Such newlines are normalized to line continuations.
//
// [identifier:] https://graphviz.org/doc/info/lang.html#ids
func (p *Printer) layoutID(doc *layout.Doc, tree *dot.Tree) {
	if tok, ok := dot.TokenFirst(tree, token.ID); ok {
		doc.Text(tok.Literal)
	}
}

func (p *Printer) layoutStmt(doc *layout.Doc, tree *dot.Tree) {
	switch tree.Kind {
	case dot.KindNodeStmt:
		p.layoutNodeStmt(doc, tree)
	case dot.KindEdgeStmt:
		p.layoutEdgeStmt(doc, tree)
	case dot.KindAttrStmt:
		p.layoutAttrStmt(doc, tree)
	case dot.KindAttribute:
		doc.Break(1)
		p.layoutAttribute(doc, tree)
	case dot.KindSubgraph:
		doc.Break(1)
		p.layoutSubgraph(doc, tree)
	}
}

func (p *Printer) layoutNodeStmt(doc *layout.Doc, tree *dot.Tree) {
	doc.Break(1).
		Group(func(d *layout.Doc) {
			// node_stmt : node_id [ attr_list ]
			if nodeID, ok := dot.TreeAt(tree, dot.KindNodeID, 0); ok {
				p.layoutNodeID(doc, nodeID)
			}
			doc.Space()
			if attrList, ok := dot.TreeLast(tree, dot.KindAttrList); ok {
				p.layoutAttrList(doc, attrList)
			}
		})
}

func (p *Printer) layoutNodeID(doc *layout.Doc, tree *dot.Tree) {
	// node_id : ID [ port ]
	if id, ok := dot.TreeAt(tree, dot.KindID, 0); ok {
		p.layoutID(doc, id)
	}

	port, ok := dot.TreeAt(tree, dot.KindPort, 1)
	if !ok {
		return
	}

	// port : ':' ID [ ':' compass_pt ] | ':' compass_pt
	if portName, ok := dot.TreeAt(port, dot.KindID, 1); ok {
		doc.Text(token.Colon.String())
		p.layoutID(doc, portName)
	}
	if cp, ok := dot.TreeFirstWithin(port, dot.KindCompassPoint, 3); ok {
		if tok, ok := dot.TokenFirst(cp, token.ID); ok && tok.Literal != "_" {
			doc.Text(token.Colon.String())
			doc.Text(tok.Literal)
		}
	}
}

func (p *Printer) layoutAttrList(doc *layout.Doc, tree *dot.Tree) {
	if tree == nil {
		return
	}

	// Build list of attribute groups (one per bracket pair)
	var lists [][]*dot.Tree
	var current []*dot.Tree
	for _, child := range tree.Children {
		if tc, ok := child.(dot.TokenChild); ok {
			switch tc.Kind {
			case token.LeftBracket:
				current = make([]*dot.Tree, 0)
			case token.RightBracket:
				lists = append(lists, current)
			}
		} else if tc, ok := child.(dot.TreeChild); ok && tc.Kind == dot.KindAList {
			for _, ac := range tc.Children {
				if attr, ok := ac.(dot.TreeChild); ok && attr.Kind == dot.KindAttribute {
					current = append(current, attr.Tree)
				}
			}
		}
	}

	if len(lists) == 0 {
		return
	}

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

func (p *Printer) layoutEdgeStmt(doc *layout.Doc, tree *dot.Tree) {
	doc.Break(1)

	doc.Group(func(d *layout.Doc) {
		// Collect operands (NodeID or Subgraph)
		var operands []*dot.Tree
		for _, child := range tree.Children {
			if tc, ok := child.(dot.TreeChild); ok {
				if tc.Kind == dot.KindNodeID || tc.Kind == dot.KindSubgraph {
					operands = append(operands, tc.Tree)
				}
			}
		}

		// Check if directed
		_, directed := dot.TokenAt(tree, token.DirectedEdge, 1)

		doc.Group(func(d *layout.Doc) {
			for i, op := range operands {
				p.layoutEdgeOperand(doc, op)
				if i < len(operands)-1 {
					doc.Space()
					if directed {
						doc.Text(token.DirectedEdge.String())
					} else {
						doc.Text(token.UndirectedEdge.String())
					}
					doc.Space()
				}
			}
		})

		doc.Space()
		if attrList, ok := dot.TreeLast(tree, dot.KindAttrList); ok {
			p.layoutAttrList(doc, attrList)
		}
	})
}

func (p *Printer) layoutEdgeOperand(doc *layout.Doc, tree *dot.Tree) {
	switch tree.Kind {
	case dot.KindNodeID:
		p.layoutNodeID(doc, tree)
	case dot.KindSubgraph:
		p.layoutSubgraph(doc, tree)
	}
}

func (p *Printer) layoutAttrStmt(doc *layout.Doc, tree *dot.Tree) {
	doc.Break(1).
		Group(func(d *layout.Doc) {
			// attr_stmt : (graph | node | edge) attr_list
			if tok, ok := dot.TokenAt(tree, token.Graph|token.Node|token.Edge, 0); ok {
				doc.Text(tok.Literal)
			}
			doc.Space()
			if attrList, ok := dot.TreeLast(tree, dot.KindAttrList); ok {
				p.layoutAttrList(doc, attrList)
			}
		})
}

func (p *Printer) layoutAttribute(doc *layout.Doc, tree *dot.Tree) {
	// attribute : attr_name '=' attr_value
	if nameTree, ok := dot.TreeAt(tree, dot.KindAttrName, 0); ok {
		if id, ok := dot.TreeAt(nameTree, dot.KindID, 0); ok {
			p.layoutID(doc, id)
		}
	}
	doc.Text(token.Equal.String())
	if valueTree, ok := dot.TreeAt(tree, dot.KindAttrValue, 2); ok {
		if id, ok := dot.TreeAt(valueTree, dot.KindID, 0); ok {
			p.layoutID(doc, id)
		}
	}
}

func (p *Printer) layoutSubgraph(doc *layout.Doc, tree *dot.Tree) {
	doc.Group(func(f *layout.Doc) {
		// subgraph : [ subgraph [ ID ] ] '{' stmt_list '}'
		if _, ok := dot.TokenAt(tree, token.Subgraph, 0); ok {
			doc.Text(token.Subgraph.String()).Space()
		}

		// ID appears at index 1 only if subgraph keyword is present
		if _, ok := dot.TokenAt(tree, token.Subgraph, 0); ok {
			if id, ok := dot.TreeAt(tree, dot.KindID, 1); ok {
				p.layoutID(doc, id)
				doc.Space()
			}
		}

		doc.Text(token.LeftBrace.String())
		doc.Group(func(f *layout.Doc) {
			doc.Indent(1, func(d *layout.Doc) {
				if stmtList, ok := dot.TreeFirst(tree, dot.KindStmtList); ok {
					p.layoutStmtList(doc, stmtList)
				}
			})

			doc.Break(1).Text(token.RightBrace.String())
		})
	})
}
