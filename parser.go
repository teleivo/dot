// Package dot provides a parser for the [DOT language].
//
// The parser implements an error-resilient recursive descent parser that produces a concrete syntax
// tree (CST) representation of DOT source code. It can parse syntactically invalid input and
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
	"fmt"
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
	return fmt.Sprintf("%d:%d: %s", e.Pos.Line, e.Pos.Column, e.Msg)
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
	curToken  token.Token
	peekToken token.Token
	errors    []Error
	directed  bool // true if parsing a digraph, false for graph
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

// nextToken advances to the next non-comment token. Comments are currently skipped.
func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	for p.peekToken = p.scanner.Next(); p.peekToken.Type == token.Comment; p.peekToken = p.scanner.Next() {
	}
}

// Errors returns all parse and scan errors collected during parsing.
func (p *Parser) Errors() []Error {
	return p.errors
}

// Parse parses the DOT source code and returns the concrete syntax tree representation.
//
// The returned [Tree] has type [File] and contains zero or more graphs. Parse always returns a
// tree, even when errors are encountered. Syntax errors are collected and can be retrieved via
// [Parser.Errors].
func (p *Parser) Parse() *Tree {
	f := &Tree{}
	first := token.Strict | token.Graph | token.Digraph
	for !p.curTokenIs(token.EOF) {
		if p.curTokenIs(first) {
			graph := p.parseGraph()
			f.appendTree(graph)
		} else {
			p.wrapErrorExpected(f, first)
		}
	}
	f.Type = KindFile
	return f
}

// parseGraph parses a graph definition.
//
//	graph : [ 'strict' ] ( 'graph' | 'digraph' ) [ ID ] '{' stmt_list '}'
func (p *Parser) parseGraph() *Tree {
	assert.That(p.curTokenIs(token.Strict|token.Graph|token.Digraph), "current token must be strict, graph, or digraph, got %s", p.curToken)
	graph := &Tree{Type: KindGraph}

	okStrict := p.optional(graph, token.Strict)

	p.directed = p.curTokenIs(token.Digraph)
	defer func() { p.directed = false }()

	var okGraph bool
	if okStrict || p.curTokenIs(token.LeftBrace) {
		okGraph = p.expect(graph, token.Graph|token.Digraph)
	} else { // optional to avoid cascading error
		okGraph = p.optional(graph, token.Graph|token.Digraph)
	}

	if okGraph && p.curTokenIs(token.ID) {
		id := p.parseID()
		graph.appendTree(id)
	}

	// consume until we find a left brace, or EOF
	const recoverySet = token.Strict | token.Graph | token.Digraph
	for !p.curTokenIs(token.LeftBrace | token.EOF) {
		// a token in recovery set could indicate a new graph so we exit
		if p.curTokenIs(recoverySet) {
			break
		}

		// consume unexpected tokens as error
		if !okGraph { // give more context to error
			p.wrapErrorExpected(graph, token.Graph|token.Digraph)
		} else {
			p.wrapError(graph)
		}
	}

	var okLeft bool
	if okGraph {
		okLeft = p.expect(graph, token.LeftBrace)
	} else { // optional to avoid cascading error
		okLeft = p.optional(graph, token.LeftBrace)
	}

	if okLeft {
		stmts := p.parseStatementList(recoverySet)
		graph.appendTree(stmts)

		p.expect(graph, token.RightBrace)
	}

	return graph
}

