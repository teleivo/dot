package tree

import (
	"testing"

	"github.com/teleivo/assertive/assert"
	"github.com/teleivo/dot"
	"github.com/teleivo/dot/token"
)

func TestComponentString(t *testing.T) {
	tests := []struct {
		comp Component
		want string
	}{
		{0, ""},
		{Graph, "Graph"},
		{Subgraph, "Subgraph"},
		{Cluster, "Cluster"},
		{Node, "Node"},
		{Edge, "Edge"},
		{Node | Edge, "Node, Edge"},
		{Graph | Node | Edge, "Graph, Node, Edge"},
		{Graph | Cluster | Node | Edge, "Graph, Cluster, Node, Edge"},
		{Graph | Subgraph | Cluster | Node | Edge, "Graph, Subgraph, Cluster, Node, Edge"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.comp.String()
			assert.EqualValuesf(t, got, tt.want, "unexpected string")
		})
	}
}

func TestFind(t *testing.T) {
	tests := map[string]struct {
		src         string
		position    token.Position
		want        dot.TreeKind
		wantComp    Component
		wantTree    dot.TreeKind // expected tree type found, 0 if nil
		wantLiteral string       // expected literal from ID token (optional)
	}{
		"AttrNameInNodeAttrListFirstChar": {
			// AttrName 'lab' is at 1:12-14, cursor at first char
			src:         `graph { A [lab] }`,
			position:    token.Position{Line: 1, Column: 12},
			want:        dot.KindAttrName | dot.KindAttrValue,
			wantComp:    Node,
			wantTree:    dot.KindAttrName,
			wantLiteral: "lab",
		},
		"AttrNameInNodeAttrListLastChar": {
			// AttrName 'lab' is at 1:12-14, cursor at last char
			src:         `graph { A [lab] }`,
			position:    token.Position{Line: 1, Column: 14},
			want:        dot.KindAttrName | dot.KindAttrValue,
			wantComp:    Node,
			wantTree:    dot.KindAttrName,
			wantLiteral: "lab",
		},
		"AttrNameInEdgeAttrList": {
			// AttrName 'arr' is at 1:19-21
			src:         `digraph { a -> b [arr] }`,
			position:    token.Position{Line: 1, Column: 19},
			want:        dot.KindAttrName | dot.KindAttrValue,
			wantComp:    Edge,
			wantTree:    dot.KindAttrName,
			wantLiteral: "arr",
		},
		"AttrNameInAttrStmtNode": {
			// AttrName 'lab' is at 1:15-17
			src:         `graph { node [lab] }`,
			position:    token.Position{Line: 1, Column: 15},
			want:        dot.KindAttrName | dot.KindAttrValue,
			wantComp:    Node,
			wantTree:    dot.KindAttrName,
			wantLiteral: "lab",
		},
		"AttrNameInAttrStmtEdge": {
			// AttrName 'lab' is at 1:15-17
			src:         `graph { edge [lab] }`,
			position:    token.Position{Line: 1, Column: 17},
			want:        dot.KindAttrName | dot.KindAttrValue,
			wantComp:    Edge,
			wantTree:    dot.KindAttrName,
			wantLiteral: "lab",
		},
		"AttrNameInAttrStmtGraph": {
			// AttrName 'lab' is at 1:16-18
			src:         `graph { graph [lab] }`,
			position:    token.Position{Line: 1, Column: 16},
			want:        dot.KindAttrName | dot.KindAttrValue,
			wantComp:    Graph,
			wantTree:    dot.KindAttrName,
			wantLiteral: "lab",
		},
		"AttrValueInNodeAttrListFirstChar": {
			// AttrValue 'box' is at 1:18-20, cursor at first char
			src:         `graph { a [shape=box] }`,
			position:    token.Position{Line: 1, Column: 18},
			want:        dot.KindAttrName | dot.KindAttrValue,
			wantComp:    Node,
			wantTree:    dot.KindAttrValue,
			wantLiteral: "box",
		},
		"AttrValueInNodeAttrListLastChar": {
			// AttrValue 'box' is at 1:18-20, cursor at last char
			src:         `graph { a [shape=box] }`,
			position:    token.Position{Line: 1, Column: 20},
			want:        dot.KindAttrName | dot.KindAttrValue,
			wantComp:    Node,
			wantTree:    dot.KindAttrValue,
			wantLiteral: "box",
		},
		"AttrValueInEdgeAttrList": {
			// AttrValue 'back' is at 1:23-26
			src:         `digraph { a -> b [dir=back] }`,
			position:    token.Position{Line: 1, Column: 23},
			want:        dot.KindAttrName | dot.KindAttrValue,
			wantComp:    Edge,
			wantTree:    dot.KindAttrValue,
			wantLiteral: "back",
		},
		"AttrNameInNodeInsideSubgraph": {
			// AttrName 'lab' is at 1:23-25
			src:         `graph { subgraph { a [lab] } }`,
			position:    token.Position{Line: 1, Column: 23},
			want:        dot.KindAttrName | dot.KindAttrValue,
			wantComp:    Node,
			wantTree:    dot.KindAttrName,
			wantLiteral: "lab",
		},
		"AttrNameInAttrStmtGraphInsideSubgraph": {
			// AttrName 'pen' is at 1:31-33
			src:         `graph { subgraph foo { graph [pen] } }`,
			position:    token.Position{Line: 1, Column: 31},
			want:        dot.KindAttrName | dot.KindAttrValue,
			wantComp:    Subgraph,
			wantTree:    dot.KindAttrName,
			wantLiteral: "pen",
		},
		"AttrNameInNodeInsideClusterSubgraph": {
			// AttrName 'pen' is at 1:35-37
			src:         `graph { subgraph cluster_foo { a [pen] } }`,
			position:    token.Position{Line: 1, Column: 35},
			want:        dot.KindAttrName | dot.KindAttrValue,
			wantComp:    Node,
			wantTree:    dot.KindAttrName,
			wantLiteral: "pen",
		},
		"AttrNameInAttrStmtGraphInsideCluster": {
			// AttrName 'pen' is at 1:39-41
			src:         `graph { subgraph cluster_foo { graph [pen] } }`,
			position:    token.Position{Line: 1, Column: 39},
			want:        dot.KindAttrName | dot.KindAttrValue,
			wantComp:    Cluster,
			wantTree:    dot.KindAttrName,
			wantLiteral: "pen",
		},
		"TwoAttrsFirstNameFirstChar": {
			// 'shape' is at 1:12-16, 'color' is at 1:22-26
			src:         `graph { a [shape=box,color=red] }`,
			position:    token.Position{Line: 1, Column: 12},
			want:        dot.KindAttrName | dot.KindAttrValue,
			wantComp:    Node,
			wantTree:    dot.KindAttrName,
			wantLiteral: "shape",
		},
		"TwoAttrsFirstNameLastChar": {
			src:         `graph { a [shape=box,color=red] }`,
			position:    token.Position{Line: 1, Column: 16},
			want:        dot.KindAttrName | dot.KindAttrValue,
			wantComp:    Node,
			wantTree:    dot.KindAttrName,
			wantLiteral: "shape",
		},
		"TwoAttrsFirstValue": {
			// 'box' is at 1:18-20
			src:         `graph { a [shape=box,color=red] }`,
			position:    token.Position{Line: 1, Column: 18},
			want:        dot.KindAttrName | dot.KindAttrValue,
			wantComp:    Node,
			wantTree:    dot.KindAttrValue,
			wantLiteral: "box",
		},
		"TwoAttrsSecondName": {
			// 'color' is at 1:22-26
			src:         `graph { a [shape=box,color=red] }`,
			position:    token.Position{Line: 1, Column: 22},
			want:        dot.KindAttrName | dot.KindAttrValue,
			wantComp:    Node,
			wantTree:    dot.KindAttrName,
			wantLiteral: "color",
		},
		"TwoAttrsSecondValue": {
			// 'red' is at 1:28-30
			src:         `graph { a [shape=box,color=red] }`,
			position:    token.Position{Line: 1, Column: 30},
			want:        dot.KindAttrName | dot.KindAttrValue,
			wantComp:    Node,
			wantTree:    dot.KindAttrValue,
			wantLiteral: "red",
		},
		"NodeStmtMatch": {
			// NodeStmt 'a' is at 1:9
			src:         `graph { a }`,
			position:    token.Position{Line: 1, Column: 9},
			want:        dot.KindNodeStmt,
			wantComp:    Node,
			wantTree:    dot.KindNodeStmt,
			wantLiteral: "a",
		},
		"NoMatchInEmptySource": {
			src:      ``,
			position: token.Position{Line: 1, Column: 1},
			want:     dot.KindAttrName | dot.KindAttrValue,
			wantComp: Graph,
			wantTree: 0,
		},
		"NoMatchOutsideAttrList": {
			// 'a' is at 1:9, looking for AttrName|AttrValue but it's a NodeStmt
			src:      `graph { a }`,
			position: token.Position{Line: 1, Column: 9},
			want:     dot.KindAttrName | dot.KindAttrValue,
			wantComp: Node,
			wantTree: 0,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ps := dot.NewParser([]byte(tt.src))
			tree := ps.Parse()

			got := Find(tree, tt.position, tt.want)

			assert.EqualValuesf(t, got.Comp, tt.wantComp, "unexpected component for %q at %s", tt.src, tt.position)
			if tt.wantTree == 0 {
				assert.Nilf(t, got.Tree, "expected nil tree for %q at %s", tt.src, tt.position)
			} else {
				assert.NotNilf(t, got.Tree, "expected non-nil tree for %q at %s", tt.src, tt.position)
				if got.Tree != nil {
					assert.EqualValuesf(t, got.Tree.Type, tt.wantTree, "unexpected tree type for %q at %s", tt.src, tt.position)
					if tt.wantLiteral != "" {
						gotLiteral := extractLiteral(got.Tree)
						assert.EqualValuesf(t, gotLiteral, tt.wantLiteral, "unexpected literal for %q at %s", tt.src, tt.position)
					}
				}
			}
		})
	}
}

// extractLiteral finds the first ID token literal in a tree by traversing down.
func extractLiteral(tree *dot.Tree) string {
	if tree == nil {
		return ""
	}
	for _, child := range tree.Children {
		switch c := child.(type) {
		case dot.TreeChild:
			if c.Type == dot.KindID && len(c.Children) > 0 {
				if tok, ok := c.Children[0].(dot.TokenChild); ok {
					return tok.Literal
				}
			}
			if lit := extractLiteral(c.Tree); lit != "" {
				return lit
			}
		}
	}
	return ""
}
