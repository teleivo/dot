// Package dot provides a parser for the [DOT language].
//
// The parser implements an error-resilient recursive descent parser that produces a concrete syntax
// tree representation of DOT source code. It can parse syntactically invalid input and
// recover to continue parsing, collecting all errors encountered during parsing.
//
// # DOT Grammar
//
// The parser implements the following grammar from the DOT language specification:
//
//	graph      : [ 'strict' ] ( 'graph' | 'digraph' ) [ ID ] '{' stmt_list '}'
//	stmt_list  : [ stmt [ ';' ] stmt_list ]
//	stmt       : node_stmt | edge_stmt | attr_stmt | ID '=' ID | subgraph
//	attr_stmt  : ( 'graph' | 'node' | 'edge' ) attr_list
//	attr_list  : '[' [ a_list ] ']' [ attr_list ]
//	a_list     : ID '=' ID [ ( ';' | ',' ) ] [ a_list ]
//	edge_stmt  : ( node_id | subgraph ) edgeRHS [ attr_list ]
//	edgeRHS    : edgeop ( node_id | subgraph ) [ edgeRHS ]
//	node_stmt  : node_id [ attr_list ]
//	node_id    : ID [ port ]
//	port       : ':' ID [ ':' compass_pt ] | ':' compass_pt
//	subgraph   : [ 'subgraph' [ ID ] ] '{' stmt_list '}'
//	compass_pt : 'n' | 'ne' | 'e' | 'se' | 's' | 'sw' | 'w' | 'nw' | 'c' | '_'
//
// Where edgeop is '--' for undirected graphs and '->' for directed graphs.
//
// [DOT language]: https://graphviz.org/doc/info/lang.html
package dot

import (
	"strings"

	"github.com/teleivo/dot/internal/assert"
	"github.com/teleivo/dot/token"
)

// Error represents a parse error in DOT source code.
// The position Pos points to the beginning of the offending token, and the error condition is
// described by Msg.
type Error struct {
	Pos token.Position
	Msg string
}

// Error formats the error as "line:column: message".
func (e Error) Error() string {
	return e.Pos.String() + ": " + e.Msg
}

// Parser parses DOT language source code into a concrete syntax tree.
//
// Parser continues parsing after encountering errors, collecting all errors for later retrieval
// via [Parser.Errors].
//
// The parser uses one token of lookahead (LL(1)) and produces a [Tree] that preserves all tokens
// from the source.
type Parser struct {
	scanner   *Scanner
	prevToken token.Token
	curToken  token.Token
	peekToken token.Token
	comments  []token.Token
	errors    []Error
	directed  bool // true if parsing a digraph, false for graph
	tree      Tree // flat tree built by Parse
}

// NewParser creates a new parser that parses the given DOT source code.
func NewParser(src []byte) *Parser {
	scanner := NewScanner(src)

	p := Parser{
		scanner: scanner,
	}

	// initialize current and peek token
	p.nextToken()
	p.nextToken()

	return &p
}

// nextToken advances to the next non-comment token. Comments are buffered in p.comments
// and flushed by addToken when the next non-comment token is consumed.
func (p *Parser) nextToken() {
	p.prevToken = p.curToken
	p.curToken = p.peekToken
	for p.peekToken = p.scanner.Next(); p.peekToken.Kind == token.Comment; p.peekToken = p.scanner.Next() {
		p.comments = append(p.comments, p.peekToken)
	}
}

// Errors returns all parse and scan errors collected during parsing.
func (p *Parser) Errors() []Error {
	return p.errors
}

// Parse parses the DOT source code and returns the concrete syntax tree.
//
// The returned [Tree] has a root node of type [KindFile] and contains zero or more graphs. Parse
// always returns a tree, even when errors are encountered. Syntax errors are collected and can be
// retrieved via [Parser.Errors].
func (p *Parser) Parse() *Tree {
	p.tree.nodes = p.tree.nodes[:0]

	fileIdx := p.openNode(KindFile)
	first := token.Strict | token.Graph | token.Digraph
	for !p.curTokenIs(token.EOF) {
		if p.curTokenIs(first) {
			p.parseGraph(fileIdx)
		} else {
			p.wrapErrorExpected(fileIdx, first)
		}
	}
	for _, comment := range p.comments {
		p.addTokenDirect(comment)
	}
	p.comments = p.comments[:0]
	p.closeNode(fileIdx)
	return &p.tree
}

func (p *Parser) openNode(kind TreeKind) int {
	i := len(p.tree.nodes)
	p.tree.nodes = append(p.tree.nodes, Node{Kind: kind})
	return i
}

