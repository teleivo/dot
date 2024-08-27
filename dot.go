// Package dot provides a parser for the dot language https://graphviz.org/doc/info/lang.html.
package dot

import (
	"fmt"
	"io"
	"slices"

	dot "github.com/teleivo/dot/internal"
	"github.com/teleivo/dot/internal/ast"
	"github.com/teleivo/dot/internal/token"
)

type Parser struct {
	lexer     *dot.Lexer
	curToken  token.Token
	peekToken token.Token
}

func New(r io.Reader) (*Parser, error) {
	lexer, err := dot.NewLexer(r)
	if err != nil {
		return nil, err
	}

	p := Parser{
		lexer: lexer,
	}

	// initialize peek token
	err = p.nextToken()
	if err != nil {
		return nil, err
	}

	return &p, nil
}

func (p *Parser) nextToken() error {
	p.curToken = p.peekToken
	tok, err := p.lexer.NextToken()
	if err != nil {
		return err
	}
	p.peekToken = tok
	fmt.Printf("%#v\n", p)

	return nil
}

func (p *Parser) Parse() (ast.Graph, error) {
	// if p.isDone() {
	if p.peekTokenIs(token.EOF) {
		var graph ast.Graph
		return graph, nil
	}

	graph, err := p.parseHeader()
	if err != nil {
		return graph, err
	}

	err = p.expectPeekTokenIsOneOf(token.LeftBrace)
	if err != nil {
		return graph, err
	}

	for ; !p.curTokenIs(token.EOF) && err == nil; err = p.nextToken() {
		// TODO move the append out
		switch p.curToken.Type {
		case token.Identifier:
			if p.peekTokenIsOneOf(token.UndirectedEgde, token.DirectedEgde) {
				var stmt ast.Stmt
				stmt, err = p.parseEdgeStatement()
				graph.Stmts = append(graph.Stmts, stmt)
			} else {
				var stmt ast.Stmt
				stmt, err = p.parseNodeStatement()
				graph.Stmts = append(graph.Stmts, stmt)
			}
		case token.Graph, token.Node, token.Edge:
			var stmt ast.Stmt
			stmt, err = p.parseAttrStatement()
			graph.Stmts = append(graph.Stmts, stmt)
		}

		if err != nil {
			return graph, err
		}
	}

	return graph, err
}

func (p *Parser) parseHeader() (ast.Graph, error) {
	var graph ast.Graph

	err := p.expectPeekTokenIsOneOf(token.Strict, token.Graph, token.Digraph)
	if err != nil {
		return graph, err
	}

	if p.curTokenIs(token.Strict) {
		graph.Strict = true

		err := p.expectPeekTokenIsOneOf(token.Graph, token.Digraph)
		if err != nil {
			return graph, err
		}
	}

	if p.curTokenIs(token.Digraph) {
		graph.Directed = true
	}

	// graph ID is optional
	hasID, err := p.advanceIfPeekTokenIsOneOf(token.Identifier)
	if err != nil {
		return graph, err
	}

	if hasID {
		graph.ID = p.curToken.Literal
	}

	return graph, nil
}

func (p *Parser) parseEdgeStatement() (*ast.EdgeStmt, error) {
	fmt.Println("parseEdgeStatement")
	es := &ast.EdgeStmt{Left: p.curToken.Literal}

	// TODO parse edgeRHS

	// attr_list is optional
	hasLeftBracket, err := p.advanceIfPeekTokenIsOneOf(token.LeftBracket)
	if err != nil {
		return es, err
	}
	if !hasLeftBracket {
		return es, nil
	}

	attrs, err := p.parseAttrList()
	if err != nil {
		return es, err
	}

	es.AttrList = attrs

	return es, nil
}

func (p *Parser) parseNodeStatement() (*ast.NodeStmt, error) {
	fmt.Println("parseNodeStatement")
	ns := &ast.NodeStmt{ID: p.curToken.Literal}

	// attr_list is optional
	hasLeftBracket, err := p.advanceIfPeekTokenIsOneOf(token.LeftBracket)
	if err != nil {
		return ns, err
	}
	if !hasLeftBracket {
		return ns, nil
	}

	attrs, err := p.parseAttrList()
	if err != nil {
		return ns, err
	}

	ns.AttrList = attrs

	return ns, nil
}

