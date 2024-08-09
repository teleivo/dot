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
	eof       bool
}

func New(r io.Reader) (*Parser, error) {
	l, err := dot.NewLexer(r)
	if err != nil {
		return nil, err
	}

	p := Parser{
		l: l,
	}

	// consume two tokens to initialize cur and next token
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

	// TODO I think can we have an error with a valid token
	// TODO is an EOF token easier to use?
	tok, err, ok := p.l.Next()
	if err != nil {
		return err
	}
	if !ok {
		p.eof = true
		return nil
	}
	p.peekToken = tok

	return nil
}

func (p *Parser) Parse() (*Graph, error) {
	g := &Graph{}

	if p.isDone() {
		return g, nil
	}

	fmt.Printf("%#v\n", p)

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

	return g, nil
}

func (p *Parser) isDone() bool {
	return p.isEOF()
}

func (p *Parser) isEOF() bool {
	return p.eof
}

func (p *Parser) curTokenIs(t token.TokenType) bool {
	return p.curToken.Type == t
}

func (p *Parser) peekTokenIs(t token.TokenType) bool {
	return p.peekToken.Type == t
}
