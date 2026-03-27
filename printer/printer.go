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
	src    []byte
	tree   *dot.Tree
	w      io.Writer
	format layout.Format
}

// New creates a new printer that formats DOT source code and writes the formatted output to w.
func New(src []byte, w io.Writer, format layout.Format) *Printer {
	return &Printer{src: src, w: w, format: format}
}

// Print parses the DOT code and writes the formatted output to the writer.
// Returns an error if parsing or formatting fails.
func (p *Printer) Print() error {
	ps := dot.NewParser(p.src)
	p.tree = ps.Parse()
	if errs := ps.Errors(); len(errs) > 0 {
		return errs[0]
	}

	root := p.tree.Root()
	if root.Start >= root.End {
		return nil
	}

	// root node is the File node
	fileIdx := 0
	var broke bool
	doc := layout.NewDoc(maxColumn)
	nr := p.tree.Children(fileIdx)
	first := true
	for i := nr.Start; i < nr.End; i = p.tree.Next(i) {
		n := p.tree.NodeAt(i)
		if n.IsToken() && n.TokenKind == token.Comment {
			if !first {
				broke = !p.layoutComment(doc, n.Literal, false)
			} else {
				broke = !p.layoutCommentText(doc, n.Literal)
			}
		} else if !n.IsToken() && n.Kind == dot.KindGraph {
			if !first {
				doc.Break(1)
			}
			broke = p.layoutBlock(doc, i)
		}
		first = false
	}
	if !broke {
		doc.Break(1)
	}

	return doc.Render(p.w, p.format)
}

// layoutBlock handles graph and subgraph layout.
func (p *Printer) layoutBlock(doc *layout.Doc, nodeIdx int) bool {
	nr := p.tree.Children(nodeIdx)

	// layout keywords and ID up to '{'
	var needsSpace bool
	i := nr.Start
	var prevEnd uint32
	isFirst := true
	for ; i < nr.End; i = p.tree.Next(i) {
		n := p.tree.NodeAt(i)
		if n.IsToken() {
			if n.TokenKind == token.LeftBrace {
				if needsSpace {
					doc.Space()
				}
				break
			} else if n.TokenKind == token.Comment {
				if !isFirst {
					isTrailing := prevEnd > 0 && n.Start.Line == prevEnd
					needsSpace = p.layoutComment(doc, n.Literal, isTrailing)
				} else {
					needsSpace = p.layoutCommentText(doc, n.Literal)
				}
				// Block comment on its own line needs a break after
				nextIdx := p.tree.Next(i)
				isOwn := !isTrailingLine(prevEnd, n.Start.Line) &&
					nextIdx < nr.End && int(n.End.Line) < p.tree.StartLine(nextIdx)
				if isOwn && needsSpace {
					doc.Break(1)
					needsSpace = false
				}
				prevEnd = n.End.Line
			} else {
				if needsSpace {
					doc.Space()
				}
				doc.Text(n.Literal).Space()
				needsSpace = false
				prevEnd = n.End.Line
			}
		} else if n.Kind == dot.KindID {
			if needsSpace {
				doc.Space()
			}
			if !p.layoutID(doc, i) {
				doc.Space()
			}
			needsSpace = false
			prevEnd = n.End.Line
		}
		isFirst = false
	}

	doc.Text(token.LeftBrace.String())
	doc.Group(func(f *layout.Doc) {
		doc.Indent(1, func(d *layout.Doc) {
			i = p.tree.Next(i) // skip past '{'
			for ; i < nr.End; i = p.tree.Next(i) {
				n := p.tree.NodeAt(i)
				if n.IsToken() {
					if n.TokenKind == token.RightBrace {
						break
					} else if n.TokenKind == token.Comment {
						isTrailing := prevEnd > 0 && n.Start.Line == prevEnd
						p.layoutComment(doc, n.Literal, isTrailing)
						prevEnd = n.End.Line
					}
				} else if n.Kind == dot.KindStmtList {
					p.layoutStmtList(doc, i)
					prevEnd = n.End.Line
				}
			}
		})

		doc.Break(1).Text(token.RightBrace.String())
	})

	var braceEndLine uint32
	n := p.tree.NodeAt(i)
	if n.IsToken() && n.TokenKind == token.RightBrace {
		braceEndLine = n.End.Line
	}
	// Handle trailing comments after '}'
	var broke bool
	i = p.tree.Next(i)
	for ; i < nr.End; i = p.tree.Next(i) {
		n := p.tree.NodeAt(i)
		if n.IsToken() && n.TokenKind == token.Comment {
			isTrailing := braceEndLine > 0 && n.Start.Line == braceEndLine
			broke = !p.layoutComment(doc, n.Literal, isTrailing)
			braceEndLine = n.End.Line
		}
	}
	return broke
}