func (p *Parser) parseAttrStatement() (*ast.AttrStmt, error) {
	fmt.Println("parseAttrStatement")
	ns := &ast.AttrStmt{ID: p.curToken.Literal}

	err := p.expectPeekTokenIsOneOf(token.LeftBracket)
	if err != nil {
		return ns, err
	}

	attrs, err := p.parseAttrList()
	if err != nil {
		return ns, err
	}

	ns.AttrList = attrs

	return ns, nil
}

func (p *Parser) parseAttrList() (*ast.AttrList, error) {
	fmt.Println("parseAttrList")
	var first, cur *ast.AttrList
	for p.curTokenIs(token.LeftBracket) {
		err := p.expectPeekTokenIsOneOf(token.RightBracket, token.Identifier)
		if err != nil {
			return first, err
		}

		// a_list is optional
		if p.curTokenIs(token.Identifier) {
			alist, err := p.parseAList()
			if err != nil {
				return first, err
			}
			if first == nil {
				first = &ast.AttrList{AList: alist}
				cur = first
			} else {
				cur.Next = &ast.AttrList{AList: alist}
				cur = cur.Next
			}

			err = p.expectPeekTokenIsOneOf(token.RightBracket)
			if err != nil {
				return first, err
			}
		}

		_, err = p.advanceIfPeekTokenIsOneOf(token.LeftBracket)
		if err != nil {
			return first, err
		}
	}

	return first, nil
}

func (p *Parser) parseAList() (*ast.AList, error) {
	fmt.Println("parseAList")
	var first, cur *ast.AList
	for p.curTokenIs(token.Identifier) {
		attr, err := p.parseAttribute()
		if err != nil {
			return first, err
		}
		if first == nil {
			first = &ast.AList{Attribute: attr}
			cur = first
		} else {
			cur.Next = &ast.AList{Attribute: attr}
			cur = cur.Next
		}

		_, err = p.advanceIfPeekTokenIsOneOf(token.Comma, token.Semicolon)
		if err != nil {
			return first, err
		}

		hasID, err := p.advanceIfPeekTokenIsOneOf(token.Identifier)
		if err != nil {
			return first, err
		}
		if !hasID {
			return first, err
		}
	}

	return first, nil
}

func (p *Parser) parseAttribute() (ast.Attribute, error) {
	fmt.Println("parseAttribute")
	attr := ast.Attribute{
		Name: p.curToken.Literal,
	}

	err := p.expectPeekTokenIsOneOf(token.Equal)
	if err != nil {
		return attr, err
	}

	err = p.expectPeekTokenIsOneOf(token.Identifier)
	if err != nil {
		return attr, err
	}
	attr.Value = p.curToken.Literal

	return attr, nil
}

func (p *Parser) isDone() bool {
	return p.isEOF()
}

func (p *Parser) isEOF() bool {
	return p.curTokenIs(token.EOF)
}

func (p *Parser) curTokenIs(t token.TokenType) bool {
	return p.curToken.Type == t
}

func (p *Parser) curTokenIsOneOf(tokens ...token.TokenType) bool {
	return slices.ContainsFunc(tokens, p.curTokenIs)
}

func (p *Parser) peekTokenIsOneOf(tokens ...token.TokenType) bool {
	return slices.ContainsFunc(tokens, p.peekTokenIs)
}

func (p *Parser) peekTokenIs(t token.TokenType) bool {
	return p.peekToken.Type == t
}

func (p *Parser) expectPeekTokenIsOneOf(want ...token.TokenType) error {
	if !p.peekTokenIsOneOf(want...) {
		if len(want) == 1 {
			return fmt.Errorf("expected next token to be %q but got %q instead", want[0], p.peekToken)
		}
		return fmt.Errorf("expected next token to be one of %q but got %q instead", want, p.peekToken)
	}

	err := p.nextToken()
	if err != nil {
		return err
	}

	return nil
}

func (p *Parser) advanceIfPeekTokenIsOneOf(tokens ...token.TokenType) (bool, error) {
	if !p.peekTokenIsOneOf(tokens...) {
		return false, nil
	}

	err := p.nextToken()
	if err != nil {
		return true, err
	}

	return true, nil
}
