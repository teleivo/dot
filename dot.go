// Package dot provides a parser for the dot language https://graphviz.org/doc/info/lang.html.
package dot

import (
	"fmt"
	"io"
	"slices"

	dot "github.com/teleivo/dot/internal"
	"github.com/teleivo/dot/internal/token"
)

type Graph struct {
	ID       string
	Strict   bool
	Directed bool
	Nodes    map[string]*Node
}

type Node struct {
	ID         string
	Attributes map[string]Attribute
}

type Attribute struct {
	Name, Value string
}

type Parser struct {
	lexer     *dot.Lexer
	curToken  token.Token
	peekToken token.Token
	graph     *Graph
}

func New(r io.Reader) (*Parser, error) {
	lexer, err := dot.NewLexer(r)
	if err != nil {
		return nil, err
	}

	p := Parser{
		lexer: lexer,
		graph: newGraph(),
	}

	// initialize cur and peek token
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

	tok, err := p.lexer.NextToken()
	if err != nil {
		return err
	}
	p.peekToken = tok
	fmt.Printf("%#v\n", p)

	return nil
}

func (p *Parser) Parse() (*Graph, error) {
	if p.isDone() {
		return p.graph, nil
	}

	// TODO always stay on last parsed token
	err := p.parseHeader()
	if err != nil {
		return nil, err
	}

	// TODO extract some func that does the expect + advance?
	if !p.curTokenIs(token.LeftBrace) {
		return nil, fmt.Errorf("expected either %q but got %q instead", token.LeftBrace, p.curToken)
	}
	err = p.nextToken()
	if err != nil {
		return nil, err
	}

	for ; !p.curTokenIs(token.EOF) && err == nil; err = p.nextToken() {
		switch p.curToken.Type {
		case token.Identifier:
			err = p.parseNodeStatement()
		}

		if err != nil {
			return p.graph, err
		}
	}

	return p.graph, err
}

func newGraph() *Graph {
	g := Graph{
		Nodes: make(map[string]*Node),
	}

	return &g
}

func (p *Parser) parseHeader() error {
	if !p.curTokenIs(token.Strict) && !p.curTokenIs(token.Graph) && !p.curTokenIs(token.Digraph) {
		return fmt.Errorf("expected either %q, %q, or %q but got %q instead", token.Strict, token.Graph, token.Digraph, p.curToken)
	}
	if p.curTokenIs(token.Strict) {
		p.graph.Strict = true

		err := p.nextToken()
		if err != nil {
			return err
		}
	}

	if !p.curTokenIs(token.Graph) && !p.curTokenIs(token.Digraph) {
		return fmt.Errorf("expected either %q, or %q but got %q instead", token.Graph, token.Digraph, p.curToken)
	}
	if p.curTokenIs(token.Digraph) {
		p.graph.Directed = true
	}
	err := p.nextToken()
	if err != nil {
		return err
	}

	if !p.curTokenIs(token.Identifier) && !p.curTokenIs(token.LeftBrace) {
		return fmt.Errorf("expected either %q, or %q but got %q instead", token.Identifier, token.LeftBrace, p.curToken)
	}
	if p.curTokenIs(token.Identifier) {
		p.graph.ID = p.curToken.Literal

		err := p.nextToken()
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *Parser) parseNodeStatement() error {
	fmt.Println("parseNodeStatement")
	id := p.curToken.Literal
	n, ok := p.graph.Nodes[id]
	if !ok {
		n = &Node{ID: id, Attributes: make(map[string]Attribute)}
		p.graph.Nodes[id] = n
	}

	// attr_list is optional
	if !p.peekTokenIs(token.LeftBracket) {
		return nil
	}
	err := p.nextToken()
	if err != nil {
		return err
	}

	attrs, err := p.parseAttributeList()
	if err != nil {
		return err
	}

	for _, attr := range attrs {
		n.Attributes[attr.Name] = attr
	}

	return nil
}

func (p *Parser) parseAttributeList() ([]Attribute, error) {
	fmt.Println("parseAttributeList")
	var attrs []Attribute

	for p.curTokenIs(token.LeftBracket) {
		err := p.expectPeekTokenIs(token.RightBracket, token.Identifier)
		if err != nil {
			return attrs, err
		}

		//TODO should I not advance?
		if p.curTokenIs(token.RightBracket) {
			continue
		}

		alist, err := p.parseAList()
		if err != nil {
			return attrs, err
		}
		attrs = append(attrs, alist...)

		err = p.advanceIfPeekTokenIsOneOf(token.RightBracket)
		if err != nil {
			return attrs, err
		}

		// TODO this is an awkward loop right now
		if !p.peekTokenIs(token.LeftBracket) {
			return attrs, nil
		}
		err = p.nextToken()
		if err != nil {
			return attrs, err
		}
	}

	return attrs, nil
}

func (p *Parser) parseAList() ([]Attribute, error) {
	fmt.Println("parseAList")
	var attrs []Attribute

	err := p.expectCurrentTokenIs(token.Identifier)
	if err != nil {
		return attrs, err
	}

	for p.curTokenIs(token.Identifier) {
		attr, err := p.parseAttribute()
		if err != nil {
			return attrs, err
		}
		attrs = append(attrs, attr)

		err = p.advanceIfPeekTokenIsOneOf(token.Comma, token.Semicolon)
		if err != nil {
			return attrs, err
		}

		// TODO this is an awkward loop right now
		if !p.peekTokenIs(token.Identifier) {
			return attrs, nil
		}
		err = p.nextToken()
		if err != nil {
			return attrs, err
		}
	}

	return attrs, nil
}

func (p *Parser) parseAttribute() (Attribute, error) {
	fmt.Println("parseAttribute")
	var attr Attribute

	err := p.expectCurrentTokenIs(token.Identifier)
	if err != nil {
		return attr, err
	}
	attr = Attribute{Name: p.curToken.Literal}

	err = p.expectPeekTokenIs(token.Equal)
	if err != nil {
		return attr, err
	}

	err = p.expectPeekTokenIs(token.Identifier)
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

func (p *Parser) expectPeekTokenIs(want ...token.TokenType) error {
	if !p.peekTokenIsOneOf(want...) {
		if len(want) == 1 {
			return fmt.Errorf("expected next token to be %q but got %q instead", want[0], p.curToken)
		}
		return fmt.Errorf("expected next token to be one of %q but got %q instead", want, p.curToken)
	}

	err := p.nextToken()
	if err != nil {
		return err
	}

	return nil
}

// TODO use a different name for this or the above function as this one does not advance but only
// validates
func (p *Parser) expectCurrentTokenIs(want ...token.TokenType) error {
	if !p.curTokenIsOneOf(want...) {
		if len(want) == 1 {
			return fmt.Errorf("expected token to be %q but got %q instead", want[0], p.curToken)
		}
		return fmt.Errorf("expected token to be one of %q but got %q instead", want, p.curToken)
	}

	// err := p.nextToken()
	// if err != nil {
	// 	return err
	// }

	return nil
}

func (p *Parser) advanceIfPeekTokenIsOneOf(tokens ...token.TokenType) error {
	if !p.peekTokenIsOneOf(tokens...) {
		return nil
	}

	err := p.nextToken()
	if err != nil {
		return err
	}

	return nil
}
