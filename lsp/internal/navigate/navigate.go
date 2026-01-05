// Package navigate provides code navigation features for DOT graph documents.
//
// This package implements LSP navigation capabilities including document symbols.
package navigate

import (
	"strings"

	"github.com/teleivo/dot"
	"github.com/teleivo/dot/lsp/internal/rpc"
	"github.com/teleivo/dot/lsp/internal/tree"
	"github.com/teleivo/dot/token"
)

// Limits for document symbols to handle large files.
const (
	// maxDepth is the maximum nesting depth for symbols.
	maxDepth = 4
	// maxItems is the maximum number of symbols to return.
	maxItems = 1000
)

// DocumentSymbols returns the document symbols for the given parse tree.
// Symbols represent navigable elements in the DOT file such as graphs, subgraphs,
// nodes, and edges. The returned symbols form a hierarchy matching the document structure.
//
// For DOT files, the symbol hierarchy is:
//   - Graph (top-level digraph/graph) with Kind=Module, Detail="digraph"/"graph"
//   - Subgraph (subgraph blocks) with Kind=Namespace, Detail="subgraph"
//   - Node (node statements) with Kind=Variable
//   - Edge (edge statements like "a -> b") with Kind=Event
//
// Anonymous graphs/subgraphs have an empty Name with the keyword in Detail.
// Attribute statements (node [...], edge [...], graph [...]) are skipped.
//
// To handle large files, symbols are limited to MaxItems total and MaxDepth nesting.
func DocumentSymbols(root *dot.Tree) []rpc.DocumentSymbol {
	var symbols []rpc.DocumentSymbol
	symbols, _ = collect(root, symbols, 0, 0)
	return symbols
}

func collect(root *dot.Tree, result []rpc.DocumentSymbol, depth, items int) ([]rpc.DocumentSymbol, int) {
	if root == nil {
		return result, items
	}

	for _, child := range root.Children {
		switch c := child.(type) {
		case dot.TreeChild:
			switch c.Kind {
			case dot.KindGraph, dot.KindSubgraph:
				if depth >= maxDepth || items >= maxItems {
					return result, items
				}
				sym := documentSymbol(c.Tree)
				var children []rpc.DocumentSymbol
				children, items = collect(c.Tree, children, depth+1, items)
				sym.Children = children

				result = append(result, *sym)
				items++
			default:
				if items >= maxItems-1 { // need 1 spot for the parent
					return result, items
				}

				sym := documentSymbol(c.Tree)
				if sym != nil {
					result = append(result, *sym)
					items++
				}

				result, items = collect(c.Tree, result, depth, items)
			}
		}
	}

	return result, items
}

func documentSymbol(t *dot.Tree) *rpc.DocumentSymbol {
	switch t.Kind {
	case dot.KindGraph:
		keyword, _ := tree.GetToken(t, token.Graph|token.Digraph)
		result := rpc.DocumentSymbol{
			Detail: keyword.Kind.String(),
			Kind:   rpc.SymbolKindModule,
			Range:  rpc.RangeFromToken(t.Start, t.End),
		}
		idTree, ok := tree.GetKind(t, dot.KindID)
		if ok {
			id, _ := tree.GetToken(idTree, token.ID)
			result.Name = id.Literal
			result.SelectionRange = rpc.RangeFromToken(id.Start, id.End)
		} else {
			result.SelectionRange = rpc.RangeFromToken(keyword.Start, keyword.End)
		}
		return &result
	case dot.KindSubgraph:
		keyword, _ := tree.GetToken(t, token.Subgraph)
		result := rpc.DocumentSymbol{
			Detail: keyword.Kind.String(),
			Kind:   rpc.SymbolKindNamespace,
			Range:  rpc.RangeFromToken(t.Start, t.End),
		}
		idTree, ok := tree.GetKind(t, dot.KindID)
		if ok {
			id, _ := tree.GetToken(idTree, token.ID)
			result.Name = id.Literal
			result.SelectionRange = rpc.RangeFromToken(id.Start, id.End)
		} else {
			result.SelectionRange = rpc.RangeFromToken(keyword.Start, keyword.End)
		}
		return &result
	case dot.KindNodeStmt:
		treeRange := rpc.RangeFromToken(t.Start, t.End)
		result := rpc.DocumentSymbol{
			Kind:           rpc.SymbolKindVariable,
			Range:          treeRange,
			SelectionRange: treeRange,
		}
		nodeID, _ := tree.GetKind(t, dot.KindNodeID)
		idTree, ok := tree.GetKind(nodeID, dot.KindID)
		if ok {
			id, _ := tree.GetToken(idTree, token.ID)
			result.Name = id.Literal
			result.SelectionRange = rpc.RangeFromToken(id.Start, id.End)
		}
		return &result
	case dot.KindEdgeStmt:
		var sb strings.Builder
		for _, child := range t.Children {
			if c, ok := child.(dot.TreeChild); ok && c.Kind == dot.KindNodeID {
				idTree, _ := tree.GetKind(c.Tree, dot.KindID)
				id, _ := tree.GetToken(idTree, token.ID)
				sb.WriteString(id.Literal)
			} else if tok, ok := child.(dot.TokenChild); ok && tok.Kind&(token.DirectedEdge|token.UndirectedEdge) != 0 {
				sb.WriteByte(' ')
				sb.WriteString(tok.Literal)
				sb.WriteByte(' ')
			}
		}
		treeRange := rpc.RangeFromToken(t.Start, t.End)
		result := rpc.DocumentSymbol{
			Kind:           rpc.SymbolKindEvent,
			Name:           sb.String(),
			Range:          treeRange,
			SelectionRange: treeRange,
		}
		return &result
	}

	return nil
}