func (p *Parser) closeNode(i int) {
	n := &p.tree.nodes[i]
	n.len = len(p.tree.nodes) - i - 1
	if n.len > 0 {
		// Set Start/End from first and last descendant
		first := p.tree.nodes[i+1]
		last := p.tree.nodes[len(p.tree.nodes)-1]
		n.Start = first.Start
		n.End = last.End
	}
}

func (p *Parser) addTokenDirect(tok token.Token) {
	p.tree.nodes = append(p.tree.nodes, Node{
		TokenKind: tok.Kind,
		Start:     tok.Start,
		End:       tok.End,
		Literal:   tok.Literal,
	})
}

// addToken appends the current token to the tree, flushing any buffered comments.
// parentIdx is used for own-line leading comments (they become siblings to the current construct).
func (p *Parser) addToken(parentIdx int) {
	_ = parentIdx // own-line comments are handled before openNode in each parse function

	// leading comments on same line as current token
	remaining := p.comments[:0]
	for _, comment := range p.comments {
		if comment.Start.Before(p.curToken.Start) {
			if comment.Start.Line == p.curToken.Start.Line {
				p.addTokenDirect(comment)
			} else {
				// own-line comment — will be placed by caller before openNode
				remaining = append(remaining, comment)
			}
		} else {
			remaining = append(remaining, comment)
		}
	}
	p.comments = remaining

	p.addTokenDirect(p.curToken)

	// trailing comments on same line
	remaining = p.comments[:0]
	for _, comment := range p.comments {
		if p.curToken.End.IsValid() && p.curToken.End.Before(comment.Start) && p.curToken.End.Line == comment.Start.Line {
			p.addTokenDirect(comment)
		} else {
			remaining = append(remaining, comment)
		}
	}
	p.comments = remaining

	p.nextToken()
}

// flushOwnLineComments appends own-line comments (before curToken, on different line) to the tree.
func (p *Parser) flushOwnLineComments() {
	remaining := p.comments[:0]
	for _, comment := range p.comments {
		if comment.Start.Before(p.curToken.Start) && comment.Start.Line != p.curToken.Start.Line {
			p.addTokenDirect(comment)
		} else {
			remaining = append(remaining, comment)
		}
	}
	p.comments = remaining
}

// consume appends the current token and advances.
func (p *Parser) consume() {
	p.addToken(0) // parent doesn't matter for consume (own-line stays in current construct)
}

func (p *Parser) expect(want token.Kind) bool {
	if p.curTokenIs(want) {
		p.consume()
		return true
	}
	p.errorExpected(want)
	return false
}

func (p *Parser) expectWithComments(parentIdx int, want token.Kind) bool {
	if p.curTokenIs(want) {
		p.flushOwnLineComments()
		p.addToken(parentIdx)
		return true
	}
	p.errorExpected(want)
	return false
}

func (p *Parser) optionalWithComments(parentIdx int, want token.Kind) bool {
	if p.curTokenIs(want) {
		p.flushOwnLineComments()
		p.addToken(parentIdx)
		return true
	}
	return false
}

func (p *Parser) curTokenIs(t token.Kind) bool {
	return p.curToken.Kind&t != 0
}

func (p *Parser) peekTokenIs(t token.Kind) bool {
	return p.peekToken.Kind&t != 0
}

// error records a parse error at current position with custom message.
func (p *Parser) error(msg string) {
	p.errors = append(p.errors, Error{
		Pos: p.curToken.Start,
		Msg: msg,
	})
}

// errorExpected records "expected X or Y" at current position.
func (p *Parser) errorExpected(want token.Kind) {
	var msg strings.Builder
	msg.WriteString("expected ")
	writeExpected(want, &msg)
	p.error(msg.String())
}

