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
		"GraphEmpty": {
			in: `strict graph {	
			}


			`,
			want: `strict graph {
}`,
		},
		"GraphWithID": {
			in: `strict graph 
					"galaxy"     {}`,
			want: `strict graph "galaxy" {
}`,
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
	"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa世界aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaab"
}`,
			want: `graph {
	"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa世界aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\
	b"
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
		"NodeStmtWithSingleAttribute": {
			in: `graph {
A        	[ 	label="blue",]
			}`,
			want: `graph {
	A [label="blue"]
}`,
		},
		"NodeStmtWithMultipleAttributes": {
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
		"NodeStmtWithMultipleAttributeLists": {
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
		"EdgeStmtDigraph": {
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
		// TODO fix this as outlined in the codes todo
		// "AttrStmtsEmpty": {
		// 	in:   `graph { node []; edge[]; graph[];}`,
		// 	want: `graph {}`,
		// },
		"AttrStmtWithSingleAttribute": {
			in: `graph {
graph     [ 	label="blue",]
			}`,
			want: `graph {
	graph [label="blue"]
}`,
		},
		// TODO think about this again. Newlines in the ID should restart the counter towards maxcolumn
		// 		"AttrStmtWithIDOfMaxColumn": {
		// 			in: `graph {
		// 	"Node1234" [label="This is a test\nof a long multi-line\nlabel where the value exceeds the max col"]
		// }`,
		// 			want: `graph {
		// 	"Node1234" [label="This is a test\nof a long multi-line\nlabel where the value exceeds the max col"]
		// }`,
		// 		},
		// 		"AttrStmtWithIDPastMaxColumn": {
		// 			in: `graph {
		// 	"Node1234" [label="This is a test\nof a long multi-line\nlabel where the value exceeds the max col."]
		// }`,
		// 			want: `graph {
		// 	"Node1234" [label="This is a test\nof a long multi-line\nlabel where the value exceeds the max co\
		// 		l."]
		// }`,
		// 		},
		"AttributeStmtWithSingleAttribute": {
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
		"CommentsWithOnlyWhitespaceAreDiscarded": {
			in: `graph {
		#    	
			//    
  /*   

			*/
}`,
			want: `graph {
}`,
		},
		"CommentsSingleLineAreChangedToCppMarker": {
			in: `graph {
		//this   is a comment! that has exactly 100 runes, 	which is the max column of dotfmt like it or not!
#this   is a comment! that has exactly 100 runes, 	which is the max column of dotfmt like it or not!
}`,
			want: `graph {
// this is a comment! that has exactly 100 runes, which is the max column of dotfmt like it or not!
// this is a comment! that has exactly 100 runes, which is the max column of dotfmt like it or not!
}`,
		},
		"CommentsSingleLineThatExceedMaxColumnAreBrokenUp": {
			in: `graph {
		//this   is a comment! that has a bit more than 100 runes, 	which is the max column of dotfmt like it or not!
#this   is a comment! that has a bit more than 100 runes, 	which is the max column of dotfmt like it or not!
// this is a comment! that has exactly 101 runes, which is the max column of dotfmt like it or knot2!
}`,
			want: `graph {
// this is a comment! that has a bit more than 100 runes, which is the max column of dotfmt like it
// or not!
// this is a comment! that has a bit more than 100 runes, which is the max column of dotfmt like it
// or not!
// this is a comment! that has exactly 101 runes, which is the max column of dotfmt like it or
// knot2!
}`,
		},
		"CommentsMultiLineThatFitOntoSingleLineAreChangedToSingleLineMarker": {
			in: `graph {
			/*	  this is a multi-line marker  
			comment that fits onto a single line                            */
}`,
			want: `graph {
// this is a multi-line marker comment that fits onto a single line
}`,
		},
		"CommentsMultiLineAreAreChangedToCppMarkerRespectingWordBoundaries": {
			in: `graph {
	/* this is a multi-line comment that will not fit onto a single line so it will stay a
			 multi-line comment but get stripped of its      superfluous    whitespace	

			nonetheless

			*/
}`,
			want: `graph {
// this is a multi-line comment that will not fit onto a single line so it will stay a multi-line
// comment but get stripped of its superfluous whitespace nonetheless
}`,
		},
		"CommentsMultiLineWithWordsWhichAreGreaterThanMaxColumnAreNotBrokenUp": {
			in: `graph {
	// this uses a single-line marker but is too long for a single line https://github.com/teleivo/dot/blob/fake/27b6dbfe4b99f67df74bfb7323e19d6c547f68fd/parser_test.go#L13
}`,
			want: `graph {
// this uses a single-line marker but is too long for a single line
// https://github.com/teleivo/dot/blob/fake/27b6dbfe4b99f67df74bfb7323e19d6c547f68fd/parser_test.go#L13
}`,
		},
		// TODO test comments on the same line as other statements
		// add a test for breaking up the comment
		// add a test for a multi-line comment like A -- B /* foo */; B -- C
		// or choose an attribute statement?
		// TODO cleanup implementation
		// "CommentsNextToStmtsAreKeptOnTheSameLine": {
		// 	in: `graph {
		// 	A [
		// 		style="filled"
		// 		color="blue"
		// 	] //  keep me
		// 	A -- B 	 //   stays on the same line
		// }`,
		// 	want: `graph {
		// 	A [
		// 		style="filled"
		// 		color="blue"
		// 	] // keep me
		// 	A -- B // stays on the same line
		// }`,
		// },
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
