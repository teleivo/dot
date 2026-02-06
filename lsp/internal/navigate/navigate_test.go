package navigate

import (
	"testing"

	"github.com/teleivo/assertive/assert"
	"github.com/teleivo/dot"
	"github.com/teleivo/dot/lsp/internal/rpc"
	"github.com/teleivo/dot/token"
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

			assert.EqualValues(t, got, tt.want, "unexpected symbols for %q", tt.src)
		})
	}

	t.Run("LimitMaxItems", func(t *testing.T) {
		// Generate a graph with more than maxItems nodes
		src := "digraph { "
		for i := 0; i < maxItems+1; i++ {
			src += "n" + string(rune('a'+i%26)) + string(rune('0'+i/26%10)) + "; "
		}
		src += "}"

		ps := dot.NewParser([]byte(src))
		tree := ps.Parse()

		got := DocumentSymbols(tree)

		// Count total symbols (graph + children)
		total := countSymbols(got)
		assert.True(t, total <= maxItems, "expected at most %d symbols, got %d", maxItems, total)
	})

	t.Run("LimitMaxItemsAcrossGraphs", func(t *testing.T) {
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
		assert.True(t, total <= maxItems, "expected at most %d symbols, got %d", maxItems, total)
	})

	t.Run("LimitMaxDepth", func(t *testing.T) {
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
		assert.True(t, depth <= maxDepth, "expected max depth %d, got %d", maxDepth, depth)
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

func TestDefinition(t *testing.T) {
	uri := rpc.DocumentURI("file:///test.dot")

	tests := map[string]struct {
		src  string
		pos  token.Position // cursor position (1-based line, column)
		want *rpc.Location
	}{
		"NodeStmtToEdgeStmt": {
			// Cursor on 'a' in node stmt, definition is 'a' in edge (first occurrence)
			src:  `digraph { a -> b; a }`,
			pos:  token.Position{Line: 1, Column: 19}, // on 'a' in node stmt
			want: &rpc.Location{URI: uri, Range: r(0, 10, 0, 11)},
		},
		"EdgeStmtToNodeStmt": {
			// Cursor on 'a' in edge stmt, definition is 'a' in node stmt (first occurrence)
			src:  `digraph { a; a -> b }`,
			pos:  token.Position{Line: 1, Column: 14}, // on 'a' in edge stmt
			want: &rpc.Location{URI: uri, Range: r(0, 10, 0, 11)},
		},
		"EdgeToEdge": {
			// Cursor on 'b' in second edge, definition is 'b' in first edge
			src:  `digraph { a -> b; b -> c }`,
			pos:  token.Position{Line: 1, Column: 19}, // on 'b' in second edge
			want: &rpc.Location{URI: uri, Range: r(0, 15, 0, 16)},
		},
		"CursorOnFirstOccurrence": {
			// Cursor is already on first occurrence, still return the location
			src:  `digraph { a; a -> b }`,
			pos:  token.Position{Line: 1, Column: 11}, // on 'a' in node stmt (first occurrence)
			want: &rpc.Location{URI: uri, Range: r(0, 10, 0, 11)},
		},
		"SingleOccurrence": {
			// Node only appears once, still return the location
			src:  `digraph { a }`,
			pos:  token.Position{Line: 1, Column: 11}, // on 'a'
			want: &rpc.Location{URI: uri, Range: r(0, 10, 0, 11)},
		},
		"CursorNotOnNodeID": {
			// Cursor on keyword, not a node ID
			src:  `digraph { a }`,
			pos:  token.Position{Line: 1, Column: 3}, // on 'digraph'
			want: nil,
		},
		"CursorOnEdgeOperator": {
			// Cursor on '->', not a node ID
			src:  `digraph { a -> b }`,
			pos:  token.Position{Line: 1, Column: 13}, // on '->'
			want: nil,
		},
		"QuotedIdentifier": {
			// Quoted IDs should match
			src:  `digraph { "foo" -> b; "foo" }`,
			pos:  token.Position{Line: 1, Column: 23}, // on second "foo"
			want: &rpc.Location{URI: uri, Range: r(0, 10, 0, 15)},
		},
		"MultilineDefinition": {
			src: `digraph {
	a -> b
	a
}`,
			pos:  token.Position{Line: 3, Column: 2}, // on 'a' in node stmt (line 3)
			want: &rpc.Location{URI: uri, Range: r(1, 1, 1, 2)},
		},
		"InSubgraph": {
			// Definition spans across subgraph boundaries
			src:  `digraph { subgraph { a }; a -> b }`,
			pos:  token.Position{Line: 1, Column: 27}, // on 'a' in edge stmt
			want: &rpc.Location{URI: uri, Range: r(0, 21, 0, 22)},
		},
		"NodeAfterSubgraph": {
			// Node 'a' appears after a subgraph containing different nodes
			// Tests that search continues past subgraph when node not found inside
			src:  `digraph { subgraph { b }; a }`,
			pos:  token.Position{Line: 1, Column: 27}, // on 'a' after subgraph
			want: &rpc.Location{URI: uri, Range: r(0, 26, 0, 27)},
		},
		"SameNodeInGraphAndSubgraph": {
			// Node 'a' appears in both graph and subgraph - no special scoping
			// First occurrence wins regardless of nesting level
			src:  `digraph { a; subgraph { a } }`,
			pos:  token.Position{Line: 1, Column: 25}, // on 'a' inside subgraph
			want: &rpc.Location{URI: uri, Range: r(0, 10, 0, 11)},
		},
		"SameNodeSubgraphFirst": {
			// Node 'a' appears in subgraph first, then in graph
			// First occurrence (in subgraph) wins
			src:  `digraph { subgraph { a }; a }`,
			pos:  token.Position{Line: 1, Column: 27}, // on 'a' after subgraph
			want: &rpc.Location{URI: uri, Range: r(0, 21, 0, 22)},
		},
		"NodeWithPort": {
			// Node ID with port - definition should find the node ID part
			src:  `digraph { a:p1 -> b; a }`,
			pos:  token.Position{Line: 1, Column: 22}, // on 'a' in node stmt
			want: &rpc.Location{URI: uri, Range: r(0, 10, 0, 11)},
		},
		"EdgeToInlineSubgraph": {
			// Edge target is inline subgraph containing 'b'
			// Cursor on 'b' inside subgraph should find itself (first occurrence)
			src:  `digraph { a -> { b }; b }`,
			pos:  token.Position{Line: 1, Column: 23}, // on 'b' after subgraph
			want: &rpc.Location{URI: uri, Range: r(0, 17, 0, 18)},
		},
		"EmptySource": {
			src:  ``,
			pos:  token.Position{Line: 1, Column: 1},
			want: nil,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ps := dot.NewParser([]byte(tt.src))
			tree := ps.Parse()

			got := Definition(tree, uri, tt.pos)

			assert.EqualValues(t, got, tt.want, "unexpected definition for %q at %v", tt.src, tt.pos)
		})
	}
}

func TestReferences(t *testing.T) {
	uri := rpc.DocumentURI("file:///test.dot")

	tests := map[string]struct {
		src  string
		pos  token.Position // cursor position (1-based line, column)
		want []rpc.Location
	}{
		"SingleOccurrence": {
			// Node only appears once
			src:  `digraph { a }`,
			pos:  token.Position{Line: 1, Column: 11}, // on 'a'
			want: []rpc.Location{{URI: uri, Range: r(0, 10, 0, 11)}},
		},
		"TwoOccurrencesNodeAndEdge": {
			// Node 'a' appears in node stmt and edge stmt
			src:  `digraph { a; a -> b }`,
			pos:  token.Position{Line: 1, Column: 11}, // on first 'a'
			want: []rpc.Location{
				{URI: uri, Range: r(0, 10, 0, 11)},
				{URI: uri, Range: r(0, 13, 0, 14)},
			},
		},
		"TwoOccurrencesEdgeThenNode": {
			// Node 'a' appears in edge stmt first, then node stmt
			src:  `digraph { a -> b; a }`,
			pos:  token.Position{Line: 1, Column: 19}, // on second 'a' in node stmt
			want: []rpc.Location{
				{URI: uri, Range: r(0, 10, 0, 11)},
				{URI: uri, Range: r(0, 18, 0, 19)},
			},
		},
		"MultipleEdges": {
			// Node 'b' appears in multiple edges
			src:  `digraph { a -> b; b -> c; d -> b }`,
			pos:  token.Position{Line: 1, Column: 16}, // on first 'b'
			want: []rpc.Location{
				{URI: uri, Range: r(0, 15, 0, 16)},
				{URI: uri, Range: r(0, 18, 0, 19)},
				{URI: uri, Range: r(0, 31, 0, 32)},
			},
		},
		"InSubgraph": {
			// Node 'a' appears both inside and outside subgraph
			src:  `digraph { subgraph { a }; a -> b }`,
			pos:  token.Position{Line: 1, Column: 22}, // on 'a' inside subgraph
			want: []rpc.Location{
				{URI: uri, Range: r(0, 21, 0, 22)},
				{URI: uri, Range: r(0, 26, 0, 27)},
			},
		},
		"QuotedIdentifier": {
			// Quoted IDs should match exactly
			src:  `digraph { "foo" -> b; "foo" }`,
			pos:  token.Position{Line: 1, Column: 11}, // on first "foo"
			want: []rpc.Location{
				{URI: uri, Range: r(0, 10, 0, 15)},
				{URI: uri, Range: r(0, 22, 0, 27)},
			},
		},
		"MultilineReferences": {
			src: `digraph {
	a -> b
	c -> a
	a
}`,
			pos: token.Position{Line: 2, Column: 2}, // on 'a' in first edge (line 2)
			want: []rpc.Location{
				{URI: uri, Range: r(1, 1, 1, 2)},
				{URI: uri, Range: r(2, 6, 2, 7)},
				{URI: uri, Range: r(3, 1, 3, 2)},
			},
		},
		"NodeWithPort": {
			// Node ID with port - should find both occurrences of 'a'
			src:  `digraph { a:p1 -> b; a }`,
			pos:  token.Position{Line: 1, Column: 22}, // on 'a' in node stmt
			want: []rpc.Location{
				{URI: uri, Range: r(0, 10, 0, 11)},
				{URI: uri, Range: r(0, 21, 0, 22)},
			},
		},
		"CursorNotOnNodeID": {
			// Cursor on keyword, not a node ID
			src:  `digraph { a }`,
			pos:  token.Position{Line: 1, Column: 3}, // on 'digraph'
			want: nil,
		},
		"CursorOnEdgeOperator": {
			// Cursor on '->', not a node ID
			src:  `digraph { a -> b }`,
			pos:  token.Position{Line: 1, Column: 13}, // on '->'
			want: nil,
		},
		"EmptySource": {
			src:  ``,
			pos:  token.Position{Line: 1, Column: 1},
			want: nil,
		},
		"EdgeChain": {
			// Node appears multiple times in edge chain
			src:  `digraph { a -> b -> a }`,
			pos:  token.Position{Line: 1, Column: 11}, // on first 'a'
			want: []rpc.Location{
				{URI: uri, Range: r(0, 10, 0, 11)},
				{URI: uri, Range: r(0, 20, 0, 21)},
			},
		},
		"NestedSubgraphs": {
			// Node appears at different nesting levels
			src:  `digraph { a; subgraph { subgraph { a } }; a }`,
			pos:  token.Position{Line: 1, Column: 11}, // on first 'a'
			want: []rpc.Location{
				{URI: uri, Range: r(0, 10, 0, 11)},
				{URI: uri, Range: r(0, 35, 0, 36)},
				{URI: uri, Range: r(0, 42, 0, 43)},
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ps := dot.NewParser([]byte(tt.src))
			tree := ps.Parse()

			got := References(tree, uri, tt.pos)

			assert.EqualValues(t, got, tt.want, "unexpected references for %q at %v", tt.src, tt.pos)
		})
	}
}

// r creates an rpc.Range from 0-based line/character positions.
// Arguments: startLine, startChar, endLine, endChar
func r(sl, sc, el, ec int) rpc.Range {
	return rpc.Range{
		Start: rpc.Position{Line: uint32(sl), Character: uint32(sc)},
		End:   rpc.Position{Line: uint32(el), Character: uint32(ec)},
	}
}
