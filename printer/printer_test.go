package printer_test

import (
	"bytes"
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
}
`,
		},
		"GraphWithID": {
			in: `strict graph
					"galaxy"     {}`,
			want: `strict graph "galaxy" {
}
`,
		},
		"NodeStmtWithAttributeIDPastMaxColumn": {
			in: `graph {
"Node1234" [label="This is a test of a long attribute value that is past the max column which should be split on word boundaries several times of course as long as this is necessary it should also respect giant URLs https://github.com/teleivo/dot/blob/fake/27b6dbfe4b99f67df74bfb7323e19d6c547f68fd/parser_test.go#L13"]
		}`,
			want: `graph {
	"Node1234" [
		label="This is a test of a long attribute value that is past the max column which should be split on word boundaries several times of course as long as this is necessary it should also respect giant URLs https://github.com/teleivo/dot/blob/fake/27b6dbfe4b99f67df74bfb7323e19d6c547f68fd/parser_test.go#L13"
	]
}
`,
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
}
`,
		},
		"NodeStmtWithSingleAttribute": {
			in: `graph {
A        	[ 	label="blue",]
			}`,
			want: `graph {
	A [label="blue"]
}
`,
		},
		"NodeStmtWithMultipleAttributes": {
			in: `graph {
A     [ 	label="blue", color=grey; size=0.1,]
			}`,
			want: `graph {
	A [label="blue",color=grey,size=0.1]
}
`,
		},
		"NodeStmtWithMultipleAttributeLists": {
			in: `graph {
A     [ 	label="blue", ] [color=grey ;	size =	0.1,] [ ]
			}`,
			want: `graph {
	A [label="blue"] [color=grey,size=0.1] []
}
`,
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
}
`,
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
}
`,
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
}
`,
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
}
`,
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
}
`,
		},
		"AttrStmtsEmpty": {
			in: `graph { node []; edge[]; graph[];}`,
			want: `graph {
	node []
	edge []
	graph []
}
`,
		},
		"AttrStmtWithEmptyAndSingleAttribute": {
			in: `graph {
graph    [] [ 	label="blue",]
			}`,
			want: `graph {
	graph [] [label="blue"]
}
`,
		},
		"AttributeStmtWithSingleAttribute": {
			in: `graph {
label="blue"; minlen=2;
 color=grey;
			}`,
			want: `graph {
	label="blue"
	minlen=2
	color=grey
}
`,
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
}
`,
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
}
`,
		},
		"MultipleGraphs": {
			in: `graph G1 { A }
