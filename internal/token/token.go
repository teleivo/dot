package token

import (
	"strings"
)

// TokenType represents the types of tokens of the dot language.
type TokenType int

const (
	LeftBrace      TokenType = iota // {
	RightBrace                      // }
	LeftBracket                     // [
	RightBracket                    // ]
	Colon                           // :
	Semicolon                       // ;
	Equal                           // =
	Comma                           // ,
	DirectedEgde                    // ->
	UndirectedEgde                  // --
	Identifier                      // like _A 12 "234"
	Comment                         // like C pre-processor ones '# 34'

	// Keywords
	Digraph  // digraph
	Edge     // edge
	Graph    // graph
	Node     // node
	Strict   // strict
	Subgraph // subgraph

	// TODO move this up under special tokens as Go does. This then leads to a problem as the zero
	// value for the current token in the parser is EOF.
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
	Comment:        "comment",
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
	"comment":    Comment,
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
	Type       TokenType
	Literal    string
	Start, End Position
}

func (t Token) String() string {
	if t.Type == Identifier {
		return t.Literal
	}

	return t.Type.String()
}

type Position struct {
	Row    int // Row is the line number starting at 1. A row of zero is not valid.
	Column int // Column is the horizontal position of in terms of runes starting at 1. A column of zero is not valid.
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

// Lookup returns the token type associated with given identifier which is either a dot keyword or a
// dot id. Dot keywords are case-insensitive. This function expects that the input is a valid dot id
// as specified in https://graphviz.org/doc/info/lang.html#ids.
func Lookup(identifier string) TokenType {
	if len(identifier) > maxKeywordLen {
		return Identifier
	}

	identifier = strings.ToLower(identifier)
	if tokenType, ok := keywords[identifier]; ok {
		return tokenType
	}

	return Identifier
}
