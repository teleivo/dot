// Package hover provides hover documentation for DOT graph elements.
package hover

import (
	"github.com/teleivo/dot"
	"github.com/teleivo/dot/lsp/internal/attribute"
	"github.com/teleivo/dot/lsp/internal/rpc"
	"github.com/teleivo/dot/lsp/internal/tree"
	"github.com/teleivo/dot/token"
)

// Info returns hover information for the symbol at the given position.
func Info(root *dot.Tree, pos token.Position) *rpc.Hover {
	matchAttr := tree.Find(root, pos, dot.KindAttribute)
	if matchAttr.Tree == nil || len(matchAttr.Tree.Children) == 0 {
		return nil
	}

	matchAttrName, ok := matchAttr.Tree.Children[0].(dot.TreeChild)
	if !ok || len(matchAttrName.Children) == 0 {
		return nil
	}

	id, ok := matchAttrName.Children[0].(dot.TreeChild)
	if !ok || len(id.Children) == 0 {
		return nil
	}
	attrNameTok, ok := id.Children[0].(dot.TokenChild)
	if !ok {
		return nil
	}

	var attrFound *attribute.Attribute
	for _, v := range attribute.Attributes {
		if attrNameTok.Literal == v.Name {
			attrFound = &v
			break
		}
	}
	if attrFound == nil {
		return nil
	}

	matchAttrValue := tree.Find(matchAttr.Tree, pos, dot.KindAttrValue)
	if matchAttrValue.Tree == nil { // hover must be on attribute name
		return &rpc.Hover{Contents: rpc.MarkupContent{Kind: "markdown", Value: attrFound.MarkdownDoc}}
	}

	// hover might be on attribute value
	id, ok = matchAttrValue.Tree.Children[0].(dot.TreeChild)
	if !ok || len(id.Children) == 0 {
		return nil
	}
	attrValueTok, ok := id.Children[0].(dot.TokenChild)
	if !ok {
		return nil
	}

	for _, v := range attrFound.Type.Values() {
		if attrValueTok.Literal == v.Value {
			return &rpc.Hover{Contents: rpc.MarkupContent{Kind: "markdown", Value: v.MarkdownDoc(attrFound.Type)}}
		}
	}

	return nil
}
