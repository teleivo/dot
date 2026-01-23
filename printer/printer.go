// Package printer prints DOT syntax trees formatted in the spirit of [gofumpt].
//
// [gofumpt]: https://github.com/mvdan/gofumpt
package printer

import (
	"io"
	"strings"

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
	var needsSpace bool
	for i = 0; i < len(tree.Children); i++ {
		child := tree.Children[i]
		if tc, ok := child.(dot.TokenChild); ok {
			if tc.Kind == token.LeftBrace {
				if needsSpace {
					doc.Space()
				}
				break
			} else if tc.Kind == token.Comment {
				prevLine := prevEndLine(tree.Children, i)
				isTrailing := prevLine > 0 && tc.Start.Line == prevLine
				if prevLine == 0 {
					doc.Text(tc.Literal)
					if isLineComment(tc.Literal) {
						doc.Break(1)
					}
				} else {
					p.layoutComment(doc, tc.Literal, isTrailing)
				}
				// Check if comment was on its own line (next element on different line)
				nextLine := nextStartLine(tree.Children, i)
				ownLine := !isTrailing && nextLine > 0 && tc.End.Line < nextLine
				if ownLine && !isLineComment(tc.Literal) {
					doc.Break(1)
					needsSpace = false
				} else {
					needsSpace = !isLineComment(tc.Literal)
				}
			} else {
				if needsSpace {
					doc.Space()
				}
				doc.Text(tc.Literal).Space()
				needsSpace = false
			}
		} else if tc, ok := child.(dot.TreeChild); ok && tc.Kind == dot.KindID {
			if needsSpace {
				doc.Space()
			}
			if !p.layoutID(doc, tc.Tree) {
				doc.Space()
			}
			needsSpace = false
		}
	}

	doc.Text(token.LeftBrace.String())
	doc.Group(func(f *layout.Doc) {
		doc.Indent(1, func(d *layout.Doc) {
			for i++; i < len(tree.Children); i++ {
				child := tree.Children[i]
				if tc, ok := child.(dot.TokenChild); ok {
					if tc.Kind == token.RightBrace {
						break
					} else if tc.Kind == token.Comment {
						p.layoutComment(doc, tc.Literal, false)
					}
				} else if tc, ok := child.(dot.TreeChild); ok && tc.Kind == dot.KindStmtList {
					p.layoutStmtList(doc, tc.Tree)
				}
			}
		})

		doc.Break(1).Text(token.RightBrace.String())
	})

	// Handle trailing comments after '}'
	braceEndLine := 0
	if i < len(tree.Children) {
		if tc, ok := tree.Children[i].(dot.TokenChild); ok && tc.Kind == token.RightBrace {
			braceEndLine = tc.End.Line
		}
	}
	for i++; i < len(tree.Children); i++ {
		if tc, ok := tree.Children[i].(dot.TokenChild); ok && tc.Kind == token.Comment {
			isTrailing := braceEndLine > 0 && tc.Start.Line == braceEndLine
			p.layoutComment(doc, tc.Literal, isTrailing)
			braceEndLine = tc.End.Line
		}
	}
}

// layoutStmtList handles: stmt_list : [ stmt [ ';' ] stmt_list ]
func (p *Printer) layoutStmtList(doc *layout.Doc, tree *dot.Tree) {
	for _, child := range tree.Children {
		if tc, ok := child.(dot.TreeChild); ok {
			p.layoutStmt(doc, tc.Tree)
		} else if tc, ok := child.(dot.TokenChild); ok && tc.Kind == token.Comment {
			p.layoutComment(doc, tc.Literal, false)
		}
	}
}

