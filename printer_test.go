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
					"galaxy"     {}`,
			want: `strict graph "galaxy" {}`,
		},
		// World in Chinese each rune is 3 bytes long 世界
		"NodeWithQuotedIDOfMaxWidth": {
			in: `graph {
	"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa世界aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
}`,
			want: `graph {
	"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa世界aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
}`,
		},
		"NodeWithQuotedIDPastMaxWidth": {
			in: `graph {
	"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa世界aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
}`,
			want: `graph {
	"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa世界aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\
aa"
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
		// TODO test attr_stmt
		// TODO test node_stmt
		// 		"SingleAttributeStatement": {
		// 			in: `graph {
		// graph     [ 	label="blue"]
		// 			}`,
		// 			want: `graph {
		// 	graph [label="blue"]
		// }`,
		// 		},
		// TODO add tests for trailing , and ; that are stripped
		// TODO add tests for A [a=b,c=d] [e=f] -> [a=b,c=d,e=f]
		"NodeStatementWithSingleAttribute": {
			in: `graph {
A     [ 	label="blue"]
			}`,
			want: `graph {
	A [label="blue"]
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
