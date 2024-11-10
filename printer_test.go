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
			in: `strict graph {}


			`,
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
		"NodeStatementsWithPorts": {
			in: `graph {
		
				A:"north":n
		B:"center":_
		
	C:"south"
				
			}`,
			want: `graph {
	A:"north":n
	B:"center"
	C:"south"
}`,
		},
		"NodeStatementWithSingleAttribute": {
			in: `graph {
A        	[ 	label="blue",]
			}`,
			want: `graph {
	A [label="blue"]
}`,
		},
		"NodeStatementWithMultipleAttributes": {
			in: `graph {
A     [ 	label="blue", color=grey; size=0.1,]
			}`,
			want: `graph {
	A [
		label="blue",
		color=grey,
		size=0.1,
	]
}`,
		},
		"NodeStatementWithMultipleAttributeLists": {
			in: `graph {
A     [ 	label="blue", ] [color=grey ;	size =	0.1,] [ ]
			}`,
			want: `graph {
	A [
		label="blue",
		color=grey,
		size=0.1,
	]
}`,
		},
		"DigraphEdgeStmt": {
			in: `digraph {
			3 	->     2->4  [
		color = "blue", len = 2.6
	]; rank=same;}
`,
			want: `digraph {
	3 -> 2 -> 4 [
		color="blue",
		len=2.6,
	]
	rank=same
}`,
		},
		"AttrStatementWithSingleAttribute": {
			in: `graph {
graph     [ 	label="blue",]
			}`,
			want: `graph {
	graph [label="blue"]
}`,
		},
		"AttributeStatementWithSingleAttribute": {
			in: `graph {
label="blue", minlen=2;
 color=grey;
			}`,
			want: `graph {
	label="blue"
	minlen=2
	color=grey
}`,
		},
		"Subgraph": {
			in: `digraph {
A;subgraph family { 
				label   = "parents";
			Parent1 -> Child1; Parent2 -> Child2
				subgraph 	"grandparents"  {
		label   = "grandparents"
Grandparent1  -> Parent1; Grandparent2 -> Parent1;
 Grandparent3  -> Parent2; Grandparent4 -> Parent2;
	  	}
			}
}`,
			want: `digraph {
	A
	subgraph family {
		label="parents"
		Parent1 -> Child1
		Parent2 -> Child2
		subgraph "grandparents" {
			label="grandparents"
			Grandparent1 -> Parent1
			Grandparent2 -> Parent1
			Grandparent3 -> Parent2
			Grandparent4 -> Parent2
		}
	}
}`,
		},
		"SubgraphWithoutKeyword": {
			in: `graph
				{
			{A -- B; C--E}
}`,
			want: `graph {
	subgraph {
		A -- B
		C -- E
	}
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