// layoutStmtList handles: stmt_list : [ stmt [ ';' ] stmt_list ]
func (p *Printer) layoutStmtList(doc *layout.Doc, nodeIdx int) {
	nr := p.tree.Children(nodeIdx)
	var prevEnd uint32
	for i := nr.Start; i < nr.End; i = p.tree.Next(i) {
		n := p.tree.NodeAt(i)
		if !n.IsToken() {
			p.layoutStmt(doc, i)
			prevEnd = n.End.Line
		} else if n.TokenKind == token.Comment {
			isTrailing := prevEnd > 0 && n.Start.Line == prevEnd
			p.layoutComment(doc, n.Literal, isTrailing)
			prevEnd = n.End.Line
		}
	}
}

// layoutStmt handles: stmt : node_stmt | edge_stmt | attr_stmt | ID '=' ID | subgraph
func (p *Printer) layoutStmt(doc *layout.Doc, nodeIdx int) {
	n := p.tree.NodeAt(nodeIdx)
	switch n.Kind {
	case dot.KindNodeStmt:
		p.layoutNodeStmt(doc, nodeIdx)
	case dot.KindEdgeStmt:
		p.layoutEdgeStmt(doc, nodeIdx)
	case dot.KindAttrStmt:
		p.layoutAttrStmt(doc, nodeIdx)
	case dot.KindAttribute:
		doc.Break(1)
		p.layoutAttribute(doc, nodeIdx)
	case dot.KindSubgraph:
		doc.Break(1)
		p.layoutSubgraph(doc, nodeIdx)
	}
}

// layoutNodeStmt handles: node_stmt : node_id [ attr_list ]
func (p *Printer) layoutNodeStmt(doc *layout.Doc, nodeIdx int) {
	doc.Break(1).
		Group(func(d *layout.Doc) {
			if nodeID, ok := p.tree.TreeAt(nodeIdx, dot.KindNodeID, 0); ok {
				if !p.layoutNodeID(doc, nodeID) {
					doc.Space()
				}
			}
			if attrList, ok := p.tree.LastTree(nodeIdx, dot.KindAttrList); ok {
				p.layoutAttrList(doc, attrList)
			}
		})
}

// layoutID prints a DOT identifier.
func (p *Printer) layoutID(doc *layout.Doc, nodeIdx int) bool {
	if tok, ok := p.tree.FirstToken(nodeIdx, token.ID); ok {
		doc.Text(tok.Literal)
	}
	return p.layoutTrailingComments(doc, nodeIdx)
}

// layoutTrailingComments emits any trailing comment tokens from direct children.
func (p *Printer) layoutTrailingComments(doc *layout.Doc, nodeIdx int) bool {
	var broke bool
	nr := p.tree.Children(nodeIdx)
	for i := nr.Start; i < nr.End; i = p.tree.Next(i) {
		n := p.tree.NodeAt(i)
		if n.IsToken() && n.TokenKind == token.Comment {
			broke = !p.layoutComment(doc, n.Literal, true)
		}
	}
	return broke
}

// layoutComment emits a comment with appropriate spacing.
func (p *Printer) layoutComment(doc *layout.Doc, literal string, trailing bool) bool {
	if trailing {
		doc.Space()
	} else {
		doc.Break(1)
	}
	return p.layoutCommentText(doc, literal)
}

