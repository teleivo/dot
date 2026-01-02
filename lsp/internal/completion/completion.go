// Package completion provides autocompletion for DOT graph attributes.
package completion

import (
	"strings"

	"github.com/teleivo/dot"
	"github.com/teleivo/dot/lsp/internal/rpc"
	"github.com/teleivo/dot/token"
)

// Items returns completion items for the given tree at the given position.
func Items(tree *dot.Tree, pos token.Position) []rpc.CompletionItem {
	attrCtx := result{AttrCtx: Graph}
	context(tree, pos, &attrCtx)

	// TODO simplify this
	var items []rpc.CompletionItem
	if attrCtx.AttrName == "" { // attribute name completion
		for _, attr := range Attributes {
			if strings.HasPrefix(attr.Name, attrCtx.Prefix) && attr.UsedBy&attrCtx.AttrCtx != 0 {
				items = append(items, attributeNameItem(attr))
			}
		}
	} else { // attribute value completion
		var at *Attribute
		for _, attr := range Attributes {
			// TODO does context still matter?
			if attr.Name == attrCtx.AttrName && attr.UsedBy&attrCtx.AttrCtx != 0 {
				at = &attr
				break
			}
		}
		if at != nil {
			values := at.Type.ValuesFor(attrCtx.AttrCtx)
			items = make([]rpc.CompletionItem, len(values))
			for i, v := range values {
				items[i] = attributeValueItem(v.Value, at.Type)
			}
		}
	}
	return items
}

// TODO where to put this? or rename completion package?
// Hover returns hover information for the symbol at the given position.
func Hover(tree *dot.Tree, pos token.Position) *rpc.Hover {
	attrCtx := result{AttrCtx: Graph}
	context(tree, pos, &attrCtx)

	// Cursor is on attribute name (Prefix contains the name)
	if attrCtx.AttrName == "" && attrCtx.Prefix != "" {
		for _, attr := range Attributes {
			if attr.Name == attrCtx.Prefix {
				return &rpc.Hover{Contents: rpc.MarkupContent{Kind: "markdown", Value: attr.MarkdownDoc}}
			}
		}
	}

	// Cursor is on attribute value (AttrName contains the attribute, Prefix contains the value)
	if attrCtx.AttrName != "" {
		for _, attr := range Attributes {
			if attr.Name == attrCtx.AttrName && attr.UsedBy&attrCtx.AttrCtx != 0 {
				for _, v := range attr.Type.Values() {
					if attrCtx.Prefix == v.Value {
						return &rpc.Hover{Contents: rpc.MarkupContent{Kind: "markdown", Value: v.markdownDoc(attr.Type)}}
					}
				}
				break
			}
		}
	}

	return nil
}

func attributeNameItem(attr Attribute) rpc.CompletionItem {
	kind := rpc.CompletionItemKindProperty
	detail := attr.Type.String()
	text := attr.Name + "="
	return rpc.CompletionItem{
		Label:      attr.Name,
		InsertText: &text,
		Kind:       &kind,
		Detail:     &detail,
		Documentation: &rpc.MarkupContent{
			Kind:  "markdown",
			Value: attr.MarkdownDoc,
		},
	}
}

func attributeValueItem(value string, attrType AttrType) rpc.CompletionItem {
	kind := rpc.CompletionItemKindValue
	detail := attrType.String()
	return rpc.CompletionItem{
		Label:  value,
		Kind:   &kind,
		Detail: &detail,
		Documentation: &rpc.MarkupContent{
			Kind:  "markdown",
			Value: "[" + attrType.String() + "](" + attrType.URL() + ")",
		},
	}
}

// result accumulates context as we traverse the tree.
type result struct {
	Prefix   string           // text typed so far (for filtering)
	AttrCtx  AttributeContext // element context (Node, Edge, Graph, etc.)
	AttrName string           // when non-empty, cursor is in value position for this attribute
}

// hasEqualToken checks if the tree has an = token child.
func hasEqualToken(tree *dot.Tree) bool {
	for _, child := range tree.Children {
		if tok, ok := child.(dot.TokenChild); ok && tok.Type == token.Equal {
			return true
		}
	}
	return false
}

// attrNameFrom extracts the attribute name from an Attribute node.
func attrNameFrom(attr *dot.Tree) string {
	// Attribute: AttrName '=' AttrValue
	if len(attr.Children) == 0 {
		return ""
	}
	nameTree, ok := attr.Children[0].(dot.TreeChild)
	if !ok || nameTree.Type != dot.KindAttrName || len(nameTree.Children) == 0 {
		return ""
	}
	idTree, ok := nameTree.Children[0].(dot.TreeChild)
	if !ok || idTree.Type != dot.KindID || len(idTree.Children) == 0 {
		return ""
	}
	tok, ok := idTree.Children[0].(dot.TokenChild)
	if !ok {
		return ""
	}
	return tok.Literal
}

// context finds the prefix text at the cursor position and determines the attribute context.
// Returns the prefix string (text before cursor within the current token), the context
// for filtering attributes (Node, Edge, Graph, etc.), and the attribute name when in value position.
func context(tree *dot.Tree, pos token.Position, result *result) {
	if tree == nil {
		return
	}

	switch tree.Type {
	case dot.KindSubgraph:
		result.AttrCtx = Subgraph
	case dot.KindNodeStmt:
		result.AttrCtx = Node
	case dot.KindEdgeStmt:
		result.AttrCtx = Edge
	}

	for i, child := range tree.Children {
		switch c := child.(type) {
		case dot.TreeChild:
			if tree.Type == dot.KindSubgraph && i == 1 && c.Type == dot.KindID && len(c.Children) > 0 {
				if id, ok := c.Children[0].(dot.TokenChild); ok && strings.HasPrefix(id.Literal, "cluster_") {
					result.AttrCtx = Cluster
				}
			}

			end := token.Position{Line: c.End.Line, Column: c.End.Column + 1}
			if !pos.Before(c.Start) && !pos.After(end) {
				// skip AttrName if cursor is past its actual end AND there's a = token
				// (meaning we're in value position, not still typing the name)
				if c.Type == dot.KindAttrName && pos.After(c.End) && hasEqualToken(tree) {
					continue
				}
				// when entering AttrValue, capture the attribute name from parent Attribute
				if c.Type == dot.KindAttrValue && tree.Type == dot.KindAttribute {
					result.AttrName = attrNameFrom(tree)
				}
				context(c.Tree, pos, result)
				return
			}
		case dot.TokenChild:
			if i == 0 && tree.Type == dot.KindAttrStmt {
				switch c.Type {
				case token.Graph: // graph [name=value]
					if result.AttrCtx != Cluster {
						result.AttrCtx = Graph
					}
				case token.Node: // node [name=value]
					result.AttrCtx = Node
				case token.Edge: // edge [name=value]
					result.AttrCtx = Edge
				}
			}

			end := token.Position{Line: c.End.Line, Column: c.End.Column + 1}
			if !pos.Before(c.Start) && !pos.After(end) {
				// cursor on or after = in Attribute means value position
				if c.Type == token.Equal && tree.Type == dot.KindAttribute {
					result.AttrName = attrNameFrom(tree)
					if pos.After(c.End) {
						continue // cursor is past =, continue to find AttrValue for prefix
					}
					return
				}
				if c.Type == token.ID {
					result.Prefix = c.String()
				}
				return
			}
		}
	}
}
