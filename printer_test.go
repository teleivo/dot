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
					"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {}`,
			want: `strict graph "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {}`,
		},
		"GraphIDAboveMaxLen": {
			in: `strict graph 
					"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaab" {}`,
			want: `strict graph "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\
b" {}`,
		},
		"DigraphWithMulipleEdges": {
			in: `strict digraph {
			3 	->     2->4
}

			`, // TODO add some semicolons in here?
			want: `strict digraph {
	3 -> 2 -> 4
}`,
		},
	}

	for _, test := range tests {
		var got bytes.Buffer
		err := dot.Print(strings.NewReader(test.in), &got)
		require.NoErrorf(t, err, "Print(%q)", test.in)

		assert.EqualValuesf(t, got.String(), test.want, "Print()")
	}
}
