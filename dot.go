// Package dot provides a parser for the dot language https://graphviz.org/doc/info/lang.html.
package dot

import (
	"io"

	dot "github.com/teleivo/dot/internal"
)

type Graph struct {
	ID       string
	Strict   bool
	Directed bool
}

type Parser struct {
	l *dot.Lexer
}

func New(r io.Reader) (*Parser, error) {
	l, err := dot.NewLexer(r)
	if err != nil {
		return nil, err
	}

	return &Parser{
		l: l,
	}, nil
}

func (p *Parser) Parse() (*Graph, error) {
	g := &Graph{}

	// var pos int
	// for tok, err := range p.l.All() {
	// TODO discern LexError and "fatal" error?
	// if err != nil {
	// 	return nil, err
	// }
	// if tok == token.Strict {
	// 	if pos == 0 {
	// 		g.Strict == true
	// 	}
	// }
	// }

	return g, nil
}