func (p *Parser) parseGraph(parentIdx int) {
	assert.That(p.curTokenIs(token.Strict|token.Graph|token.Digraph), "current token must be strict, graph, or digraph, got %s", p.curToken)

	p.flushOwnLineComments()
	graphIdx := p.openNode(KindGraph)

	okStrict := p.optionalWithComments(parentIdx, token.Strict)

	p.directed = p.curTokenIs(token.Digraph)
	defer func() { p.directed = false }()

	var okGraph bool
	if okStrict || p.curTokenIs(token.LeftBrace) {
		okGraph = p.expectWithComments(graphIdx, token.Graph|token.Digraph)
	} else {
		okGraph = p.optionalWithComments(graphIdx, token.Graph|token.Digraph)
	}

	if okGraph && p.curTokenIs(token.ID) {
		p.parseID()
	}

	const recoverySet = token.Strict | token.Graph | token.Digraph
	for !p.curTokenIs(token.LeftBrace | token.EOF) {
		if p.curTokenIs(recoverySet) {
			break
		}
		if !okGraph {
			p.wrapErrorExpected(graphIdx, token.Graph|token.Digraph)
		} else {
			p.wrapError(graphIdx)
		}
	}

	var okLeft bool
	if okGraph {
		okLeft = p.expectWithComments(parentIdx, token.LeftBrace)
	} else {
		okLeft = p.optionalWithComments(parentIdx, token.LeftBrace)
	}

	if okLeft {
		p.parseStatementList(recoverySet)
		p.expectWithComments(graphIdx, token.RightBrace)
	}

	p.closeNode(graphIdx)
}

func (p *Parser) parseStatementList(recoverySet token.Kind) {
	stmtsIdx := p.openNode(KindStmtList)
	recoverySet |= token.RightBrace | token.Semicolon
	for !p.curTokenIs(token.RightBrace | token.EOF) {
		if p.curTokenIs(token.ID) && p.peekTokenIs(token.Equal) {
			p.parseAttribute(stmtsIdx)
		} else if p.curTokenIs(token.Edge | token.Graph | token.Node) {
			stmtIdx := p.openNode(0) // kind set later
			p.consume()
			if p.curTokenIs(token.LeftBracket) {
				p.parseAttrList(stmtsIdx, recoverySet|token.Edge|token.Graph|token.Node)
			} else {
				p.error("expected [ to start attribute list")
			}
			p.tree.nodes[stmtIdx].Kind = KindAttrStmt
			p.closeNode(stmtIdx)
		} else if p.curTokenIs(token.ID | token.Subgraph | token.LeftBrace) {
			if p.curTokenIs(token.ID) {
				nodeIDIdx := p.parseNodeID(stmtsIdx)
				if p.curTokenIs(token.UndirectedEdge | token.DirectedEdge) {
					_ = nodeIDIdx
					edgeIdx := p.wrapInNode(nodeIDIdx, KindEdgeStmt)
					p.parseEdgeRHS(stmtsIdx, recoverySet)
					if p.curTokenIs(token.LeftBracket) {
						p.parseAttrList(stmtsIdx, recoverySet|token.Edge|token.Graph|token.Node)
					}
					p.closeNode(edgeIdx)
				} else {
					nodeStmtIdx := p.wrapInNode(nodeIDIdx, KindNodeStmt)
					if p.curTokenIs(token.LeftBracket) {
						p.parseAttrList(stmtsIdx, recoverySet|token.Edge|token.Graph|token.Node)
					}
					p.closeNode(nodeStmtIdx)
				}
			} else {
				subIdx := p.parseSubgraph(stmtsIdx, recoverySet)
				if p.curTokenIs(token.UndirectedEdge | token.DirectedEdge) {
					edgeIdx := p.wrapInNode(subIdx, KindEdgeStmt)
					p.parseEdgeRHS(stmtsIdx, recoverySet)
					if p.curTokenIs(token.LeftBracket) {
						p.parseAttrList(stmtsIdx, recoverySet|token.Edge|token.Graph|token.Node)
					}
					p.closeNode(edgeIdx)
				}
			}
		} else if p.curTokenIs(token.Semicolon) {
			p.consume()
		} else if p.curTokenIs(recoverySet) {
			break
		} else {
			p.wrapErrorMsg(stmtsIdx, "cannot start a statement")
		}
	}
	p.tree.nodes[stmtsIdx].Kind = KindStmtList
	p.closeNode(stmtsIdx)
}

// wrapInNode inserts a new tree node before childIdx that will contain childIdx and subsequent
// nodes as children. Returns the index of the new wrapper node.
func (p *Parser) wrapInNode(childIdx int, kind TreeKind) int {
	// Insert a node at childIdx position, shifting everything after it
	p.tree.nodes = append(p.tree.nodes, Node{})
	copy(p.tree.nodes[childIdx+1:], p.tree.nodes[childIdx:])
	p.tree.nodes[childIdx] = Node{Kind: kind}
	return childIdx
}

