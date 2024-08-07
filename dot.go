// Package dot provides a parser for the dot language https://graphviz.org/doc/info/lang.html.
package dot

import (
	"io"
)

type Graph struct {
	ID       string
	Strict   bool
	Directed bool
}

type Parser struct {
}

func (p *Parser) Parse(r io.Reader) *Graph {
	g := &Graph{}

	return g
}