// layoutID prints a DOT [identifier]. newlines without preceding '\' are not mentioned as legal but
// are supported by the DOT tooling. Such newlines are normalized to line continuations.
// Returns true if a trailing break was emitted (line comment or multi-line block comment).
//
// [identifier:] https://graphviz.org/doc/info/lang.html#ids
func (p *Printer) layoutID(doc *layout.Doc, tree *dot.Tree) bool {
	if tok, ok := dot.TokenFirst(tree, token.ID); ok {
		doc.Text(tok.Literal)
	}
	return p.layoutTrailingComments(doc, tree)
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
				if !p.layoutNodeID(doc, nodeID) {
					doc.Space()
				}
			}
			if attrList, ok := dot.TreeLast(tree, dot.KindAttrList); ok {
				p.layoutAttrList(doc, attrList)
			}
		})
}

// layoutComment emits a comment with appropriate spacing.
// If trailing, adds Space before; otherwise adds Break before.
// Returns true if a space is needed after (block comments end with text).
func (p *Printer) layoutComment(doc *layout.Doc, literal string, trailing bool) bool {
	if trailing {
		doc.Space()
	} else {
		doc.Break(1)
	}
	if isLineComment(literal) {
		doc.Text(literal)
		doc.Break(1)
		return false
	}
	var start int
	for i, r := range literal {
		if r == '\n' {
			doc.Text(strings.TrimSpace(literal[start:i])).Break(1)
			start = i + 1
		}
	}
	if start < len(literal) {
		doc.Text(strings.TrimSpace(literal[start:]))
	}
	return true
}

// layoutTrailingComments emits any trailing comment tokens from tree's direct children.
// Returns true if a line comment was emitted (ends with break).
func (p *Printer) layoutTrailingComments(doc *layout.Doc, tree *dot.Tree) bool {
	var broke bool
	for _, child := range tree.Children {
		if tc, ok := child.(dot.TokenChild); ok && tc.Kind == token.Comment {
			if !p.layoutComment(doc, tc.Literal, true) {
				broke = true
			}
		}
	}
	return broke
}

// isLineComment reports whether the comment literal is a line comment (// or #).
func isLineComment(s string) bool {
	return (len(s) > 0 && s[0] == '#') || (len(s) > 1 && s[0] == '/' && s[1] == '/')
}

// prevEndLine returns the end line of the child at index i-1, or 0 if i==0.
func prevEndLine(children []dot.Child, i int) int {
	if i <= 0 {
		return 0
	}
	prev := children[i-1]
	if tc, ok := prev.(dot.TokenChild); ok {
		return tc.End.Line
	}
	if tc, ok := prev.(dot.TreeChild); ok {
		return tc.End.Line
	}
	return 0
}

// nextStartLine returns the start line of the child at index i+1, or 0 if i is last.
func nextStartLine(children []dot.Child, i int) int {
	if i >= len(children)-1 {
		return 0
	}
	next := children[i+1]
	if tc, ok := next.(dot.TokenChild); ok {
		return tc.Start.Line
	}
	if tc, ok := next.(dot.TreeChild); ok {
		return tc.Start.Line
	}
	return 0
}

