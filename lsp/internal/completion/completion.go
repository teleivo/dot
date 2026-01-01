package completion

import (
	"strings"

	"github.com/teleivo/dot"
	"github.com/teleivo/dot/lsp/internal/rpc"
	"github.com/teleivo/dot/token"
)

// Items returns completion items for the given tree at the given position.
func Items(tree *dot.Tree, pos token.Position) []rpc.CompletionItem {
	ctx := context(tree, pos)

	var candidates []Attribute
	for _, attr := range Attributes {
		if strings.HasPrefix(attr.Name, ctx.Prefix) && attr.UsedBy&ctx.AttrCtx != 0 {
			candidates = append(candidates, attr)
		}
	}

	items := make([]rpc.CompletionItem, len(candidates))
	for i, candidate := range candidates {
		items[i] = completionItem(candidate)
	}
	return items
}

func completionItem(attr Attribute) rpc.CompletionItem {
	kind := rpc.CompletionItemKindProperty
	detail := attr.UsedBy.String()
	text := attr.Name + "="
	return rpc.CompletionItem{
		Label:         attr.Name,
		InsertText:    &text,
		Kind:          &kind,
		Detail:        &detail,
		Documentation: &attr.Documentation,
	}
}

// result accumulates context as we traverse the tree.
type result struct {
	Prefix   string           // text typed so far (for filtering)
	AttrCtx  AttributeContext // element context (Node, Edge, Graph, etc.)
	AttrName string           // when non-empty, cursor is in value position for this attribute
}

// context finds the prefix text at the cursor position and determines the attribute context.
// Returns the prefix string (text before cursor within the current token), the context
// for filtering attributes (Node, Edge, Graph, etc.), and the attribute name when in value position.
// Falls back to empty prefix and Graph context if unable to determine a more specific context.
func context(tree *dot.Tree, pos token.Position) result {
	r := result{AttrCtx: Graph}
	contextRec(tree, pos, &r)
	return r
}

func contextRec(tree *dot.Tree, pos token.Position, result *result) {
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

	// capture attribute name if pos is on or past =
	if tree.Type == dot.KindAttribute && len(tree.Children) >= 2 {
		nameTree, okTree := tree.Children[0].(dot.TreeChild)
		equal, okEqual := tree.Children[1].(dot.TokenChild)
		if okTree && nameTree.Type == dot.KindID && len(nameTree.Children) > 0 {
			if id, okID := nameTree.Children[0].(dot.TokenChild); okID && okEqual && equal.Type == token.Equal && !pos.Before(equal.End) {
				result.AttrName = id.Literal
			}
		}
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
				contextRec(c.Tree, pos, result)
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
				if c.Type == token.ID {
					// If AttrName is set and Prefix equals AttrName, we're on the name ID in value position - clear Prefix
					if result.AttrName != "" && c.String() == result.AttrName {
						return
					}
					result.Prefix = c.String()
				}
				// continue to find the value ID
				if c.Type != token.Equal {
					return
				}
			}
		}
	}
}
