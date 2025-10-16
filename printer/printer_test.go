package printer_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/teleivo/assertive/require"
	"github.com/teleivo/dot/internal/layout"
	"github.com/teleivo/dot/printer"
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
		"NodeStmtWithAttributeIDPastMaxColumn": {
			in: `graph {
"Node1234" [label="This is a test of a long attribute value that is past the max column which should be split on word boundaries several times of course as long as this is necessary it should also respect giant URLs https://github.com/teleivo/dot/blob/fake/27b6dbfe4b99f67df74bfb7323e19d6c547f68fd/parser_test.go#L13"]
		}`,
			want: `graph {
	"Node1234" [
		label="This is a test of a long attribute value that is past the max column which should be split on word boundaries several times of course as long as this is necessary it should also respect giant URLs https://github.com/teleivo/dot/blob/fake/27b6dbfe4b99f67df74bfb7323e19d6c547f68fd/parser_test.go#L13"
	]
}`,
		},
		// TODO do I want this below instead of the above? if so how to achieve that? this splits the quoted ID using line
		// continuation
		// 		"NodeStmtWithAttributeIDPastMaxColumn": {
		// 			in: `graph {
		// "Node1234" [label="This is a test of a long attribute value that is past the max column which should be split on word boundaries several times of course as long as this is necessary it should also respect giant URLs https://github.com/teleivo/dot/blob/fake/27b6dbfe4b99f67df74bfb7323e19d6c547f68fd/parser_test.go#L13"]
		// 		}`,
		// 			want: `graph {
		// 	"Node1234" [label="This is a test of a long attribute value that is past the max column which \
		// should be split on word boundaries several times of course as long as this is necessary it should \
		// also respect giant URLs \
		// https://github.com/teleivo/dot/blob/fake/27b6dbfe4b99f67df74bfb7323e19d6c547f68fd/parser_test.go#L13"]
		// }`,
		// 		},
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
	A [label="blue",color=grey,size=0.1]
}`,
		},
		"NodeStmtWithMultipleAttributeLists": {
			in: `graph {
A     [ 	label="blue", ] [color=grey ;	size =	0.1,] [ ]
			}`,
			want: `graph {
	A [label="blue"] [color=grey,size=0.1] []
}`,
		},
		"EdgeStmtDigraph": {
			in: `digraph {
			3 	->     2->4  [
		color = "blue", len = 2.6
	]; rank=same;}
`,
			want: `digraph {
	3 -> 2 -> 4 [color="blue",len=2.6]
	rank=same
}`,
		},
		"EdgeStmtWithAttributesPastMaxColumn": {
			in: `digraph {
			3 	->     2->4 -> "five" -> "sixteen"  [
		color = "blue", len = 2.6 font	= "Helvetica patched" background = "transparent red" arrowtail = "halfopen"]; rank=same;}
`,
			want: `digraph {
	3 -> 2 -> 4 -> "five" -> "sixteen" [
		color="blue"
		len=2.6
		font="Helvetica patched"
		background="transparent red"
		arrowtail="halfopen"
	]
	rank=same
}`,
		},
		"EdgeStmtWithFirstAttributeListFitting": {
			in: `digraph {
			3 	->     2->4 -> "five" -> "sixteen"  [
		color = "blue", len = 2.6] [arrowtail = "halfopen",arrowhead=diamond]; rank=same;}
`,
			want: `digraph {
	3 -> 2 -> 4 -> "five" -> "sixteen" [color="blue",len=2.6] [
		arrowtail="halfopen"
		arrowhead=diamond
	]
	rank=same
}`,
		},
		"EdgeStmtWithMultipleAttributeListsPastMaxColumn": {
			in: `digraph {
			3 	->     2->4 -> "five" -> "sixteen"  [
		color = "blue", len = 2.6 font	= "Helvetica patched" background = "transparent red" ] [arrowtail = "halfopen",arrowhead=diamond][ arrowtail="halfopen" arrowhead=diamond beautify=true taillabel="tail" ]; rank=same;}
`,
			want: `digraph {
	3 -> 2 -> 4 -> "five" -> "sixteen" [
		color="blue"
		len=2.6
		font="Helvetica patched"
		background="transparent red"
	] [arrowtail="halfopen",arrowhead=diamond] [
		arrowtail="halfopen"
		arrowhead=diamond
		beautify=true
		taillabel="tail"
	]
	rank=same
}`,
		},
		"EdgeStmtWithSubgraphs": {
			in: `
