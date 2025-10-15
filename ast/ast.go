// Package ast contains an abstract syntax tree representation of the [DOT language].
//
// [DOT language]: https://graphviz.org/doc/info/lang.html
package ast

import (
	"strings"

	"github.com/teleivo/dot/token"
)

// Graph is a directed or undirected dot graph.
type Graph struct {
	StrictStart *token.Position // StrictStart is the starting position of the optional 'strict' keyword.
	GraphStart  token.Position  // GraphStart is the starting position of the 'graph' or 'digraph' keyword.
	Directed    bool            // Directed indicates that the graph is a directed graph.
	ID          *ID             // ID is the optional identifier of a graph.
	LeftBrace   token.Position  // Position of the opening '{'.
	Stmts       []Stmt          // Stmts lists all the graphs statements.
	RightBrace  token.Position  // Position of the closing '}'.
	Comments    []Comment       // List of all comments in the graph.
}

// IsStrict indicates whether the graph is declared as strict. Refer to [Lexical and Semantic Notes]
// for its meaning.
//
// [Lexical and Semantic Notes]: https://graphviz.org/doc/info/lang.html#lexical-and-semantic-notes
func (g *Graph) IsStrict() bool {
	return g.StrictStart != nil
}

func (g *Graph) String() string {
	var out strings.Builder
	if g.IsStrict() {
		out.WriteString("strict ")
	}
	if g.Directed {
		out.WriteString("digraph")
	} else {
		out.WriteString("graph")
	}
	if g.ID != nil {
		out.WriteRune(' ')
		out.WriteString(g.ID.String())
	}
	out.WriteString(" {")
	if len(g.Stmts) > 0 {
		out.WriteRune('\n')
	}
	for _, stmt := range g.Stmts {
		out.WriteRune('\t')
		out.WriteString(stmt.String())
		out.WriteRune('\n')
	}
	out.WriteRune('}')

	return out.String()
}

// Start returns the starting position of the first rune belonging to the graph. This can be the
// first rune of 'strict', 'graph', 'digraph' or the opening '{'. Use	the corresponding fields on
// the [Graph] if you need to access the individual starting positions. There might be leading
// comments that you can access via [Graph.Comments].
func (g *Graph) Start() token.Position {
	if g.StrictStart != nil {
		return *g.StrictStart
	}
	return g.GraphStart
}

// End returns the position of the closing '}' of the graph. There might be trailing comments which
// you can access via [Graph.Comments].
func (g *Graph) End() token.Position {
	return g.RightBrace
}

// Node represents an AST node of a dot graph.
type Node interface {
	String() string        // String returns a string representation of the AST node.
	Start() token.Position // Starting position returns the position of the first rune of the AST node.
	End() token.Position   // Starting position returns the position of the last rune of the AST node.
}

// Stmt nodes implement the Stmt interface.
type Stmt interface {
	Node
	stmtNode()
}

// ID is a DOT [identifier]. HTML strings are not supported.
//
// [identifier]: https://graphviz.org/doc/info/lang.html#ids
type ID struct {
	Literal  string         // Identifier literal
	StartPos token.Position // Position of the first rune of the ID
	EndPos   token.Position // Position of the last rune of the ID
}

func (id ID) String() string {
	return string(id.Literal)
}

func (id ID) Start() token.Position {
	return id.StartPos
}

func (id ID) End() token.Position {
	return id.EndPos
}

// NodeStmt is a dot node statement defining a node with optional attributes.
type NodeStmt struct {
	NodeID   NodeID    // NodeID is the identifier of the node targeted by the node statement.
	AttrList *AttrList // AttrList is an optional list of attributes for the node targeted by the node statement.
}

func (ns *NodeStmt) String() string {
	var out strings.Builder

	out.WriteString(ns.NodeID.String())
	if ns.AttrList != nil {
		out.WriteRune(' ')
		out.WriteString(ns.AttrList.String())
	}

	return out.String()
}

func (ns *NodeStmt) Start() token.Position {
	return ns.NodeID.Start()
}

