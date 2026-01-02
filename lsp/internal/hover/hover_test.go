package hover

import (
	"strings"
	"testing"

	"github.com/teleivo/assertive/assert"
	"github.com/teleivo/dot"
	"github.com/teleivo/dot/token"
)

func TestInfo(t *testing.T) {
	tests := map[string]struct {
		src      string
		position token.Position
		wantNil  bool   // expect nil result
		wantDoc  string // substring expected in markdown doc
	}{
		"AttrNameShape": {
			// 'shape' is at 1:12-16
			src:      `graph { a [shape=box] }`,
			position: token.Position{Line: 1, Column: 12},
			wantDoc:  "Shape of a node",
		},
		"AttrValueBox": {
			// 'box' is at 1:18-20
			src:      `graph { a [shape=box] }`,
			position: token.Position{Line: 1, Column: 18},
			wantDoc:  "shape", // value doc links to shape type
		},
		"AttrNameDir": {
			// 'dir' is at 1:19-21
			src:      `digraph { a -> b [dir=back] }`,
			position: token.Position{Line: 1, Column: 19},
			wantDoc:  "Edge type for drawing arrowheads",
		},
		"AttrValueBack": {
			// 'back' is at 1:23-26
			src:      `digraph { a -> b [dir=back] }`,
			position: token.Position{Line: 1, Column: 23},
			wantDoc:  "dirType", // value doc links to dirType
		},
		"UnknownAttrName": {
			// 'foo' is not a known attribute
			src:      `graph { a [foo=bar] }`,
			position: token.Position{Line: 1, Column: 12},
			wantNil:  true,
		},
		"UnknownAttrValue": {
			// 'unknown' is not a known value for shape
			src:      `graph { a [shape=unknown] }`,
			position: token.Position{Line: 1, Column: 18},
			wantNil:  true,
		},
		"OutsideAttrList": {
			// cursor on node 'a', not in attribute list
			src:      `graph { a [shape=box] }`,
			position: token.Position{Line: 1, Column: 9},
			wantNil:  true,
		},
		"EmptySource": {
			src:      ``,
			position: token.Position{Line: 1, Column: 1},
			wantNil:  true,
		},
		"AttrNameWithoutValue": {
			// 'shape' at 1:12-16, no value assigned yet
			src:      `graph { a [shape] }`,
			position: token.Position{Line: 1, Column: 12},
			wantDoc:  "Shape of a node",
		},
		"QuotedValue": {
			// "box" at 1:18-22 (with quotes) - literal includes quotes so no match
			src:      `graph { a [shape="box"] }`,
			position: token.Position{Line: 1, Column: 19},
			wantNil:  true,
		},
		"MultipleAttrsSecond": {
			// 'color' at 1:22-26
			src:      `graph { a [shape=box,color=red] }`,
			position: token.Position{Line: 1, Column: 22},
			wantDoc:  "Basic drawing color",
		},
		"CursorOnEquals": {
			// '=' at 1:17 - falls back to attr name hover since no value match at this position
			src:      `graph { a [shape=box] }`,
			position: token.Position{Line: 1, Column: 17},
			wantDoc:  "Shape of a node",
		},
		"AttrStmtNode": {
			// 'shape' at 1:15-19
			src:      `graph { node [shape=box] }`,
			position: token.Position{Line: 1, Column: 15},
			wantDoc:  "Shape of a node",
		},
		"ClusterStyle": {
			// 'style' at 1:39-43
			src:      `graph { subgraph cluster_a { graph [style=filled] } }`,
			position: token.Position{Line: 1, Column: 37},
			wantDoc:  "Style information",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ps := dot.NewParser([]byte(tt.src))
			tree := ps.Parse()

			got := Info(tree, tt.position)

			if tt.wantNil {
				assert.Nilf(t, got, "expected nil hover for %q at %s", tt.src, tt.position)
			} else {
				assert.NotNilf(t, got, "expected non-nil hover for %q at %s", tt.src, tt.position)
				if got != nil {
					assert.Truef(t, strings.Contains(got.Contents.Value, tt.wantDoc),
						"hover doc %q should contain %q for %q at %s", got.Contents.Value, tt.wantDoc, tt.src, tt.position)
				}
			}
		})
	}
}
