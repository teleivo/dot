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

	var items []rpc.CompletionItem
	if ctx.AttrName == "" { // attribute name completion
		for _, attr := range Attributes {
			if strings.HasPrefix(attr.Name, ctx.Prefix) && attr.UsedBy&ctx.Comp != 0 {
				items = append(items, attributeNameItem(attr, ctx.HasEqual))
			}
		}
	} else if attr := findAttr(ctx.AttrName, ctx.Comp); attr != nil { // attribute value completion
		for _, v := range attr.Type.ValuesFor(ctx.Comp) {
			if strings.HasPrefix(v.Value, ctx.Prefix) {
				items = append(items, attributeValueItem(v, attr.Type))
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

func findAttr(name string, comp tree.Component) *Attribute {
	for _, attr := range Attributes {
		if attr.Name == name && attr.UsedBy&comp != 0 {
			return &attr
		}
	}
	return nil
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

// context finds the prefix text at the cursor position and determines the attribute context.
func context(root *dot.Tree, pos token.Position, res *result) {
	if root == nil {
		return
	}

	switch root.Type {
	case dot.KindSubgraph:
		res.Comp = tree.Subgraph
	case dot.KindNodeStmt:
		if tree.HasAttrList(root) {
			res.Comp = tree.Node
		}
	case dot.KindEdgeStmt:
		res.Comp = tree.Edge
	}

	for i, child := range root.Children {
		switch c := child.(type) {
		case dot.TreeChild:
			if root.Type == dot.KindSubgraph && i == 1 && c.Type == dot.KindID && len(c.Children) > 0 {
				if id, ok := c.Children[0].(dot.TokenChild); ok && strings.HasPrefix(id.Literal, "cluster_") {
					res.Comp = tree.Cluster
				}
			}

			end := token.Position{Line: c.End.Line, Column: c.End.Column + 1}
			if !pos.Before(c.Start) && !pos.After(end) {
				// skip AttrName if cursor is past its actual end AND there's a = token
				// (meaning we're in value position, not still typing the name)
				if c.Type == dot.KindAttrName && pos.After(c.End) && tree.HasEqualSign(root) {
					continue
				}
				if root.Type == dot.KindAttribute {
					switch c.Type {
					case dot.KindAttrName:
						res.HasEqual = tree.HasEqualSign(root)
					case dot.KindAttrValue:
						res.AttrName = tree.AttrName(root)
					}
				}
				context(c.Tree, pos, res)
				return
			}
		case dot.TokenChild:
			if i == 0 && root.Type == dot.KindAttrStmt {
				switch c.Type {
				case token.Graph: // graph [name=value]
					if res.Comp != tree.Cluster && res.Comp != tree.Subgraph {
						res.Comp = tree.Graph
					}
				case token.Node: // node [name=value]
					res.Comp = tree.Node
				case token.Edge: // edge [name=value]
					res.Comp = tree.Edge
				}
			}

			end := token.Position{Line: c.End.Line, Column: c.End.Column + 1}
			if !pos.Before(c.Start) && !pos.After(end) {
				// cursor on or after = in Attribute means value position
				if c.Type == token.Equal && root.Type == dot.KindAttribute {
					res.AttrName = tree.AttrName(root)
					if pos.After(c.End) {
						continue // cursor is past =, continue to find AttrValue for prefix
					}
					return
				}
				if c.Type == token.ID {
					res.Prefix = c.String()
				}
				return
			}
		}
	}
}
