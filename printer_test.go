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
		// TODO are there any special characters that require me to keep things quoted, yes escaped
		// quotes
		// TODO add/adjust test with a rune > 1 byte
		"GraphWithQuotedIDThatIsAKeyword": {
			in: `strict graph 
					"graph"     {}`,
			want: `strict graph "graph" {}`,
		},
		"GraphWithQuotedID": {
			in: `strict graph 
					"galaxy"     {}`,
			want: `strict graph galaxy {}`,
		},
		"NodeWithQuotedIDOfMaxWidthThatCanBeUnquoted": { // as the resulting ID is below maxColumn
			in: `graph {
	"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaab"
}`,
			want: `graph {
	aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaab
}`,
		},
		"NodeWithQuotedIDPastMaxWidthThatCannotBeUnquoted": { // as the resulting ID would be above maxColumn
			in: `graph {
	"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaab"
}`,
			want: `graph {
	"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\
aaab"
}`,
		},
		"NodeWithUnquotedIDOfMaxWidth": {
			in: `graph {
	aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaab
}`,
			want: `graph {
	aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaab
}`,
		},
		"NodeWithUnquotedIDPastMaxWidth": {
			in: `graph {
	aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaab
}`,
			want: `graph {
	"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\
aab"
}`,
		},
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
			p := dot.NewPrinter(strings.NewReader(test.in), &got)
			err := p.Print()
			require.NoErrorf(t, err, "Print(%q)", test.in)

			assert.EqualValuesf(t, got.String(), test.want, "Print(%q)", test.in)
		})
	}
}
