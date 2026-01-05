package navigate

import (
	"testing"

	"github.com/teleivo/assertive/assert"
	"github.com/teleivo/dot"
	"github.com/teleivo/dot/lsp/internal/rpc"
)

func TestDocumentSymbols(t *testing.T) {
	tests := map[string]struct {
		src  string
		want []rpc.DocumentSymbol
	}{
		"EmptySource": {
			src:  ``,
			want: nil,
		},
		"SingleGraph": {
			src: `digraph foo { }`,
			want: []rpc.DocumentSymbol{
				{Name: "foo", Detail: "digraph", Kind: rpc.SymbolKindModule, Range: r(0, 0, 0, 15), SelectionRange: r(0, 8, 0, 11)},
			},
		},
		"AnonymousGraph": {
			src: `digraph { }`,
			want: []rpc.DocumentSymbol{
				{Name: "", Detail: "digraph", Kind: rpc.SymbolKindModule, Range: r(0, 0, 0, 11), SelectionRange: r(0, 0, 0, 7)},
			},
		},
		"UndirectedGraph": {
			src: `graph bar { }`,
			want: []rpc.DocumentSymbol{
				{Name: "bar", Detail: "graph", Kind: rpc.SymbolKindModule, Range: r(0, 0, 0, 13), SelectionRange: r(0, 6, 0, 9)},
			},
		},
		"CaseInsensitiveKeywords": {
			src: `DIGRAPH G { SUBGRAPH S { } }`,
			want: []rpc.DocumentSymbol{
				{
					Name: "G", Detail: "digraph", Kind: rpc.SymbolKindModule, Range: r(0, 0, 0, 28), SelectionRange: r(0, 8, 0, 9),
					Children: []rpc.DocumentSymbol{
						{Name: "S", Detail: "subgraph", Kind: rpc.SymbolKindNamespace, Range: r(0, 12, 0, 26), SelectionRange: r(0, 21, 0, 22)},
					},
				},
			},
		},
		"MultipleGraphs": {
			src: `digraph first { a } graph second { a; b }`,
			want: []rpc.DocumentSymbol{
				{
					Name: "first", Detail: "digraph", Kind: rpc.SymbolKindModule, Range: r(0, 0, 0, 19), SelectionRange: r(0, 8, 0, 13),
					Children: []rpc.DocumentSymbol{
						{Name: "a", Kind: rpc.SymbolKindVariable, Range: r(0, 16, 0, 17), SelectionRange: r(0, 16, 0, 17)},
					},
				},
				{
					Name: "second", Detail: "graph", Kind: rpc.SymbolKindModule, Range: r(0, 20, 0, 41), SelectionRange: r(0, 26, 0, 32),
					Children: []rpc.DocumentSymbol{
						{Name: "a", Kind: rpc.SymbolKindVariable, Range: r(0, 35, 0, 36), SelectionRange: r(0, 35, 0, 36)},
						{Name: "b", Kind: rpc.SymbolKindVariable, Range: r(0, 38, 0, 39), SelectionRange: r(0, 38, 0, 39)},
					},
				},
			},
		},
		"GraphWithNode": {
			src: `digraph { a }`,
			want: []rpc.DocumentSymbol{
				{
					Name: "", Detail: "digraph", Kind: rpc.SymbolKindModule, Range: r(0, 0, 0, 13), SelectionRange: r(0, 0, 0, 7),
					Children: []rpc.DocumentSymbol{
						{Name: "a", Kind: rpc.SymbolKindVariable, Range: r(0, 10, 0, 11), SelectionRange: r(0, 10, 0, 11)},
					},
				},
			},
		},
		"GraphWithMultipleNodes": {
			src: `digraph { a; b; c }`,
			want: []rpc.DocumentSymbol{
				{
					Name: "", Detail: "digraph", Kind: rpc.SymbolKindModule, Range: r(0, 0, 0, 19), SelectionRange: r(0, 0, 0, 7),
					Children: []rpc.DocumentSymbol{
						{Name: "a", Kind: rpc.SymbolKindVariable, Range: r(0, 10, 0, 11), SelectionRange: r(0, 10, 0, 11)},
						{Name: "b", Kind: rpc.SymbolKindVariable, Range: r(0, 13, 0, 14), SelectionRange: r(0, 13, 0, 14)},
						{Name: "c", Kind: rpc.SymbolKindVariable, Range: r(0, 16, 0, 17), SelectionRange: r(0, 16, 0, 17)},
					},
				},
			},
		},
		"GraphWithEdge": {
			src: `digraph { a -> b }`,
			want: []rpc.DocumentSymbol{
				{
					Name: "", Detail: "digraph", Kind: rpc.SymbolKindModule, Range: r(0, 0, 0, 18), SelectionRange: r(0, 0, 0, 7),
					Children: []rpc.DocumentSymbol{
						{Name: "a -> b", Kind: rpc.SymbolKindEvent, Range: r(0, 10, 0, 16), SelectionRange: r(0, 10, 0, 16)},
					},
				},
			},
		},
		"GraphWithUndirectedEdge": {
			src: `graph { a -- b }`,
			want: []rpc.DocumentSymbol{
				{
					Name: "", Detail: "graph", Kind: rpc.SymbolKindModule, Range: r(0, 0, 0, 16), SelectionRange: r(0, 0, 0, 5),
					Children: []rpc.DocumentSymbol{
						{Name: "a -- b", Kind: rpc.SymbolKindEvent, Range: r(0, 8, 0, 14), SelectionRange: r(0, 8, 0, 14)},
					},
				},
			},
		},
		"EdgeChainWithAttributes": {
			src: `digraph { a -> b -> c [label="path"] }`,
			want: []rpc.DocumentSymbol{
				{
					Name: "", Detail: "digraph", Kind: rpc.SymbolKindModule, Range: r(0, 0, 0, 38), SelectionRange: r(0, 0, 0, 7),
					Children: []rpc.DocumentSymbol{
						{Name: "a -> b -> c", Kind: rpc.SymbolKindEvent, Range: r(0, 10, 0, 36), SelectionRange: r(0, 10, 0, 36)},
					},
				},
			},
		},
		"GraphWithSubgraph": {
			src: `digraph { subgraph cluster_a { } }`,
			want: []rpc.DocumentSymbol{
				{
					Name: "", Detail: "digraph", Kind: rpc.SymbolKindModule, Range: r(0, 0, 0, 34), SelectionRange: r(0, 0, 0, 7),
					Children: []rpc.DocumentSymbol{
						{Name: "cluster_a", Detail: "subgraph", Kind: rpc.SymbolKindNamespace, Range: r(0, 10, 0, 32), SelectionRange: r(0, 19, 0, 28)},
					},
				},
			},
		},
		"GraphWithAnonymousSubgraph": {
			src: `digraph { subgraph { a } }`,
			want: []rpc.DocumentSymbol{
				{
					Name: "", Detail: "digraph", Kind: rpc.SymbolKindModule, Range: r(0, 0, 0, 26), SelectionRange: r(0, 0, 0, 7),
					Children: []rpc.DocumentSymbol{
						{
							Name: "", Detail: "subgraph", Kind: rpc.SymbolKindNamespace, Range: r(0, 10, 0, 24), SelectionRange: r(0, 10, 0, 18),
							Children: []rpc.DocumentSymbol{
								{Name: "a", Kind: rpc.SymbolKindVariable, Range: r(0, 21, 0, 22), SelectionRange: r(0, 21, 0, 22)},
							},
						},
					},
				},
			},
		},
		"NestedSubgraphs": {
			src: `digraph { subgraph outer { subgraph inner { a } } }`,
			want: []rpc.DocumentSymbol{
				{
					Name: "", Detail: "digraph", Kind: rpc.SymbolKindModule, Range: r(0, 0, 0, 51), SelectionRange: r(0, 0, 0, 7),
					Children: []rpc.DocumentSymbol{
						{
							Name: "outer", Detail: "subgraph", Kind: rpc.SymbolKindNamespace, Range: r(0, 10, 0, 49), SelectionRange: r(0, 19, 0, 24),
							Children: []rpc.DocumentSymbol{
								{
									Name: "inner", Detail: "subgraph", Kind: rpc.SymbolKindNamespace, Range: r(0, 27, 0, 47), SelectionRange: r(0, 36, 0, 41),
									Children: []rpc.DocumentSymbol{
										{Name: "a", Kind: rpc.SymbolKindVariable, Range: r(0, 44, 0, 45), SelectionRange: r(0, 44, 0, 45)},
									},
								},
							},
						},
					},
				},
			},
		},
		"SkipAttrStatements": {
			src: `digraph { node [shape=box]; edge [color=red]; a -> b }`,
			want: []rpc.DocumentSymbol{
				{
					Name: "", Detail: "digraph", Kind: rpc.SymbolKindModule, Range: r(0, 0, 0, 54), SelectionRange: r(0, 0, 0, 7),
					Children: []rpc.DocumentSymbol{
						{Name: "a -> b", Kind: rpc.SymbolKindEvent, Range: r(0, 46, 0, 52), SelectionRange: r(0, 46, 0, 52)},
					},
				},
			},
		},
		"QuotedIdentifiers": {
			src: `digraph { "node with spaces" -> "another node" }`,
			want: []rpc.DocumentSymbol{
				{
					Name: "", Detail: "digraph", Kind: rpc.SymbolKindModule, Range: r(0, 0, 0, 48), SelectionRange: r(0, 0, 0, 7),
					Children: []rpc.DocumentSymbol{
						{Name: `"node with spaces" -> "another node"`, Kind: rpc.SymbolKindEvent, Range: r(0, 10, 0, 46), SelectionRange: r(0, 10, 0, 46)},
					},
				},
			},
		},
		// Error recovery: parser creates ErrorTree nodes for invalid syntax,
		// but valid parts of the tree should still produce symbols
		"ErrorRecoveryMissingClosingBrace": {
			src: `digraph G { a`,
			want: []rpc.DocumentSymbol{
				{
					Name: "G", Detail: "digraph", Kind: rpc.SymbolKindModule, Range: r(0, 0, 0, 13), SelectionRange: r(0, 8, 0, 9),
					Children: []rpc.DocumentSymbol{
						{Name: "a", Kind: rpc.SymbolKindVariable, Range: r(0, 12, 0, 13), SelectionRange: r(0, 12, 0, 13)},
					},
				},
			},
		},
		"ErrorRecoveryInvalidEdgeExtractsValidNodes": {
			src: `digraph { a; b; -> }`,
			want: []rpc.DocumentSymbol{
				{
					Name: "", Detail: "digraph", Kind: rpc.SymbolKindModule, Range: r(0, 0, 0, 20), SelectionRange: r(0, 0, 0, 7),
					Children: []rpc.DocumentSymbol{
						{Name: "a", Kind: rpc.SymbolKindVariable, Range: r(0, 10, 0, 11), SelectionRange: r(0, 10, 0, 11)},
						{Name: "b", Kind: rpc.SymbolKindVariable, Range: r(0, 13, 0, 14), SelectionRange: r(0, 13, 0, 14)},
					},
				},
			},
		},
		"ErrorRecoveryIncompleteEdgeExtractsSecondGraph": {
			src: `digraph first { a -> } digraph second { b }`,
			want: []rpc.DocumentSymbol{
				{
					Name: "first", Detail: "digraph", Kind: rpc.SymbolKindModule, Range: r(0, 0, 0, 22), SelectionRange: r(0, 8, 0, 13),
					Children: []rpc.DocumentSymbol{
						{Name: "a -> ", Kind: rpc.SymbolKindEvent, Range: r(0, 16, 0, 20), SelectionRange: r(0, 16, 0, 20)},
					},
				},
				{
					Name: "second", Detail: "digraph", Kind: rpc.SymbolKindModule, Range: r(0, 23, 0, 43), SelectionRange: r(0, 31, 0, 37),
					Children: []rpc.DocumentSymbol{
						{Name: "b", Kind: rpc.SymbolKindVariable, Range: r(0, 40, 0, 41), SelectionRange: r(0, 40, 0, 41)},
					},
				},
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ps := dot.NewParser([]byte(tt.src))
			tree := ps.Parse()

			got := DocumentSymbols(tree)

			assert.EqualValuesf(t, got, tt.want, "unexpected symbols for %q", tt.src)
		})
	}
}

