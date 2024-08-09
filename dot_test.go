package dot_test

import (
	"strings"
	"testing"

	"github.com/teleivo/assertive/assert"
	"github.com/teleivo/assertive/require"
	"github.com/teleivo/dot"
)

func TestParser(t *testing.T) {
	tests := map[string]struct {
		in   string
		want dot.Graph
		err  error
	}{
		"Empty": {
			in:   "",
			want: dot.Graph{},
		},
		"EmptyDirectedGraph": {
			in: "digraph {}",
			want: dot.Graph{
				Directed: true,
			},
		},
		"EmptyUndirectedGraph": {
			in:   "graph {}",
			want: dot.Graph{},
		},
		// "StrictDirectedNamedGraph": {
		// 	in: `strict digraph dependencies {}`,
		// 	want: dot.Graph{
		// 		Strict:   true,
		// 		Directed: true,
		// 	},
		// },
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			p, err := dot.New(strings.NewReader(test.in))

			require.NoErrorf(t, err, "New(%q)", test.in)

			g, err := p.Parse()

			assert.NoErrorf(t, err, "Parse(%q)", test.in)
			assert.EqualValuesf(t, g, &test.want, "Parse(%q)", test.in)
		})
	}
}
