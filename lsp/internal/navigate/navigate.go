// Package navigate provides code navigation features for DOT graph documents.
//
// This package implements LSP navigation capabilities including document symbols,
// go-to-definition, and find references.
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
	symbols, _ = collectSymbols(root, symbols, 0, 0)
	return symbols
}

func collectSymbols(root *dot.Tree, result []rpc.DocumentSymbol, depth, items int) ([]rpc.DocumentSymbol, int) {
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
				children, items = collectSymbols(c.Tree, children, depth+1, items)
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

				result, items = collectSymbols(c.Tree, result, depth, items)
			}
		}
	}

	return result, items
}

func documentSymbol(t *dot.Tree) *rpc.DocumentSymbol {
	switch t.Kind {
	case dot.KindGraph:
		keyword, _ := dot.TokenFirst(t, token.Graph|token.Digraph)
		result := rpc.DocumentSymbol{
			Detail: keyword.Kind.String(),
			Kind:   rpc.SymbolKindModule,
			Range:  rpc.RangeFromToken(t.Start, t.End),
		}
		if id, ok := dot.FirstID(t); ok {
			result.Name = id.Literal
			result.SelectionRange = rpc.RangeFromToken(id.Start, id.End)
		} else {
			result.SelectionRange = rpc.RangeFromToken(keyword.Start, keyword.End)
		}
		return &result
	case dot.KindSubgraph:
		keyword, _ := dot.TokenFirst(t, token.Subgraph)
		result := rpc.DocumentSymbol{
			Detail: keyword.Kind.String(),
			// Namespace: subgraphs group statements but don't create scope in DOT
			Kind:  rpc.SymbolKindNamespace,
			Range: rpc.RangeFromToken(t.Start, t.End),
		}
		if id, ok := dot.FirstID(t); ok {
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
		nodeID, _ := dot.TreeFirst(t, dot.KindNodeID)
		if id, ok := dot.FirstID(nodeID); ok {
			result.Name = id.Literal
			result.SelectionRange = rpc.RangeFromToken(id.Start, id.End)
		}
		return &result
	case dot.KindEdgeStmt:
		var sb strings.Builder
		for _, child := range t.Children {
			if c, ok := child.(dot.TreeChild); ok && c.Kind == dot.KindNodeID {
				if id, ok := dot.FirstID(c.Tree); ok {
					sb.WriteString(id.Literal)
				}
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

// Definition returns the location of the definition for the node ID at the given position.
// A definition is the first occurrence of a node ID in the document, whether it appears
// in a node statement or an edge statement.
//
// Returns nil if the position is not on a node ID or the tree is nil.
func Definition(root *dot.Tree, uri rpc.DocumentURI, pos token.Position) *rpc.Location {
	match := tree.Find(root, pos, dot.KindNodeID)
	if match.Tree == nil {
		return nil
	}
	id, ok := dot.FirstID(match.Tree)
	if !ok {
		return nil
	}

	def := firstNodeID(root, id.Literal)
	return &rpc.Location{URI: uri, Range: rpc.RangeFromToken(def.Start, def.End)}
}

func firstNodeID(root *dot.Tree, name string) *token.Token {
	if root == nil {
		return nil
	}

	for _, child := range root.Children {
		switch c := child.(type) {
		case dot.TreeChild:
			if c.Kind == dot.KindNodeID {
				if id, ok := dot.FirstID(c.Tree); ok && id.Literal == name {
					return &id
				}
			} else if found := firstNodeID(c.Tree, name); found != nil {
				return found
			}
		}
	}

	return nil
}

// References returns all locations where the node ID at the given position is used.
// This finds all occurrences of the same node ID throughout the document, whether they
// appear in node statements or edge statements.
//
// Returns nil if the position is not on a node ID or the tree is nil.
func References(root *dot.Tree, uri rpc.DocumentURI, pos token.Position) []rpc.Location {
	match := tree.Find(root, pos, dot.KindNodeID)
	if match.Tree == nil {
		return nil
	}
	id, ok := dot.FirstID(match.Tree)
	if !ok {
		return nil
	}

	var result []rpc.Location
	return collectReferences(root, uri, id.Literal, result)
}

func collectReferences(root *dot.Tree, uri rpc.DocumentURI, name string, result []rpc.Location) []rpc.Location {
	if root == nil {
		return result
	}

	for _, child := range root.Children {
		switch c := child.(type) {
		case dot.TreeChild:
			if c.Kind == dot.KindNodeID {
				if id, ok := dot.FirstID(c.Tree); ok && id.Literal == name {
					result = append(result, rpc.Location{URI: uri, Range: rpc.RangeFromToken(id.Start, id.End)})
				}
			} else {
				result = collectReferences(c.Tree, uri, name, result)
			}
		}
	}

	return result
}
