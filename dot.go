// Package dot provides a parser for the dot language https://graphviz.org/doc/info/lang.html.
package dot

import (
	"fmt"
	"io"

	dot "github.com/teleivo/dot/internal"
	"github.com/teleivo/dot/internal/token"
)

type Graph struct {
	ID       string
	Strict   bool
	Directed bool
}

type Parser struct {
	l         *dot.Lexer
	curToken  token.Token
	peekToken token.Token
}

func New(r io.Reader) (*Parser, error) {
	l, err := dot.NewLexer(r)
	if err != nil {
		return nil, err
	}

	p := Parser{
		l: l,
	}

	// consume two tokens to initialize cur and peek token
	err = p.nextToken()
	if err != nil {
		return nil, err
	}
	err = p.nextToken()
	if err != nil {
		return nil, err
	}

	return &p, nil
}

func (p *Parser) nextToken() error {
	p.curToken = p.peekToken
	if p.isDone() {
		return nil
	}

	tok, err := p.l.NextToken()
	if err != nil {
		return err
	}
	p.peekToken = tok

	return nil
}

func (p *Parser) Parse() (*Graph, error) {
	g := &Graph{}

	if p.isDone() {
		return g, nil
	}

	if !p.curTokenIs(token.Strict) && !p.curTokenIs(token.Graph) && !p.curTokenIs(token.Digraph) {
		return nil, fmt.Errorf("expected either %q, %q, or %q but got %q instead", token.Strict, token.Graph, token.Digraph, p.curToken)
	}
	if p.curTokenIs(token.Strict) {
		g.Strict = true

		err := p.nextToken()
		if err != nil {
			return nil, err
		}
	}

	if !p.curTokenIs(token.Graph) && !p.curTokenIs(token.Digraph) {
		return nil, fmt.Errorf("expected either %q, or %q but got %q instead", token.Graph, token.Digraph, p.curToken)
	}
	if p.curTokenIs(token.Digraph) {
		g.Directed = true
	}
	err := p.nextToken()
	if err != nil {
		return nil, err
	}

	if !p.curTokenIs(token.Identifier) && !p.curTokenIs(token.LeftBrace) {
		return nil, fmt.Errorf("expected either %q, or %q but got %q instead", token.Identifier, token.LeftBrace, p.curToken)
	}
	if p.curTokenIs(token.Identifier) {
		g.ID = p.curToken.Literal

		err := p.nextToken()
		if err != nil {
			return nil, err
		}
	}

	if !p.curTokenIs(token.LeftBrace) {
		return nil, fmt.Errorf("expected either %q but got %q instead", token.LeftBrace, p.curToken)
	}
	// TODO count opening braces and brackets and decrement them on closing to validate they match?
	// or is that to simplistic as there are rules as to when you are allowed/have to close them?
	err = p.nextToken()
	if err != nil {
		return nil, err
	}

	return g, nil
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

func (p *Parser) peekTokenIs(t token.TokenType) bool {
	return p.peekToken.Type == t
}
