package token

import "strings"

type TokenType string

const (
	LeftBrace    = "{"
	RightBrace   = "}"
	LeftBracket  = "["
	RightBracket = "]"
	Colon        = ":"
	Semicolon    = ";"
	Equal        = "="
	Comma        = ","

	DirectedEgde   = "->"
	UndirectedEgde = "--"

	Identifier = "identifier"

	// Keywords
	Digraph  = "digraph"
	Edge     = "edge"
	Graph    = "graph"
	Node     = "node"
	Strict   = "strict"
	Subgraph = "subgraph"
)

// Token represents a token of the dot language.
type Token struct {
	Type    TokenType
	Literal string
}

// maxKeywordLen is the length of the longest dot keyword which is "subgraph".
const maxKeywordLen = len(Subgraph)

var keywords = map[string]TokenType{
	"digraph":  Digraph,
	"edge":     Edge,
	"graph":    Graph,
	"node":     Node,
	"strict":   Strict,
	"subgraph": Subgraph,
}

// LookupIdentifier returns the token type associated with given identifier which is either a dot
// keyword or a dot id. Dot keywords are case-insensitive. This function expects that the input is a
// valid dot id as specified in https://graphviz.org/doc/info/lang.html#ids.
func LookupIdentifier(identifier string) TokenType {
	if len(identifier) <= maxKeywordLen {
		identifier = strings.ToLower(identifier)
	}
	tokenType, ok := keywords[identifier]
	if ok {
		return tokenType
	}

	return Identifier
}