func (ns *NodeStmt) End() token.Position {
	if ns.AttrList != nil {
		return ns.AttrList.End()
	}

	return ns.NodeID.End()
}

func (ns *NodeStmt) stmtNode() {}

// NodeID identifies a dot node with an optional port.
type NodeID struct {
	ID   ID    // ID is the identifier of the node.
	Port *Port // Port is an optioal port an edge can attach to.
}

func (ni NodeID) String() string {
	var out strings.Builder

	out.WriteString(ni.ID.String())
	if ni.Port != nil {
		out.WriteRune(':')
		out.WriteString(ni.Port.String())
	}

	return out.String()
}

func (ni NodeID) Start() token.Position {
	return ni.ID.StartPos
}

func (ni NodeID) End() token.Position {
	if ni.Port != nil {
		return ni.Port.End()
	}
	return ni.ID.EndPos
}

func (ni NodeID) edgeOperand() {}

// Port defines a node [port] where an edge can attach to. At least one of name and compass point
// must be defined.
//
// [port]: https://graphviz.org/doc/info/lang.html
type Port struct {
	Name         *ID           // Name is the identifier of the port.
	CompassPoint *CompassPoint // CompassPoint is the position at which an edge can attach to.
}

func (p Port) String() string {
	if p.Name == nil {
		return ":" + p.CompassPoint.String()
	} else if p.CompassPoint == nil {
		return p.Name.String()
	}

	return p.Name.String() + ":" + p.CompassPoint.String()
}

func (p Port) Start() token.Position {
	if p.Name != nil {
		return token.Position{
			Row:    p.Name.StartPos.Row,
			Column: p.Name.StartPos.Column - 1, // account for leading ':'
		}
	}

	return token.Position{
		Row:    p.CompassPoint.StartPos.Row,
		Column: p.CompassPoint.StartPos.Column - 1, // account for leading ':'
	}
}

func (p Port) End() token.Position {
	if p.CompassPoint == nil {
		return p.Name.EndPos
	}

	return p.CompassPoint.EndPos
}

// CompassPoint is the [position] at which an edge can attach to a node.
//
// [position]: https://graphviz.org/docs/attr-types/portPos
type CompassPoint struct {
	Type     CompassPointType
	StartPos token.Position // Position of the first rune of the compass point
	EndPos   token.Position // Position of the last rune of the compass point
}

func (cp CompassPoint) String() string {
	return cp.Type.String()
}

type CompassPointType int

const (
	CompassPointUnderscore CompassPointType = iota // Underscore is the default compass point in a port with a name https://graphviz.org/docs/attr-types/portPos.
	CompassPointNorth
	CompassPointNorthEast
	CompassPointEast
	CompassPointSouthEast
	CompassPointSouth
	CompassPointSouthWest
	CompassPointWest
	CompassPointNorthWest
	CompassPointCenter
)

func (cpt CompassPointType) String() string {
	return compassPointStrings[cpt]
}

var compassPointStrings = map[CompassPointType]string{
	CompassPointUnderscore: "_",
	CompassPointNorth:      "n",
	CompassPointNorthEast:  "ne",
	CompassPointEast:       "e",
	CompassPointSouthEast:  "se",
	CompassPointSouth:      "s",
	CompassPointSouthWest:  "sw",
	CompassPointWest:       "w",
	CompassPointNorthWest:  "nw",
	CompassPointCenter:     "c",
}

var compassPoints = map[string]CompassPointType{
	"_":  CompassPointUnderscore,
	"n":  CompassPointNorth,
	"ne": CompassPointNorthEast,
	"e":  CompassPointEast,
	"se": CompassPointSouthEast,
	"s":  CompassPointSouth,
	"sw": CompassPointSouthWest,
	"w":  CompassPointWest,
	"nw": CompassPointNorthWest,
	"c":  CompassPointCenter,
}

func IsCompassPoint(in string) (CompassPointType, bool) {
	v, ok := compassPoints[in]
	return v, ok
}

