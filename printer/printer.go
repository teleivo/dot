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

	var endsWithBreak bool
	first := true
	doc := layout.NewDoc(maxColumn)
	for _, child := range file.Children {
		if tc, ok := child.(dot.TokenChild); ok && tc.Kind == token.Comment {
			// file-level comments are always on their own line
			if !first {
				doc.Break(1)
			}
			doc.Text(tc.Literal)
			if isLineComment(tc.Literal) {
				doc.Break(1)
				endsWithBreak = true
			}
			first = false
		} else if tc, ok := child.(dot.TreeChild); ok && tc.Kind == dot.KindGraph {
			if !first {
				doc.Break(1)
			}
			p.layoutGraph(doc, tc.Tree)
			endsWithBreak = false
			first = false
		}
	}
	if !endsWithBreak {
		doc.Break(1)
	}
	if err := doc.Render(p.w, p.format); err != nil {
		return err
	}

	return nil
}

// layoutGraph handles: graph : [ 'strict' ] ( 'graph' | 'digraph' ) [ ID ] '{' stmt_list '}'
func (p *Printer) layoutGraph(doc *layout.Doc, tree *dot.Tree) {
	p.layoutBlock(doc, tree)
}

// layoutBlock handles graph and subgraph layout:
//
//	graph    : [ 'strict' ] ( 'graph' | 'digraph' ) [ ID ] '{' stmt_list '}'
//	subgraph : [ 'subgraph' [ ID ] ] '{' stmt_list '}'
func (p *Printer) layoutBlock(doc *layout.Doc, tree *dot.Tree) {
	var i int

	// layout [ 'strict' ] ( 'graph' | 'digraph' ) [ ID ] up to '{'
	// emittedToken tracks if we emitted a non-comment token, to distinguish trailing vs leading comments
	// emittedAnything tracks if we emitted anything at all, to avoid leading break/space on first element
	var emittedToken, emittedAnything bool
	for i = 0; i < len(tree.Children); i++ {
		child := tree.Children[i]
		if tc, ok := child.(dot.TokenChild); ok {
			if tc.Kind == token.LeftBrace {
				break
			} else if tc.Kind == token.Comment {
				if emittedAnything {
					p.layoutComment(doc, tc.Literal, !emittedToken)
				} else { // comment is first element (token/tree)
					doc.Text(tc.Literal)
					if isLineComment(tc.Literal) {
						doc.Break(1)
					}
				}
				emittedToken = false
				emittedAnything = true
			} else {
				doc.Text(tc.Literal).Space()
				emittedToken = true
				emittedAnything = true
			}
		} else if tc, ok := child.(dot.TreeChild); ok && tc.Kind == dot.KindID {
			p.layoutID(doc, tc.Tree)
			doc.Space()
			emittedToken = true
			emittedAnything = true
		}
	}

	doc.Text(token.LeftBrace.String())
	doc.Group(func(f *layout.Doc) {
		// continue after '{' and emit all before '}'
		doc.Indent(1, func(d *layout.Doc) {
			for i++; i < len(tree.Children); i++ {
				child := tree.Children[i]
				if tc, ok := child.(dot.TokenChild); ok {
					if tc.Kind == token.RightBrace {
						break
					} else if tc.Kind == token.Comment {
						p.layoutComment(doc, tc.Literal, true)
					}
				} else if tc, ok := child.(dot.TreeChild); ok && tc.Kind == dot.KindStmtList {
					p.layoutStmtList(doc, tc.Tree)
				}
			}
		})

		doc.Break(1).Text(token.RightBrace.String())
	})

	// Handle trailing comments after '}'
	for i++; i < len(tree.Children); i++ {
		if tc, ok := tree.Children[i].(dot.TokenChild); ok && tc.Kind == token.Comment {
			p.layoutComment(doc, tc.Literal, false)
		}
	}
}

