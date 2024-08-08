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

func New(r io.Reader) *Parser {
	return &Parser{
		l: dot.New(r),
	}
}

func (p *Parser) Parse() (*Graph, error) {
	g := &Graph{}

	return g, nil
}