// EdgeStmt is a dot edge statement connecting nodes or subgraphs.
type EdgeStmt struct {
	Left     EdgeOperand // Left is the left node identifier or subgraph of the edge statement.
	Right    EdgeRHS     // Right is the edge statements right hand side.
	AttrList *AttrList   // AttrList is an optional list of attributes for the edge.
}

func (ns *EdgeStmt) String() string {
	var out strings.Builder

	out.WriteString(ns.Left.String())
	out.WriteString(ns.Right.String())
	if ns.AttrList != nil {
		out.WriteRune(' ')
		out.WriteString(ns.AttrList.String())
	}

	return out.String()
}

func (ns *EdgeStmt) Start() token.Position {
	return ns.Left.Start()
}

func (ns *EdgeStmt) End() token.Position {
	if ns.AttrList != nil {
		return ns.AttrList.End()
	}

	return ns.Right.End()
}

func (ns *EdgeStmt) stmtNode() {}

// EdgeRHS is the right-hand side of an edge statement.
type EdgeRHS struct {
	StartPos token.Position // StartPos is the starting position of the edge operator '--' or '->' as specified by [EdgeRHS.Directed].
	Directed bool           // Directed indicates that this is a directed edge.
	Right    EdgeOperand    // Right is the right node identifier or subgraph of the edge right hand side.
	Next     *EdgeRHS       // Next is an optional edge right hand side.
}

func (er EdgeRHS) String() string {
	var out strings.Builder

	if er.Directed {
		out.WriteString(" -> ")
	} else {
		out.WriteString(" -- ")
	}
	out.WriteString(er.Right.String())

	for cur := er.Next; cur != nil; cur = cur.Next {
		if cur.Directed {
			out.WriteString(" -> ")
		} else {
			out.WriteString(" -- ")
		}
		out.WriteString(cur.Right.String())
	}

	return out.String()
}

func (er EdgeRHS) Start() token.Position {
	return er.StartPos
}

func (er EdgeRHS) End() token.Position {
	var last EdgeOperand
	for cur := &er; cur != nil; cur = cur.Next {
		last = cur.Right
	}
	return last.End()
}

// EdgeOperand is an operand in an edge statement that can either be a graph or a subgraph.
type EdgeOperand interface {
	Node
	edgeOperand()
}

// AttrStmt is an attribute list defining default attributes for graphs, nodes or edges defined
// after this statement. The attr_stmt production requires an attr_list
//
//	attr_stmt :	(graph | node | edge) attr_list
//
// while the attr_list only requires opening and closing brackets.
//
//	attr_list :	'[' [ a_list ] ']' [ attr_list ]
//
// This means that the attr_list might be empty.
type AttrStmt struct {
	ID       ID       // ID is either graph, node or edge.
	AttrList AttrList // AttrList is a list of attributes for the graph, node or edge keyword.
}

func (ns *AttrStmt) String() string {
	var out strings.Builder

	out.WriteString(ns.ID.String())
	out.WriteRune(' ')
	out.WriteString(ns.AttrList.String())

	return out.String()
}

func (ns *AttrStmt) Start() token.Position {
	return ns.ID.Start()
}

func (ns *AttrStmt) End() token.Position {
	return ns.AttrList.End()
}

func (ns *AttrStmt) stmtNode() {}

// AttrList is a list of attributes as defined by [Attributes].
//
// [Attributes]: https://graphviz.org/doc/info/attrs.html
type AttrList struct {
	LeftBracket  token.Position // Position of the opening '['.
	AList        *AList         // AList is an optional list of attributes.
	RightBracket token.Position // Position of the first closing ']'. Note this might not be last ']' if
	// there are Next AttrList which themselves have '[]'. Use [AttrList.End] to get the position of
	// the last closing ']'.
	Next *AttrList // Next optionally points to the attribute list following this one.
}

