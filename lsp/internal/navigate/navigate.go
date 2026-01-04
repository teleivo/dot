// Package navigate provides code navigation features for DOT graph documents.
//
// This package implements LSP navigation capabilities including document symbols,
// go-to-definition, and find-references. These features enable users to efficiently
// navigate and explore DOT graph structures within their editor.
//
// Document symbols expose the hierarchical structure of a DOT file: graphs contain
// subgraphs, nodes, and edges, allowing users to quickly jump to any element.
package navigate

import (
	"github.com/teleivo/dot"
	"github.com/teleivo/dot/lsp/internal/rpc"
)

// Limits for document symbols to handle large files.
const (
	// MaxItems is the maximum number of symbols to return.
	MaxItems = 1000
	// MaxDepth is the maximum nesting depth for symbols.
	MaxDepth = 4
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
	// TODO(human): implement document symbol extraction from the parse tree
	return nil
}
