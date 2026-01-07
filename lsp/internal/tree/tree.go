// Package tree provides utilities for traversing DOT syntax trees.
package tree

import (
	"strings"

	"github.com/teleivo/dot"
	"github.com/teleivo/dot/token"
)

// Component represents which DOT graph component an attribute applies to.
// These correspond to the "Used By" column in the Graphviz attribute documentation.
type Component uint

const (
	Graph    Component = 1 << iota // Graph-level attributes (e.g., rankdir, splines)
	Subgraph                       // Subgraph attributes (e.g., rank)
	Cluster                        // Cluster subgraph attributes (subgraph with "cluster_" prefix)
	Node                           // Node attributes (e.g., shape, label)
	Edge                           // Edge attributes (e.g., arrowhead, style)

	All = Graph | Subgraph | Cluster | Node | Edge
)

// String returns the string representation of the component.
func (c Component) String() string {
	if c == 0 {
		return ""
	}

	contexts := make([]Component, 0, 5)
	for remaining := c; remaining != 0; {
		bit := remaining & -remaining
		contexts = append(contexts, bit)
		remaining &^= bit
	}

	var result strings.Builder
	for i, ctx := range contexts {
		if i > 0 {
			result.WriteString(", ")
		}
		switch ctx {
		case Graph:
			result.WriteString("Graph")
		case Subgraph:
			result.WriteString("Subgraph")
		case Cluster:
			result.WriteString("Cluster")
		case Node:
			result.WriteString("Node")
		case Edge:
			result.WriteString("Edge")
		}
	}
	return result.String()
}

// Match holds the result of a tree search.
type Match struct {
	Tree *dot.Tree
	Comp Component
}

// Find locates the deepest tree node matching any of the given kinds at the specified position.
// It also determines the component context (Graph, Node, Edge, etc.) based on the tree structure.
func Find(tree *dot.Tree, pos token.Position, want dot.TreeKind) Match {
	match := Match{Comp: Graph}
	find(tree, pos, want, &match)
	return match
}

func find(tree *dot.Tree, pos token.Position, want dot.TreeKind, match *Match) {
	if tree == nil {
		return
	}
	if pos.Before(tree.Start) || pos.After(tree.End) {
		return
	}

	switch tree.Kind {
	case dot.KindSubgraph:
		match.Comp = Subgraph
		if id, ok := dot.FirstID(tree); ok && strings.HasPrefix(id.Literal, "cluster_") {
			match.Comp = Cluster
		}
	case dot.KindNodeStmt:
		match.Comp = Node
	case dot.KindEdgeStmt:
		match.Comp = Edge
	case dot.KindAttrStmt:
		if tok, ok := dot.TokenAt(tree, token.Graph|token.Node|token.Edge, 0); ok {
			switch tok.Kind {
			case token.Graph:
				if match.Comp != Cluster && match.Comp != Subgraph {
					match.Comp = Graph
				}
			case token.Node:
				match.Comp = Node
			case token.Edge:
				match.Comp = Edge
			}
		}
	}

	if tree.Kind&want != 0 {
		match.Tree = tree
	}

	for _, child := range tree.Children {
		if c, ok := child.(dot.TreeChild); ok {
			if !pos.Before(c.Start) && !pos.After(c.End) {
				find(c.Tree, pos, want, match)
				return
			}
		}
	}
}

// AttrName extracts the attribute name from an Attribute node.
func AttrName(attr *dot.Tree) string {
	// Attribute: AttrName '=' AttrValue
	nameTree, ok := dot.TreeFirst(attr, dot.KindAttrName)
	if !ok {
		return ""
	}
	tok, ok := dot.FirstID(nameTree)
	if !ok {
		return ""
	}
	return tok.Literal
}