func (atl *AttrList) String() string {
	var out strings.Builder

	for cur := atl; cur != nil; cur = cur.Next {
		out.WriteRune('[')
		out.WriteString(cur.AList.String())
		out.WriteRune(']')
		if cur.Next != nil {
			out.WriteRune(' ')
		}
	}

	return out.String()
}

func (atl *AttrList) Start() token.Position {
	return atl.LeftBracket
}

func (atl *AttrList) End() token.Position {
	var end token.Position
	for cur := atl; cur != nil; cur = cur.Next {
		end = cur.RightBracket
	}
	return end
}

// AList is a list of name-value [attribute pairs].
//
// [attribute pairs]: https://graphviz.org/doc/info/attrs.html
type AList struct {
	Attribute Attribute // Attribute is the name-value attribute pair.
	Next      *AList    // Next optionally points to the attribute following this one.
}

func (al *AList) String() string {
	var out strings.Builder

	for cur := al; cur != nil; cur = cur.Next {
		out.WriteString(cur.Attribute.String())
		if cur.Next != nil {
			out.WriteRune(',')
		}
	}

	return out.String()
}

func (al *AList) Start() token.Position {
	return al.Attribute.Start()
}

func (al *AList) End() token.Position {
	var last Attribute
	for cur := al; cur != nil; cur = cur.Next {
		last = cur.Attribute
	}
	return last.End()
}

// Attribute is a name-value [attribute pair]. Note that this name is not defined in the abstract
// grammar of the DOT language. It is defined as a statement and as part of the a_list as ID '=' ID.
//
// [attribute pair]: https://graphviz.org/doc/info/attrs.html
type Attribute struct {
	Name  ID // Name is an identifier naming the attribute.
	Value ID // Value is the identifier representing the value of the attribute.
}

func (a Attribute) String() string {
	var out strings.Builder

	out.WriteString(a.Name.String())
	out.WriteString("=")
	out.WriteString(a.Value.String())

	return out.String()
}

func (a Attribute) Start() token.Position {
	return a.Name.Start()
}

func (a Attribute) End() token.Position {
	return a.Value.End()
}

func (a Attribute) stmtNode() {}

// Subgraph is a dot subgraph.
type Subgraph struct {
	SubgraphStart *token.Position // SubgraphStart is the starting position of the optional keyword 'subgraph'.
	ID            *ID             // ID is the optional identifier.
	LeftBrace     token.Position  // LeftBrace is the position of the opening '{'.
	Stmts         []Stmt          // Stmts contains all the subraphs statements.
	RightBrace    token.Position  // RightBrace is the position of the closing '}'.
}

func (s Subgraph) String() string {
	var out strings.Builder

	out.WriteString("subgraph ")
	if s.ID != nil {
		out.WriteString(s.ID.String())
		out.WriteRune(' ')
	}
	out.WriteRune('{')
	for i, stmt := range s.Stmts {
		out.WriteString(stmt.String())
		if i != len(s.Stmts)-1 {
			out.WriteRune(' ')
		}
	}
	out.WriteRune('}')

	return out.String()
}

// Start returns the position of the first token belonging to the subgraph. This is either the
// 'subraph' keyword or the opening left brace.
func (s Subgraph) Start() token.Position {
	if s.SubgraphStart != nil {
		return *s.SubgraphStart
	}
	return s.LeftBrace
}

// End returns the position of the last token belonging to the subgraph which is the closing brace.
func (s Subgraph) End() token.Position {
	return s.RightBrace
}

func (s Subgraph) stmtNode()    {}
func (s Subgraph) edgeOperand() {}

// Comment is a DOT [comment].
//
// [comment]: https://graphviz.org/doc/info/lang.html#comments-and-optional-formatting
type Comment struct {
	Text     string         // Comment text including any opening and closing markers.
	StartPos token.Position // Position of the first rune of the comment.
	EndPos   token.Position // Position of the last rune of the comment.
}

func (c Comment) String() string {
	return c.Text
}

func (c Comment) Start() token.Position {
	return c.StartPos
}

func (c Comment) End() token.Position {
	return c.EndPos
}
