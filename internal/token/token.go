package token

import (
	"strings"
)

// TokenType represents the types of tokens of the dot language.
type TokenType int

const (
	LeftBrace TokenType = iota
	RightBrace
	LeftBracket
	RightBracket
	Colon
	Semicolon
	Equal
	Comma
	DirectedEgde
	UndirectedEgde
	Identifier
	// Keywords
	Digraph
	Edge
	Graph
	Node
	Strict
	Subgraph
	// EOF is not part of the dot language and is used to indicate the end of the file or stream. No
	// language token should follow the EOF token.
	EOF
)

var typeStrings map[TokenType]string = map[TokenType]string{
	LeftBrace:      "{",
	RightBrace:     "}",
	LeftBracket:    "[",
	RightBracket:   "]",
	Colon:          ":",
	Semicolon:      ";",
	Equal:          "=",
	Comma:          ",",
	DirectedEgde:   "->",
	UndirectedEgde: "--",
	Identifier:     "identifier",
	// Keywords,
	Digraph:  "digraph",
	Edge:     "edge",
	Graph:    "graph",
	Node:     "node",
	Strict:   "strict",
	Subgraph: "subgraph",
	EOF:      "EOF",
}

var types map[string]TokenType = map[string]TokenType{
	"{":          LeftBrace,
	"}":          RightBrace,
	"[":          LeftBracket,
	"]":          RightBracket,
	":":          Colon,
	";":          Semicolon,
	"=":          Equal,
	",":          Comma,
	"->":         DirectedEgde,
	"--":         UndirectedEgde,
	"identifier": Identifier,
	// Keywords,
	"digraph":  Digraph,
	"edge":     Edge,
	"graph":    Graph,
	"node":     Node,
	"strict":   Strict,
	"subgraph": Subgraph,
}

func (tt TokenType) String() string {
	return typeStrings[tt]
}

func Type(in string) (TokenType, bool) {
	v, ok := types[in]
	return v, ok
}

// Token represents a token of the dot language.
type Token struct {
	Type    TokenType
	Literal string
}

func (t Token) String() string {
	if t.Type == Identifier {
		return t.Literal
	}

	return t.Type.String()
}

// maxKeywordLen is the length of the longest dot keyword which is "subgraph".
const maxKeywordLen = 8

var keywords = map[string]TokenType{
	"digraph":  Digraph,
	"edge":     Edge,
	"graph":    Graph,
	"node":     Node,
	"strict":   Strict,
	"subgraph": Subgraph,
}

// LookupKeyword returns the token type associated with given identifier which is either a dot
// keyword or a dot id. Dot keywords are case-insensitive. This function expects that the input is a
// valid dot id as specified in https://graphviz.org/doc/info/lang.html#ids.
func LookupKeyword(identifier string) TokenType {
	if len(identifier) <= maxKeywordLen {
		identifier = strings.ToLower(identifier)
	}
	tokenType, ok := keywords[identifier]
	if ok {
		return tokenType
	}

	return Identifier
}
