package dot

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/teleivo/dot/internal/assert"
	"github.com/teleivo/dot/token"
)

// Format specifies the output representation for rendering a [Tree].
type Format int

const (
	// Default renders the formatted output as indented text.
	Default Format = iota
	// Scheme renders the tree as S-expressions with position annotations. Each node is rendered
	// as (NodeType (@ startLine startCol endLine endCol) children...) and tokens are rendered as
	// ('token' (@ startLine startCol endLine endCol)).
	Scheme
)

var formats = map[string]Format{
	"default": Default,
	"scheme":  Scheme,
}

var validFormats = [...]string{"default", "scheme"}

// NewFormat converts a string to a [Format] constant. Valid values are "default" and "scheme".
// Returns an error if the format string is invalid.
func NewFormat(format string) (Format, error) {
	if f, ok := formats[format]; ok {
		return f, nil
	}
	return Default, fmt.Errorf("invalid format string: %q, valid ones are: %q", format, validFormats)
}

// TreeKind represents the type of syntax tree node (non-terminals).
type TreeKind uint32

const (
	KindErrorTree TreeKind = 1 << iota

	// Graph structure
	KindFile
	KindGraph
	KindSubgraph

	// Statements
	KindStmtList
	KindNodeStmt
	KindEdgeStmt
	KindAttrStmt

	// Node and Edge components
	KindNodeID
	KindPort
	KindCompassPoint

	// Attributes
	KindAttrList
	KindAList
	KindAttribute
	KindAttrName
	KindAttrValue

	KindID
)

// String returns the name of the tree kind.
func (tk TreeKind) String() string {
	switch tk {
	case KindErrorTree:
		return "ErrorTree"
	case KindFile:
		return "File"
	case KindGraph:
		return "Graph"
	case KindSubgraph:
		return "Subgraph"
	case KindStmtList:
		return "StmtList"
	case KindNodeStmt:
		return "NodeStmt"
	case KindEdgeStmt:
		return "EdgeStmt"
	case KindAttrStmt:
		return "AttrStmt"
	case KindNodeID:
		return "NodeID"
	case KindPort:
		return "Port"
	case KindCompassPoint:
		return "CompassPoint"
	case KindAttrList:
		return "AttrList"
	case KindAList:
		return "AList"
	case KindAttribute:
		return "Attribute"
	case KindAttrName:
		return "AttrName"
	case KindAttrValue:
		return "AttrValue"
	case KindID:
		return "ID"
	default:
		panic(fmt.Errorf("TreeKind Stringer missing case for %d", tk))
	}
}

// Tree is a flat, contiguous concrete syntax tree. Nodes are stored in depth-first order in a single
// slice. Tree nodes have a len field that encodes the number of descendant nodes, enabling O(1)
// subtree skipping. Token nodes have len 0.
//
// A tree node at index i has children spanning [i+1, i+1+nodes[i].len). Sibling advancement is
// done via [Tree.Next].
type Tree struct {
	nodes []Node
}

// Node is a node in the flat concrete syntax tree. It represents either a tree node (a syntactic
// construct like a graph, statement, or attribute) or a token node (a terminal symbol from the
// source). Use [Node.IsToken] to distinguish between the two.
type Node struct {
	Kind       TreeKind       // syntactic construct for tree nodes; zero for token nodes
	TokenKind  token.Kind     // token kind for token nodes; zero for tree nodes
	len        uint32         // number of descendant nodes in the subtree; zero for token nodes
	Start, End token.Position // source positions
	Literal    string         // token literal for token nodes; empty for tree nodes
}

// NodeRange represents a half-open range [Start, End) of child nodes in the Tree.
type NodeRange struct {
	Start, End int
}

// IsToken reports whether the node is a token node.
func (n Node) IsToken() bool {
	return n.TokenKind != 0
}

// Token returns the node's data as a token.Token. Only meaningful for token nodes.
func (n Node) Token() token.Token {
	return token.Token{Kind: n.TokenKind, Literal: n.Literal, Start: n.Start, End: n.End}
}

// NodeAt returns a pointer to the node at index i. The pointer is into the backing slice and must
// not be held across modifications to the Tree.
func (t *Tree) NodeAt(i int) *Node {
	return &t.nodes[i]
}

// Root returns the NodeRange spanning all top-level nodes in the tree.
func (t *Tree) Root() NodeRange {
	return NodeRange{0, len(t.nodes)}
}

// Children returns the NodeRange of children for the tree node at index i.
func (t *Tree) Children(i int) NodeRange {
	return NodeRange{i + 1, i + 1 + int(t.nodes[i].len)}
}

