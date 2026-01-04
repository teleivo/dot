// Package completion provides autocompletion for DOT graph attributes.
package completion

import (
	"strings"

	"github.com/teleivo/dot"
	"github.com/teleivo/dot/lsp/internal/rpc"
	"github.com/teleivo/dot/lsp/internal/tree"
	"github.com/teleivo/dot/token"
)

// Items returns completion items for the given tree at the given position.
func Items(root *dot.Tree, pos token.Position) []rpc.CompletionItem {
	ctx := result{Comp: tree.Graph}
	context(root, pos, &ctx)

	// TODO simplify this
	var items []rpc.CompletionItem
	if ctx.AttrName == "" { // attribute name completion
		for _, attr := range Attributes {
			if strings.HasPrefix(attr.Name, ctx.Prefix) && attr.UsedBy&ctx.Comp != 0 {
				items = append(items, attributeNameItem(attr, ctx.HasEqual))
			}
		}
	} else { // attribute value completion
		var at *Attribute
		for _, attr := range Attributes {
			// TODO does context still matter?
			if attr.Name == ctx.AttrName && attr.UsedBy&ctx.Comp != 0 {
				at = &attr
				break
			}
		}
		if at != nil {
			for _, v := range at.Type.ValuesFor(ctx.Comp) {
				if strings.HasPrefix(v.Value, ctx.Prefix) {
					items = append(items, attributeValueItem(v, at.Type))
				}
			}
		}
	}
	return items
}

func attributeNameItem(attr Attribute, hasEqual bool) rpc.CompletionItem {
	kind := rpc.CompletionItemKindProperty
	detail := attr.Type.String()
	text := attr.Name
	if !hasEqual {
		text += "="
	}
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

func attributeValueItem(v AttrValue, attrType AttrType) rpc.CompletionItem {
	kind := rpc.CompletionItemKindValue
	detail := attrType.String()
	return rpc.CompletionItem{
		Label:  v.Value,
		Kind:   &kind,
		Detail: &detail,
		Documentation: &rpc.MarkupContent{
			Kind:  "markdown",
			Value: v.markdownDoc(attrType),
		},
	}
}

// result accumulates context as we traverse the tree.
type result struct {
	Prefix   string         // text typed so far (for filtering)
	Comp     tree.Component // element context (Node, Edge, Graph, etc.)
	AttrName string         // when non-empty, cursor is in value position for this attribute
	HasEqual bool           // true when = already exists in the attribute
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
func context(root *dot.Tree, pos token.Position, result *result) {
	if root == nil {
		return
	}

	switch root.Type {
	case dot.KindSubgraph:
		result.Comp = tree.Subgraph
	case dot.KindNodeStmt:
		result.Comp = tree.Node
	case dot.KindEdgeStmt:
		result.Comp = tree.Edge
	}

	for i, child := range root.Children {
		switch c := child.(type) {
		case dot.TreeChild:
			if root.Type == dot.KindSubgraph && i == 1 && c.Type == dot.KindID && len(c.Children) > 0 {
				if id, ok := c.Children[0].(dot.TokenChild); ok && strings.HasPrefix(id.Literal, "cluster_") {
					result.Comp = tree.Cluster
				}
			}

			end := token.Position{Line: c.End.Line, Column: c.End.Column + 1}
			if !pos.Before(c.Start) && !pos.After(end) {
				// skip AttrName if cursor is past its actual end AND there's a = token
				// (meaning we're in value position, not still typing the name)
				if c.Type == dot.KindAttrName && pos.After(c.End) && hasEqualToken(root) {
					continue
				}
				if root.Type == dot.KindAttribute {
					switch c.Type {
					case dot.KindAttrName:
						result.HasEqual = hasEqualToken(root)
					case dot.KindAttrValue:
						result.AttrName = attrNameFrom(root)
					}
				}
				context(c.Tree, pos, result)
				return
			}
		case dot.TokenChild:
			if i == 0 && root.Type == dot.KindAttrStmt {
				switch c.Type {
				case token.Graph: // graph [name=value]
					if result.Comp != tree.Cluster && result.Comp != tree.Subgraph {
						result.Comp = tree.Graph
					}
				case token.Node: // node [name=value]
					result.Comp = tree.Node
				case token.Edge: // edge [name=value]
					result.Comp = tree.Edge
				}
			}

			end := token.Position{Line: c.End.Line, Column: c.End.Column + 1}
			if !pos.Before(c.Start) && !pos.After(end) {
				// cursor on or after = in Attribute means value position
				if c.Type == token.Equal && root.Type == dot.KindAttribute {
					result.AttrName = attrNameFrom(root)
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
