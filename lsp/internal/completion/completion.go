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
func Items(t *dot.Tree, pos token.Position) []rpc.CompletionItem {
	ctx := result{Comp: tree.Graph}
	context(t, 0, pos, &ctx)

	var items []rpc.CompletionItem
	if ctx.AttrName == "" {
		items = attributeNameItems(ctx)
	} else if attr := findAttr(ctx.AttrName, ctx.Comp); attr != nil {
		items = attributeValueItems(attr, ctx)
	}
	return items
}

func attributeNameItems(ctx result) []rpc.CompletionItem {
	var items []rpc.CompletionItem
	for _, attr := range attribute.Attributes {
		if strings.HasPrefix(attr.Name, ctx.Prefix) && attr.UsedBy&ctx.Comp != 0 {
			items = append(items, attributeNameItem(attr, ctx.HasEqual))
		}
	}
	return items
}

func attributeValueItems(attr *attribute.Attribute, ctx result) []rpc.CompletionItem {
	var items []rpc.CompletionItem
	for _, v := range attr.Type.ValuesFor(ctx.Comp) {
		if strings.HasPrefix(v.Value, ctx.Prefix) {
			items = append(items, attributeValueItem(v, attr.Type))
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

func context(t *dot.Tree, nodeIdx int, pos token.Position, res *result) {
	n := t.NodeAt(nodeIdx)
	if n.IsToken() {
		return
	}

	switch n.Kind {
	case dot.KindSubgraph:
		res.Comp = tree.Subgraph
		if id, ok := t.FirstID(nodeIdx); ok && strings.HasPrefix(id.Literal, "cluster_") {
			res.Comp = tree.Cluster
		}
	case dot.KindNodeStmt:
		if _, ok := t.FirstTree(nodeIdx, dot.KindAttrList); ok {
			res.Comp = tree.Node
		}
	case dot.KindEdgeStmt:
		res.Comp = tree.Edge
	case dot.KindAttrStmt:
		if tok, ok := t.TokenAt(nodeIdx, token.Graph|token.Node|token.Edge, 0); ok {
			switch tok.Kind {
			case token.Graph:
				if res.Comp != tree.Cluster && res.Comp != tree.Subgraph {
					res.Comp = tree.Graph
				}
			case token.Node:
				res.Comp = tree.Node
			case token.Edge:
				res.Comp = tree.Edge
			}
		}
	}

	prevIdx := -1
	nr := t.Children(nodeIdx)
	for i := nr.Start; i < nr.End; i = t.Next(i) {
		cn := t.NodeAt(i)
		if !cn.IsToken() {
			end := token.Position{Line: cn.End.Line, Column: cn.End.Column + 1}
			if !pos.Before(cn.Start) && !pos.After(end) {
				if cn.Kind == dot.KindAttrName && pos.After(cn.End) {
					if _, ok := t.FirstToken(nodeIdx, token.Equal); ok {
						prevIdx = i
						continue
					}
				}
				if n.Kind == dot.KindAttribute {
					switch cn.Kind {
					case dot.KindAttrName:
						_, res.HasEqual = t.FirstToken(nodeIdx, token.Equal)
					case dot.KindAttrValue:
						res.AttrName = tree.AttrName(t, nodeIdx)
					}
				}
				if cn.Kind == dot.KindErrorTree && (n.Kind == dot.KindAList || n.Kind == dot.KindStmtList) && prevIdx != -1 {
					prevNode := t.NodeAt(prevIdx)
					_, hasEqual := t.FirstToken(prevIdx, token.Equal)
					_, hasValue := t.FirstTree(prevIdx, dot.KindAttrValue)
					if prevNode.Kind == dot.KindAttribute && hasEqual && !hasValue {
						res.AttrName = tree.AttrName(t, prevIdx)
					}
				}
				context(t, i, pos, res)
				return
			}
			prevIdx = i
		} else {
			end := token.Position{Line: cn.End.Line, Column: cn.End.Column + 1}
			if !pos.Before(cn.Start) && !pos.After(end) {
				if cn.TokenKind == token.Equal && n.Kind == dot.KindAttribute {
					res.AttrName = tree.AttrName(t, nodeIdx)
					if pos.After(cn.End) {
						continue
					}
					return
				}
				if cn.TokenKind == token.ID {
					res.Prefix = cn.Token().String()
				}
				return
			}
		}
	}
}
