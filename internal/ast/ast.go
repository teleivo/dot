// Package ast contains an abstract syntax tree representation of the dot language https://graphviz.org/doc/info/lang.html.
package ast

import (
	"strings"
)

// Graph is a directed or undirected dot graph.
type Graph struct {
	Strict   bool
	Directed bool   // Directed indicates that the graph is a directed graph.
	ID       string // ID is the optional identifier of a graph.
	Stmts    []Stmt
}

func (g Graph) String() string {
	var out strings.Builder
	if g.Strict {
		out.WriteString("strict ")
	}
	if g.Directed {
		out.WriteString("digraph ")
	} else {
		out.WriteString("graph ")
	}
	out.WriteRune('{')
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

// TODO add another marker as this right now means that any Stringer is an AST node
// Node represents an AST node of a dot graph.
type Node interface {
	String() string
}

// Statement nodes implement the Stmt interface.
type Stmt interface {
	Node
	stmtNode()
}

// NodeStmt is a dot node statement defining a node with optional attributes.
type NodeStmt struct {
	ID       NodeID    // ID is the identifier of the node targeted by the node statement.
	AttrList *AttrList // AttrList is an optional list of attributes for the node targeted by the node statement.
}

func (ns *NodeStmt) String() string {
	var out strings.Builder

	out.WriteString(ns.ID.String())
	if ns.AttrList != nil {
		out.WriteRune(' ')
		out.WriteString(ns.AttrList.String())
	}

	return out.String()
}

func (ns *NodeStmt) stmtNode() {}

// NodeID identifies a dot node with an optional port.
type NodeID struct {
	ID   string // ID is the identifier of the node.
	Port *Port  // Port is an optioal port an edge can attach to.
}

func (ni NodeID) String() string {
	var out strings.Builder

	out.WriteString(ni.ID)
	if ni.Port != nil {
		out.WriteRune(':')
		out.WriteString(ni.Port.String())
	}

	return out.String()
}

func (ni NodeID) edgeOperand() {}

// Port defines a node port where an edge can attach to.
type Port struct {
	Name         string       // Name is the identifier of the port.
	CompassPoint CompassPoint // Position at which an edge can attach to.
}

func (p Port) String() string {
	return p.Name + ":" + p.CompassPoint.String()
}

// CompassPoint position at which an edge can attach to a node https://graphviz.org/docs/attr-types/portPos.
type CompassPoint int

const (
	Underscore CompassPoint = iota // Underscore is the default compass point in a port with a name https://graphviz.org/docs/attr-types/portPos.
	North
	NorthEast
	East
	SouthEast
	South
	SouthWest
	West
	NorthWest
	Center
)

var compassPointStrings = map[CompassPoint]string{
	Underscore: "_",
	North:      "n",
	NorthEast:  "ne",
	East:       "e",
	SouthEast:  "se",
	South:      "s",
	SouthWest:  "sw",
	West:       "w",
	NorthWest:  "nw",
	Center:     "c",
}

func (cp CompassPoint) String() string {
	return compassPointStrings[cp]
}

var compassPoints = map[string]CompassPoint{
	"_":  Underscore,
	"n":  North,
	"ne": NorthEast,
	"e":  East,
	"se": SouthEast,
	"s":  South,
	"sw": SouthWest,
	"w":  West,
	"nw": NorthWest,
	"c":  Center,
}

func IsCompassPoint(in string) (CompassPoint, bool) {
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

func (ns *EdgeStmt) stmtNode() {}

// EdgeRHS is the right-hand side of an edge statement.
type EdgeRHS struct {
	Directed bool        // Directed indicates that this is a directed edge.
	Right    EdgeOperand // Right is the right node identifier or subgraph of the edge right hand side.
	Next     *EdgeRHS    // Next is an optional edge right hand side.
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

// EdgeOperand is an operand in an edge statement that can either be a graph or a subgraph.
type EdgeOperand interface {
	Node
	edgeOperand()
}

// TODO the AttrList is not optional in (graph|node|edge) attr_list
// AttrStmt is an attribute list defining default attributes for graphs, nodes or edges defined
// after this statement.
type AttrStmt struct {
	ID       string    // ID is either graph, node or edge.
	AttrList *AttrList // AttrList is a list of attributes for the graph, node or edge keyword.
}

func (ns *AttrStmt) String() string {
	var out strings.Builder

	out.WriteString(ns.ID)
	if ns.AttrList != nil {
		out.WriteRune(' ')
		out.WriteString(ns.AttrList.String())
	}

	return out.String()
}

func (ns *AttrStmt) stmtNode() {}

// AttrList is a list of attributes as defined by https://graphviz.org/doc/info/attrs.html.
type AttrList struct {
	AList *AList    // AList is an optional list of attributes.
	Next  *AttrList // Next optionally points to the attribute list following this one.
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

// AList is a list of name-value attribute pairs https://graphviz.org/doc/info/attrs.html.
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

// Attribute is a name-value attribute pair https://graphviz.org/doc/info/attrs.html. Note that this
// name is not defined in the abstract grammar of the dot language. It is defined as a statement and
// as part of the a_list as ID '=' ID.
type Attribute struct {
	Name  string // Name is an identifier naming the attribute.
	Value string // Value is the identifier representing the value of the attribute.
}

func (a Attribute) String() string {
	var out strings.Builder

	out.WriteString(a.Name)
	out.WriteString("=")
	out.WriteString(a.Value)

	return out.String()
}

func (a Attribute) stmtNode() {}

// Subgraph is a dot subgraph.
type Subgraph struct {
	ID    string // ID is the optional identifier.
	Stmts []Stmt
}

func (s Subgraph) String() string {
	var out strings.Builder

	out.WriteString("subgraph ")
	if s.ID != "" {
		out.WriteString(s.ID)
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

func (s Subgraph) stmtNode()    {}
func (s Subgraph) edgeOperand() {}
