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

		B:"center":_ C:"south"
			D:n
				
			}`,
			want: `graph {
	A:"north":n
	B:"center"
	C:"south"
	D:n
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
		label="blue"
		color=grey
		size=0.1
	]
}`,
		},
		"NodeStatementWithMultipleAttributeLists": {
			in: `graph {
A     [ 	label="blue", ] [color=grey ;	size =	0.1,] [ ]
			}`,
			want: `graph {
	A [
		label="blue"
		color=grey
		size=0.1
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
		color="blue"
		len=2.6
	]
	rank=same
}`,
		},
		"EdgeStmtWithSubgraphs": {
			in: `
graph {
{1;2} -- subgraph "numbers" {node [color=blue;style=filled]; 3; 4}
}
`,
			want: `graph {
	subgraph {
		1
		2
	} -- subgraph "numbers" {
		node [
			color=blue
			style=filled
		]
		3
		4
	}
}`,
		},
		"EmptyAttrStatements": {
			in:   `graph { node []; edge[]; graph[];}`,
			want: `graph {}`,
		},
		"AttrStatementWithSingleAttribute": {
			in: `graph {
graph     [ 	label="blue",]
			}`,
			want: `graph {
	graph [label="blue"]
}`,
		},
		"AttrStatementWithIDOfMaxWidth": {
			in: `graph {
	"Node1234" [label="This is a test\nof a long multi-line\nlabel where the value exceeds the max col"]
}`,
			want: `graph {
	"Node1234" [label="This is a test\nof a long multi-line\nlabel where the value exceeds the max col"]
}`,
		},
		"AttrStatementWithIDPastMaxWidth": {
			in: `graph {
	"Node1234" [label="This is a test\nof a long multi-line\nlabel where the value exceeds the max col."]
}`,
			want: `graph {
	"Node1234" [label="This is a test\nof a long multi-line\nlabel where the value exceeds the max co\
l."]
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
		// TODO align closing marker as gofumpt does, I need to know keep track of p.row or maybe
		// know the tokens range
		// TODO break up comments that are too long
		// TODO test comments on the same line as other statements
		"EmptyComments": {
			in: `graph {
		#    	
			//    
  /*    */
}`,
			want: `graph {}`,
		},
		"CommentsGetOneLeadingSpace": {
			in: `graph {
//indent and add one space
		#		indent and remove leading whitespace, adding one space
			/*	  this is a multi-line marker comment on a single line */
  			/*	  this is a multi-line comment
		next line gets the current indentation added

			*/
}`,
			want: `graph {
	// indent and add one space
	# indent and remove leading whitespace, adding one space
	/* this is a multi-line marker comment on a single line */
	/* this is a multi-line comment
		next line gets the current indentation added

			*/
}`,
		},
	}
	/*
		asdf asdff sad asd  as dfasd
	*/

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
