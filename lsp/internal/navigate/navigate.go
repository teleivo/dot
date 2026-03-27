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

// DocumentSymbols returns the document symbols for the given syntax tree.
func DocumentSymbols(t *dot.Tree, root int) []rpc.DocumentSymbol {
	var symbols []rpc.DocumentSymbol
	symbols, _ = collectSymbols(t, root, symbols, 0, 0)
	return symbols
}

func collectSymbols(t *dot.Tree, parent int, result []rpc.DocumentSymbol, depth, items int) ([]rpc.DocumentSymbol, int) {
	nr := t.Children(parent)
	for i := nr.Start; i < nr.End; i = t.Next(i) {
		n := t.NodeAt(i)
		if n.IsToken() {
			continue
		}
		switch n.Kind {
		case dot.KindGraph, dot.KindSubgraph:
			if depth >= maxDepth || items >= maxItems {
				return result, items
			}
			sym := documentSymbol(t, i)
			var children []rpc.DocumentSymbol
			children, items = collectSymbols(t, i, children, depth+1, items)
			sym.Children = children

			result = append(result, *sym)
			items++
		default:
			if items >= maxItems-1 {
				return result, items
			}

			sym := documentSymbol(t, i)
			if sym != nil {
				result = append(result, *sym)
				items++
			}

			result, items = collectSymbols(t, i, result, depth, items)
		}
	}

	return result, items
}

func documentSymbol(t *dot.Tree, i int) *rpc.DocumentSymbol {
	n := t.NodeAt(i)
	switch n.Kind {
	case dot.KindGraph:
		keyword, _ := t.FirstToken(i, token.Graph|token.Digraph)
		result := rpc.DocumentSymbol{
			Detail: keyword.Kind.String(),
			Kind:   rpc.SymbolKindModule,
			Range:  rpc.RangeFromToken(n.Start, n.End),
		}
		if id, ok := t.FirstID(i); ok {
			result.Name = id.Literal
			result.SelectionRange = rpc.RangeFromToken(id.Start, id.End)
		} else {
			result.SelectionRange = rpc.RangeFromToken(keyword.Start, keyword.End)
		}
		return &result
	case dot.KindSubgraph:
		keyword, _ := t.FirstToken(i, token.Subgraph)
		result := rpc.DocumentSymbol{
			Detail: keyword.Kind.String(),
			Kind:   rpc.SymbolKindNamespace,
			Range:  rpc.RangeFromToken(n.Start, n.End),
		}
		if id, ok := t.FirstID(i); ok {
			result.Name = id.Literal
			result.SelectionRange = rpc.RangeFromToken(id.Start, id.End)
		} else {
			result.SelectionRange = rpc.RangeFromToken(keyword.Start, keyword.End)
		}
		return &result
	case dot.KindNodeStmt:
		treeRange := rpc.RangeFromToken(n.Start, n.End)
		result := rpc.DocumentSymbol{
			Kind:           rpc.SymbolKindVariable,
			Range:          treeRange,
			SelectionRange: treeRange,
		}
		if nodeIDIdx, ok := t.FirstTree(i, dot.KindNodeID); ok {
			if id, ok := t.FirstID(nodeIDIdx); ok {
				result.Name = id.Literal
				result.SelectionRange = rpc.RangeFromToken(id.Start, id.End)
			}
		}
		return &result
	case dot.KindEdgeStmt:
		var sb strings.Builder
		nr := t.Children(i)
		for j := nr.Start; j < nr.End; j = t.Next(j) {
			cn := t.NodeAt(j)
			if !cn.IsToken() && cn.Kind == dot.KindNodeID {
				if id, ok := t.FirstID(j); ok {
					sb.WriteString(id.Literal)
				}
			} else if cn.IsToken() && cn.TokenKind&(token.DirectedEdge|token.UndirectedEdge) != 0 {
				sb.WriteByte(' ')
				sb.WriteString(cn.Literal)
				sb.WriteByte(' ')
			}
		}
		treeRange := rpc.RangeFromToken(n.Start, n.End)
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
func Definition(t *dot.Tree, root int, uri rpc.DocumentURI, pos token.Position) *rpc.Location {
	match := tree.Find(t, pos, dot.KindNodeID)
	if match.Index == -1 {
		return nil
	}
	id, ok := t.FirstID(match.Index)
	if !ok {
		return nil
	}

	def := firstNodeID(t, root, id.Literal)
	if def == nil {
		return nil
	}
	return &rpc.Location{URI: uri, Range: rpc.RangeFromToken(def.Start, def.End)}
}

func firstNodeID(t *dot.Tree, parent int, name string) *token.Token {
	nr := t.Children(parent)
	for i := nr.Start; i < nr.End; i = t.Next(i) {
		n := t.NodeAt(i)
		if n.IsToken() {
			continue
		}
		if n.Kind == dot.KindNodeID {
			if id, ok := t.FirstID(i); ok && id.Literal == name {
				return &id
			}
		} else if found := firstNodeID(t, i, name); found != nil {
			return found
		}
	}
	return nil
}

// References returns all locations where the node ID at the given position is used.
func References(t *dot.Tree, root int, uri rpc.DocumentURI, pos token.Position) []rpc.Location {
	match := tree.Find(t, pos, dot.KindNodeID)
	if match.Index == -1 {
		return nil
	}
	id, ok := t.FirstID(match.Index)
	if !ok {
		return nil
	}

	var result []rpc.Location
	return collectReferences(t, root, uri, id.Literal, result)
}

func collectReferences(t *dot.Tree, parent int, uri rpc.DocumentURI, name string, result []rpc.Location) []rpc.Location {
	nr := t.Children(parent)
	for i := nr.Start; i < nr.End; i = t.Next(i) {
		n := t.NodeAt(i)
		if n.IsToken() {
			continue
		}
		if n.Kind == dot.KindNodeID {
			if id, ok := t.FirstID(i); ok && id.Literal == name {
				result = append(result, rpc.Location{URI: uri, Range: rpc.RangeFromToken(id.Start, id.End)})
			}
		} else {
			result = collectReferences(t, i, uri, name, result)
		}
	}
	return result
}