// isTrailingComment reports whether the comment at index i is on the same line as the previous element.
func isTrailingComment(comment dot.TokenChild, children []dot.Child, i int) bool {
	prevLine := prevEndLine(children, i)
	return prevLine > 0 && comment.Start.Line == prevLine
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

// layoutNodeID handles: node_id : ID [ port ]
// Returns true if a trailing break was emitted (line comment or multi-line block comment).
func (p *Printer) layoutNodeID(doc *layout.Doc, tree *dot.Tree) bool {
	var broke bool
	if id, ok := dot.TreeAt(tree, dot.KindID, 0); ok {
		broke = p.layoutID(doc, id)
	}
	if port, ok := dot.TreeAt(tree, dot.KindPort, 1); ok {
		broke = p.layoutPort(doc, port)
	}
	return broke
}

// layoutPort handles: port : ':' ID [ ':' compass_pt ] | ':' compass_pt
// Returns true if a trailing break was emitted (line comment or multi-line block comment).
func (p *Printer) layoutPort(doc *layout.Doc, tree *dot.Tree) bool {
	var pendingColon, broke, needsSpace bool
	var colonLine int
	for i, child := range tree.Children {
		if tc, ok := child.(dot.TokenChild); ok {
			switch tc.Kind {
			case token.Colon:
				pendingColon = true
				colonLine = tc.End.Line
			case token.Comment:
				if pendingColon {
					doc.Text(token.Colon.String())
					pendingColon = false
				}
				prevLine := prevEndLine(tree.Children, i)
				if colonLine > 0 && prevLine == 0 {
					prevLine = colonLine
				}
				isTrailing := prevLine > 0 && tc.Start.Line == prevLine
				needsSpace = p.layoutComment(doc, tc.Literal, isTrailing)
				if !needsSpace {
					broke = true
				}
				colonLine = 0
			}
		} else if tc, ok := child.(dot.TreeChild); ok {
			if tc.Kind == dot.KindID || tc.Kind == dot.KindCompassPoint {
				if tok, ok := dot.TokenFirst(tc.Tree, token.ID); ok && tok.Literal != "_" {
					if needsSpace {
						doc.Space()
					}
					if pendingColon {
						doc.Text(token.Colon.String())
					}
					idBroken := p.layoutID(doc, tc.Tree)
					if idBroken {
						broke = true
						needsSpace = false
					} else {
						needsSpace = hasComment(tc.Tree)
					}
				}
				pendingColon = false
				colonLine = 0
			}
		}
	}
	return broke
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
					p.layoutComment(doc, tc.Literal, emittedBracket)
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
							p.layoutComment(doc, tc.Literal, false)
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
			p.layoutComment(doc, tc.Literal, false)
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
		var needsSpace, lastBroke bool
		doc.Group(func(d *layout.Doc) {
			for i, child := range tree.Children {
				if tc, ok := child.(dot.TokenChild); ok {
					switch tc.Kind {
					case token.DirectedEdge, token.UndirectedEdge:
						if needsSpace {
							doc.Space()
						}
						doc.Text(tc.Literal)
						needsSpace = true
					case token.Comment:
						isTrailing := isTrailingComment(tc, tree.Children, i)
						needsSpace = p.layoutComment(doc, tc.Literal, isTrailing)
					}
				} else if tc, ok := child.(dot.TreeChild); ok {
					if tc.Kind == dot.KindNodeID || tc.Kind == dot.KindSubgraph {
						if needsSpace {
							doc.Space()
						}
						lastBroke = p.layoutEdgeOperand(doc, tc.Tree)
						needsSpace = true
					}
				}
			}
		})
		if !lastBroke {
			doc.Space()
		}

		if attrList, ok := dot.TreeLast(tree, dot.KindAttrList); ok {
			p.layoutAttrList(doc, attrList)
		}
	})
}

// layoutEdgeOperand handles a node_id or subgraph in an edge statement.
// Returns true if a trailing break was emitted (line comment or multi-line block comment).
func (p *Printer) layoutEdgeOperand(doc *layout.Doc, tree *dot.Tree) bool {
	switch tree.Kind {
	case dot.KindNodeID:
		return p.layoutNodeID(doc, tree)
	case dot.KindSubgraph:
		p.layoutSubgraph(doc, tree)
	}
	return false
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
	var needsSpace bool
	for i, child := range tree.Children {
		if tc, ok := child.(dot.TokenChild); ok {
			switch tc.Kind {
			case token.Equal:
				if needsSpace {
					doc.Space()
				}
				doc.Text(token.Equal.String())
				needsSpace = false
			case token.Comment:
				isTrailing := isTrailingComment(tc, tree.Children, i)
				needsSpace = p.layoutComment(doc, tc.Literal, isTrailing)
			}
		} else if tc, ok := child.(dot.TreeChild); ok {
			if tc.Kind == dot.KindAttrName || tc.Kind == dot.KindAttrValue {
				if id, ok := dot.TreeAt(tc.Tree, dot.KindID, 0); ok {
					if needsSpace {
						doc.Space()
					}
					broke := p.layoutID(doc, id)
					needsSpace = !broke && hasComment(id)
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