func (p *Parser) parseEdgeRHS(parentIdx int, recoverySet token.Kind) {
	assert.That(p.curTokenIs(token.DirectedEdge|token.UndirectedEdge), "current token must be directed or undirected edge, got %s", p.curToken)

	for p.curTokenIs(token.DirectedEdge | token.UndirectedEdge) {
		if p.directed && p.curTokenIs(token.UndirectedEdge) {
			p.error("expected '->' for edge in directed graph")
		} else if !p.directed && p.curTokenIs(token.DirectedEdge) {
			p.error("expected '--' for edge in undirected graph")
		}
		p.consume()

		if p.curTokenIs(token.ID) {
			p.parseNodeID(parentIdx)
		} else if p.curTokenIs(token.LeftBrace | token.Subgraph) {
			p.parseSubgraph(parentIdx, recoverySet)
		} else if p.curTokenIs(recoverySet) {
			p.error("expected node or subgraph as edge operand")
			break
		} else {
			p.wrapErrorMsg(parentIdx, "is not a valid edge operand")
		}
	}
}

func (p *Parser) parseNodeID(parentIdx int) int {
	assert.That(p.curTokenIs(token.ID), "current token must be ID, got %s", p.curToken)

	p.flushOwnLineComments()
	nidIdx := p.openNode(KindNodeID)
	p.parseID()

	if p.curTokenIs(token.Colon) {
		p.parsePort(parentIdx)
	}

	p.closeNode(nidIdx)
	return nidIdx
}

func (p *Parser) parseID() {
	assert.That(p.curTokenIs(token.ID), "current token must be ID, got %s", p.curToken)

	idIdx := p.openNode(KindID)
	p.consume()
	p.closeNode(idIdx)
}

func (p *Parser) parsePort(parentIdx int) {
	assert.That(p.curTokenIs(token.Colon), "current token must be colon, got %s", p.curToken)

	portIdx := p.openNode(KindPort)
	p.expectWithComments(parentIdx, token.Colon)

	firstCompass := p.curToken.IsCompassPoint()
	var firstIDIdx int
	if p.curTokenIs(token.ID) {
		firstIDIdx = len(p.tree.nodes)
		p.parseID()
	} else {
		p.error("expected ID for port")
	}

	if p.curTokenIs(token.Colon) {
		p.expectWithComments(portIdx, token.Colon)

		secondCompass := p.curToken.IsCompassPoint()
		if p.curTokenIs(token.ID) {
			secondIDIdx := len(p.tree.nodes)
			p.parseID()
			if secondCompass {
				p.tree.nodes[secondIDIdx].Kind = KindCompassPoint
			} else {
				p.error("expected compass point (c, e, n, ne, nw, s, se, sw, w, or _)")
			}
		} else {
			p.error("expected compass point (c, e, n, ne, nw, s, se, sw, w, or _)")
		}
	} else if firstCompass && firstIDIdx > 0 {
		p.tree.nodes[firstIDIdx].Kind = KindCompassPoint
	}

	p.closeNode(portIdx)
}

func (p *Parser) parseAttrList(parentIdx int, recoverySet token.Kind) {
	assert.That(p.curTokenIs(token.LeftBracket), "current token must be [, got %s", p.curToken)

	attrListIdx := p.openNode(KindAttrList)
	for p.curTokenIs(token.LeftBracket) && !p.curTokenIs(token.EOF) {
		p.consume()

		if p.curTokenIs(token.ID) {
			p.parseAList(parentIdx, recoverySet|token.LeftBracket|token.RightBracket)
		}

		if p.curTokenIs(token.RightBracket) {
			p.consume()
		} else {
			p.error("expected ] to close attribute list")
		}
	}

	p.closeNode(attrListIdx)
}

func (p *Parser) parseAList(parentIdx int, recoverySet token.Kind) {
	assert.That(p.curTokenIs(token.ID), "current token must be ID, got %s", p.curToken)

	var hasID bool
	aListIdx := p.openNode(KindAList)
	for !p.curTokenIs(token.RightBracket) && !p.curTokenIs(token.EOF) {
		if p.curTokenIs(token.ID) {
			hasID = true
			p.parseAttribute(parentIdx)

			if p.curTokenIs(token.Semicolon | token.Comma) {
				p.consume()
			}
		} else if p.curTokenIs(recoverySet) {
			if !hasID {
				p.error("expected attribute name")
			}
			break
		} else if !p.curTokenIs(token.LeftBracket) {
			p.wrapErrorMsg(aListIdx, "is not a valid attribute name")
		}
	}

	p.closeNode(aListIdx)
}