// layoutStmtList handles: stmt_list : [ stmt [ ';' ] stmt_list ]
func (p *Printer) layoutStmtList(doc *layout.Doc, tree *dot.Tree) {
	for _, child := range tree.Children {
		if tc, ok := child.(dot.TreeChild); ok {
			p.layoutStmt(doc, tc.Tree)
		} else if tc, ok := child.(dot.TokenChild); ok && tc.Kind == token.Comment {
			p.layoutComment(doc, tc.Literal, true)
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
	p.layoutTrailingComments(doc, tree)
}

// layoutStmt handles: stmt : node_stmt | edge_stmt | attr_stmt | ID '=' ID | subgraph
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

// layoutNodeStmt handles: node_stmt : node_id [ attr_list ]
func (p *Printer) layoutNodeStmt(doc *layout.Doc, tree *dot.Tree) {
	doc.Break(1).
		Group(func(d *layout.Doc) {
			if nodeID, ok := dot.TreeAt(tree, dot.KindNodeID, 0); ok {
				p.layoutNodeID(doc, nodeID)
				p.spaceOrBreak(doc, nodeID)
			}
			if attrList, ok := dot.TreeLast(tree, dot.KindAttrList); ok {
				p.layoutAttrList(doc, attrList)
			}
		})
}

// layoutComment emits a comment with appropriate spacing.
// leading=true for own-line comments (Break before), leading=false for trailing (Space before).
func (p *Printer) layoutComment(doc *layout.Doc, literal string, leading bool) {
	if leading {
		doc.Break(1)
	} else {
		doc.Space()
	}
	doc.Text(literal)
	if isLineComment(literal) {
		doc.Break(1)
	}
}

// layoutTrailingComments emits any trailing comment tokens from tree's direct children.
// Returns true if a line comment was emitted (which forces a break).
func (p *Printer) layoutTrailingComments(doc *layout.Doc, tree *dot.Tree) bool {
	var broken bool
	for _, child := range tree.Children {
		if tc, ok := child.(dot.TokenChild); ok && tc.Kind == token.Comment {
			p.layoutComment(doc, tc.Literal, false)
			if isLineComment(tc.Literal) {
				broken = true
			}
		}
	}
	return broken
}

// isLineComment reports whether the comment literal is a line comment (// or #).
func isLineComment(s string) bool {
	return (len(s) > 0 && s[0] == '#') || (len(s) > 1 && s[0] == '/' && s[1] == '/')
}

// hasComment reports whether tree or any of its descendants contain a comment.
func hasComment(tree *dot.Tree) bool {
	for _, child := range tree.Children {
		if tc, ok := child.(dot.TokenChild); ok && tc.Kind == token.Comment {
			return true
		}
		if tc, ok := child.(dot.TreeChild); ok && hasComment(tc.Tree) {
			return true
		}
	}
	return false
}

func (p *Printer) spaceOrBreak(doc *layout.Doc, tree *dot.Tree) {
	if hasComment(tree) {
		doc.Break(1)
	} else {
		doc.Space()
	}
}

// layoutNodeID handles: node_id : ID [ port ]
func (p *Printer) layoutNodeID(doc *layout.Doc, tree *dot.Tree) {
	if id, ok := dot.TreeAt(tree, dot.KindID, 0); ok {
		p.layoutID(doc, id)
	}
	if port, ok := dot.TreeAt(tree, dot.KindPort, 1); ok {
		p.layoutPort(doc, port)
	}
}

// layoutPort handles: port : ':' ID [ ':' compass_pt ] | ':' compass_pt
func (p *Printer) layoutPort(doc *layout.Doc, tree *dot.Tree) {
	// emittedColon tracks if we just emitted ':', to distinguish trailing vs leading comments
	// pendingColon tracks if we have a ':' that needs to be printed before the next ID
	emittedColon := false
	pendingColon := false
	for _, child := range tree.Children {
		if tc, ok := child.(dot.TokenChild); ok {
			switch tc.Kind {
			case token.Colon:
				// don't print colon yet and see if next child is "_" or comment
				pendingColon = true
				emittedColon = false
			case token.Comment:
				// print pending colon before comment so comment can trail it
				if pendingColon {
					doc.Text(token.Colon.String())
					pendingColon = false
					emittedColon = true
				}
				p.layoutComment(doc, tc.Literal, !emittedColon)
				emittedColon = false
			}
		} else if tc, ok := child.(dot.TreeChild); ok {
			if tc.Kind == dot.KindID || tc.Kind == dot.KindCompassPoint {
				// skip printing "_" compass point and its preceding ':'
				if tok, ok := dot.TokenFirst(tc.Tree, token.ID); ok && tok.Literal != "_" {
					if pendingColon {
						doc.Text(token.Colon.String())
					}
					p.layoutID(doc, tc.Tree)
					emittedColon = false
				}
				pendingColon = false
			}
		}
	}
}

// layoutAttrList handles: attr_list : '[' [ a_list ] ']' [ attr_list ]
func (p *Printer) layoutAttrList(doc *layout.Doc, tree *dot.Tree) {
	emittedBracket := false // for space between consecutive bracket pairs: [a=b] [c=d]
	doc.Group(func(d *layout.Doc) {
		for i := 0; i < len(tree.Children); i++ {
			if tc, ok := tree.Children[i].(dot.TokenChild); ok {
				switch tc.Kind {
				case token.LeftBracket:
					if emittedBracket {
						doc.Space()
					}
					i = p.layoutBracketBlock(doc, tree, i)
					emittedBracket = true
				case token.Comment:
					p.layoutComment(doc, tc.Literal, !emittedBracket)
				}
			}
		}
	})
}

// layoutBracketBlock handles a single [...] block starting at index i (the '[').
// Returns the index of the closing ']'.
func (p *Printer) layoutBracketBlock(doc *layout.Doc, tree *dot.Tree, i int) int {
	doc.Group(func(d *layout.Doc) {
		doc.Text(token.LeftBracket.String()).
			BreakIf(1, layout.Broken).
			Indent(1, func(d *layout.Doc) {
				emittedAttr := false
				for i++; i < len(tree.Children); i++ {
					if tc, ok := tree.Children[i].(dot.TokenChild); ok {
						if tc.Kind == token.RightBracket {
							break
						} else if tc.Kind == token.Comment {
							p.layoutComment(doc, tc.Literal, true)
						}
					} else if tc, ok := tree.Children[i].(dot.TreeChild); ok && tc.Kind == dot.KindAList {
						emittedAttr = p.layoutAList(doc, tc.Tree, emittedAttr)
					}
				}
			})
		doc.BreakIf(1, layout.Broken).Text(token.RightBracket.String())
	})
	return i
}

// layoutAList handles: a_list : ID '=' ID [ ( ';' | ',' ) ] [ a_list ]
// Returns true if any attribute was emitted.
func (p *Printer) layoutAList(doc *layout.Doc, tree *dot.Tree, emittedAttr bool) bool {
	for _, child := range tree.Children {
		if tc, ok := child.(dot.TokenChild); ok && tc.Kind == token.Comment {
			p.layoutComment(doc, tc.Literal, true)
		} else if tc, ok := child.(dot.TreeChild); ok && tc.Kind == dot.KindAttribute {
			if emittedAttr {
				doc.TextIf(token.Comma.String(), layout.Flat)
				doc.BreakIf(1, layout.Broken)
			}
			p.layoutAttribute(doc, tc.Tree)
			emittedAttr = true
		}
	}
	return emittedAttr
}

// layoutEdgeStmt handles:
//
//	edge_stmt : (node_id | subgraph) edgeRHS [ attr_list ]
//	edgeRHS   : edgeop (node_id | subgraph) [ edgeRHS ]
func (p *Printer) layoutEdgeStmt(doc *layout.Doc, tree *dot.Tree) {
	doc.Break(1)
	doc.Group(func(d *layout.Doc) {
		// emittedEdgeOp tracks if we just emitted an edge operator, to distinguish
		// trailing comments (space before: A -> // c1) from leading comments
		// (break before: A // c1 \n ->).
		//
		// needsSpace tracks if the next element needs a leading space. After an
		// operand we need a space before the edge operator. After a leading comment
		// (which ends with a break), we don't need a space.
		emittedEdgeOp := false
		needsSpace := false
		var lastOperand *dot.Tree
		doc.Group(func(d *layout.Doc) {
			for _, child := range tree.Children {
				if tc, ok := child.(dot.TokenChild); ok {
					switch tc.Kind {
					case token.DirectedEdge, token.UndirectedEdge:
						if needsSpace {
							doc.Space()
						}
						doc.Text(tc.Literal)
						emittedEdgeOp = true
						needsSpace = true
					case token.Comment:
						// comment after edge op is trailing (same line), otherwise leading (own line)
						leading := !emittedEdgeOp
						p.layoutComment(doc, tc.Literal, leading)
						emittedEdgeOp = false
						// line comments end with a break, so no space needed after
						needsSpace = !isLineComment(tc.Literal)
					}
				} else if tc, ok := child.(dot.TreeChild); ok {
					if tc.Kind == dot.KindNodeID || tc.Kind == dot.KindSubgraph {
						if needsSpace {
							doc.Space()
						}
						p.layoutEdgeOperand(doc, tc.Tree)
						lastOperand = tc.Tree
						emittedEdgeOp = false
						needsSpace = true
					}
				}
			}
		})
		if lastOperand != nil {
			p.spaceOrBreak(doc, lastOperand)
		}

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

// layoutAttrStmt handles: attr_stmt : ( 'graph' | 'node' | 'edge' ) attr_list
func (p *Printer) layoutAttrStmt(doc *layout.Doc, tree *dot.Tree) {
	doc.Break(1).
		Group(func(d *layout.Doc) {
			if tok, ok := dot.TokenAt(tree, token.Graph|token.Node|token.Edge, 0); ok {
				doc.Text(tok.Literal)
			}
			if !p.layoutTrailingComments(doc, tree) {
				doc.Space()
			}
			if attrList, ok := dot.TreeLast(tree, dot.KindAttrList); ok {
				p.layoutAttrList(doc, attrList)
			}
		})
}

// layoutAttribute handles: ID '=' ID
func (p *Printer) layoutAttribute(doc *layout.Doc, tree *dot.Tree) {
	emittedToken := false
	for _, child := range tree.Children {
		if tc, ok := child.(dot.TokenChild); ok {
			switch tc.Kind {
			case token.Equal:
				doc.Text(token.Equal.String())
				emittedToken = true
			case token.Comment:
				p.layoutComment(doc, tc.Literal, !emittedToken)
				emittedToken = false
			}
		} else if tc, ok := child.(dot.TreeChild); ok {
			if tc.Kind == dot.KindAttrName || tc.Kind == dot.KindAttrValue {
				if id, ok := dot.TreeAt(tc.Tree, dot.KindID, 0); ok {
					p.layoutID(doc, id)
					emittedToken = true
				}
			}
		}
	}
}

// layoutSubgraph handles: subgraph : [ 'subgraph' [ ID ] ] '{' stmt_list '}'
func (p *Printer) layoutSubgraph(doc *layout.Doc, tree *dot.Tree) {
	doc.Group(func(f *layout.Doc) {
		p.layoutBlock(doc, tree)
	})
}
