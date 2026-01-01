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

// TreeKind represents the type of syntax tree node (non-terminals)
type TreeKind int

const (
	KindErrorTree TreeKind = iota

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

// Tree represents a node in the concrete syntax tree.
//
// Type identifies the syntactic construct (e.g., [Graph], [NodeStmt], [ID]). Children contains the
// node's children in source order, which may be either [TreeChild] (subtrees) or [TokenChild]
// (tokens). Start and End mark the source positions.
type Tree struct {
	Type       TreeKind
	Children   []Child
	Start, End token.Position
}

func (tree *Tree) appendToken(child token.Token) {
	if len(tree.Children) == 0 {
		tree.Start = child.Start
	}
	tree.End = child.End
	tree.Children = append(tree.Children, TokenChild{child})
}

func (tree *Tree) appendTree(child *Tree) {
	if len(tree.Children) == 0 {
		tree.Start = child.Start
	}
	tree.End = child.End
	tree.Children = append(tree.Children, TreeChild{child})
}

// String returns the tree formatted using the [Default] format.
func (tree *Tree) String() string {
	if tree == nil {
		return ""
	}

	var sb strings.Builder
	_ = tree.Render(&sb, Default)
	return sb.String()
}

// renderDefault writes the tree in default format (indented text without positions or parentheses)
// to the buffered writer.
func renderDefault(bw *bufio.Writer, tree *Tree, indent int) error {
	if tree == nil {
		return nil
	}

	err := writeIndentBuffered(bw, indent)
	if err != nil {
		return err
	}
	_, err = bw.WriteString(tree.Type.String())
	if err != nil {
		return err
	}

	for _, child := range tree.Children {
		err = bw.WriteByte('\n')
		if err != nil {
			return err
		}
		switch c := child.(type) {
		case TokenChild:
			err = writeIndentBuffered(bw, indent+1)
			if err != nil {
				return err
			}
			err = bw.WriteByte('\'')
			if err != nil {
				return err
			}
			_, err = bw.WriteString(c.String())
			if err != nil {
				return err
			}
			err = bw.WriteByte('\'')
			if err != nil {
				return err
			}
		case TreeChild:
			err = renderDefault(bw, c.Tree, indent+1)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func writeIndentBuffered(bw *bufio.Writer, columns int) error {
	for range columns {
		err := bw.WriteByte('\t')
		if err != nil {
			return err
		}
	}
	return nil
}

// Render writes the tree to w in the specified format. See [Format] for available formats.
func (tree *Tree) Render(w io.Writer, format Format) error {
	if tree == nil {
		return nil
	}
	bw := bufio.NewWriter(w)

	var err error
	switch format {
	case Default:
		err = renderDefault(bw, tree, 0)
	case Scheme:
		err = renderScheme(bw, tree, 0)
	default:
		panic(fmt.Errorf("rendering tree in format %q is not implemented", format))
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

// renderScheme writes the tree in scheme format (S-expressions with position annotations) to the
// buffered writer.
func renderScheme(bw *bufio.Writer, tree *Tree, indent int) error {
	if tree == nil {
		return nil
	}

	err := writeIndentBuffered(bw, indent)
	if err != nil {
		return err
	}
	err = bw.WriteByte('(')
	if err != nil {
		return err
	}
	_, err = bw.WriteString(tree.Type.String())
	if err != nil {
		return err
	}
	err = renderPosition(bw, tree.Start, tree.End)
	if err != nil {
		return err
	}

	for _, child := range tree.Children {
		err = bw.WriteByte('\n')
		if err != nil {
			return err
		}
		switch c := child.(type) {
		case TokenChild:
			err = writeIndentBuffered(bw, indent+1)
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
			_, err = bw.WriteString(c.String())
			if err != nil {
				return err
			}
			err = bw.WriteByte('\'')
			if err != nil {
				return err
			}
			err = renderPosition(bw, c.Start, c.End)
			if err != nil {
				return err
			}
			err = bw.WriteByte(')')
			if err != nil {
				return err
			}
		case TreeChild:
			err = renderScheme(bw, c.Tree, indent+1)
			if err != nil {
				return err
			}
		}
	}
	err = bw.WriteByte(')')
	if err != nil {
		return err
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

// Child is a marker interface for tree node children. Implementations are [TreeChild] and
// [TokenChild].
type Child interface {
	child()
}

// TreeChild wraps a [Tree] as a child of another tree node.
type TreeChild struct {
	*Tree
}

func (TreeChild) child() {}

// TokenChild wraps a [token.Token] as a child of a tree node.
type TokenChild struct {
	token.Token
}

func (TokenChild) child() {}