digraph G2 { B -> C }`,
			want: `graph G1 {
	A
}
digraph G2 {
	B -> C
}
`,
		},
		"EscapeSequencesInStrings": {
			in: `graph {
A [label="line1\nline2"]
B [label="tab\there"]
C [label="quote\"here"]
D [label="backslash\\here"]
}`,
			want: `graph {
	A [label="line1\nline2"]
	B [label="tab\there"]
	C [label="quote\"here"]
	D [label="backslash\\here"]
}
`,
		},
		// Comment tests
		//
		// Line comments are preserved as-is with only indentation adjusted.
		// Content is never modified: no line wrapping, no whitespace normalization.

		// Basic placement
		"CommentLineBeforeStmt": {
			in: `graph {
	// comment before A
A
}`,
			want: `graph {
	// comment before A
	A
}
`,
		},
		"CommentTrailingID": {
			in: `graph {
A    //   trailing   spaces   preserved
}`,
			want: `graph {
	A //   trailing   spaces   preserved
}
`,
		},
		"CommentTrailingAttrStmtTarget": {
			in: `graph {
node    //   trailing
[color=red]
}`,
			want: `graph {
	node //   trailing
	[color=red]
}
`,
		},
		"CommentTrailingNodeID": {
			in: `graph {
A    //   trailing
[color=red]
}`,
			want: `graph {
	A //   trailing
	[color=red]
}
`,
		},
		"CommentTrailingEdgeStmt": {
			in: `digraph {
A -> B    //   trailing
[color=red]
}`,
			want: `digraph {
	A -> B //   trailing
	[color=red]
}
`,
		},
		"CommentTrailingSubgraphKeyword": {
			in: `graph {
subgraph    //   trailing
{
A
}
}`,
			want: `graph {
	subgraph //   trailing
	{
		A
	}
}
`,
		},
		"CommentTrailingCompassPoint": {
			in: `graph {
A:n    //   trailing
}`,
			want: `graph {
	A:n //   trailing
}
`,
		},
		"CommentTrailingPortWithCompassPoint": {
			in: `graph {
A:port:sw    //   trailing
}`,
			want: `graph {
	A:port:sw //   trailing
}
`,
		},
		"CommentTrailingAttrName": {
			in: `graph {
color    //   trailing
= red
}`,
			want: `graph {
	color //   trailing
	=red
}
`,
		},
		"CommentTrailingAttrValue": {
			in: `graph {
color = red    //   trailing
}`,
			want: `graph {
	color=red //   trailing
}
`,
		},
		// TODO: block comments
		// "CommentBlockBeforeStmt": {
		// 	in: `graph {
		// 		/* comment before A */
		// A
		// }`,
		// 	want: `graph {
		// 	/* comment before A */
		// 	A
		// }`,
		// },
		// "CommentBlockAfterStmt": {
		// 	in: `graph {
		// A /* trailing comment */
		// }`,
		// 	want: `graph {
		// 	A /* trailing comment */
		// }`,
		// },

		// Indentation correction
		// "CommentIndentationCorrectedInNestedSubgraph": {
		// 	in: `graph {
		// // wrong indent
		// 	A
		// 	subgraph {
		// 			// wrong indent in subgraph
		// 		B
		// 		subgraph {
		// // deeply wrong indent
		// 			C
		// 		}
		// 	}
		// }`,
		// 	want: `graph {
		// 	// wrong indent
		// 	A
		// 	subgraph {
		// 		// wrong indent in subgraph
		// 		B
		// 		subgraph {
		// 			// deeply wrong indent
		// 			C
		// 		}
		// 	}
		// }`,
		// },
		// Content preservation - max column is NOT applied to comments
		// "CommentLineExceedingMaxColumnPreserved": {
		// 	in: `graph {
		// 	// this is a very long comment that exceeds the max column limit but should be preserved exactly as written without any line breaking or wrapping
		// 	A
		// }`,
		// 	want: `graph {
		// 	// this is a very long comment that exceeds the max column limit but should be preserved exactly as written without any line breaking or wrapping
		// 	A
		// }`,
		// },
		// TODO: block comments
		// "CommentBlockExceedingMaxColumnPreserved": {
		// 	in: `graph {
		// 	/* this is a very long block comment that exceeds the max column limit but should be preserved exactly as written without any line breaking or wrapping */
		// 	A
		// }`,
		// 	want: `graph {
		// 	/* this is a very long block comment that exceeds the max column limit but should be preserved exactly as written without any line breaking or wrapping */
		// 	A
		// }`,
		// },
		// "CommentBlockMultilineFormattingPreserved": {
		// 	in: `graph {
		// 	/*
		// 	 * This block comment has
		// 	 * intentional formatting with
		// 	 * aligned asterisks that must
		// 	 * be preserved exactly
		// 	 */
		// 	A
		// }`,
		// 	want: `graph {
		// 	/*
		// 	 * This block comment has
		// 	 * intentional formatting with
		// 	 * aligned asterisks that must
		// 	 * be preserved exactly
		// 	 */
		// 	A
		// }`,
		// },
		// "CommentInternalWhitespacePreserved": {
		// 	in: `graph {
		// 	//    multiple   spaces   preserved
		// 	A
		// }`,
		// 	want: `graph {
		// 	//    multiple   spaces   preserved
		// 	A
		// }`,
		// },

		// Multiple comment types
		// "CommentPreprocessorStyle": {
		// 	in: `# preprocessor comment
		// graph {
		// 		# inside graph
		// 	A
		// }`,
		// 	want: `# preprocessor comment
		// graph {
		// 	# inside graph
		// 	A
		// }`,
		// },
		// TODO: block comments
		// "CommentMixedTypes": {
		// 	in: `// line comment
		// # preprocessor comment
		// /* block comment */
		// graph {
		// 	A
		// }`,
		// 	want: `// line comment
		// # preprocessor comment
		// /* block comment */
		// graph {
		// 	A
		// }`,
		// },

		// Edge cases
		// "CommentOnlyFile": {
		// 	in: `// just a comment`,
		// 	want: `// just a comment`,
		// },
		// TODO: block comments
		// "CommentInEmptyGraph": {
		// 	in: `graph {
		// 		/* comment in empty graph */
		// }`,
		// 	want: `graph {
		// 	/* comment in empty graph */
		// }`,
		// },
		// "CommentBetweenGraphs": {
		// 	in: `graph G1 {
		// 	A
		// }
		// // between graphs
		// graph G2 {
		// 	B
		// }`,
		// 	want: `graph G1 {
		// 	A
		// }
		// // between graphs
		// graph G2 {
		// 	B
		// }`,
		// },
		// TODO: block comments
		// "CommentAroundAttributes": {
		// 	in: `graph {
		// 	A [
		// 		/* before attr */ color=red /* after value */
		// 	]
		// }`,
		// 	want: `graph {
		// 	A [/* before attr */ color=red /* after value */]
		// }`,
		// },
		"CommentBeforeClosingBrace": {
			in: `graph {
	A
	// c1
}`,
			want: `graph {
	A
	// c1
}
`,
		},
		"CommentBeforeClosingBraceInSubgraph": {
			in: `graph {
	subgraph {
		A
		// c1
	}
}`,
			want: `graph {
	subgraph {
		A
		// c1
	}
}
`,
		},
		"CommentBeforeClosingBracket": {
			in: `graph {
	A [
		color=red
		// c1
	]
}`,
			want: `graph {
	A [
		color=red
		// c1
	]
}
`,
		},
		"CommentTrailingAttrEquals": {
			in: `graph {
	a= // c1
b
}`,
			want: `graph {
	a= // c1
	b
}
`,
		},
		"CommentTrailingEdgeOperator": {
			in: `digraph {
A -> // c1
B
}`,
			want: `digraph {
	A -> // c1
	B
}
`,
		},
		"CommentLeadingEdgeOperator": {
			in: `digraph {
A
// c1
-> B
}`,
			want: `digraph {
	A
	// c1
	-> B
}
`,
		},
		// File-level comments (between graphs)
		"CommentFile": {
			in: `// c1

// c2
graph {}
// c3
graph {}
// c4`,
			want: `// c1
// c2
graph {
}
// c3
graph {
}
// c4
`,
		},
		"CommentTrailingFirstGraph": {
			in: `graph {} // c1
graph {}`,
			want: `graph {
} // c1
graph {
}
`,
		},
		"CommentSingleHash": {
			in: `#!/usr/local/bin/dot
# comment
#
digraph G {}`,
			want: `#!/usr/local/bin/dot
# comment
#
digraph G {
}
`,
		},
		// Port comments
		"CommentTrailingPortColon": {
			in: `graph {
A: // c1
port
}`,
			want: `graph {
	A: // c1
	port
}
`,
		},
		"CommentTrailingPortName": {
			in: `graph {
A:port // c1
:n
}`,
			want: `graph {
	A:port // c1
	:n
}
`,
		},
		// Comment on its own line before second ':' stays inside Port.
		"CommentLeadingPortCompassColon": {
			in: `graph {
A:port
// c1
:n
}`,
			want: `graph {
	A:port
	// c1
	:n
}
`,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			var gotFirst bytes.Buffer
			p := printer.New([]byte(test.in), &gotFirst, layout.Default)
			err := p.Print()
			require.NoErrorf(t, err, "Print(%q)", test.in)

			if gotFirst.String() != test.want {
				t.Fatalf("\n\nin:\n%s\n\ngot:\n%s\n\n\nwant:\n%s\n", test.in, gotFirst.String(), test.want)
			}

			t.Logf("print again with the previous output as the input to ensure printing is idempotent")

			var gotSecond bytes.Buffer
			p = printer.New(gotFirst.Bytes(), &gotSecond, layout.Default)
			err = p.Print()
			require.NoErrorf(t, err, "Print(%q)", gotFirst.String())

			if gotSecond.String() != gotFirst.String() {
				t.Errorf("\n\nin:\n%s\n\ngot:\n%s\n\n\nwant:\n%s\n", gotFirst.String(), gotSecond.String(), gotFirst.String())
			}
		})
	}
}

func TestPrintErrorReturnsError(t *testing.T) {
	input := "graph { a = }"

	var output bytes.Buffer
	p := printer.New([]byte(input), &output, layout.Default)

	err := p.Print()

	require.NotNilf(t, err, "Print(%q) should return an error when parsing fails", input)

	// Print() should not write anything to the writer when parsing fails. The implementation
	// returns early on parse error, ensuring the output writer remains empty.
	got := output.String()
	if got != "" {
		t.Errorf("Print() wrote to output on parse error, got: %q, want empty string", got)
	}
}