func (p *Parser) parseAttribute(parentIdx int) {
	assert.That(p.curTokenIs(token.ID), "current token must be ID, got %s", p.curToken)

	p.flushOwnLineComments()
	attrIdx := p.openNode(KindAttribute)

	nameIdx := p.openNode(KindAttrName)
	p.parseID()
	p.closeNode(nameIdx)

	okEqual := p.expect(token.Equal)

	if p.curTokenIs(token.ID) {
		valueIdx := p.openNode(KindAttrValue)
		p.parseID()
		p.closeNode(valueIdx)
	} else if okEqual {
		p.error("expected attribute value")
	}

	p.closeNode(attrIdx)
}

func (p *Parser) parseSubgraph(parentIdx int, recoverySet token.Kind) int {
	assert.That(p.curTokenIs(token.LeftBrace|token.Subgraph), "current token must be { or subgraph, got %s", p.curToken)

	p.flushOwnLineComments()
	subIdx := p.openNode(KindSubgraph)

	okSubgraph := p.optionalWithComments(parentIdx, token.Subgraph)

	if okSubgraph && p.curTokenIs(token.ID) {
		p.parseID()
	}

	for !p.curTokenIs(token.LeftBrace | token.EOF) {
		if p.curTokenIs(recoverySet) {
			break
		}
		if !okSubgraph {
			p.wrapErrorExpected(subIdx, token.Subgraph)
		} else {
			p.wrapError(subIdx)
		}
	}

	var okLeft bool
	if okSubgraph {
		okLeft = p.expectWithComments(parentIdx, token.LeftBrace)
	} else {
		okLeft = p.optionalWithComments(parentIdx, token.LeftBrace)
	}

	if okLeft {
		p.parseStatementList(recoverySet)
		p.expectWithComments(subIdx, token.RightBrace)
	}

	p.closeNode(subIdx)
	return subIdx
}

// wrapError consumes curToken into ErrorTree, records error, advances.
// For ERROR tokens, uses the scanner's error message; otherwise records "unexpected token X".
func (p *Parser) wrapError(parentIdx int) {
	if p.curToken.Kind == token.ERROR {
		p.error(p.curToken.Error)
	} else {
		var msg strings.Builder
		msg.WriteString("unexpected token ")
		writeToken(p.curToken, &msg)
		p.error(msg.String())
	}

	errIdx := p.openNode(KindErrorTree)
	p.consume()
	p.closeNode(errIdx)
}

// wrapErrorMsg consumes curToken into ErrorTree, records error, advances.
// For ERROR tokens, uses the scanner's error message; otherwise records "'X' msg".
func (p *Parser) wrapErrorMsg(parentIdx int, suffix string) {
	if p.curToken.Kind == token.ERROR {
		p.error(p.curToken.Error)
	} else {
		var msg strings.Builder
		writeToken(p.curToken, &msg)
		msg.WriteByte(' ')
		msg.WriteString(suffix)
		p.error(msg.String())
	}

	errIdx := p.openNode(KindErrorTree)
	p.consume()
	p.closeNode(errIdx)
}

// wrapErrorExpected consumes curToken into ErrorTree, records error, advances.
// For ERROR tokens, uses the scanner's error message; otherwise records "unexpected token X, expected Y".
func (p *Parser) wrapErrorExpected(parentIdx int, want token.Kind) {
	if p.curToken.Kind == token.ERROR {
		p.error(p.curToken.Error)
	} else {
		var msg strings.Builder
		msg.WriteString("unexpected token ")
		writeToken(p.curToken, &msg)
		msg.WriteString(", expected ")
		writeExpected(want, &msg)
		p.error(msg.String())
	}

	errIdx := p.openNode(KindErrorTree)
	p.consume()
	p.closeNode(errIdx)
}

// writeExpected writes "X, Y or Z" to w based on token.Kind bitmask.
func writeExpected(want token.Kind, w *strings.Builder) {
	// Pre-allocate for max token kinds in want. Increase capacity if callers pass more tokens.
	tokens := make([]token.Kind, 0, 4)
	for remaining := want; remaining != 0; {
		bit := remaining & -remaining
		tokens = append(tokens, bit)
		remaining &^= bit
	}

	for i, t := range tokens {
		if i > 0 {
			if i == len(tokens)-1 {
				w.WriteString(" or ")
			} else {
				w.WriteString(", ")
			}
		}
		w.WriteString(t.String())
	}
}

func writeToken(tok token.Token, msg *strings.Builder) {
	if tok.IsKeyword() {
		msg.WriteString(tok.Literal)
		return
	}

	if tok.Kind == token.ID {
		msg.WriteString(tok.Kind.String())
		msg.WriteRune(' ')
	}
	msg.WriteRune('\'')
	msg.WriteString(tok.Literal)
	msg.WriteRune('\'')
}
