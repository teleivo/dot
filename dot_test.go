package dot_test

import (
	"strings"
	"testing"

	"github.com/teleivo/assertive/assert"
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
			in:   "digraph {}",
			want: dot.Graph{},
		},
		"EmptyUndirectedGraph": {
			in:   "graph {}",
			want: dot.Graph{},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			p := dot.New(strings.NewReader(test.in))

			g, err := p.Parse()

			assert.NoError(t, err)
			assert.EqualValues(t, g, &test.want)
		})
	}
}
