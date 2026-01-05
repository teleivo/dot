// Package completion provides autocompletion for DOT graph attributes.
package completion

import (
	"strings"

	"github.com/teleivo/dot"
	"github.com/teleivo/dot/lsp/internal/attribute"
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
		for _, attr := range attribute.Attributes {
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

func attributeNameItem(attr attribute.Attribute, hasEqual bool) rpc.CompletionItem {
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

func findAttr(name string, comp tree.Component) *attribute.Attribute {
	for _, attr := range attribute.Attributes {
		if attr.Name == name && attr.UsedBy&comp != 0 {
			return &attr
		}
	}
	return nil
}

func attributeValueItem(v attribute.AttrValue, attrType attribute.AttrType) rpc.CompletionItem {
	kind := rpc.CompletionItemKindValue
	detail := attrType.String()
	return rpc.CompletionItem{
		Label:  v.Value,
		Kind:   &kind,
		Detail: &detail,
		Documentation: &rpc.MarkupContent{
			Kind:  "markdown",
			Value: v.MarkdownDoc(attrType),
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

	switch root.Kind {
	case dot.KindSubgraph:
		res.Comp = tree.Subgraph
	case dot.KindNodeStmt:
		if tree.HasKind(root, dot.KindAttrList) {
			res.Comp = tree.Node
		}
	case dot.KindEdgeStmt:
		res.Comp = tree.Edge
	}

	for i, child := range root.Children {
		switch c := child.(type) {
		case dot.TreeChild:
			if root.Kind == dot.KindSubgraph && i == 1 && c.Kind == dot.KindID && len(c.Children) > 0 {
				if id, ok := c.Children[0].(dot.TokenChild); ok && strings.HasPrefix(id.Literal, "cluster_") {
					res.Comp = tree.Cluster
				}
			}

			end := token.Position{Line: c.End.Line, Column: c.End.Column + 1}
			if !pos.Before(c.Start) && !pos.After(end) {
				// skip AttrName if cursor is past its actual end AND there's a = token
				// (meaning we're in value position, not still typing the name)
				if c.Kind == dot.KindAttrName && pos.After(c.End) && tree.HasToken(root, token.Equal) {
					continue
				}
				if root.Kind == dot.KindAttribute {
					switch c.Kind {
					case dot.KindAttrName:
						res.HasEqual = tree.HasToken(root, token.Equal)
					case dot.KindAttrValue:
						res.AttrName = tree.AttrName(root)
					}
				}
				// When cursor is inside ErrorTree following an incomplete Attribute
				// (has = but no AttrValue), we're in value position for that attribute.
				// This can happen in AList (inside [...]) or StmtList (top-level attrs).
				if c.Kind == dot.KindErrorTree && (root.Kind == dot.KindAList || root.Kind == dot.KindStmtList) && i > 0 {
					if prev, ok := root.Children[i-1].(dot.TreeChild); ok {
						if prev.Kind == dot.KindAttribute && tree.HasToken(prev.Tree, token.Equal) && !tree.HasKind(prev.Tree, dot.KindAttrValue) {
							res.AttrName = tree.AttrName(prev.Tree)
						}
					}
				}
				context(c.Tree, pos, res)
				return
			}
		case dot.TokenChild:
			if i == 0 && root.Kind == dot.KindAttrStmt {
				switch c.Kind {
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
				if c.Kind == token.Equal && root.Kind == dot.KindAttribute {
					res.AttrName = tree.AttrName(root)
					if pos.After(c.End) {
						continue // cursor is past =, continue to find AttrValue for prefix
					}
					return
				}
				if c.Kind == token.ID {
					res.Prefix = c.String()
				}
				return
			}
		}
	}
}