func TestDocumentSymbolsLimits(t *testing.T) {
	t.Run("MaxItems", func(t *testing.T) {
		// Generate a graph with more than maxItems nodes
		src := "digraph { "
		for i := 0; i < maxItems+100; i++ {
			src += "n" + string(rune('a'+i%26)) + string(rune('0'+i/26%10)) + "; "
		}
		src += "}"

		ps := dot.NewParser([]byte(src))
		tree := ps.Parse()

		got := DocumentSymbols(tree)

		// Count total symbols (graph + children)
		total := countSymbols(got)
		assert.Truef(t, total <= maxItems, "expected at most %d symbols, got %d", maxItems, total)
	})

	t.Run("MaxItemsAcrossGraphs", func(t *testing.T) {
		// Generate multiple graphs, each with nodes that together exceed maxItems
		// This tests that the item counter is shared across siblings, not reset per graph
		nodesPerGraph := maxItems / 3
		src := ""
		for g := 0; g < 5; g++ {
			src += "digraph g" + string(rune('0'+g)) + " { "
			for i := 0; i < nodesPerGraph; i++ {
				src += "n" + string(rune('a'+i%26)) + string(rune('0'+i/26%10)) + "; "
			}
			src += "} "
		}

		ps := dot.NewParser([]byte(src))
		tree := ps.Parse()

		got := DocumentSymbols(tree)

		total := countSymbols(got)
		assert.Truef(t, total <= maxItems, "expected at most %d symbols, got %d", maxItems, total)
	})

	t.Run("MaxDepth", func(t *testing.T) {
		// Generate deeply nested subgraphs
		src := "digraph { "
		for i := 0; i < maxDepth+2; i++ {
			src += "subgraph s" + string(rune('0'+i)) + " { "
		}
		src += "a "
		for i := 0; i < maxDepth+2; i++ {
			src += "} "
		}
		src += "}"

		ps := dot.NewParser([]byte(src))
		tree := ps.Parse()

		got := DocumentSymbols(tree)

		depth := maxSymbolDepth(got)
		assert.Truef(t, depth <= maxDepth, "expected max depth %d, got %d", maxDepth, depth)
	})
}

func countSymbols(symbols []rpc.DocumentSymbol) int {
	count := len(symbols)
	for _, s := range symbols {
		count += countSymbols(s.Children)
	}
	return count
}

func maxSymbolDepth(symbols []rpc.DocumentSymbol) int {
	if len(symbols) == 0 {
		return 0
	}
	maxChild := 0
	for _, s := range symbols {
		childDepth := maxSymbolDepth(s.Children)
		if childDepth > maxChild {
			maxChild = childDepth
		}
	}
	return 1 + maxChild
}

// r creates an rpc.Range from 0-based line/character positions.
// Arguments: startLine, startChar, endLine, endChar
func r(sl, sc, el, ec int) rpc.Range {
	return rpc.Range{
		Start: rpc.Position{Line: uint32(sl), Character: uint32(sc)},
		End:   rpc.Position{Line: uint32(el), Character: uint32(ec)},
	}
}
