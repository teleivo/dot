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
	tests := []struct {
		in   string
		want string
	}{
		{
			in:   `graph {}`,
			want: `graph {}`,
		},
		{
			in:   `strict graph {}`,
			want: `strict graph {}`,
		},
		{
			in: `strict graph 
					"galaxy" {}`,
			want: `strict graph "galaxy" {}`,
		},
		{
			in: `strict digraph {
			3 	->     2
}

			`, // TODO add some semicolons in here?
			want: `strict digraph {
	3 -> 2
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
