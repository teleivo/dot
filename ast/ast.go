// Package ast provides an abstract syntax tree representation for DOT graphs.
//
// The AST types wrap the concrete syntax tree produced by [dot.Parser] and provide a high-level,
// semantic view of DOT source code. Use [NewGraph] to create a Graph from a parsed tree.
package ast

import (
	"github.com/teleivo/dot"
	"github.com/teleivo/dot/internal/assert"
	"github.com/teleivo/dot/token"
)

// Stmt represents a statement in a DOT graph or subgraph.
type Stmt interface {
	stmtNode()
}

// EdgeOperand represents a node or subgraph that can appear in an edge statement.
type EdgeOperand interface {
	edgeOperand()
}

// Graph represents a DOT graph, either directed (digraph) or undirected (graph).
type Graph struct {
	tree *dot.Tree
}

// NewGraph returns all graphs from a parsed [dot.Tree]. Returns nil if the tree is not a File.
func NewGraph(tree *dot.Tree) []*Graph {
	if tree.Kind != dot.KindFile {
		return nil
	}

	var result []*Graph
	for _, child := range tree.Children {
		if tc, ok := child.(dot.TreeChild); ok && tc.Kind == dot.KindGraph {
			result = append(result, &Graph{tc.Tree})
		}
	}
	return result
}

// IsStrict reports whether the graph was declared with the "strict" keyword.
func (g Graph) IsStrict() bool {
	// graph : [ strict ] (graph | digraph) [ ID ] '{' stmt_list '}'
	_, ok := tokenAt(g.tree, token.Strict, 0)
	return ok
}

// Directed reports whether the graph is directed (digraph) or undirected (graph).
func (g Graph) Directed() bool {
	// graph : [ strict ] (graph | digraph) [ ID ] '{' stmt_list '}'
	_, _, ok := tokenFirst(g.tree, token.Digraph, 1)
	return ok
}

// tokenAt returns the token at index if it matches want.
func tokenAt(tree *dot.Tree, want token.Kind, at int) (token.Token, bool) {
	var tok token.Token
	if at >= len(tree.Children) {
		return tok, false
	}

	if tc, ok := tree.Children[at].(dot.TokenChild); ok && tc.Kind&want != 0 {
		return tc.Token, true
	}
	return tok, false
}

// tokenFirst returns the first token matching want within children[0:last] (inclusive).
func tokenFirst(tree *dot.Tree, want token.Kind, last int) (token.Token, int, bool) {
	for i, child := range tree.Children {
		if last < 0 {
			break
		}

		if tc, ok := child.(dot.TokenChild); ok && tc.Kind&want != 0 {
			return tc.Token, i, true
		}
		last--
	}
	var tok token.Token
	return tok, 0, false
}

// treeAt returns the tree at index if it matches want.
func treeAt(tree *dot.Tree, want dot.TreeKind, at int) (*dot.Tree, bool) {
	if at >= len(tree.Children) {
		return nil, false
	}

	if tc, ok := tree.Children[at].(dot.TreeChild); ok && tc.Kind == want {
		return tc.Tree, true
	}
	return nil, false
}

// idAt returns the ID at index if present.
func idAt(tree *dot.Tree, at int) (*ID, bool) {
	if id, ok := treeAt(tree, dot.KindID, at); ok {
		tok, ok := id.Children[0].(dot.TokenChild)
		assert.That(ok, "ID missing required token child")
		return &ID{tok.Token}, true
	}
	return nil, false
}

// treeFirst returns the first tree matching want within children[0:last] (inclusive).
func treeFirst(tree *dot.Tree, want dot.TreeKind, last int) (*dot.Tree, bool) {
	for _, child := range tree.Children {
		if last < 0 {
			break
		}

		if tc, ok := child.(dot.TreeChild); ok && tc.Kind == want {
			return tc.Tree, true
		}
		last--
	}
	return nil, false
}

// ID returns the graph identifier, or nil if not present.
func (g Graph) ID() *ID {
	// graph : [ strict ] (graph | digraph) [ ID ] '{' stmt_list '}'
	_, i, _ := tokenFirst(g.tree, token.Graph|token.Digraph, 1)
	id, _ := idAt(g.tree, i+1)
	return id
}

// Stmts returns the statements in the graph body.
func (g Graph) Stmts() []Stmt {
	return stmts(g.tree)
}