// Next returns the index of the next sibling after the node at index i, skipping over any
// descendants.
func (t *Tree) Next(i int) int {
	return i + 1 + int(t.nodes[i].len)
}

// FirstTree returns the index of the first child tree node matching want within the children of
// node at index parent. Returns -1, false if not found.
func (t *Tree) FirstTree(parent int, want TreeKind) (int, bool) {
	nr := t.Children(parent)
	for i := nr.Start; i < nr.End; i = t.Next(i) {
		n := t.nodes[i]
		if !n.IsToken() && n.Kind&want != 0 {
			return i, true
		}
	}
	return -1, false
}

// LastTree returns the index of the last child tree node matching want within the children of node
// at index parent. Returns -1, false if not found.
func (t *Tree) LastTree(parent int, want TreeKind) (int, bool) {
	nr := t.Children(parent)
	result := -1
	for i := nr.Start; i < nr.End; i = t.Next(i) {
		n := t.nodes[i]
		if !n.IsToken() && n.Kind&want != 0 {
			result = i
		}
	}
	if result == -1 {
		return -1, false
	}
	return result, true
}

// FirstToken returns the first child token matching want within the children of node at index
// parent. Returns the zero token and false if not found.
func (t *Tree) FirstToken(parent int, want token.Kind) (token.Token, bool) {
	nr := t.Children(parent)
	for i := nr.Start; i < nr.End; i = t.Next(i) {
		n := t.nodes[i]
		if n.IsToken() && n.TokenKind&want != 0 {
			return n.Token(), true
		}
	}
	return token.Token{}, false
}

// FirstID returns the token.ID of the first KindID child tree within the children of node at index
// parent.
func (t *Tree) FirstID(parent int) (token.Token, bool) {
	i, ok := t.FirstTree(parent, KindID)
	if !ok {
		return token.Token{}, false
	}
	return t.FirstToken(i, token.ID)
}

// TreeAt returns the index of the child tree at semantic index at if it matches want. Comments are
// skipped when counting the semantic index. Returns -1, false if not found.
func (t *Tree) TreeAt(parent int, want TreeKind, at int) (int, bool) {
	nr := t.Children(parent)
	var pos int
	for i := nr.Start; i < nr.End; i = t.Next(i) {
		n := t.nodes[i]
		if n.IsToken() && n.TokenKind == token.Comment {
			continue
		}
		if pos == at {
			if !n.IsToken() && n.Kind&want != 0 {
				return i, true
			}
			return -1, false
		}
		pos++
	}
	return -1, false
}

// TokenAt returns the child token at semantic index at if it matches want. Comments are skipped
// when counting the semantic index. Returns the zero token and false if not found.
func (t *Tree) TokenAt(parent int, want token.Kind, at int) (token.Token, bool) {
	nr := t.Children(parent)
	var pos int
	for i := nr.Start; i < nr.End; i = t.Next(i) {
		n := t.nodes[i]
		if n.IsToken() && n.TokenKind == token.Comment {
			continue
		}
		if pos == at {
			if n.IsToken() && n.TokenKind&want != 0 {
				return n.Token(), true
			}
			return token.Token{}, false
		}
		pos++
	}
	return token.Token{}, false
}

// FirstTokenWithin returns the first child token matching want within semantic index [0, last].
// Comments are skipped. Returns the zero token, 0, and false if not found.
func (t *Tree) FirstTokenWithin(parent int, want token.Kind, last int) (token.Token, int, bool) {
	nr := t.Children(parent)
	var pos int
	for i := nr.Start; i < nr.End; i = t.Next(i) {
		n := t.nodes[i]
		if n.IsToken() && n.TokenKind == token.Comment {
			continue
		}
		if pos > last {
			break
		}
		if n.IsToken() && n.TokenKind&want != 0 {
			return n.Token(), pos, true
		}
		pos++
	}
	return token.Token{}, 0, false
}

// FirstTreeWithin returns the index of the first child tree matching want within semantic index
// [0, last]. Comments are skipped. Returns -1, false if not found.
func (t *Tree) FirstTreeWithin(parent int, want TreeKind, last int) (int, bool) {
	nr := t.Children(parent)
	var pos int
	for i := nr.Start; i < nr.End; i = t.Next(i) {
		n := t.nodes[i]
		if n.IsToken() && n.TokenKind == token.Comment {
			continue
		}
		if pos > last {
			break
		}
		if !n.IsToken() && n.Kind&want != 0 {
			return i, true
		}
		pos++
	}
	return -1, false
}

// HasComment reports whether the node at index i or any of its descendants contain a comment token.
func (t *Tree) HasComment(i int) bool {
	end := t.Next(i)
	for j := i + 1; j < end; j++ {
		if t.nodes[j].IsToken() && t.nodes[j].TokenKind == token.Comment {
			return true
		}
	}
	return false
}

