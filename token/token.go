// Package token defines constants representing the lexical tokens of the DOT language together with
// operations like printing, detecting Keywords or identifiers.
package token

import (
	"fmt"
	"strings"
)

// Kind represents the types of lexical tokens of the DOT language.
// Token kinds are powers of 2 and can be combined using bitwise OR
// to create token sets for efficient membership testing.
type Kind uint

const (
	ERROR Kind = 1 << iota
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

// terminalSet is the set of terminal symbols (punctuation and operators)
const terminalSet = LeftBrace | RightBrace | LeftBracket | RightBracket | Colon | Semicolon | Equal | Comma

// String returns the string representation of the token type.
func (k Kind) String() string {
	switch k {
	case ERROR:
		return "ERROR"
	case EOF:
		return "EOF"
	case ID:
		return "ID"
	case Comment:
		return "COMMENT"
	case LeftBrace:
		return "{"
	case RightBrace:
		return "}"
	case LeftBracket:
		return "["
	case RightBracket:
		return "]"
	case Colon:
		return ":"
	case Semicolon:
		return ";"
	case Equal:
		return "="
	case Comma:
		return ","
	case DirectedEdge:
		return "->"
	case UndirectedEdge:
		return "--"
	case Digraph:
		return "digraph"
	case Edge:
		return "edge"
	case Graph:
		return "graph"
	case Node:
		return "node"
	case Strict:
		return "strict"
	case Subgraph:
		return "subgraph"
	default:
		panic(fmt.Sprintf("missing String() case for token.Kind: %d", k))
	}
}

// IsTerminal reports whether the token type is a terminal symbol (punctuation or operator).
// Terminal symbols include braces, brackets, colon, semicolon, equal, and comma.
func (k Kind) IsTerminal() bool {
	return k&terminalSet != 0
}

// Token represents a token of the DOT language.
type Token struct {
	Type       Kind
	Literal    string
	Error      string // Error message for ERROR tokens, empty otherwise
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

func (t Token) IsKeyword() bool {
	switch t.Type {
	case Digraph, Edge, Graph, Node, Strict, Subgraph:
		return true
	default:
		return false

	}
}

func (t Token) IsCompassPoint() bool {
	if t.Type != ID {
		return false
	}

	switch t.Literal {
	case "_", "n", "ne", "e", "se", "s", "sw", "w", "nw", "c":
		return true
	default:
		return false
	}
}

// maxKeywordLen is the length of the longest DOT keyword which is "subgraph".
const maxKeywordLen = 8

// Lookup returns the token type associated with given identifier which is either a DOT keyword or a
// DOT ID. DOT keywords are case-insensitive. This function expects that the input is a valid DOT ID
// as specified in [IDs].
//
// [IDs]: https://graphviz.org/doc/info/lang.html#ids
func Lookup(identifier string) Kind {
	if len(identifier) > maxKeywordLen {
		return ID
	}

	switch strings.ToLower(identifier) {
	case "digraph":
		return Digraph
	case "edge":
		return Edge
	case "graph":
		return Graph
	case "node":
		return Node
	case "strict":
		return Strict
	case "subgraph":
		return Subgraph
	default:
		return ID
	}
}
