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
				{Name: "foo", Detail: "digraph", Kind: rpc.SymbolKindModule},
			},
		},
		"AnonymousGraph": {
			src: `digraph { }`,
			want: []rpc.DocumentSymbol{
				{Name: "", Detail: "digraph", Kind: rpc.SymbolKindModule},
			},
		},
		"UndirectedGraph": {
			src: `graph bar { }`,
			want: []rpc.DocumentSymbol{
				{Name: "bar", Detail: "graph", Kind: rpc.SymbolKindModule},
			},
		},
		"MultipleGraphs": {
			src: `digraph first { }
graph second { }`,
			want: []rpc.DocumentSymbol{
				{Name: "first", Detail: "digraph", Kind: rpc.SymbolKindModule},
				{Name: "second", Detail: "graph", Kind: rpc.SymbolKindModule},
			},
		},
		"GraphWithNode": {
			src: `digraph { a }`,
			want: []rpc.DocumentSymbol{
				{
					Name: "", Detail: "digraph", Kind: rpc.SymbolKindModule,
					Children: []rpc.DocumentSymbol{
						{Name: "a", Kind: rpc.SymbolKindVariable},
					},
				},
			},
		},
		"GraphWithMultipleNodes": {
			src: `digraph { a; b; c }`,
			want: []rpc.DocumentSymbol{
				{
					Name: "", Detail: "digraph", Kind: rpc.SymbolKindModule,
					Children: []rpc.DocumentSymbol{
						{Name: "a", Kind: rpc.SymbolKindVariable},
						{Name: "b", Kind: rpc.SymbolKindVariable},
						{Name: "c", Kind: rpc.SymbolKindVariable},
					},
				},
			},
		},
		"GraphWithEdge": {
			src: `digraph { a -> b }`,
			want: []rpc.DocumentSymbol{
				{
					Name: "", Detail: "digraph", Kind: rpc.SymbolKindModule,
					Children: []rpc.DocumentSymbol{
						{Name: "a -> b", Kind: rpc.SymbolKindEvent},
					},
				},
			},
		},
		"GraphWithUndirectedEdge": {
			src: `graph { a -- b }`,
			want: []rpc.DocumentSymbol{
				{
					Name: "", Detail: "graph", Kind: rpc.SymbolKindModule,
					Children: []rpc.DocumentSymbol{
						{Name: "a -- b", Kind: rpc.SymbolKindEvent},
					},
				},
			},
		},
		"GraphWithEdgeChain": {
			src: `digraph { a -> b -> c }`,
			want: []rpc.DocumentSymbol{
				{
					Name: "", Detail: "digraph", Kind: rpc.SymbolKindModule,
					Children: []rpc.DocumentSymbol{
						{Name: "a -> b -> c", Kind: rpc.SymbolKindEvent},
					},
				},
			},
		},
		"GraphWithSubgraph": {
			src: `digraph { subgraph cluster_a { } }`,
			want: []rpc.DocumentSymbol{
				{
					Name: "", Detail: "digraph", Kind: rpc.SymbolKindModule,
					Children: []rpc.DocumentSymbol{
						{Name: "cluster_a", Detail: "subgraph", Kind: rpc.SymbolKindNamespace},
					},
				},
			},
		},
		"GraphWithAnonymousSubgraph": {
			src: `digraph { subgraph { a } }`,
			want: []rpc.DocumentSymbol{
				{
					Name: "", Detail: "digraph", Kind: rpc.SymbolKindModule,
					Children: []rpc.DocumentSymbol{
						{
							Name: "", Detail: "subgraph", Kind: rpc.SymbolKindNamespace,
							Children: []rpc.DocumentSymbol{
								{Name: "a", Kind: rpc.SymbolKindVariable},
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
					Name: "", Detail: "digraph", Kind: rpc.SymbolKindModule,
					Children: []rpc.DocumentSymbol{
						{
							Name: "outer", Detail: "subgraph", Kind: rpc.SymbolKindNamespace,
							Children: []rpc.DocumentSymbol{
								{
									Name: "inner", Detail: "subgraph", Kind: rpc.SymbolKindNamespace,
									Children: []rpc.DocumentSymbol{
										{Name: "a", Kind: rpc.SymbolKindVariable},
									},
								},
							},
						},
					},
				},
			},
		},
		"SkipAttrStatements": {
			// node [...], edge [...], graph [...] should not appear as symbols
			src: `digraph { node [shape=box]; edge [color=red]; a -> b }`,
			want: []rpc.DocumentSymbol{
				{
					Name: "", Detail: "digraph", Kind: rpc.SymbolKindModule,
					Children: []rpc.DocumentSymbol{
						{Name: "a -> b", Kind: rpc.SymbolKindEvent},
					},
				},
			},
		},
		"QuotedIdentifiers": {
			src: `digraph { "node with spaces" -> "another node" }`,
			want: []rpc.DocumentSymbol{
				{
					Name: "", Detail: "digraph", Kind: rpc.SymbolKindModule,
					Children: []rpc.DocumentSymbol{
						{Name: `"node with spaces" -> "another node"`, Kind: rpc.SymbolKindEvent},
					},
				},
			},
		},
		"ComplexGraph": {
			src: `digraph G {
				subgraph cluster_0 {
					a; b
					a -> b
				}
				subgraph cluster_1 {
					c -> d
				}
				a -> c
			}`,
			want: []rpc.DocumentSymbol{
				{
					Name: "G", Detail: "digraph", Kind: rpc.SymbolKindModule,
					Children: []rpc.DocumentSymbol{
						{
							Name: "cluster_0", Detail: "subgraph", Kind: rpc.SymbolKindNamespace,
							Children: []rpc.DocumentSymbol{
								{Name: "a", Kind: rpc.SymbolKindVariable},
								{Name: "b", Kind: rpc.SymbolKindVariable},
								{Name: "a -> b", Kind: rpc.SymbolKindEvent},
							},
						},
						{
							Name: "cluster_1", Detail: "subgraph", Kind: rpc.SymbolKindNamespace,
							Children: []rpc.DocumentSymbol{
								{Name: "c -> d", Kind: rpc.SymbolKindEvent},
							},
						},
						{Name: "a -> c", Kind: rpc.SymbolKindEvent},
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

			assert.EqualValuesf(t, got, tt.want, "unexpected symbols")
		})
	}
}

func TestDocumentSymbolsLimits(t *testing.T) {
	t.Run("MaxItems", func(t *testing.T) {
		// Generate a graph with more than MaxItems nodes
		src := "digraph { "
		for i := 0; i < MaxItems+100; i++ {
			src += "n" + string(rune('a'+i%26)) + string(rune('0'+i/26%10)) + "; "
		}
		src += "}"

		ps := dot.NewParser([]byte(src))
		tree := ps.Parse()

		got := DocumentSymbols(tree)

		// Count total symbols (graph + children)
		total := countSymbols(got)
		assert.Truef(t, total <= MaxItems, "expected at most %d symbols, got %d", MaxItems, total)
	})

	t.Run("MaxDepth", func(t *testing.T) {
		// Generate deeply nested subgraphs
		src := "digraph { "
		for i := 0; i < MaxDepth+2; i++ {
			src += "subgraph s" + string(rune('0'+i)) + " { "
		}
		src += "a "
		for i := 0; i < MaxDepth+2; i++ {
			src += "} "
		}
		src += "}"

		ps := dot.NewParser([]byte(src))
		tree := ps.Parse()

		got := DocumentSymbols(tree)

		depth := maxSymbolDepth(got)
		assert.Truef(t, depth <= MaxDepth, "expected max depth %d, got %d", MaxDepth, depth)
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