// EndLine returns the end line of the node at index i.
func (t *Tree) EndLine(i int) int {
	return int(t.nodes[i].End.Line)
}

// StartLine returns the start line of the node at index i.
func (t *Tree) StartLine(i int) int {
	return int(t.nodes[i].Start.Line)
}

// String returns the tree formatted using the [Default] format.
func (t *Tree) String() string {
	if t == nil || len(t.nodes) == 0 {
		return ""
	}

	var sb strings.Builder
	_ = t.Render(&sb, Default)
	return sb.String()
}

// Render writes the tree to w in the specified format. See [Format] for available formats.
func (t *Tree) Render(w io.Writer, format Format) error {
	if t == nil || len(t.nodes) == 0 {
		return nil
	}
	bw := bufio.NewWriter(w)

	var err error
	switch format {
	case Default:
		err = t.renderDefault(bw, 0, 0)
	case Scheme:
		err = t.renderScheme(bw, 0, 0)
	default:
		panic(fmt.Errorf("rendering tree in format %d is not implemented", format))
	}
	if err != nil {
		return err
	}
	err = bw.WriteByte('\n')
	if err != nil {
		return err
	}

	return bw.Flush()
}

// renderDefault writes the tree in default format (indented text without positions or parentheses)
// to the buffered writer.
func (t *Tree) renderDefault(bw *bufio.Writer, idx, indent int) error {
	n := t.nodes[idx]
	if n.IsToken() {
		err := writeIndent(bw, indent)
		if err != nil {
			return err
		}
		err = bw.WriteByte('\'')
		if err != nil {
			return err
		}
		_, err = bw.WriteString(n.Token().String())
		if err != nil {
			return err
		}
		err = bw.WriteByte('\'')
		if err != nil {
			return err
		}
		return nil
	}

	err := writeIndent(bw, indent)
	if err != nil {
		return err
	}
	_, err = bw.WriteString(n.Kind.String())
	if err != nil {
		return err
	}

	nr := t.Children(idx)
	for i := nr.Start; i < nr.End; i = t.Next(i) {
		err = bw.WriteByte('\n')
		if err != nil {
			return err
		}
		err = t.renderDefault(bw, i, indent+1)
		if err != nil {
			return err
		}
	}

	return nil
}

// renderScheme writes the tree in scheme format (S-expressions with position annotations) to the
// buffered writer.
func (t *Tree) renderScheme(bw *bufio.Writer, idx, indent int) error {
	n := t.nodes[idx]
	if n.IsToken() {
		err := writeIndent(bw, indent)
		if err != nil {
			return err
		}
		err = bw.WriteByte('(')
		if err != nil {
			return err
		}
		err = bw.WriteByte('\'')
		if err != nil {
			return err
		}
		_, err = bw.WriteString(n.Token().String())
		if err != nil {
			return err
		}
		err = bw.WriteByte('\'')
		if err != nil {
			return err
		}
		err = renderPosition(bw, n.Start, n.End)
		if err != nil {
			return err
		}
		err = bw.WriteByte(')')
		if err != nil {
			return err
		}
		return nil
	}

	err := writeIndent(bw, indent)
	if err != nil {
		return err
	}
	err = bw.WriteByte('(')
	if err != nil {
		return err
	}
	_, err = bw.WriteString(n.Kind.String())
	if err != nil {
		return err
	}
	err = renderPosition(bw, n.Start, n.End)
	if err != nil {
		return err
	}

	nr := t.Children(idx)
	for i := nr.Start; i < nr.End; i = t.Next(i) {
		err = bw.WriteByte('\n')
		if err != nil {
			return err
		}
		err = t.renderScheme(bw, i, indent+1)
		if err != nil {
			return err
		}
	}
	err = bw.WriteByte(')')
	if err != nil {
		return err
	}

	return nil
}

func writeIndent(bw *bufio.Writer, columns int) error {
	for range columns {
		err := bw.WriteByte('\t')
		if err != nil {
			return err
		}
	}
	return nil
}

func renderPosition(bw *bufio.Writer, start, end token.Position) error {
	assert.That(start.IsValid() == end.IsValid(), "tree position invariant violated: both Start and End must be valid or both invalid, got Start=%v End=%v", start, end)

	if !start.IsValid() && !end.IsValid() { // empty File will not have positions
		return nil
	}

	_, err := bw.WriteString(" (@ ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(bw, "%d %d %d %d", start.Line, start.Column, end.Line, end.Column)
	if err != nil {
		return err
	}
	err = bw.WriteByte(')')
	if err != nil {
		return err
	}
	return nil
}