// parseStatementList parses a list of statements.
//
//	stmt_list : [ stmt [ ';' ] stmt_list ]
//	stmt      : node_stmt | edge_stmt | attr_stmt | ID '=' ID | subgraph
func (p *Parser) parseStatementList(recoverySet token.Kind) *Tree {
	stmts := &Tree{}
	recoverySet |= token.RightBrace | token.Semicolon
	for !p.curTokenIs(token.RightBrace | token.EOF) {
		if p.curTokenIs(token.ID) && p.peekTokenIs(token.Equal) { // ID '=' ID
			stmt := p.parseAttribute()
			stmts.appendTree(stmt)
		} else if p.curTokenIs(token.Edge | token.Graph | token.Node) { // attr_stmt  : (graph | node | edge) attr_list
			stmt := &Tree{}
			p.consume(stmt)

			if p.curTokenIs(token.LeftBracket) { // attr_list is required
				attrs := p.parseAttrList(recoverySet | token.Edge | token.Graph | token.Node)
				stmt.appendTree(attrs)
			} else {
				p.error("expected [ to start attribute list")
			}

			stmt.Type = KindAttrStmt
			stmts.appendTree(stmt)
		} else if p.curTokenIs(token.ID | token.Subgraph | token.LeftBrace) { // edge_stmt | node_stmt | subgraph
			// Parse the operand (node_id or subgraph)
			var operand *Tree
			var isSubgraph bool
			if p.curTokenIs(token.ID) {
				operand = p.parseNodeID()
			} else {
				operand = p.parseSubgraph(recoverySet)
				isSubgraph = true
			}

			if p.curTokenIs(token.UndirectedEdge | token.DirectedEdge) { // edge_stmt
				stmt := &Tree{Type: KindEdgeStmt}
				stmt.appendTree(operand)
				p.parseEdgeRHS(stmt, recoverySet)
				if p.curTokenIs(token.LeftBracket) {
					attrs := p.parseAttrList(recoverySet | token.Edge | token.Graph | token.Node)
					stmt.appendTree(attrs)
				}
				stmts.appendTree(stmt)
			} else if isSubgraph { // standalone subgraph
				stmts.appendTree(operand)
			} else { // node_stmt
				stmt := &Tree{Type: KindNodeStmt}
				stmt.appendTree(operand)
				if p.curTokenIs(token.LeftBracket) {
					attrs := p.parseAttrList(recoverySet | token.Edge | token.Graph | token.Node)
					stmt.appendTree(attrs)
				}
				stmts.appendTree(stmt)
			}
		} else if p.curTokenIs(token.Semicolon) {
			p.consume(stmts)
		} else if p.curTokenIs(recoverySet) {
			break
		} else {
			// we must consume the current token to make progress if we didn't parse a statement,
			// didn't consume a semicolon, and cannot recover in parent
			p.wrapErrorMsg(stmts, "cannot start a statement")
		}
	}

	stmts.Type = KindStmtList
	return stmts
}

// parseEdgeRHS parses the right-hand side of an edge statement.
//
//	edgeRHS : edgeop ( node_id | subgraph ) [ edgeRHS ]
//
// Where edgeop is '--' for undirected graphs and '->' for directed graphs.
func (p *Parser) parseEdgeRHS(stmt *Tree, recoverySet token.Kind) {
	assert.That(p.curTokenIs(token.DirectedEdge|token.UndirectedEdge), "current token must be directed or undirected edge, got %s", p.curToken)

	for p.curTokenIs(token.DirectedEdge | token.UndirectedEdge) {
		if p.directed && p.curTokenIs(token.UndirectedEdge) {
			p.error("expected '->' for edge in directed graph")
		} else if !p.directed && p.curTokenIs(token.DirectedEdge) {
			p.error("expected '--' for edge in undirected graph")
		}
		p.consume(stmt)

		if p.curTokenIs(token.ID) {
			operand := p.parseNodeID()
			stmt.appendTree(operand)
		} else if p.curTokenIs(token.LeftBrace | token.Subgraph) {
			operand := p.parseSubgraph(recoverySet)
			stmt.appendTree(operand)
		} else if p.curTokenIs(recoverySet) {
			p.error("expected node or subgraph as edge operand")
			break
		} else {
			// consume the current token to make progress
			p.wrapErrorMsg(stmt, "is not a valid edge operand")
		}
	}
}

// parseNodeID parses a node identifier with optional port.
//
//	node_id : ID [ port ]
func (p *Parser) parseNodeID() *Tree {
	assert.That(p.curTokenIs(token.ID), "current token must be ID, got %s", p.curToken)

	nid := &Tree{Type: KindNodeID}
	id := p.parseID()
	nid.appendTree(id)

	if p.curTokenIs(token.Colon) {
		port := p.parsePort()
		nid.appendTree(port)
	}

	return nid
}

// parseID parses an identifier.
func (p *Parser) parseID() *Tree {
	assert.That(p.curTokenIs(token.ID), "current token must be ID, got %s", p.curToken)

	id := &Tree{Type: KindID}
	p.expect(id, token.ID)
	return id
}

