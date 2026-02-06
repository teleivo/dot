package attribute

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
		"WithLayoutType": {
			name: "layout",
			want: "Which layout engine to use\n\n**Type:** [layout](https://graphviz.org/docs/layouts/): `circo` | `dot` | `fdp` | `neato` | `osage` | `patchwork` | `sfdp` | `twopi`\n\n[Docs](https://graphviz.org/docs/attrs/layout/)",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			i := slices.IndexFunc(Attributes, func(a Attribute) bool { return a.Name == tt.name })
			got := Attributes[i].MarkdownDoc
			assert.EqualValues(t, got, tt.want, "unexpected markdown")
		})
	}
}
