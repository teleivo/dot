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
		"NodeWithQuotedIDOfMaxColumn": {
			in: `graph {
	"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa世界aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
}`,
			want: `graph {
	"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa世界aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
}`,
		},
		"NodeWithQuotedIDPastMaxColumn": {
			in: `graph {
	"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa世界aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
}`,
			want: `graph {
	"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa世界aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\
aa"
}`,
		},
		"NodeWithUnquotedIDOfMaxColumn": {
			in: `graph {
	aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaab
}`,
			want: `graph {
	aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaab
}`,
		},
		"NodeWithUnquotedIDPastMaxColumn": {
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
		"AttrStatementWithIDOfMaxColumn": {
			in: `graph {
	"Node1234" [label="This is a test\nof a long multi-line\nlabel where the value exceeds the max col"]
}`,
			want: `graph {
	"Node1234" [label="This is a test\nof a long multi-line\nlabel where the value exceeds the max col"]
}`,
		},
		"AttrStatementWithIDPastMaxColumn": {
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
		// TODO fix current test
		// TODO add test showing that single/multi-line comments can be 100 runes wide
		// TODO improve by breaking up at - as well? is - valid in urls or only percent encoded?
		// TODO test comments on the same line as other statements
		// TODO cleanup implementation
		"EmptyCommentsAreDiscarded": {
			in: `graph {
		#    	
			//    
  /*   

			*/
}`,
			want: `graph {}`,
		},
		"CommentsSingleLineUseSameMarker": {
			in: `graph {
//indent and add one space  
		#		indent and remove leading whitespace, adding one space  
}`,
			want: `graph {
	// indent and add one space
	// indent and remove leading whitespace, adding one space
}`,
		},
		"CommentsMultiLineThatFitsOntoSingleLineIsTransformed": {
			in: `graph {
			/*	  this is a multi-line marker  
			comment that fits onto a single line                            */
}`,
			want: `graph {
	// this is a multi-line marker comment that fits onto a single line
}`,
		},
		"CommentsMultiLineIsBrokenUpAtWordBoundary": {
			in: `graph {
	/* this is a multi-line comment that will not fit onto a single line so it will stay a
			 multi-line comment but get stripped of its      superfluous    whitespace	

			nonetheless

			*/
}`,
			want: `graph {
	/*
		this is a multi-line comment that will not fit onto a single line so it will stay a multi-line
		comment but get stripped of its superfluous whitespace nonetheless
	*/
}`,
		},
		"CommentsMultiLineDoesNotBreakupWordsWhichAreGreaterThanMaxColumn": {
			in: `graph {
	// this uses a single-line marker but is too long for a single line https://github.com/teleivo/dot/blob/fake/27b6dbfe4b99f67df74bfb7323e19d6c547f68fd/parser_test.go#L13
}`,
			want: `graph {
	/*
		this uses a single-line marker but is too long for a single line
		https://github.com/teleivo/dot/blob/fake/27b6dbfe4b99f67df74bfb7323e19d6c547f68fd/parser_test.go#L13
	*/
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