func stmts(tree *dot.Tree) []Stmt {
	var result []Stmt
	// graph : [ strict ] (graph | digraph) [ ID ] '{' stmt_list '}'
	stmtList, ok := treeFirst(tree, dot.KindStmtList, 4)
	if !ok {
		return result
	}

	for _, child := range stmtList.Children {
		if tc, ok := child.(dot.TreeChild); ok {
			switch tc.Kind {
			case dot.KindAttribute:
				result = append(result, Attribute{tc.Tree})
			case dot.KindAttrStmt:
				result = append(result, AttrStmt{tc.Tree})
			case dot.KindEdgeStmt:
				result = append(result, EdgeStmt{tc.Tree})
			case dot.KindSubgraph:
				result = append(result, Subgraph{tc.Tree})
			case dot.KindNodeStmt:
				result = append(result, NodeStmt{tc.Tree})
			}
		}
	}
	return result
}

// ID represents an identifier in DOT (node names, attribute names/values, graph names).
type ID struct {
	tok token.Token
}

// Literal returns the identifier text as it appears in the source.
func (id ID) Literal() string {
	return id.tok.Literal
}

// NodeStmt represents a node statement that declares a node with optional attributes.
type NodeStmt struct {
	tree *dot.Tree
}

// NodeID returns the node identifier with optional port.
func (n NodeStmt) NodeID() NodeID {
	// node_stmt : node_id [ attr_list ]
	nid, ok := treeAt(n.tree, dot.KindNodeID, 0)
	assert.That(ok, "NodeStmt missing required NodeID child")
	return NodeID{nid}
}

// AttrList returns the attribute list. Check Lists() for empty to detect absence.
func (n NodeStmt) AttrList() AttrList {
	t, _ := treeAt(n.tree, dot.KindAttrList, len(n.tree.Children)-1)
	return AttrList{t}
}

func (NodeStmt) stmtNode() {}

// NodeID identifies a node, optionally with a port specification.
type NodeID struct {
	tree *dot.Tree
}

// ID returns the node name.
func (n NodeID) ID() ID {
	// node_id : ID [ port ]
	id, ok := idAt(n.tree, 0)
	assert.That(ok, "NodeID missing required ID child")
	return *id
}

// Port returns the port specification, or nil if not present.
func (n NodeID) Port() *Port {
	// node_id : ID [ port ]
	if port, ok := treeAt(n.tree, dot.KindPort, 1); ok {
		return &Port{port}
	}
	return nil
}

func (NodeID) edgeOperand() {}

// Port specifies where an edge attaches to a node.
type Port struct {
	tree *dot.Tree
}

// Name returns the port name, or nil if only a compass point is specified.
func (p Port) Name() *ID {
	// port : ':' ID [ ':' compass_pt ] | ':' compass_pt
	id, _ := idAt(p.tree, 1)
	return id
}

// CompassPoint returns the compass point, or nil if not present.
func (p Port) CompassPoint() *CompassPoint {
	// port : ':' ID [ ':' compass_pt ] | ':' compass_pt
	if id, ok := treeFirst(p.tree, dot.KindCompassPoint, 3); ok {
		tok, _, _ := tokenFirst(id, token.ID, 0)
		return &CompassPoint{tok}
	}
	return nil
}

// CompassPointType represents a compass direction for edge attachment.
type CompassPointType int

const (
	CompassPointUnderscore CompassPointType = iota // "_" - default/center
	CompassPointNorth                              // "n"
	CompassPointNorthEast                          // "ne"
	CompassPointEast                               // "e"
	CompassPointSouthEast                          // "se"
	CompassPointSouth                              // "s"
	CompassPointSouthWest                          // "sw"
	CompassPointWest                               // "w"
	CompassPointNorthWest                          // "nw"
	CompassPointCenter                             // "c"
)

// CompassPoint represents a compass direction where an edge attaches to a node.
type CompassPoint struct {
	tok token.Token
}

// Type returns the compass direction.
func (cp CompassPoint) Type() CompassPointType {
	switch cp.tok.Literal {
	case "n":
		return CompassPointNorth
	case "ne":
		return CompassPointNorthEast
	case "e":
		return CompassPointEast
	case "se":
		return CompassPointSouthEast
	case "s":
		return CompassPointSouth
	case "sw":
		return CompassPointSouthWest
	case "w":
		return CompassPointWest
	case "nw":
		return CompassPointNorthWest
	case "c":
		return CompassPointCenter
	case "_":
		fallthrough
	default:
		return CompassPointUnderscore
	}
}

// String returns the compass point as it appears in the source (e.g., "n", "se").
func (cp CompassPoint) String() string {
	return cp.tok.Literal
}

// EdgeStmt represents an edge statement connecting nodes or subgraphs.
type EdgeStmt struct {
	tree *dot.Tree
}

// Directed reports whether this is a directed edge (->).
func (e EdgeStmt) Directed() bool {
	// edge_stmt : (node_id | subgraph) edgeRHS [ attr_list ]
	// edgeRHS   : edgeop (node_id | subgraph) [ edgeRHS ]
	_, ok := tokenAt(e.tree, token.DirectedEdge, 1)
	return ok
}