// parsePort parses a port specification.
//
//	port       : ':' ID [ ':' compass_pt ] | ':' compass_pt
//	compass_pt : 'n' | 'ne' | 'e' | 'se' | 's' | 'sw' | 'w' | 'nw' | 'c' | '_'
func (p *Parser) parsePort() *Tree {
	assert.That(p.curTokenIs(token.Colon), "current token must be colon, got %s", p.curToken)

	port := &Tree{Type: KindPort}
	p.expect(port, token.Colon)

	firstCompass := p.curToken.IsCompassPoint()
	var firstID *Tree
	if p.curTokenIs(token.ID) {
		firstID = p.parseID()
		port.appendTree(firstID)
	} else {
		p.error("expected ID for port")
	}

	if p.curTokenIs(token.Colon) {
		p.expect(port, token.Colon)

		secondCompass := p.curToken.IsCompassPoint()
		if p.curTokenIs(token.ID) {
			secondID := p.parseID()
			if secondCompass {
				secondID.Type = KindCompassPoint
			} else {
				p.error("expected compass point (c, e, n, ne, nw, s, se, sw, w, or _)")
			}
			port.appendTree(secondID)
		} else {
			p.error("expected compass point (c, e, n, ne, nw, s, se, sw, w, or _)")
		}
	} else {
		// first is only a compass point if its one and there is no second :
		if firstCompass {
			firstID.Type = KindCompassPoint
		}
	}
	return port
}

// parseAttrList parses an attribute list.
//
//	attr_list : '[' [ a_list ] ']' [ attr_list ]
func (p *Parser) parseAttrList(recoverySet token.Kind) *Tree {
	assert.That(p.curTokenIs(token.LeftBracket), "current token must be [, got %s", p.curToken)

	attrList := &Tree{Type: KindAttrList}
	for p.curTokenIs(token.LeftBracket) && !p.curTokenIs(token.EOF) {
		p.consume(attrList)

		if p.curTokenIs(token.ID) { // a_list is optional
			aList := p.parseAList(recoverySet | token.LeftBracket | token.RightBracket)
			attrList.appendTree(aList)
		}

		if p.curTokenIs(token.RightBracket) {
			p.consume(attrList)
		} else {
			p.error("expected ] to close attribute list")
		}
	}

	return attrList
}

// parseAList parses a list of attributes within brackets.
//
//	a_list : ID '=' ID [ ( ';' | ',' ) ] [ a_list ]
func (p *Parser) parseAList(recoverySet token.Kind) *Tree {
	assert.That(p.curTokenIs(token.ID), "current token must be ID, got %s", p.curToken)

	var hasID bool
	aList := &Tree{Type: KindAList}
	for !p.curTokenIs(token.RightBracket) && !p.curTokenIs(token.EOF) {
		if p.curTokenIs(token.ID) {
			hasID = true
			attr := p.parseAttribute()
			aList.appendTree(attr)

			if p.curTokenIs(token.Semicolon | token.Comma) { // ; and , are optional
				p.consume(aList)
			}
		} else if p.curTokenIs(recoverySet) {
			if !hasID {
				p.error("expected attribute name")
			}
			break
		} else if !p.curTokenIs(token.LeftBracket) {
			p.wrapErrorMsg(aList, "is not a valid attribute name")
		}
	}

	return aList
}

// parseAttribute parses a single attribute.
//
//	ID '=' ID
func (p *Parser) parseAttribute() *Tree {
	assert.That(p.curTokenIs(token.ID), "current token must be ID, got %s", p.curToken)

	attr := &Tree{Type: KindAttribute}
	id := p.parseID()
	attr.appendTree(id)

	okEqual := p.expect(attr, token.Equal)

	if p.curTokenIs(token.ID) {
		id := p.parseID()
		attr.appendTree(id)
	} else if okEqual { // reduce noise by only reporting missing rhs ID if we've seen a =
		p.error("expected attribute value")
	}

	return attr
}

