package printer_test

import (
	"bytes"
	"testing"

	"github.com/teleivo/assertive/assert"
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
		// Comment tests: comments are preserved as-is with only indentation adjusted.
		// Content is never modified: no line wrapping, no whitespace normalization.
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
		"CommentBlockFile": {
			in: `/* c1 */

/* c2 */
graph {}
/* c3 */
graph {}
/* c4 */`,
			want: `/* c1 */
/* c2 */
graph {
}
/* c3 */
graph {
}
/* c4 */
`,
		},
		"CommentBlockLeadingFile": {
			in: `/* c1 */

strict graph {}`,
			want: `/* c1 */
strict graph {
}
`,
		},
		"CommentBlockInlineBeforeGraphKeyword": {
			in:   `/* c1 */ graph {}`,
			want: `/* c1 */ graph {
}
`,
		},
		"CommentBlockMultiLineBeforeGraphKeyword": {
			in: `/* line1
   line2 */ graph {}`,
			want: `/* line1
line2 */ graph {
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
		"CommentTrailingFirstGraph": {
			in: `graph {} // c1
graph {}`,
			want: `graph {
} // c1
graph {
}
`,
		},
		"CommentBlockTrailingFirstGraph": {
			in: `graph {} /* c1 */
graph {}`,
			want: `graph {
} /* c1 */
graph {
}
`,
		},
		"CommentTrailingStrict": {
			in: `strict // c1
graph {}`,
			want: `strict // c1
graph {
}
`,
		},
		"CommentBlockTrailingStrict": {
			in: `strict /* c1 */ graph {}`,
			want: `strict /* c1 */ graph {
}
`,
		},
		"CommentTrailingGraphKeyword": {
			in: `graph // c1
{}`,
			want: `graph // c1
{
}
`,
		},
		"CommentBlockTrailingGraphKeyword": {
			in: `graph /* c1 */ {}`,
			want: `graph /* c1 */ {
}
`,
		},
		"CommentTrailingGraphID": {
			in: `graph G // c1
{}`,
			want: `graph G // c1
{
}
`,
		},
		"CommentBlockTrailingGraphID": {
			in: `graph G /* c1 */ {}`,
			want: `graph G /* c1 */ {
}
`,
		},
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
		"CommentBlockBeforeClosingBrace": {
			in: `graph {
	A
	/* c1 */
}`,
			want: `graph {
	A
	/* c1 */
}
`,
		},
		"CommentBeforeStmt": {
			in: `graph {
	// c1
A
}`,
			want: `graph {
	// c1
	A
}
`,
		},
		"CommentBlockBeforeStmt": {
			in: `graph {
	/* c1 */
A
}`,
			want: `graph {
	/* c1 */
	A
}
`,
		},
		"CommentTrailingAttrName": {
			in: `graph {
color    // c1
= red
}`,
			want: `graph {
	color // c1
	=red
}
`,
		},
		"CommentBlockTrailingAttrName": {
			in: `graph {
color    /* c1 */
= red
}`,
			want: `graph {
	color /* c1 */ =red
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
		"CommentBlockTrailingAttrEquals": {
			in: `graph {
	a= /* c1 */
b
}`,
			want: `graph {
	a= /* c1 */ b
}
`,
		},
		"CommentTrailingAttrValue": {
			in: `graph {
color = red    // c1
}`,
			want: `graph {
	color=red // c1
}
`,
		},
		"CommentBlockTrailingAttrValue": {
			in: `graph {
color = red    /* c1 */
}`,
			want: `graph {
	color=red /* c1 */
}
`,
		},
		"CommentTrailingAttrStmtTarget": {
			in: `graph {
node    // c1
[color=red]
}`,
			want: `graph {
	node // c1
	[color=red]
}
`,
		},
		"CommentBlockTrailingAttrStmtTarget": {
			in: `graph {
node    /* c1 */
[color=red]
}`,
			want: `graph {
	node /* c1 */ [color=red]
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
		"CommentBlockBeforeClosingBracket": {
			in: `graph {
	A [
		color=red
		/* c1 */
	]
}`,
			want: `graph {
	A [
		color=red
		/* c1 */
	]
}
`,
		},
		"CommentBetweenAttrs": {
			in: `graph {
	A [a=b // c1
c=d]
}`,
			want: `graph {
	A [
		a=b // c1
		c=d
	]
}
`,
		},
		"CommentBlockBetweenAttrs": {
			in: `graph {
	A [a=b /* c1 */ c=d]
}`,
			want: `graph {
	A [a=b /* c1 */,c=d]
}
`,
		},
		"CommentTrailingNodeID": {
			in: `graph {
A    // c1
[color=red]
}`,
			want: `graph {
	A // c1
	[color=red]
}
`,
		},
		"CommentBlockTrailingNodeID": {
			in: `graph {
A    /* c1 */
[color=red]
}`,
			want: `graph {
	A /* c1 */ [color=red]
}
`,
		},
		"CommentTrailingStmtAfterSemicolon": {
			in: `graph {
A; // c1
B
}`,
			want: `graph {
	A // c1
	B
}
`,
		},
		"CommentBlockTrailingStmtAfterSemicolon": {
			in: `graph {
A; /* c1 */
B
}`,
			want: `graph {
	A /* c1 */
	B
}
`,
		},
		"CommentTrailingID": {
			in: `graph {
A    //   c1    c1    c1
}`,
			want: `graph {
	A //   c1    c1    c1
}
`,
		},
		"CommentBlockTrailingID": {
			in: `graph {
A    /* c1    c1    c1 */
}`,
			want: `graph {
	A /* c1    c1    c1 */
}
`,
		},
		"CommentBlockMultiLineTrailingID": {
			in: `graph {
A /* line1
line2
line3 */
}`,
			want: `graph {
	A /* line1
	line2
	line3 */
}
`,
		},
		"CommentBlockMultiLineTrailingIDEmptyLine": {
			in: `graph {
A /* line1

line3 */
}`,
			want: "graph {\n\tA /* line1\n\t\n\tline3 */\n}\n",
		},
		"CommentBlockMultiLineTrailingNodeIDFits": {
			in: `graph {
A /* line1
line2 */ [color=red]
}`,
			want: `graph {
	A /* line1
	line2 */ [color=red]
}
`,
		},
		"CommentBlockMultiLineTrailingNodeIDForcesBreak": {
			in: `graph {
A /* line1
this is a long line that will push the attr list past the 80 column limit */ [color=red]
}`,
			want: `graph {
	A /* line1
	this is a long line that will push the attr list past the 80 column limit */ [
		color=red
	]
}
`,
		},
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
		"CommentBlockTrailingPortColon": {
			in: `graph {
A: /* c1 */
port
}`,
			want: `graph {
	A: /* c1 */ port
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
		"CommentBlockTrailingPortName": {
			in: `graph {
A:port /* c1 */
:n
}`,
			want: `graph {
	A:port /* c1 */ :n
}
`,
		},
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
		"CommentBlockLeadingPortCompassColon": {
			in: `graph {
A:port/* c1 */:n
}`,
			want: `graph {
	A:port /* c1 */ :n
}
`,
		},
		"CommentTrailingPortWithCompassPoint": {
			in: `graph {
A:port:sw    // c1
}`,
			want: `graph {
	A:port:sw // c1
}
`,
		},
		"CommentBlockTrailingPortWithCompassPoint": {
			in: `graph {
A:port:sw    /* c1 */
}`,
			want: `graph {
	A:port:sw /* c1 */
}
`,
		},
		"CommentTrailingCompassPoint": {
			in: `graph {
A:n    // c1
}`,
			want: `graph {
	A:n // c1
}
`,
		},
		"CommentBlockTrailingCompassPoint": {
			in: `graph {
A:n    /* c1 */
}`,
			want: `graph {
	A:n /* c1 */
}
`,
		},
		"CommentBlockMultiLineTrailingPortName": {
			in: `graph {
A:port /* line1
line2 */
}`,
			want: `graph {
	A:port /* line1
	line2 */
}
`,
		},
		"CommentBlockMultiLineTrailingCompassPoint": {
			in: `graph {
A:n /* line1
line2 */
}`,
			want: `graph {
	A:n /* line1
	line2 */
}
`,
		},
		"CommentTrailingEdgeStmt": {
			in: `digraph {
A -> B    // c1
[color=red]
}`,
			want: `digraph {
	A -> B // c1
	[color=red]
}
`,
		},
		"CommentBlockTrailingEdgeStmt": {
			in: `digraph {
A -> B    /* c1 */
[color=red]
}`,
			want: `digraph {
	A -> B /* c1 */ [color=red]
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
		"CommentBlockTrailingEdgeOperator": {
			in: `digraph {
A -> /* c1 */
B
}`,
			want: `digraph {
	A -> /* c1 */ B
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
		"CommentBlockLeadingEdgeOperator": {
			in: `digraph {
A
/* c1 */
-> B
}`,
			want: `digraph {
	A
	/* c1 */ -> B
}
`,
		},
		"CommentBlockMultiLineTrailingEdgeOperator": {
			in: `digraph {
A -> /* line1
line2 */ B
}`,
			want: `digraph {
	A -> /* line1
	line2 */ B
}
`,
		},
		"CommentBlockMultiLineLeadingEdgeOperator": {
			in: `digraph {
A /* line1
line2 */
-> B
}`,
			want: `digraph {
	A /* line1
	line2 */ -> B
}
`,
		},
		"CommentTrailingSubgraphKeyword": {
			in: `graph {
subgraph    // c1
{
A
}
}`,
			want: `graph {
	subgraph // c1
	{
		A
	}
}
`,
		},
		"CommentBlockTrailingSubgraphKeyword": {
			in: `graph {
subgraph    /* c1 */
{
A
}
}`,
			want: `graph {
	subgraph /* c1 */ {
		A
	}
}
`,
		},
		"CommentTrailingSubgraphID": {
			in: `graph {
subgraph S // c1
{
A
}
}`,
			want: `graph {
	subgraph S // c1
	{
		A
	}
}
`,
		},
		"CommentBlockTrailingSubgraphID": {
			in: `graph {
subgraph S /* c1 */ {
A
}
}`,
			want: `graph {
	subgraph S /* c1 */ {
		A
	}
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
		"CommentBlockBeforeClosingBraceInSubgraph": {
			in: `graph {
	subgraph {
		A
		/* c1 */
	}
}`,
			want: `graph {
	subgraph {
		A
		/* c1 */
	}
}
`,
		},
		"CommentBlockMultiLineTrailingSubgraphKeyword": {
			in: `graph {
subgraph /* line1
line2 */ {
A
}
}`,
			want: `graph {
	subgraph /* line1
	line2 */ {
		A
	}
}
`,
		},
		"CommentBlockMultiLineBeforeClosingBrace": {
			in: `graph {
A
/* line1
line2 */
}`,
			want: `graph {
	A
	/* line1
	line2 */
}
`,
		},
		// Long comments (>80 cols) are preserved as-is and force attr_list to break
		"CommentLongLineInAttrList": {
			in: `graph {
	A [color=red // this is a very long comment that exceeds the 80 column limit for sure
]
}`,
			want: `graph {
	A [
		color=red // this is a very long comment that exceeds the 80 column limit for sure
	]
}
`,
		},
		"CommentLongBlockInAttrList": {
			in: `graph {
	A [color=red /* this is a very long block comment that exceeds the 80 column limit */ shape=box]
}`,
			want: `graph {
	A [
		color=red /* this is a very long block comment that exceeds the 80 column limit */
		shape=box
	]
}
`,
		},
		"CommentLongLineTrailingNodeID": {
			in: `graph {
A // this is a very long comment that exceeds eighty columns for sure sure sure
}`,
			want: `graph {
	A // this is a very long comment that exceeds eighty columns for sure sure sure
}
`,
		},
		"CommentBlockMultiLineTrailingAttrStmtTarget": {
			in: `graph {
node /* line1
line2 */ [color=red]
}`,
			want: `graph {
	node /* line1
	line2 */ [color=red]
}
`,
		},
		"CommentBlockBetweenAttrListBrackets": {
			in: `graph {
A [a=b] /* c1 */ [c=d]
}`,
			want: `graph {
	A [a=b] /* c1 */ [c=d]
}
`,
		},
		"CommentBlockBetweenAttrListBracketsForcesBreak": {
			in: `graph {
A [label="This is a long label value"] /* comment between brackets */ [color=red,shape=box]
}`,
			want: `graph {
	A [label="This is a long label value"] /* comment between brackets */ [
		color=red
		shape=box
	]
}
`,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			var gotFirst bytes.Buffer
			p := printer.New([]byte(test.in), &gotFirst, layout.Default)
			err := p.Print()
			require.NoError(t, err, "Print(%q)", test.in)

			require.NoDiff(t, gotFirst.String(), test.want)

			t.Logf("print again with the previous output as the input to ensure printing is idempotent")

			var gotSecond bytes.Buffer
			p = printer.New(gotFirst.Bytes(), &gotSecond, layout.Default)
			err = p.Print()
			require.NoError(t, err, "Print(%q)", gotFirst.String())

			assert.NoDiff(t, gotSecond.String(), gotFirst.String())
		})
	}
}

func TestPrintErrorReturnsError(t *testing.T) {
	input := "graph { a = }"

	var output bytes.Buffer
	p := printer.New([]byte(input), &output, layout.Default)

	err := p.Print()

	require.NotNil(t, err, "Print(%q) should return an error when parsing fails", input)

	// Print() should not write anything to the writer when parsing fails. The implementation
	// returns early on parse error, ensuring the output writer remains empty.
	got := output.String()
	if got != "" {
		t.Errorf("Print() wrote to output on parse error, got: %q, want empty string", got)
	}
}
