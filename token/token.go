// Package token defines constants representing the lexical tokens of the DOT language together with
// operations like printing, detecting Keywords or identifiers.
package token

import (
	"strings"
)

// TokenType represents the types of lexical tokens of the DOT language.
type TokenType int

const (
	ERROR TokenType = iota
	// EOF is not part of the DOT language and is used to indicate the end of the file or stream. No
	// language token should follow the EOF token.
	EOF

	ID      // like _A 12 "234"
	Comment // like C pre-processor ones '# 34'

	LeftBrace      // {
	RightBrace     // }
	LeftBracket    // [
	RightBracket   // ]
	Colon          // :
	Semicolon      // ;
	Equal          // =
	Comma          // ,
	DirectedEdge   // ->
	UndirectedEdge // --

	// Keywords
	Digraph  // digraph
	Edge     // edge
	Graph    // graph
	Node     // node
	Strict   // strict
	Subgraph // subgraph
)

var typeStrings map[TokenType]string = map[TokenType]string{
	ERROR: "ERROR",
	EOF:   "EOF",

	ID:      "ID",
	Comment: "COMMENT",

	LeftBrace:      "{",
	RightBrace:     "}",
	LeftBracket:    "[",
	RightBracket:   "]",
	Colon:          ":",
	Semicolon:      ";",
	Equal:          "=",
	Comma:          ",",
	DirectedEdge:   "->",
	UndirectedEdge: "--",

	// Keywords
	Digraph:  "digraph",
	Edge:     "edge",
	Graph:    "graph",
	Node:     "node",
	Strict:   "strict",
	Subgraph: "subgraph",
}

var types map[string]TokenType = map[string]TokenType{
	"{":  LeftBrace,
	"}":  RightBrace,
	"[":  LeftBracket,
	"]":  RightBracket,
	":":  Colon,
	";":  Semicolon,
	"=":  Equal,
	",":  Comma,
	"->": DirectedEdge,
	"--": UndirectedEdge,

	// Keywords,
	"digraph":  Digraph,
	"edge":     Edge,
	"graph":    Graph,
	"node":     Node,
	"strict":   Strict,
	"subgraph": Subgraph,
}

// String returns the string representation of the token type.
func (tt TokenType) String() string {
	return typeStrings[tt]
}

// IsTerminal reports whether the token type is a terminal symbol (punctuation or operator).
// Terminal symbols include braces, brackets, colon, semicolon, equal, and comma.
func (tt TokenType) IsTerminal() bool {
	switch tt {
	case LeftBrace, RightBrace, LeftBracket, RightBracket, Colon, Semicolon, Equal, Comma:
		return true
	}
	return false
}

// Type returns the [TokenType] for the given string. Returns false if the string does not
// correspond to a token type (operator, keyword, or punctuation).
func Type(in string) (TokenType, bool) {
	v, ok := types[in]
	return v, ok
}

// Token represents a token of the DOT language.
type Token struct {
	Type       TokenType
	Literal    string
	Start, End Position
}

// String returns the string representation of the token. For IDs, it returns the literal
// value. For other token types, it returns the token type's string representation.
func (t Token) String() string {
	if t.Type == ID {
		return t.Literal
	}

	return t.Type.String()
}

// maxKeywordLen is the length of the longest DOT keyword which is "subgraph".
const maxKeywordLen = 8

var keywords = map[string]TokenType{
	"digraph":  Digraph,
	"edge":     Edge,
	"graph":    Graph,
	"node":     Node,
	"strict":   Strict,
	"subgraph": Subgraph,
}

// Lookup returns the token type associated with given identifier which is either a DOT keyword or a
// DOT ID. DOT keywords are case-insensitive. This function expects that the input is a valid DOT ID
// as specified in [IDs].
//
// [IDs]: https://graphviz.org/doc/info/lang.html#ids
func Lookup(identifier string) TokenType {
	if len(identifier) > maxKeywordLen {
		return ID
	}

	identifier = strings.ToLower(identifier)
	if tokenType, ok := keywords[identifier]; ok {
		return tokenType
	}

	return ID
}
