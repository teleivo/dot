// Package hover provides hover documentation for DOT graph elements.
package hover

import (
	"strings"

	"github.com/teleivo/dot"
	"github.com/teleivo/dot/lsp/internal/attribute"
	"github.com/teleivo/dot/lsp/internal/rpc"
	"github.com/teleivo/dot/lsp/internal/tree"
	"github.com/teleivo/dot/token"
)

// Info returns hover information for the symbol at the given position.
func Info(t *dot.Tree, pos token.Position) *rpc.Hover {
	matchAttr := tree.Find(t, pos, dot.KindAttribute)
	if matchAttr.Index == -1 {
		return nil
	}

	attrName := tree.AttrName(t, matchAttr.Index)
	if attrName == "" {
		return nil
	}

	var attrFound *attribute.Attribute
	for _, v := range attribute.Attributes {
		if attrName == v.Name {
			attrFound = &v
			break
		}
	}
	if attrFound == nil {
		return nil
	}

	matchAttrValue := tree.Find(t, pos, dot.KindAttrValue)
	if matchAttrValue.Index == -1 {
		return &rpc.Hover{Contents: rpc.MarkupContent{Kind: "markdown", Value: attrFound.MarkdownDoc}}
	}

	attrValueTok, ok := t.FirstID(matchAttrValue.Index)
	if !ok {
		return nil
	}

	unquoted := strings.Trim(attrValueTok.Literal, "\"")
	for _, v := range attrFound.Type.Values() {
		if unquoted == v.Value {
			return &rpc.Hover{Contents: rpc.MarkupContent{Kind: "markdown", Value: v.MarkdownDoc(attrFound.Type)}}
		}
	}

	return nil
}