// layoutCommentText emits comment text without leading spacing.
func (p *Printer) layoutCommentText(doc *layout.Doc, literal string) bool {
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

// layoutNodeID handles: node_id : ID [ port ]
func (p *Printer) layoutNodeID(doc *layout.Doc, nodeIdx int) bool {
	var broke bool
	if id, ok := p.tree.TreeAt(nodeIdx, dot.KindID, 0); ok {
		broke = p.layoutID(doc, id)
	}
	if port, ok := p.tree.TreeAt(nodeIdx, dot.KindPort, 1); ok {
		broke = p.layoutPort(doc, port)
	}
	return broke
}

// layoutPort handles: port : ':' ID [ ':' compass_pt ] | ':' compass_pt
func (p *Printer) layoutPort(doc *layout.Doc, nodeIdx int) bool {
	var pendingColon, broke, needsSpace bool
	var colonLine uint32
	nr := p.tree.Children(nodeIdx)
	var prevEnd uint32
	for i := nr.Start; i < nr.End; i = p.tree.Next(i) {
		n := p.tree.NodeAt(i)
		if n.IsToken() {
			switch n.TokenKind {
			case token.Colon:
				pendingColon = true
				colonLine = n.End.Line
				prevEnd = n.End.Line
			case token.Comment:
				if pendingColon {
					doc.Text(token.Colon.String())
					pendingColon = false
				}
				pl := prevEnd
				if colonLine > 0 && pl == 0 {
					pl = colonLine
				}
				isTrailing := pl > 0 && n.Start.Line == pl
				needsSpace = p.layoutComment(doc, n.Literal, isTrailing)
				if !needsSpace {
					broke = true
				}
				colonLine = 0
				prevEnd = n.End.Line
			}
		} else {
			if n.Kind == dot.KindID || n.Kind == dot.KindCompassPoint {
				if tok, ok := p.tree.FirstToken(i, token.ID); ok && tok.Literal != "_" {
					if needsSpace {
						doc.Space()
					}
					if pendingColon {
						doc.Text(token.Colon.String())
					}
					idBroken := p.layoutID(doc, i)
					if idBroken {
						broke = true
						needsSpace = false
					} else {
						needsSpace = p.tree.HasComment(i)
					}
				}
				pendingColon = false
				colonLine = 0
				prevEnd = n.End.Line
			}
		}
	}
	return broke
}

// layoutAttrList handles: attr_list : '[' [ a_list ] ']' [ attr_list ]
func (p *Printer) layoutAttrList(doc *layout.Doc, nodeIdx int) {
	emittedBracket := false
	doc.Group(func(d *layout.Doc) {
		nr := p.tree.Children(nodeIdx)
		for i := nr.Start; i < nr.End; i = p.tree.Next(i) {
			n := p.tree.NodeAt(i)
			if n.IsToken() {
				switch n.TokenKind {
				case token.LeftBracket:
					if emittedBracket {
						doc.Space()
					}
					i = p.layoutBracketBlock(doc, nodeIdx, i)
					emittedBracket = true
				case token.Comment:
					p.layoutComment(doc, n.Literal, emittedBracket)
				}
			}
		}
	})
}

// layoutBracketBlock handles a single [...] block starting at index i (the '[').
func (p *Printer) layoutBracketBlock(doc *layout.Doc, parentIdx, startIdx int) int {
	nr := p.tree.Children(parentIdx)
	i := startIdx
	var prevEnd uint32
	doc.Group(func(d *layout.Doc) {
		doc.Text(token.LeftBracket.String()).
			BreakIf(1, layout.Broken).
			Indent(1, func(d *layout.Doc) {
				emittedAttr := false
				i = p.tree.Next(i) // skip '['
				for ; i < nr.End; i = p.tree.Next(i) {
					n := p.tree.NodeAt(i)
					if n.IsToken() {
						if n.TokenKind == token.RightBracket {
							break
						} else if n.TokenKind == token.Comment {
							isTrailing := prevEnd > 0 && n.Start.Line == prevEnd
							p.layoutComment(doc, n.Literal, isTrailing)
							prevEnd = n.End.Line
						}
					} else if n.Kind == dot.KindAList {
						emittedAttr = p.layoutAList(doc, i, emittedAttr)
						prevEnd = n.End.Line
					}
				}
			})
		doc.BreakIf(1, layout.Broken).Text(token.RightBracket.String())
	})
	return i
}

// layoutAList handles: a_list : ID '=' ID [ ( ';' | ',' ) ] [ a_list ]
func (p *Printer) layoutAList(doc *layout.Doc, nodeIdx int, emittedAttr bool) bool {
	nr := p.tree.Children(nodeIdx)
	var prevEnd uint32
	for i := nr.Start; i < nr.End; i = p.tree.Next(i) {
		n := p.tree.NodeAt(i)
		if n.IsToken() && n.TokenKind == token.Comment {
			isTrailing := prevEnd > 0 && n.Start.Line == prevEnd
			p.layoutComment(doc, n.Literal, isTrailing)
			prevEnd = n.End.Line
		} else if !n.IsToken() && n.Kind == dot.KindAttribute {
			if emittedAttr {
				doc.TextIf(token.Comma.String(), layout.Flat)
				doc.BreakIf(1, layout.Broken)
			}
			p.layoutAttribute(doc, i)
			emittedAttr = true
			prevEnd = n.End.Line
		}
	}
	return emittedAttr
}

// layoutEdgeStmt handles edge_stmt.
func (p *Printer) layoutEdgeStmt(doc *layout.Doc, nodeIdx int) {
	doc.Break(1)
	doc.Group(func(d *layout.Doc) {
		var needsSpace, lastBroke bool
		doc.Group(func(d *layout.Doc) {
			nr := p.tree.Children(nodeIdx)
			var prevEnd uint32
			for i := nr.Start; i < nr.End; i = p.tree.Next(i) {
				n := p.tree.NodeAt(i)
				if n.IsToken() {
					switch n.TokenKind {
					case token.DirectedEdge, token.UndirectedEdge:
						if needsSpace {
							doc.Space()
						}
						doc.Text(n.Literal)
						needsSpace = true
						prevEnd = n.End.Line
					case token.Comment:
						isTrailing := prevEnd > 0 && n.Start.Line == prevEnd
						needsSpace = p.layoutComment(doc, n.Literal, isTrailing)
						prevEnd = n.End.Line
					}
				} else {
					if n.Kind == dot.KindNodeID || n.Kind == dot.KindSubgraph {
						if needsSpace {
							doc.Space()
						}
						lastBroke = p.layoutEdgeOperand(doc, i)
						needsSpace = true
						prevEnd = n.End.Line
					}
				}
			}
		})
		if !lastBroke {
			doc.Space()
		}

		if attrList, ok := p.tree.LastTree(nodeIdx, dot.KindAttrList); ok {
			p.layoutAttrList(doc, attrList)
		}
	})
}

// layoutEdgeOperand handles a node_id or subgraph in an edge statement.
func (p *Printer) layoutEdgeOperand(doc *layout.Doc, nodeIdx int) bool {
	n := p.tree.NodeAt(nodeIdx)
	switch n.Kind {
	case dot.KindNodeID:
		return p.layoutNodeID(doc, nodeIdx)
	case dot.KindSubgraph:
		return p.layoutSubgraph(doc, nodeIdx)
	}
	return false
}

// layoutAttrStmt handles: attr_stmt : ( 'graph' | 'node' | 'edge' ) attr_list
func (p *Printer) layoutAttrStmt(doc *layout.Doc, nodeIdx int) {
	doc.Break(1).
		Group(func(d *layout.Doc) {
			if tok, ok := p.tree.TokenAt(nodeIdx, token.Graph|token.Node|token.Edge, 0); ok {
				doc.Text(tok.Literal)
			}
			if !p.layoutTrailingComments(doc, nodeIdx) {
				doc.Space()
			}
			if attrList, ok := p.tree.LastTree(nodeIdx, dot.KindAttrList); ok {
				p.layoutAttrList(doc, attrList)
			}
		})
}

// layoutAttribute handles: ID '=' ID
func (p *Printer) layoutAttribute(doc *layout.Doc, nodeIdx int) {
	var needsSpace bool
	nr := p.tree.Children(nodeIdx)
	var prevEnd uint32
	for i := nr.Start; i < nr.End; i = p.tree.Next(i) {
		n := p.tree.NodeAt(i)
		if n.IsToken() {
			switch n.TokenKind {
			case token.Equal:
				if needsSpace {
					doc.Space()
				}
				doc.Text(token.Equal.String())
				needsSpace = false
				prevEnd = n.End.Line
			case token.Comment:
				isTrailing := prevEnd > 0 && n.Start.Line == prevEnd
				needsSpace = p.layoutComment(doc, n.Literal, isTrailing)
				prevEnd = n.End.Line
			}
		} else {
			if n.Kind == dot.KindAttrName || n.Kind == dot.KindAttrValue {
				if id, ok := p.tree.TreeAt(i, dot.KindID, 0); ok {
					if needsSpace {
						doc.Space()
					}
					broke := p.layoutID(doc, id)
					needsSpace = !broke && p.tree.HasComment(id)
				}
				prevEnd = n.End.Line
			}
		}
	}
}

// layoutSubgraph handles: subgraph : [ 'subgraph' [ ID ] ] '{' stmt_list '}'
func (p *Printer) layoutSubgraph(doc *layout.Doc, nodeIdx int) bool {
	var broke bool
	doc.Group(func(f *layout.Doc) {
		broke = p.layoutBlock(doc, nodeIdx)
	})
	return broke
}

// isLineComment reports whether the comment literal is a line comment (// or #).
func isLineComment(s string) bool {
	return (len(s) > 0 && s[0] == '#') || (len(s) > 1 && s[0] == '/' && s[1] == '/')
}

func isTrailingLine(prevEndLine, commentStartLine uint32) bool {
	return prevEndLine > 0 && commentStartLine == prevEndLine
}
