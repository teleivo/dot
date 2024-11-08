package dot_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/teleivo/assertive/assert"
	"github.com/teleivo/assertive/require"
	"github.com/teleivo/dot"
)

func TestPrint(t *testing.T) {
	tests := map[string]struct {
		in   string
		want string
	}{
		"EmptyGraph": {
			in:   `strict graph {}`,
			want: `strict graph {}`,
		},
		"GraphWithID": {
			in: `strict graph 
					galaxy     {}`,
			want: `strict graph galaxy {}`,
		},
		"NodeWithQuotedIDOfMaxWidth": {
			in: `graph {
	"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaab"
}`,
			want: `graph {
	"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaab"
}`,
		},
		// TODO add escaped quote in here
		"NodeWithQuotedIDPastMaxWidth": {
			in: `graph {
	"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaab"
}`,
			want: `graph {
	"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\
ab"
}`,
		},
		// TODO strip quotes unless needed = quoted keyword or > 100
		"DigraphWithMulipleEdges": {
			in: `digraph {
			3 	->     2->4
}

			`, // TODO add some semicolons in here?
			want: `digraph {
	3 -> 2 -> 4
}`,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			var got bytes.Buffer
			err := dot.Print(strings.NewReader(test.in), &got)
			require.NoErrorf(t, err, "Print(%q)", test.in)

			assert.EqualValuesf(t, got.String(), test.want, "Print(%q)", test.in)
		})
	}
}
