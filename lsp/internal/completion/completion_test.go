package completion

import (
	"testing"

	"github.com/teleivo/assertive/assert"
	"github.com/teleivo/dot"
	"github.com/teleivo/dot/token"
)

func TestItems(t *testing.T) {
	tests := map[string]struct {
		src            string
		position       token.Position
		want           []string
		wantInsertText map[string]string
	}{
		// Name completion: node context with prefix filtering
		"NameInNodeAttrList": {
			src:      `graph { A [lab] }`,
			position: token.Position{Line: 1, Column: 15},
			want:     []string{"label", "labelloc"},
		},
		"NameInNodeAttrListMultiLine": {
			src:      "graph {\n  A [lab]\n}",
			position: token.Position{Line: 2, Column: 9},
			want:     []string{"label", "labelloc"},
		},
		"NameInEdgeAttrList": {
			src:      `digraph { a -> b [arr] }`,
			position: token.Position{Line: 1, Column: 22},
			want:     []string{"arrowhead", "arrowsize", "arrowtail"},
		},
		"NameInAttrStmtEdge": {
			src:      `graph { edge [labe] }`,
			position: token.Position{Line: 1, Column: 19},
			want:     []string{"label", "labelURL", "labelangle", "labeldistance", "labelfloat", "labelfontcolor", "labelfontname", "labelfontsize", "labelhref", "labeltarget", "labeltooltip"},
		},
		"NameInAttrStmtGraph": {
			src:      `graph { graph [labe] }`,
			position: token.Position{Line: 1, Column: 20},
			want:     []string{"label", "label_scheme", "labeljust", "labelloc"},
		},
		"NameInNodeInsideSubgraph": {
			src:      `graph { subgraph { a [lab] } }`,
			position: token.Position{Line: 1, Column: 26},
			want:     []string{"label", "labelloc"},
		},
		// graph [...] inside non-cluster subgraph only gets Subgraph attrs (rank, cluster)
		"NameInAttrStmtGraphInsideSubgraph": {
			src:      `graph { subgraph { graph [ran] } }`,
			position: token.Position{Line: 1, Column: 29},
			want:     []string{"rank"},
		},
		"NameInAttrStmtGraphInsideSubgraphNoPenAttrs": {
			src:      `graph { subgraph { graph [pen] } }`,
			position: token.Position{Line: 1, Column: 29},
			want:     []string{}, // penwidth is N|E|C, not S
		},
		"NameInNodeInsideClusterSubgraph": {
			src:      `graph { subgraph cluster_foo { a [pen] } }`,
			position: token.Position{Line: 1, Column: 38},
			want:     []string{"penwidth"},
		},
		// graph [...] inside cluster gets cluster-specific attrs like pencolor
		"NameInAttrStmtGraphInsideClusterSubgraph": {
			src:      `graph { subgraph cluster_foo { graph [pen] } }`,
			position: token.Position{Line: 1, Column: 42},
			want:     []string{"pencolor", "penwidth"},
		},
		"NameWithPrefixFiltering": {
			src:      `graph { a [sha] }`,
			position: token.Position{Line: 1, Column: 15},
			want:     []string{"shape", "shapefile"},
		},
		// When no = exists, insert one
		"NameWithoutEquals": {
			src:            `graph { a [lab] }`,
			position:       token.Position{Line: 1, Column: 14},
			want:           []string{"label", "labelloc"},
			wantInsertText: map[string]string{"label": "label=", "labelloc": "labelloc="},
		},
		// When = already exists, don't insert another one
		"NameWithExistingEquals": {
			src:            `graph { a [lab=foo] }`,
			position:       token.Position{Line: 1, Column: 14},
			want:           []string{"label", "labelloc"},
			wantInsertText: map[string]string{"label": "label", "labelloc": "labelloc"},
		},

		// Value completion: shape values
		"ValueShapeEmpty": {
			src:      `graph { a [shape=] }`,
			position: token.Position{Line: 1, Column: 18},
			want: []string{
				"Mcircle", "Mdiamond", "Mrecord", "Msquare",
				"assembly", "box", "box3d", "cds", "circle", "component", "cylinder",
				"diamond", "doublecircle", "doubleoctagon",
				"egg", "ellipse",
				"fivepoverhang", "folder",
				"hexagon", "house",
				"insulator", "invhouse", "invtrapezium", "invtriangle",
				"larrow", "lpromoter",
				"none", "note", "noverhang",
				"octagon", "oval",
				"parallelogram", "pentagon", "plain", "plaintext", "point", "polygon",
				"primersite", "promoter", "proteasesite", "proteinstab",
				"rarrow", "record", "rect", "rectangle", "restrictionsite", "ribosite",
				"rnastab", "rpromoter",
				"septagon", "signature", "square", "star",
				"tab", "terminator", "threepoverhang", "trapezium", "triangle", "tripleoctagon",
				"underline", "utr",
			},
		},
		"ValueShapeAfterComma": {
			src:      `graph { a [label=foo, shape=] }`,
			position: token.Position{Line: 1, Column: 28},
			want: []string{
				"Mcircle", "Mdiamond", "Mrecord", "Msquare",
				"assembly", "box", "box3d", "cds", "circle", "component", "cylinder",
				"diamond", "doublecircle", "doubleoctagon",
				"egg", "ellipse",
				"fivepoverhang", "folder",
				"hexagon", "house",
				"insulator", "invhouse", "invtrapezium", "invtriangle",
				"larrow", "lpromoter",
				"none", "note", "noverhang",
				"octagon", "oval",
				"parallelogram", "pentagon", "plain", "plaintext", "point", "polygon",
				"primersite", "promoter", "proteasesite", "proteinstab",
				"rarrow", "record", "rect", "rectangle", "restrictionsite", "ribosite",
				"rnastab", "rpromoter",
				"septagon", "signature", "square", "star",
				"tab", "terminator", "threepoverhang", "trapezium", "triangle", "tripleoctagon",
				"underline", "utr",
			},
		},
		"ValueShapeMultiLine": {
			src:      "graph {\n  a [shape=\n    bo]\n}",
			position: token.Position{Line: 3, Column: 6},
			want:     []string{"box", "box3d"},
		},

		// Value completion: dir values (edge)
		"ValueDirEmpty": {
			src:      `digraph { a -> b [dir=] }`,
			position: token.Position{Line: 1, Column: 22},
			want:     []string{"back", "both", "forward", "none"},
		},
		"ValueDirPartial": {
			src:      `digraph { a -> b [dir=ba] }`,
			position: token.Position{Line: 1, Column: 24},
			want:     []string{"back"},
		},

		// Value completion: rankdir values (graph)
		"ValueRankdirPartial": {
			src:      `digraph { rankdir=L }`,
			position: token.Position{Line: 1, Column: 19},
			want:     []string{"LR"},
		},

		// Value completion: style values
		"StyleValuesForNode": {
			src:      `graph { a [style=] }`,
			position: token.Position{Line: 1, Column: 18},
			want:     []string{"solid", "dashed", "dotted", "bold", "invis", "filled", "striped", "wedged", "diagonals", "rounded", "radial"},
		},
		"StyleValuesForEdge": {
			src:      `digraph { a -> b [style=] }`,
			position: token.Position{Line: 1, Column: 25},
			want:     []string{"solid", "dashed", "dotted", "bold", "invis", "tapered"},
		},
		"StyleValuesForCluster": {
			src:      `graph { subgraph cluster_a { graph [style=] } }`,
			position: token.Position{Line: 1, Column: 43},
			want:     []string{"filled", "striped", "rounded", "radial"},
		},

		// No values for free-form types
		"NoValuesForColor": {
			src:      `graph { a [color=] }`,
			position: token.Position{Line: 1, Column: 18},
			want:     []string{},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ps := dot.NewParser([]byte(tt.src))
			root := ps.Parse()

			items := Items(root, tt.position)
			got := make([]string, len(items))
			for i, item := range items {
				got[i] = item.Label
			}

			assert.EqualValuesf(t, got, tt.want, "unexpected items")

			if tt.wantInsertText != nil {
				gotInsertText := make(map[string]string)
				for _, item := range items {
					if item.InsertText != nil {
						gotInsertText[item.Label] = *item.InsertText
					}
				}
				assert.EqualValuesf(t, gotInsertText, tt.wantInsertText, "unexpected InsertText")
			}
		})
	}
}
