package completion

import (
	"testing"

	"github.com/teleivo/assertive/assert"
	"github.com/teleivo/dot"
	"github.com/teleivo/dot/token"
)

func TestContext(t *testing.T) {
	tests := map[string]struct {
		src          string
		position     token.Position // 1-based line and column
		wantPrefix   string
		wantAttrCtx  AttributeContext
		wantAttrName string // empty means completing name, non-empty means completing value
	}{
		"NameInNodeAttrList": {
			src:         `graph { A [lab] }`,
			position:    token.Position{Line: 1, Column: 15},
			wantPrefix:  "lab",
			wantAttrCtx: Node,
		},
		"NameInNodeAttrListMultiLine": {
			src:         "graph {\n  A [lab]\n}",
			position:    token.Position{Line: 2, Column: 9},
			wantPrefix:  "lab",
			wantAttrCtx: Node,
		},
		"NameInNodeAttrListMultiLineColumnLessThanGraphStart": {
			src:         "  graph {\nA\n}",
			position:    token.Position{Line: 2, Column: 2},
			wantPrefix:  "A",
			wantAttrCtx: Node,
		},
		"NameInEdgeAttrList": {
			src:         `digraph { a -> b [arr] }`,
			position:    token.Position{Line: 1, Column: 22},
			wantPrefix:  "arr",
			wantAttrCtx: Edge,
		},
		"NameEmptyAfterOpenBracket": {
			src:         `graph { a [ }`,
			position:    token.Position{Line: 1, Column: 12},
			wantPrefix:  "",
			wantAttrCtx: Node,
		},
		"NameEmptyAfterComma": {
			src:         `graph { a [label=red,] }`,
			position:    token.Position{Line: 1, Column: 22},
			wantPrefix:  "",
			wantAttrCtx: Node,
		},
		"NameInAttrStmtNode": {
			src:         `graph { node [lab] }`,
			position:    token.Position{Line: 1, Column: 18},
			wantPrefix:  "lab",
			wantAttrCtx: Node,
		},
		"NameInAttrStmtEdge": {
			src:         `graph { edge [lab] }`,
			position:    token.Position{Line: 1, Column: 18},
			wantPrefix:  "lab",
			wantAttrCtx: Edge,
		},
		"NameInAttrStmtGraph": {
			src:         `graph { graph [lab] }`,
			position:    token.Position{Line: 1, Column: 19},
			wantPrefix:  "lab",
			wantAttrCtx: Graph,
		},
		"NameInNodeInsideSubgraph": {
			src:         `graph { subgraph { a [lab] } }`,
			position:    token.Position{Line: 1, Column: 26},
			wantPrefix:  "lab",
			wantAttrCtx: Node,
		},
		"NameInEmptySource": {
			src:         ``,
			position:    token.Position{Line: 1, Column: 1},
			wantPrefix:  "",
			wantAttrCtx: Graph,
		},
		"NameInNodeInsideAnonymousSubgraph": {
			src:         `graph { subgraph { a [pen] } }`,
			position:    token.Position{Line: 1, Column: 26},
			wantPrefix:  "pen",
			wantAttrCtx: Node,
		},
		"NameInNodeInsideNamedSubgraph": {
			src:         `graph { subgraph foo { a [pen] } }`,
			position:    token.Position{Line: 1, Column: 30},
			wantPrefix:  "pen",
			wantAttrCtx: Node,
		},
		"NameInNodeInsideClusterSubgraph": {
			src:         `graph { subgraph cluster_foo { a [pen] } }`,
			position:    token.Position{Line: 1, Column: 38},
			wantPrefix:  "pen",
			wantAttrCtx: Node,
		},
		"NameInAttrStmtGraphInsideClusterSubgraph": {
			src:         `graph { subgraph cluster_foo { graph [pen] } }`,
			position:    token.Position{Line: 1, Column: 42},
			wantPrefix:  "pen",
			wantAttrCtx: Cluster,
		},
		"NameWithoutEquals": {
			src:         `graph { a [sha] }`,
			position:    token.Position{Line: 1, Column: 15},
			wantPrefix:  "sha",
			wantAttrCtx: Node,
		},
		"ValueEmptyAfterEquals": {
			src:          `graph { a [shape=] }`,
			position:     token.Position{Line: 1, Column: 18},
			wantPrefix:   "",
			wantAttrCtx:  Node,
			wantAttrName: "shape",
		},
		"ValuePartialInNodeAttrList": {
			src:          `graph { a [shape=bo] }`,
			position:     token.Position{Line: 1, Column: 20},
			wantPrefix:   "bo",
			wantAttrCtx:  Node,
			wantAttrName: "shape",
		},
		"ValueEmptyInEdgeAttrList": {
			src:          `digraph { a -> b [dir=] }`,
			position:     token.Position{Line: 1, Column: 22},
			wantPrefix:   "",
			wantAttrCtx:  Edge,
			wantAttrName: "dir",
		},
		"ValuePartialInEdgeAttrList": {
			src:          `digraph { a -> b [dir=ba] }`,
			position:     token.Position{Line: 1, Column: 24},
			wantPrefix:   "ba",
			wantAttrCtx:  Edge,
			wantAttrName: "dir",
		},
		"ValueEmptySecondAttrAfterComma": {
			src:          `graph { a [label=foo, shape=] }`,
			position:     token.Position{Line: 1, Column: 28},
			wantPrefix:   "",
			wantAttrCtx:  Node,
			wantAttrName: "shape",
		},
		"ValuePartialGraphLevelAttr": {
			src:          `digraph { rankdir=L }`,
			position:     token.Position{Line: 1, Column: 19},
			wantPrefix:   "L",
			wantAttrCtx:  Graph,
			wantAttrName: "rankdir",
		},
		"ValueUnclosedQuoteReturnsEmptyPrefix": {
			src:         `graph { a [shape="bo] }`,
			position:    token.Position{Line: 1, Column: 21},
			wantPrefix:  "",
			wantAttrCtx: Node,
		},
		"ValuePartialMultiLine": {
			src:          "graph {\n  a [shape=\n    bo]\n}",
			position:     token.Position{Line: 3, Column: 6},
			wantPrefix:   "bo",
			wantAttrCtx:  Node,
			wantAttrName: "shape",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ps := dot.NewParser([]byte(tt.src))
			tree := ps.Parse()

			got := result{AttrCtx: Graph}
			context(tree, tt.position, &got)
			want := result{Prefix: tt.wantPrefix, AttrCtx: tt.wantAttrCtx, AttrName: tt.wantAttrName}

			assert.EqualValuesf(t, got, want, "for %q at %s", tt.src, tt.position)
		})
	}
}

func TestItems(t *testing.T) {
	tests := map[string]struct {
		src      string
		position token.Position
		want     []string
	}{
		"StyleValuesForNode": {
			src:      `graph { a [style=] }`,
			position: token.Position{Line: 1, Column: 18},
			want:     []string{"solid", "dashed", "dotted", "bold", "invis", "filled", "striped", "wedged", "diagonals", "rounded", "radial"},
		},
		"StyleValuesForEdge": {
			src:      `digraph { a -> b [style=] }`,
			position: token.Position{Line: 1, Column: 25},
			want:     []string{"solid", "dashed", "dotted", "bold", "invis", "filled", "tapered"},
		},
		"StyleValuesForCluster": {
			src:      `graph { subgraph cluster_a { graph [style=] } }`,
			position: token.Position{Line: 1, Column: 43},
			want:     []string{"filled", "striped", "rounded", "radial"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ps := dot.NewParser([]byte(tt.src))
			tree := ps.Parse()

			items := Items(tree, tt.position)
			got := make([]string, len(items))
			for i, item := range items {
				got[i] = item.Label
			}

			assert.EqualValuesf(t, got, tt.want, "unexpected style values")
		})
	}
}