// Operands returns all edge operands in order (e.g., for "A -> B -> C" returns [A, B, C]).
func (e EdgeStmt) Operands() []EdgeOperand {
	var result []EdgeOperand
	for _, child := range e.tree.Children {
		if tc, ok := child.(dot.TreeChild); ok {
			switch tc.Kind {
			case dot.KindNodeID:
				result = append(result, NodeID{tc.Tree})
			case dot.KindSubgraph:
				result = append(result, Subgraph{tc.Tree})
			}
		}
	}
	return result
}

// AttrList returns the attribute list. Check Lists() for empty to detect absence.
func (e EdgeStmt) AttrList() AttrList {
	t, _ := treeAt(e.tree, dot.KindAttrList, len(e.tree.Children)-1)
	return AttrList{t}
}

func (EdgeStmt) stmtNode() {}

// AttrStmt represents a default attribute statement for graph, node, or edge.
type AttrStmt struct {
	tree *dot.Tree
}

// Target returns the target type (graph, node, or edge).
func (a AttrStmt) Target() ID {
	// attr_stmt : (graph | node | edge) attr_list
	tok, ok := tokenAt(a.tree, token.Graph|token.Node|token.Edge, 0)
	assert.That(ok, "AttrStmt missing required graph, node, or edge token")
	return ID{tok}
}

// AttrList returns the attribute list.
func (a AttrStmt) AttrList() AttrList {
	t, _ := treeAt(a.tree, dot.KindAttrList, len(a.tree.Children)-1)
	return AttrList{t}
}

func (AttrStmt) stmtNode() {}

// AttrList represents one or more bracketed attribute lists.
type AttrList struct {
	tree *dot.Tree
}

// Lists returns attribute groups, preserving bracket structure
// (e.g., [a=1][b=2] returns [[{a,1}], [{b,2}]]).
func (a AttrList) Lists() [][]Attribute {
	if a.tree == nil {
		return nil
	}
	var result [][]Attribute
	var current []Attribute
	for _, child := range a.tree.Children {
		if tc, ok := child.(dot.TokenChild); ok {
			switch tc.Kind {
			case token.LeftBracket:
				current = make([]Attribute, 0)
			case token.RightBracket:
				result = append(result, current)
			}
		} else if tc, ok := child.(dot.TreeChild); ok && tc.Kind == dot.KindAList {
			for _, ac := range tc.Children {
				if attr, ok := ac.(dot.TreeChild); ok && attr.Kind == dot.KindAttribute {
					current = append(current, Attribute{attr.Tree})
				}
			}
		}
	}
	return result
}

// Attribute represents a name=value attribute assignment.
type Attribute struct {
	tree *dot.Tree
}

// Name returns the attribute name.
func (a Attribute) Name() ID {
	// a_list : ID '=' ID [ (';' | ',') ] [ a_list ]
	nameTree, ok := treeAt(a.tree, dot.KindAttrName, 0)
	assert.That(ok, "Attribute missing required AttrName child")
	id, ok := idAt(nameTree, 0)
	assert.That(ok, "AttrName missing required ID child")
	return *id
}

// Value returns the attribute value.
func (a Attribute) Value() ID {
	// a_list : ID '=' ID [ (';' | ',') ] [ a_list ]
	valueTree, ok := treeAt(a.tree, dot.KindAttrValue, 2)
	assert.That(ok, "Attribute missing required AttrValue child")
	id, ok := idAt(valueTree, 0)
	assert.That(ok, "AttrValue missing required ID child")
	return *id
}

func (Attribute) stmtNode() {}

// Subgraph represents a subgraph definition.
type Subgraph struct {
	tree *dot.Tree
}

// HasKeyword reports whether the subgraph was declared with the "subgraph" keyword.
// A subgraph can be declared without the keyword (just braces).
func (s Subgraph) HasKeyword() bool {
	// subgraph : [ subgraph [ ID ] ] '{' stmt_list '}'
	_, ok := tokenAt(s.tree, token.Subgraph, 0)
	return ok
}

// ID returns the subgraph identifier, or nil if not present.
func (s Subgraph) ID() *ID {
	// subgraph : [ subgraph [ ID ] ] '{' stmt_list '}'
	if _, ok := tokenAt(s.tree, token.Subgraph, 0); !ok {
		return nil
	}
	id, _ := idAt(s.tree, 1)
	return id
}

// Stmts returns the statements in the subgraph body.
func (s Subgraph) Stmts() []Stmt {
	return stmts(s.tree)
}

func (Subgraph) stmtNode()    {}
func (Subgraph) edgeOperand() {}