// parseSubgraph parses a subgraph definition.
//
//	subgraph : [ 'subgraph' [ ID ] ] '{' stmt_list '}'
func (p *Parser) parseSubgraph(recoverySet token.Kind) *Tree {
	assert.That(p.curTokenIs(token.LeftBrace|token.Subgraph), "current token must be { or subgraph, got %s", p.curToken)
	subgraph := &Tree{Type: KindSubgraph}

	okSubgraph := p.optional(subgraph, token.Subgraph)

	if okSubgraph && p.curTokenIs(token.ID) {
		id := p.parseID()
		subgraph.appendTree(id)
	}

	// consume until we find a left brace, or EOF
	for !p.curTokenIs(token.LeftBrace | token.EOF) {
		// a token in recovery set could indicate a new graph so we exit
		if p.curTokenIs(recoverySet) {
			break
		}

		// consume unexpected tokens as error
		if !okSubgraph { // give more context to error
			p.wrapErrorExpected(subgraph, token.Subgraph)
		} else {
			p.wrapError(subgraph)
		}
	}

	var okLeft bool
	if okSubgraph {
		okLeft = p.expect(subgraph, token.LeftBrace)
	} else { // optional to avoid cascading error
		okLeft = p.optional(subgraph, token.LeftBrace)
	}

	if okLeft {
		stmts := p.parseStatementList(recoverySet)
		subgraph.appendTree(stmts)

		p.expect(subgraph, token.RightBrace)
	}

	return subgraph
}

func (p *Parser) curTokenIs(t token.Kind) bool {
	return p.curToken.Type&t != 0
}

func (p *Parser) peekTokenIs(t token.Kind) bool {
	return p.peekToken.Type&t != 0
}

// optional checks if the current token matches one of the wanted kinds. If it does, consumes it.
// Returns true if the token was consumed, false otherwise.
func (p *Parser) optional(t *Tree, want token.Kind) bool {
	if p.curTokenIs(want) {
		t.appendToken(p.curToken)
		p.nextToken()
		return true
	}
	return false
}

// expect checks if the current token matches one of the wanted kinds. If it does, consumes it.
// If not, reports an error but does NOT advance.
// Returns true if the token was consumed, false otherwise.
func (p *Parser) expect(t *Tree, want token.Kind) bool {
	if p.curTokenIs(want) {
		t.appendToken(p.curToken)
		p.nextToken()
		return true
	}

	p.errorExpected(want)

	return false
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

func (p *Parser) consume(t *Tree) {
	t.appendToken(p.curToken)
	p.nextToken()
}

// wrapError consumes curToken into ErrorTree, records error, advances.
// For ERROR tokens, uses the scanner's error message; otherwise records "unexpected token X".
func (p *Parser) wrapError(t *Tree) {
	errTree := &Tree{Type: KindErrorTree}
	errTree.appendToken(p.curToken)
	t.appendTree(errTree)

	if p.curToken.Type == token.ERROR { // scanner error
		p.error(p.curToken.Error)
	} else { // parsing error
		var msg strings.Builder
		msg.WriteString("unexpected token ")
		writeToken(p.curToken, &msg)
		p.error(msg.String())
	}

	p.nextToken()
}

// wrapErrorMsg consumes curToken into ErrorTree, records error, advances.
// For ERROR tokens, uses the scanner's error message; otherwise records "'X' msg".
func (p *Parser) wrapErrorMsg(t *Tree, suffix string) {
	errTree := &Tree{Type: KindErrorTree}
	errTree.appendToken(p.curToken)
	t.appendTree(errTree)

	if p.curToken.Type == token.ERROR { // scanner error
		p.error(p.curToken.Error)
	} else { // parsing error
		var msg strings.Builder
		writeToken(p.curToken, &msg)
		msg.WriteByte(' ')
		msg.WriteString(suffix)
		p.error(msg.String())
	}

	p.nextToken()
}

// wrapErrorExpected consumes curToken into ErrorTree, records error, advances.
// For ERROR tokens, uses the scanner's error message; otherwise records "unexpected token X, expected Y".
func (p *Parser) wrapErrorExpected(t *Tree, want token.Kind) {
	errTree := &Tree{Type: KindErrorTree}
	errTree.appendToken(p.curToken)
	t.appendTree(errTree)

	if p.curToken.Type == token.ERROR { // scanner error
		p.error(p.curToken.Error)
	} else { // parsing error
		var msg strings.Builder
		msg.WriteString("unexpected token ")
		writeToken(p.curToken, &msg)
		msg.WriteString(", expected ")
		writeExpected(want, &msg)
		p.error(msg.String())
	}

	p.nextToken()
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

	if tok.Type == token.ID {
		msg.WriteString(tok.Type.String())
		msg.WriteRune(' ')
	}
	msg.WriteRune('\'')
	msg.WriteString(tok.Literal)
	msg.WriteRune('\'')
}