graph {
{1;2--{3;4}} -- subgraph "numbers" {node [color=blue;style=filled]; 3; 4}-- subgraph "numbers" {node [color=blue;style=filled]; 3; 4}
}
`,
			want: `graph {
	{
		1
		2 -- {
			3
			4
		}
	} -- subgraph "numbers" {
		node [color=blue,style=filled]
		3
		4
	} -- subgraph "numbers" {
		node [color=blue,style=filled]
		3
		4
	}
}`,
		},
		"AttrStmtsEmpty": {
			in: `graph { node []; edge[]; graph[];}`,
			want: `graph {
	node []
	edge []
	graph []
}`,
		},
		"AttrStmtWithEmptyAndSingleAttribute": {
			in: `graph {
graph    [] [ 	label="blue",]
			}`,
			want: `graph {
	graph [] [label="blue"]
}`,
		},
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
	{
		A -- B
		C -- E
	}
}`,
		},
		// 		"CommentsWithOnlyWhitespaceAreDiscarded": {
		// 			in: `graph {
		// 		#
		// 			//
		//   /*
		//
		// 			*/
		// }`,
		// 			want: `graph {
		// }`,
		// 		},
		// 		"CommentsSingleLineAreChangedToCppMarker": {
		// 			in: `graph {
		// 		//this   is a comment! that has exactly 100 runes, 	which is the max column of dotfmt like it or not!
		// #this   is a comment! that has exactly 100 runes, 	which is the max column of dotfmt like it or not!
		// }`,
		// 			want: `graph {
		// // this is a comment! that has exactly 100 runes, which is the max column of dotfmt like it or not!
		// // this is a comment! that has exactly 100 runes, which is the max column of dotfmt like it or not!
		// }`,
		// 		},
		// 		"CommentsSingleLineThatExceedMaxColumnAreBrokenUp": {
		// 			in: `graph {
		// 		//this   is a comment! that has a bit more than 100 runes, 	which is the max column of dotfmt like it or not!
		// #this   is a comment! that has a bit more than 100 runes, 	which is the max column of dotfmt like it or not!
		// // this is a comment! that has exactly 101 runes, which is the max column of dotfmt like it or knot2!
		// }`,
		// 			want: `graph {
		// // this is a comment! that has a bit more than 100 runes, which is the max column of dotfmt like it
		// // or not!
		// // this is a comment! that has a bit more than 100 runes, which is the max column of dotfmt like it
		// // or not!
		// // this is a comment! that has exactly 101 runes, which is the max column of dotfmt like it or
		// // knot2!
		// }`,
		// 		},
		// 		"CommentsMultiLineThatFitOntoSingleLineAreChangedToSingleLineMarker": {
		// 			in: `graph {
		// 			/*	  this is a multi-line marker
		// 			comment that fits onto a single line                            */
		// }`,
		// 			want: `graph {
		// // this is a multi-line marker comment that fits onto a single line
		// }`,
		// 		},
		// 		"CommentsMultiLineAreAreChangedToCppMarkerRespectingWordBoundaries": {
		// 			in: `graph {
		// 	/* this is a multi-line comment that will not fit onto a single line so it will stay a
		// 			 multi-line comment but get stripped of its      superfluous    whitespace
		//
		// 			nonetheless
		//
		// 			*/
		// }`,
		// 			want: `graph {
		// // this is a multi-line comment that will not fit onto a single line so it will stay a multi-line
		// // comment but get stripped of its superfluous whitespace nonetheless
		// }`,
		// 		},
		// 		"CommentsMultiLineWithWordsWhichAreGreaterThanMaxColumnAreNotBrokenUp": {
		// 			in: `graph {
		// 	// this uses a single-line marker but is too long for a single line https://github.com/teleivo/dot/blob/fake/27b6dbfe4b99f67df74bfb7323e19d6c547f68fd/parser_test.go#L13
		// }`,
		// 			want: `graph {
		// // this uses a single-line marker but is too long for a single line
		// // https://github.com/teleivo/dot/blob/fake/27b6dbfe4b99f67df74bfb7323e19d6c547f68fd/parser_test.go#L13
		// }`,
		// 		},
		// 		"CommentsWithSingleWord": {
		// 			in: `graph {//graph
		// 	C -- subgraph {//subgraph
		// 	}
		// }`,
		// 			want: `graph { // graph
		// 	C -- subgraph { // subgraph
		// 	}
		// }`,
		// 		},
		// 		"CommentsOnSubgraphStickToPreviousTokens": {
		// 			in: `graph {
		// 	C -- subgraph {
		// 		D   //D is cool
		// // stay with E
		// 		E
		// 	} // comment the subgraph
		//
		// 		}`,
		// 			want: `graph {
		// 	C -- subgraph {
		// 		D // D is cool
		// 		// stay with E
		// 		E
		// 	} // comment the subgraph
		// }`,
		// 		},
		// 		"CommentsStickToAttributes": {
		// 			in: `graph {
		// 	A [
		// 		style="filled" // always
		// 		color="pink" // what else!
		// 	] //  keep me
		// }`,
		// 			want: `graph {
		// 	A [
		// 		style="filled" // always
		// 		color="pink" // what else!
		// 	] // keep me
		// }`,
		// 		},
		// 		"CommentsBeforeGraph": {
		// 			in: `
		// 			// this is my graph
		// 							// and I do what I want to!
		//
		// 			graph {
		// 		}
		//
		// `,
		// 			want: `// this is my graph
		// // and I do what I want to!
		// graph {
		// }`,
		// 		},
		// 		"CommentsAfterGraph": {
		// 			in: `graph {
		// 		}//this is kept here
		//
		// 			//	 oh wait !`,
		// 			want: `graph {
		// } // this is kept here
		// // oh wait !`,
		// 		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			var gotFirst bytes.Buffer
			p := printer.NewPrinter(strings.NewReader(test.in), &gotFirst, layout.Default)
			err := p.Print()
			require.NoErrorf(t, err, "Print(%q)", test.in)

			if gotFirst.String() != test.want {
				t.Fatalf("\n\nin:\n%s\n\ngot:\n%s\n\n\nwant:\n%s\n", test.in, gotFirst.String(), test.want)
			}

			t.Logf("print again with the previous output as the input to ensure printing is idempotent")

			var gotSecond bytes.Buffer
			p = printer.NewPrinter(strings.NewReader(gotFirst.String()), &gotSecond, layout.Default)
			err = p.Print()
			require.NoErrorf(t, err, "Print(%q)", gotFirst.String())

			if gotSecond.String() != gotFirst.String() {
				t.Errorf("\n\nin:\n%s\n\ngot:\n%s\n\n\nwant:\n%s\n", gotFirst.String(), gotSecond.String(), gotFirst.String())
			}
		})
	}
}
