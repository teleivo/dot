package completion

import (
	"slices"
	"testing"

	"github.com/teleivo/assertive/assert"
)

func TestAttributeMarkdownDoc(t *testing.T) {
	tests := map[string]struct {
		name string
		want string
	}{
		"WithEnumType": {
			name: "dir",
			want: "Edge type for drawing arrowheads\n\n**Type:** [dirType](https://graphviz.org/docs/attr-types/dirType/): `back` | `both` | `forward` | `none`\n\n[Docs](https://graphviz.org/docs/attrs/dir/)",
		},
		"WithNonEnumType": {
			name: "color",
			want: "Basic drawing color for graphics\n\n**Type:** [color](https://graphviz.org/docs/attr-types/color/)\n\nColor value. Format: #rrggbb, #rrggbbaa, H,S,V, or name\n\n[Docs](https://graphviz.org/docs/attrs/color/)",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			i := slices.IndexFunc(Attributes, func(a Attribute) bool { return a.Name == tt.name })
			got := Attributes[i].MarkdownDoc
			assert.EqualValuesf(t, got, tt.want, "unexpected markdown")
		})
	}
}

func TestAttributeContextString(t *testing.T) {
	tests := []struct {
		ctx  AttributeContext
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
			got := tt.ctx.String()
			assert.EqualValuesf(t, got, tt.want, "unexpected string")
		})
	}
}
