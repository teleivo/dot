package completion

import (
	"testing"

	"github.com/teleivo/assertive/assert"
	"github.com/teleivo/dot"
	"github.com/teleivo/dot/token"
)

func TestAttributeMarkdownDoc(t *testing.T) {
	attr := Attribute{
		Name:          "dir",
		Type:          TypeDirType,
		UsedBy:        Edge,
		Documentation: "Edge type for drawing arrowheads",
	}

	got := attr.markdownDoc()
	want := `Edge type for drawing arrowheads

**Type:** [dirType](https://graphviz.org/docs/attr-types/dirType/): ` + "`forward` | `back` | `both` | `none`" + `

[Docs](https://graphviz.org/docs/attrs/dir/)`

	assert.EqualValuesf(t, got, want, "unexpected markdown")
}

func TestContext(t *testing.T) {
	tests := map[string]struct {
		src          string
		position     token.Position // 1-based line and column
		wantPrefix   string
		wantAttrCtx  AttributeContext
		wantAttrName string // empty means completing name, non-empty means completing value
	}{
		// === Attribute name completion ===

		// Cursor inside node's attr_list after typing "lab"
		// Input: `graph { A [lab] }`
		//                       ^-- cursor at line 1, col 15 (after "lab")
		// Tree structure:
		//   NodeStmt > AttrList > AList > Attribute > ID > 'lab'
		"NodeAttrListPartialAttribute": {
			src:         `graph { A [lab] }`,
			position:    token.Position{Line: 1, Column: 15},
			wantPrefix:  "lab",
			wantAttrCtx: Node,
		},
		// Multi-line: cursor on line 2 should still be inside a node that starts on line 1
		// Input:
		//   graph {
		//     A [lab]
		//   }
		// Tree: 'lab' (@ 2 6 2 8), ']' (@ 2 9 2 9)
		// Cursor at line 2, col 9 (after "lab", on "]")
		"MultiLineNodeAttr": {
			src:         "graph {\n  A [lab]\n}",
			position:    token.Position{Line: 2, Column: 9},
			wantPrefix:  "lab",
			wantAttrCtx: Node,
		},
		// Bug case: naive column check fails when pos.Line > start.Line but pos.Column < start.Column
		// Input (note leading spaces on line 1):
		//   "  graph {\nA\n}"
		// Tree: Graph (@ 1 3 ...), 'A' (@ 2 1 2 1)
		// Cursor at line 2, col 2 (after "A")
		// Naive check: pos.Column (2) < Graph.Start.Column (3) → incorrectly returns "not inside"
		"MultiLineColumnBug": {
			src:         "  graph {\nA\n}",
			position:    token.Position{Line: 2, Column: 2},
			wantPrefix:  "A",
			wantAttrCtx: Node,
		},
		// Edge attributes: cursor after "arr" in edge attr list
		// Input: `digraph { a -> b [arr] }`
		// Tree: EdgeStmt > AttrList > AList > Attribute > ID > 'arr' (@ 1 19 1 21)
		// Cursor at line 1, col 22 (after "arr", on "]")
		"EdgeAttrList": {
			src:         `digraph { a -> b [arr] }`,
			position:    token.Position{Line: 1, Column: 22},
			wantPrefix:  "arr",
			wantAttrCtx: Edge,
		},
		// Empty prefix: cursor right after "[" with nothing typed yet
		// Input: `graph { a [ }` (malformed but parser recovers)
		// Tree: AttrList '[' (@ 1 11 1 11), no AList children
		// Cursor at line 1, col 12 (after "[")
		// Should return empty prefix, Node context
		"EmptyPrefixAfterBracket": {
			src:         `graph { a [ }`,
			position:    token.Position{Line: 1, Column: 12},
			wantPrefix:  "",
			wantAttrCtx: Node,
		},
		// Cursor after comma in attr list - ready for next attribute
		// Input: `graph { a [label=red,] }`
		// Tree: ',' (@ 1 21 1 21), ']' (@ 1 22 1 22)
		// Cursor at line 1, col 22 (after ",", on "]")
		"AfterCommaInAttrList": {
			src:         `graph { a [label=red,] }`,
			position:    token.Position{Line: 1, Column: 22},
			wantPrefix:  "",
			wantAttrCtx: Node,
		},
		// AttrStmt with "node" keyword - sets default node attributes
		// Input: `graph { node [lab] }`
		// Tree: AttrStmt > 'node' > AttrList > AList > Attribute > ID > 'lab' (@ 1 15 1 17)
		// Cursor at line 1, col 18 (after "lab")
		// Context should be Node (from AttrStmt with "node" keyword)
		"AttrStmtNode": {
			src:         `graph { node [lab] }`,
			position:    token.Position{Line: 1, Column: 18},
			wantPrefix:  "lab",
			wantAttrCtx: Node,
		},
		// AttrStmt with "edge" keyword - sets default edge attributes
		// Input: `graph { edge [lab] }`
		// Tree: AttrStmt > 'edge' > AttrList > AList > Attribute > ID > 'lab' (@ 1 15 1 17)
		// Cursor at line 1, col 18 (after "lab")
		// Context should be Edge (from AttrStmt with "edge" keyword)
		"AttrStmtEdge": {
			src:         `graph { edge [lab] }`,
			position:    token.Position{Line: 1, Column: 18},
			wantPrefix:  "lab",
			wantAttrCtx: Edge,
		},
		// AttrStmt with "graph" keyword - sets graph attributes
		// Input: `graph { graph [lab] }`
		// Tree: AttrStmt > 'graph' > AttrList > AList > Attribute > ID > 'lab' (@ 1 16 1 18)
		// Cursor at line 1, col 19 (after "lab")
		// Context should be Graph (from AttrStmt with "graph" keyword)
		"AttrStmtGraph": {
			src:         `graph { graph [lab] }`,
			position:    token.Position{Line: 1, Column: 19},
			wantPrefix:  "lab",
			wantAttrCtx: Graph,
		},
		// Subgraph: node inside subgraph still gets Node context
		// Input: `graph { subgraph { a [lab] } }`
		// Tree: Subgraph > StmtList > NodeStmt > AttrList > AList > Attribute > ID > 'lab' (@ 1 23 1 25)
		// Cursor at line 1, col 26 (after "lab")
		"NodeInSubgraph": {
			src:         `graph { subgraph { a [lab] } }`,
			position:    token.Position{Line: 1, Column: 26},
			wantPrefix:  "lab",
			wantAttrCtx: Node,
		},
		// Nil tree: should return empty prefix and Graph context
		"NilTree": {
			src:         ``,
			position:    token.Position{Line: 1, Column: 1},
			wantPrefix:  "",
			wantAttrCtx: Graph,
		},
		// Anonymous subgraph: node attributes inside anonymous subgraph get Node context
		// Input: `graph { subgraph { a [pen] } }`
		// Tree: Subgraph > StmtList > NodeStmt > AttrList > AList > Attribute > ID > 'pen' (@ 1 23 1 25)
		// Cursor at line 1, col 26 (after "pen")
		"AnonymousSubgraphNodeAttr": {
			src:         `graph { subgraph { a [pen] } }`,
			position:    token.Position{Line: 1, Column: 26},
			wantPrefix:  "pen",
			wantAttrCtx: Node,
		},
		// Named subgraph (non-cluster): node attributes inside named subgraph get Node context
		// Input: `graph { subgraph foo { a [pen] } }`
		// Tree: Subgraph > ID('foo') > StmtList > NodeStmt > AttrList > AList > Attribute > ID > 'pen' (@ 1 27 1 29)
		// Cursor at line 1, col 30 (after "pen")
		"NamedSubgraphNodeAttr": {
			src:         `graph { subgraph foo { a [pen] } }`,
			position:    token.Position{Line: 1, Column: 30},
			wantPrefix:  "pen",
			wantAttrCtx: Node,
		},
		// Cluster subgraph: node attributes inside cluster subgraph still get Node context
		// Input: `graph { subgraph cluster_foo { a [pen] } }`
		// Tree: Subgraph > ID('cluster_foo') > StmtList > NodeStmt > AttrList > AList > Attribute > ID > 'pen' (@ 1 35 1 37)
		// Cursor at line 1, col 38 (after "pen")
		// Context should be Node because we're on a NodeStmt, not the cluster itself
		"ClusterSubgraphNodeAttr": {
			src:         `graph { subgraph cluster_foo { a [pen] } }`,
			position:    token.Position{Line: 1, Column: 38},
			wantPrefix:  "pen",
			wantAttrCtx: Node,
		},
		// Cluster subgraph: graph attributes inside cluster get Cluster context
		// Input: `graph { subgraph cluster_foo { graph [pen] } }`
		// Tree: Subgraph > ID('cluster_foo') > StmtList > AttrStmt('graph') > AttrList > AList > Attribute > ID > 'pen' (@ 1 39 1 41)
		// Cursor at line 1, col 42 (after "pen")
		// Context should be Cluster because AttrStmt 'graph' inside a cluster_ subgraph
		"ClusterSubgraphGraphAttr": {
			src:         `graph { subgraph cluster_foo { graph [pen] } }`,
			position:    token.Position{Line: 1, Column: 42},
			wantPrefix:  "pen",
			wantAttrCtx: Cluster,
		},
		// Still completing name (no = yet)
		// Input: `graph { a [sha] }`
		// Cursor at line 1, col 15 (after "sha")
		"StillCompletingName": {
			src:         `graph { a [sha] }`,
			position:    token.Position{Line: 1, Column: 15},
			wantPrefix:  "sha",
			wantAttrCtx: Node,
		},

		// === Attribute value completion ===

		// Cursor right after "=" - ready to type value
		// Input: `graph { a [shape=] }`
		// Cursor at line 1, col 18 (after "=")
		"ValueAfterEquals": {
			src:          `graph { a [shape=] }`,
			position:     token.Position{Line: 1, Column: 18},
			wantPrefix:   "",
			wantAttrCtx:  Node,
			wantAttrName: "shape",
		},
		// Cursor after partial value
		// Input: `graph { a [shape=bo] }`
		// Cursor at line 1, col 20 (after "bo")
		"ValuePartial": {
			src:          `graph { a [shape=bo] }`,
			position:     token.Position{Line: 1, Column: 20},
			wantPrefix:   "bo",
			wantAttrCtx:  Node,
			wantAttrName: "shape",
		},
		// Edge with dir attribute
		// Input: `digraph { a -> b [dir=] }`
		// Cursor at line 1, col 22 (after "=")
		"ValueEdgeDir": {
			src:          `digraph { a -> b [dir=] }`,
			position:     token.Position{Line: 1, Column: 22},
			wantPrefix:   "",
			wantAttrCtx:  Edge,
			wantAttrName: "dir",
		},
		// Partial dir value
		// Input: `digraph { a -> b [dir=ba] }`
		// Cursor at line 1, col 24 (after "ba")
		"ValueEdgeDirPartial": {
			src:          `digraph { a -> b [dir=ba] }`,
			position:     token.Position{Line: 1, Column: 24},
			wantPrefix:   "ba",
			wantAttrCtx:  Edge,
			wantAttrName: "dir",
		},
		// Second attribute value after comma
		// Input: `graph { a [label=foo, shape=] }`
		// Cursor at line 1, col 28 (after second "=")
		"ValueSecondAttr": {
			src:          `graph { a [label=foo, shape=] }`,
			position:     token.Position{Line: 1, Column: 28},
			wantPrefix:   "",
			wantAttrCtx:  Node,
			wantAttrName: "shape",
		},
		// Graph-level rankdir
		// Input: `digraph { rankdir=L }`
		// Cursor at line 1, col 19 (after "L")
		"ValueGraphRankdir": {
			src:          `digraph { rankdir=L }`,
			position:     token.Position{Line: 1, Column: 19},
			wantPrefix:   "L",
			wantAttrCtx:  Graph,
			wantAttrName: "rankdir",
		},
		// Quoted value - unclosed quote creates error node outside Attribute,
		// can't determine we're in value position
		// Input: `graph { a [shape="bo] }`
		// Cursor at line 1, col 21 (inside ErrorTree, not Attribute)
		"ValueQuotedPartial": {
			src:         `graph { a [shape="bo] }`,
			position:    token.Position{Line: 1, Column: 21},
			wantPrefix:  "",
			wantAttrCtx: Node,
		},
		// Multi-line: value on next line
		// Input:
		//   graph {
		//     a [shape=
		//       box]
		//   }
		// Cursor at line 3, col 6 (after "bo")
		"ValueMultiLine": {
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
