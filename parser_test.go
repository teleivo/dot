package dot_test

import (
	"strings"
	"testing"

	"github.com/teleivo/assertive/assert"
	"github.com/teleivo/dot"
)

func TestParser(t *testing.T) {
	tests := map[string]struct {
		in         string
		want       string
		wantScheme string // optional - verify positions via Render(Scheme)
		wantErrors []string
	}{
		// Incremental graph construction - simulating user typing
		"Empty": {
			in: "",
			want: `File
`,
			wantScheme: `(File)
`,
		},
		"Strict": {
			in: "strict",
			want: `File
	Graph
		'strict'
`,
			wantErrors: []string{
				"1:7: expected digraph or graph",
			},
		},
		"StrictGraph": {
			in: "strict graph",
			want: `File
	Graph
		'strict'
		'graph'
`,
			wantErrors: []string{
				"1:13: expected {",
			},
		},
		"StrictGraphID": {
			in: "strict graph fruits",
			want: `File
	Graph
		'strict'
		'graph'
		ID
			'fruits'
`,
			wantErrors: []string{
				"1:20: expected {",
			},
		},
		"StrictGraphIDLeftBrace": {
			in: "strict graph fruits {",
			want: `File
	Graph
		'strict'
		'graph'
		ID
			'fruits'
		'{'
		StmtList
`,
			wantErrors: []string{
				"1:22: expected }",
			},
		},
		"StrictGraphIDEmpty": {
			in: `strict graph fruits {
}`,
			want: `File
	Graph
		'strict'
		'graph'
		ID
			'fruits'
		'{'
		StmtList
		'}'
`,
		},
		"StrictGraphIDWithID": {
			in: `strict graph fruits {
	rank
}`,
			want: `File
	Graph
		'strict'
		'graph'
		ID
			'fruits'
		'{'
		StmtList
			NodeStmt
				NodeID
					ID
						'rank'
		'}'
`,
		},
		"StrictGraphIDWithIDEquals": {
			in: `strict graph fruits {
	rank =
}`,
			want: `File
	Graph
		'strict'
		'graph'
		ID
			'fruits'
		'{'
		StmtList
			Attribute
				ID
					'rank'
				'='
		'}'
`,
			wantErrors: []string{
				"3:1: expected attribute value",
			},
		},
		"StrictGraphIDWithAttribute": {
			in: `strict graph fruits {
	rank = same
}`,
			want: `File
	Graph
		'strict'
		'graph'
		ID
			'fruits'
		'{'
		StmtList
			Attribute
				ID
					'rank'
				'='
				ID
					'same'
		'}'
`,
		},
		"ScannerErrorInvalidCharacter": {
			in: `digraph { a@b }`,
			want: `File
	Graph
		'digraph'
		'{'
		StmtList
			ErrorTree
				'ERROR'
		'}'
`,
			wantErrors: []string{
				"1:11: invalid character '@': unquoted IDs can only contain letters, digits, and underscores",
			},
		},
		"StrictGraphIDWithEdgeIncompleteOperator": {
			in: `strict graph fruits {
	A -
}`,
			want: `File
	Graph
		'strict'
		'graph'
		ID
			'fruits'
		'{'
		StmtList
			NodeStmt
				NodeID
					ID
						'A'
			ErrorTree
				'ERROR'
		'}'
`,
			wantErrors: []string{
				"2:4: invalid character U+000A: ambiguous: quote for ID, or add digit for number like '-.1' or '-0.'",
			},
		},
		"StrictGraphIDWithEdgeCompleteOperatorMissingRHS": {
			in: `strict graph fruits {
	A --
}`,
			want: `File
	Graph
		'strict'
		'graph'
		ID
			'fruits'
		'{'
		StmtList
			EdgeStmt
				NodeID
					ID
						'A'
				'--'
		'}'
`,
			wantErrors: []string{
				"3:1: expected node or subgraph as edge operand",
			},
		},
		"StrictGraphIDWithEdge": {
			in: `strict graph fruits {
	A -- B
}`,
			want: `File
	Graph
		'strict'
		'graph'
		ID
			'fruits'
		'{'
		StmtList
			EdgeStmt
				NodeID
					ID
						'A'
				'--'
				NodeID
					ID
						'B'
		'}'
`,
		},
		"StrictGraphIDWithEdgeChainIncomplete": {
			in: `strict graph fruits {
	A -- B --
}`,
			want: `File
	Graph
		'strict'
		'graph'
		ID
			'fruits'
		'{'
		StmtList
			EdgeStmt
				NodeID
					ID
						'A'
				'--'
				NodeID
					ID
						'B'
				'--'
		'}'
`,
			wantErrors: []string{
				"3:1: expected node or subgraph as edge operand",
			},
		},
		"StrictGraphIDWithEdgeChain": {
			in: `strict graph fruits {
	A -- B -- C
}`,
			want: `File
	Graph
		'strict'
		'graph'
		ID
			'fruits'
		'{'
		StmtList
			EdgeStmt
				NodeID
					ID
						'A'
				'--'
				NodeID
					ID
						'B'
				'--'
				NodeID
					ID
						'C'
		'}'
`,
		},
		"GraphWithNodeAttrStmt": {
			in: `graph {
	A -- B -- C
	node
}`,
			want: `File
	Graph
		'graph'
		'{'
		StmtList
			EdgeStmt
				NodeID
					ID
						'A'
				'--'
				NodeID
					ID
						'B'
				'--'
				NodeID
					ID
						'C'
			AttrStmt
				'node'
		'}'
`,
			wantErrors: []string{
				"4:1: expected [ to start attribute list",
			},
		},
		"GraphWithNodeAttrStmtLeftBracket": {
			in: `graph {
	A -- B -- C
	node [
}`,
			want: `File
	Graph
		'graph'
		'{'
		StmtList
			EdgeStmt
				NodeID
					ID
						'A'
				'--'
				NodeID
					ID
						'B'
				'--'
				NodeID
					ID
						'C'
			AttrStmt
				'node'
				AttrList
					'['
		'}'
`,
			wantErrors: []string{
				"4:1: expected ] to close attribute list",
			},
		},
		"GraphWithNodeAttrStmtEmpty": {
			in: `graph {
	A -- B -- C
	node []
}`,
			want: `File
	Graph
		'graph'
		'{'
		StmtList
			EdgeStmt
				NodeID
					ID
						'A'
				'--'
				NodeID
					ID
						'B'
				'--'
				NodeID
					ID
						'C'
			AttrStmt
				'node'
				AttrList
					'['
					']'
		'}'
`,
		},
		"GraphWithEdgeAttrStmt": {
			in: `graph {
	A -- B -- C
	edge []
}`,
			want: `File
	Graph
		'graph'
		'{'
		StmtList
			EdgeStmt
				NodeID
					ID
						'A'
				'--'
				NodeID
					ID
						'B'
				'--'
				NodeID
					ID
						'C'
			AttrStmt
				'edge'
				AttrList
					'['
					']'
		'}'
`,
		},
		"GraphWithGraphAttrStmt": {
			in: `graph {
	A -- B -- C
	graph []
}`,
			want: `File
	Graph
		'graph'
		'{'
		StmtList
			EdgeStmt
				NodeID
					ID
						'A'
				'--'
				NodeID
					ID
						'B'
				'--'
				NodeID
					ID
						'C'
			AttrStmt
				'graph'
				AttrList
					'['
					']'
		'}'
`,
		},
		"GraphWithNodeAttrStmtWithSemicolon": {
			in: `graph {
	A -- B -- C
	node [];
}`,
			want: `File
	Graph
		'graph'
		'{'
		StmtList
			EdgeStmt
				NodeID
					ID
						'A'
				'--'
				NodeID
					ID
						'B'
				'--'
				NodeID
					ID
						'C'
			AttrStmt
				'node'
				AttrList
					'['
					']'
			';'
		'}'
`,
		},
		"GraphWithEdgeAttrStmtNoSemicolon": {
			in: `graph {
	A -- B -- C
	edge []
	graph []
}`,
			want: `File
	Graph
		'graph'
		'{'
		StmtList
			EdgeStmt
				NodeID
					ID
						'A'
				'--'
				NodeID
					ID
						'B'
				'--'
				NodeID
					ID
						'C'
			AttrStmt
				'edge'
				AttrList
					'['
					']'
			AttrStmt
				'graph'
				AttrList
					'['
					']'
		'}'
`,
		},
		"GraphWithAllThreeAttrStmtsSeparatedBySemicolon": {
			in: `graph {
	node []; edge []; graph []
}`,
			want: `File
	Graph
		'graph'
		'{'
		StmtList
			AttrStmt
				'node'
				AttrList
					'['
					']'
			';'
			AttrStmt
				'edge'
				AttrList
					'['
					']'
			';'
			AttrStmt
				'graph'
				AttrList
					'['
					']'
		'}'
`,
		},
		"GraphWithAttrStmtIncompleteAttribute": {
			in: `graph {
	node [color]
}`,
			want: `File
	Graph
		'graph'
		'{'
		StmtList
			AttrStmt
				'node'
				AttrList
					'['
					AList
						Attribute
							ID
								'color'
					']'
		'}'
`,
			wantErrors: []string{
				"2:13: expected =",
			},
		},
		"GraphWithAttrStmtMissingValue": {
			in: `graph {
	node [color=]
}`,
			want: `File
	Graph
		'graph'
		'{'
		StmtList
			AttrStmt
				'node'
				AttrList
					'['
					AList
						Attribute
							ID
								'color'
							'='
					']'
		'}'
`,
			wantErrors: []string{
				"2:14: expected attribute value",
			},
		},
		"GraphWithAttrStmtValidAndIncomplete": {
			in: `graph {
	node [color=blue font]
}`,
			want: `File
	Graph
		'graph'
		'{'
		StmtList
			AttrStmt
				'node'
				AttrList
					'['
					AList
						Attribute
							ID
								'color'
							'='
							ID
								'blue'
						Attribute
							ID
								'font'
					']'
		'}'
`,
			wantErrors: []string{
				"2:23: expected =",
			},
		},
		"GraphWithAttrStmtRecoveryOnEdgeKeyword": {
			in: `graph {
	node [blue edge [a=b]]
}`,
			want: `File
	Graph
		'graph'
		'{'
		StmtList
			AttrStmt
				'node'
				AttrList
					'['
					AList
						Attribute
							ID
								'blue'
			AttrStmt
				'edge'
				AttrList
					'['
					AList
						Attribute
							ID
								'a'
							'='
							ID
								'b'
					']'
			ErrorTree
				']'
		'}'
`,
			wantErrors: []string{
				"2:13: expected =",
				"2:13: expected ] to close attribute list",
				"2:23: ']' cannot start a statement",
			},
		},
		"GraphWithAttrStmtRecoveryOnNodeKeyword": {
			in: `graph {
	edge [color=blue node [shape=box]
}`,
			want: `File
	Graph
		'graph'
		'{'
		StmtList
			AttrStmt
				'edge'
				AttrList
					'['
					AList
						Attribute
							ID
								'color'
							'='
							ID
								'blue'
			AttrStmt
				'node'
				AttrList
					'['
					AList
						Attribute
							ID
								'shape'
							'='
							ID
								'box'
					']'
		'}'
`,
			wantErrors: []string{
				"2:19: expected ] to close attribute list",
			},
		},
		"GraphWithAttrStmtRecoveryOnLeftBracket": {
			in: `graph {
	node [color [ font=arial]
}`,
			want: `File
	Graph
		'graph'
		'{'
		StmtList
			AttrStmt
				'node'
				AttrList
					'['
					AList
						Attribute
							ID
								'color'
					'['
					AList
						Attribute
							ID
								'font'
							'='
							ID
								'arial'
					']'
		'}'
`,
			wantErrors: []string{
				"2:14: expected =",
				"2:14: expected ] to close attribute list",
			},
		},
		"GraphWithAttrStmtComplexRecovery": {
			in: `graph {
	node [blue font edge [a=b]
}`,
			want: `File
	Graph
		'graph'
		'{'
		StmtList
			AttrStmt
				'node'
				AttrList
					'['
					AList
						Attribute
							ID
								'blue'
							ID
								'font'
			AttrStmt
				'edge'
				AttrList
					'['
					AList
						Attribute
							ID
								'a'
							'='
							ID
								'b'
					']'
		'}'
`,
			wantErrors: []string{
				"2:13: expected =",
				"2:18: expected ] to close attribute list",
			},
		},
		"GraphWithAttrStmtMissingClosingBracketWithSubsequentAttrList": {
			in: `graph {
	node [a=b[c=d]
}`,
			want: `File
	Graph
		'graph'
		'{'
		StmtList
			AttrStmt
				'node'
				AttrList
					'['
					AList
						Attribute
							ID
								'a'
							'='
							ID
								'b'
					'['
					AList
						Attribute
							ID
								'c'
							'='
							ID
								'd'
					']'
		'}'
`,
			wantErrors: []string{
				"2:11: expected ] to close attribute list",
			},
		},
		"GraphIDWithEdgeGarbageBetween": {
			in: `graph fruits {
	A -- = B
}`,
			want: `File
	Graph
		'graph'
		ID
			'fruits'
		'{'
		StmtList
			EdgeStmt
				NodeID
					ID
						'A'
				'--'
				ErrorTree
					'='
			NodeStmt
				NodeID
					ID
						'B'
		'}'
`,
			wantErrors: []string{
				"2:7: '=' is not a valid edge operand",
			},
		},
		"GraphIDWithEdgeSemicolon": {
			in: `graph fruits {
	A -- ;
}`,
			want: `File
	Graph
		'graph'
		ID
			'fruits'
		'{'
		StmtList
			EdgeStmt
				NodeID
					ID
						'A'
				'--'
			';'
		'}'
`,
			wantErrors: []string{
				"2:7: expected node or subgraph as edge operand",
			},
		},
		"GraphIDWithEdgeComma": {
			in: `graph fruits {
	A -- ,
}`,
			want: `File
	Graph
		'graph'
		ID
			'fruits'
		'{'
		StmtList
			EdgeStmt
				NodeID
					ID
						'A'
				'--'
				ErrorTree
					','
		'}'
`,
			wantErrors: []string{
				"2:7: ',' is not a valid edge operand",
			},
		},
		"StrictGraphIDWithAttributeTrailingSemicolon": {
			in: `strict graph fruits {
	rank = same;
}`,
			want: `File
	Graph
		'strict'
		'graph'
		ID
			'fruits'
		'{'
		StmtList
			Attribute
				ID
					'rank'
				'='
				ID
					'same'
			';'
		'}'
`,
		},
		"StrictGraphIDWithTwoAttributesWithSemicolon": {
			in: `strict graph fruits {
	rank = same; ; color = red
}`,
			want: `File
	Graph
		'strict'
		'graph'
		ID
			'fruits'
		'{'
		StmtList
			Attribute
				ID
					'rank'
				'='
				ID
					'same'
			';'
			';'
			Attribute
				ID
					'color'
				'='
				ID
					'red'
		'}'
`,
		},
		"StrictGraphIDWithTwoAttributesNoSemicolon": {
			in: `strict graph fruits {
	rank = same
	color = red
}`,
			want: `File
	Graph
		'strict'
		'graph'
		ID
			'fruits'
		'{'
		StmtList
			Attribute
				ID
					'rank'
				'='
				ID
					'same'
			Attribute
				ID
					'color'
				'='
				ID
					'red'
		'}'
`,
		},
		"AttributeRecoveryAtDigraph": {
			in: `graph { A = digraph { C = D }`,
			want: `File
	Graph
		'graph'
		'{'
		StmtList
			Attribute
				ID
					'A'
				'='
	Graph
		'digraph'
		'{'
		StmtList
			Attribute
				ID
					'C'
				'='
				ID
					'D'
		'}'
`,
			wantErrors: []string{
				"1:13: expected attribute value",
				"1:13: expected }",
			},
		},
		"EmptyDirectedGraph": {
			in: "digraph {}",
			want: `File
	Graph
		'digraph'
		'{'
		StmtList
		'}'
`,
			wantScheme: `(File (@ 1 1 1 10)
	(Graph (@ 1 1 1 10)
		('digraph' (@ 1 1 1 7))
		('{' (@ 1 9 1 9))
		(StmtList)
		('}' (@ 1 10 1 10))))
`,
		},
		"TypoInStrict": {
			in: "stict graph {}",
			want: `File
	ErrorTree
		'stict'
	Graph
		'graph'
		'{'
		StmtList
		'}'
`,
			wantErrors: []string{
				`1:1: unexpected token ID 'stict', expected digraph, graph or strict`,
			},
		},
		"TypoInDigraph": {
			in: "disgraph {}",
			want: `File
	ErrorTree
		'disgraph'
	ErrorTree
		'{'
	ErrorTree
		'}'
`,
			wantErrors: []string{
				`1:1: unexpected token ID 'disgraph', expected digraph, graph or strict`,
				`1:10: unexpected token '{', expected digraph, graph or strict`,
				`1:11: unexpected token '}', expected digraph, graph or strict`,
			},
		},
		"WrongKeywordBeforeGraph": {
			in: "public graph {}",
			want: `File
	ErrorTree
		'public'
	Graph
		'graph'
		'{'
		StmtList
		'}'
`,
			wantErrors: []string{
				`1:1: unexpected token ID 'public', expected digraph, graph or strict`,
			},
		},
		"MultipleWrongKeywordsBeforeGraph": {
			in: "public def graph {}",
			want: `File
	ErrorTree
		'public'
	ErrorTree
		'def'
	Graph
		'graph'
		'{'
		StmtList
		'}'
`,
			wantScheme: `(File (@ 1 1 1 19)
	(ErrorTree (@ 1 1 1 6)
		('public' (@ 1 1 1 6)))
	(ErrorTree (@ 1 8 1 10)
		('def' (@ 1 8 1 10)))
	(Graph (@ 1 12 1 19)
		('graph' (@ 1 12 1 16))
		('{' (@ 1 18 1 18))
		(StmtList)
		('}' (@ 1 19 1 19))))
`,
			wantErrors: []string{
				`1:1: unexpected token ID 'public', expected digraph, graph or strict`,
				`1:8: unexpected token ID 'def', expected digraph, graph or strict`,
			},
		},
		"MultipleGraphsInFile": {
			in: `graph G1 {}
digraph G2 {}
strict graph G3 {}`,
			want: `File
	Graph
		'graph'
		ID
			'G1'
		'{'
		StmtList
		'}'
	Graph
		'digraph'
		ID
			'G2'
		'{'
		StmtList
		'}'
	Graph
		'strict'
		'graph'
		ID
			'G3'
		'{'
		StmtList
		'}'
`,
		},
		"StrictWithoutGraphKeyword": {
			in: "strict id {}",
			want: `File
	Graph
		'strict'
		ErrorTree
			'id'
		'{'
		StmtList
		'}'
`,
			wantErrors: []string{
				`1:8: expected digraph or graph`,
				`1:8: unexpected token ID 'id', expected digraph or graph`,
			},
		},
		"StrictWithoutGraphKeywordNoBrace": {
			in: "strict {}",
			want: `File
	Graph
		'strict'
		'{'
		StmtList
		'}'
`,
			wantErrors: []string{
				`1:8: expected digraph or graph`,
			},
		},
		"StrictWithTypoInGraph": {
			in: "strict gaph id {}",
			want: `File
	Graph
		'strict'
		ErrorTree
			'gaph'
		ErrorTree
			'id'
		'{'
		StmtList
		'}'
`,
			wantErrors: []string{
				`1:8: expected digraph or graph`,
				`1:8: unexpected token ID 'gaph', expected digraph or graph`,
				`1:13: unexpected token ID 'id', expected digraph or graph`,
			},
		},
		"StrictWithMultipleErrorsBeforeBrace": {
			in: "strict foo \"weee\" {",
			want: `File
	Graph
		'strict'
		ErrorTree
			'foo'
		ErrorTree
			'"weee"'
		'{'
		StmtList
`,
			wantErrors: []string{
				`1:8: expected digraph or graph`,
				`1:8: unexpected token ID 'foo', expected digraph or graph`,
				`1:12: unexpected token ID '"weee"', expected digraph or graph`,
				`1:20: expected }`,
			},
		},
		"StrictWithRecoveryAtNextGraph": {
			in: `strict foo graph {}`,
			want: `File
	Graph
		'strict'
		ErrorTree
			'foo'
	Graph
		'graph'
		'{'
		StmtList
		'}'
`,
			wantErrors: []string{
				`1:8: expected digraph or graph`,
				`1:8: unexpected token ID 'foo', expected digraph or graph`,
			},
		},
		"GraphMissingBrace": {
			in: "graph id",
			want: `File
	Graph
		'graph'
		ID
			'id'
`,
			wantErrors: []string{
				`1:9: expected {`,
			},
		},
		"GraphAlone": {
			in: "graph",
			want: `File
	Graph
		'graph'
`,
			wantErrors: []string{
				`1:6: expected {`,
			},
		},
		"DuplicateStrict": {
			in: "strict strict graph {}",
			want: `File
	Graph
		'strict'
	Graph
		'strict'
		'graph'
		'{'
		StmtList
		'}'
`,
			wantErrors: []string{
				`1:8: expected digraph or graph`,
			},
		},
		"GraphIDWithGarbageBeforeBrace": {
			in: `graph G foo bar {
}`,
			want: `File
	Graph
		'graph'
		ID
			'G'
		ErrorTree
			'foo'
		ErrorTree
			'bar'
		'{'
		StmtList
		'}'
`,
			wantErrors: []string{
				`1:9: unexpected token ID 'foo'`,
				`1:13: unexpected token ID 'bar'`,
			},
		},
		"GraphWithSemicolonBeforeBrace": {
			in: `graph G ; {
}`,
			want: `File
	Graph
		'graph'
		ID
			'G'
		ErrorTree
			';'
		'{'
		StmtList
		'}'
`,
			wantErrors: []string{
				`1:9: unexpected token ';'`,
			},
		},
		"GraphMultipleIDs": {
			in: "graph id1 id2 id3 {}",
			want: `File
	Graph
		'graph'
		ID
			'id1'
		ErrorTree
			'id2'
		ErrorTree
			'id3'
		'{'
		StmtList
		'}'
`,
			wantErrors: []string{
				`1:11: unexpected token ID 'id2'`,
				`1:15: unexpected token ID 'id3'`,
			},
		},
		"GraphAsID": {
			in: "graph graph {}",
			want: `File
	Graph
		'graph'
	Graph
		'graph'
		'{'
		StmtList
		'}'
`,
			wantErrors: []string{
				`1:7: expected {`,
			},
		},
		"AttributeSingle": {
			in: "graph { rank = same; }",
			want: `File
	Graph
		'graph'
		'{'
		StmtList
			Attribute
				ID
					'rank'
				'='
				ID
					'same'
			';'
		'}'
`,
		},
		"QuotedAttributeValueSpanningMultipleLines": {
			in: `graph { 	label="Rainy days
				in summer"
}`,
			want: `File
	Graph
		'graph'
		'{'
		StmtList
			Attribute
				ID
					'label'
				'='
				ID
					'"Rainy days
				in summer"'
		'}'
`,
		},
		"QuotedAttributeValueSpanningMultipleLinesWithBackslashFollowedByNewline": {
			in: `graph { 	label="Rainy days\
				in summer"
}`,
			want: `File
	Graph
		'graph'
		'{'
		StmtList
			Attribute
				ID
					'label'
				'='
				ID
					'"Rainy days\
				in summer"'
		'}'
`,
		},
		"StmtListDisambiguation": {
			in: `graph {
	a=1
	b c
	d -- e
}`,
			want: `File
	Graph
		'graph'
		'{'
		StmtList
			Attribute
				ID
					'a'
				'='
				ID
					'1'
			NodeStmt
				NodeID
					ID
						'b'
			NodeStmt
				NodeID
					ID
						'c'
			EdgeStmt
				NodeID
					ID
						'd'
				'--'
				NodeID
					ID
						'e'
		'}'
`,
		},
		"NodeStmtSingle": {
			in: `graph { A }`,
			want: `File
	Graph
		'graph'
		'{'
		StmtList
			NodeStmt
				NodeID
					ID
						'A'
		'}'
`,
		},
		"NodeStmtMultiple": {
			in: `graph { A; B; C }`,
			want: `File
	Graph
		'graph'
		'{'
		StmtList
			NodeStmt
				NodeID
					ID
						'A'
			';'
			NodeStmt
				NodeID
					ID
						'B'
			';'
			NodeStmt
				NodeID
					ID
						'C'
		'}'
`,
		},
		"NodeStmtWithPort": {
			in: `graph { A:port1 }`,
			want: `File
	Graph
		'graph'
		'{'
		StmtList
			NodeStmt
				NodeID
					ID
						'A'
					Port
						':'
						ID
							'port1'
		'}'
`,
		},
		"NodeStmtWithPortAndCompassPoint": {
			in: `graph { A:port1:n }`,
			want: `File
	Graph
		'graph'
		'{'
		StmtList
			NodeStmt
				NodeID
					ID
						'A'
					Port
						':'
						ID
							'port1'
						':'
						CompassPoint
							'n'
		'}'
`,
		},
		"NodeStmtWithCompassPointOnly": {
			in: `graph { A:ne }`,
			want: `File
	Graph
		'graph'
		'{'
		StmtList
			NodeStmt
				NodeID
					ID
						'A'
					Port
						':'
						CompassPoint
							'ne'
		'}'
`,
		},
		"NodeStmtWithEmptyAttrList": {
			in: `graph { A [] }`,
			want: `File
	Graph
		'graph'
		'{'
		StmtList
			NodeStmt
				NodeID
					ID
						'A'
				AttrList
					'['
					']'
		'}'
`,
		},
		"NodeStmtWithSingleAttribute": {
			in: `graph { A [color=red] }`,
			want: `File
	Graph
		'graph'
		'{'
		StmtList
			NodeStmt
				NodeID
					ID
						'A'
				AttrList
					'['
					AList
						Attribute
							ID
								'color'
							'='
							ID
								'red'
					']'
		'}'
`,
		},
		"NodeStmtWithMultipleAttributes": {
			in: `graph { A [color=red, shape=box] }`,
			want: `File
	Graph
		'graph'
		'{'
		StmtList
			NodeStmt
				NodeID
					ID
						'A'
				AttrList
					'['
					AList
						Attribute
							ID
								'color'
							'='
							ID
								'red'
						','
						Attribute
							ID
								'shape'
							'='
							ID
								'box'
					']'
		'}'
`,
		},
		"NodeStmtWithMultipleAttrLists": {
			in: `graph { A [color=red][shape=box] }`,
			want: `File
	Graph
		'graph'
		'{'
		StmtList
			NodeStmt
				NodeID
					ID
						'A'
				AttrList
					'['
					AList
						Attribute
							ID
								'color'
							'='
							ID
								'red'
					']'
					'['
					AList
						Attribute
							ID
								'shape'
							'='
							ID
								'box'
					']'
		'}'
`,
		},
		"NodeIDIncompletePort": {
			in: `graph { A: }`,
			want: `File
	Graph
		'graph'
		'{'
		StmtList
			NodeStmt
				NodeID
					ID
						'A'
					Port
						':'
		'}'
`,
			wantErrors: []string{
				"1:12: expected ID for port",
			},
		},
		"NodeIDPortMissingCompassPoint": {
			in: `graph { A:port1: }`,
			want: `File
	Graph
		'graph'
		'{'
		StmtList
			NodeStmt
				NodeID
					ID
						'A'
					Port
						':'
						ID
							'port1'
						':'
		'}'
`,
			wantErrors: []string{
				"1:18: expected compass point (c, e, n, ne, nw, s, se, sw, w, or _)",
			},
		},
		"NodeIDPortNumeric": {
			in: `graph { A:123 }`,
			want: `File
	Graph
		'graph'
		'{'
		StmtList
			NodeStmt
				NodeID
					ID
						'A'
					Port
						':'
						ID
							'123'
		'}'
`,
		},
		"NodeIDPortNumericWithCompassPoint": {
			in: `graph { A:123:sw }`,
			want: `File
	Graph
		'graph'
		'{'
		StmtList
			NodeStmt
				NodeID
					ID
						'A'
					Port
						':'
						ID
							'123'
						':'
						CompassPoint
							'sw'
		'}'
`,
		},
		"NodeIDPortQuoted": {
			in: `graph { A:"my port" }`,
			want: `File
	Graph
		'graph'
		'{'
		StmtList
			NodeStmt
				NodeID
					ID
						'A'
					Port
						':'
						ID
							'"my port"'
		'}'
`,
		},
		"NodeIDPortQuotedWithCompassPoint": {
			in: `graph { A:"port name":nw }`,
			want: `File
	Graph
		'graph'
		'{'
		StmtList
			NodeStmt
				NodeID
					ID
						'A'
					Port
						':'
						ID
							'"port name"'
						':'
						CompassPoint
							'nw'
		'}'
`,
		},
		"NodeIDPortAllCompassPoints": {
			in: `graph { A:c B:w C:e D:s }`,
			want: `File
	Graph
		'graph'
		'{'
		StmtList
			NodeStmt
				NodeID
					ID
						'A'
					Port
						':'
						CompassPoint
							'c'
			NodeStmt
				NodeID
					ID
						'B'
					Port
						':'
						CompassPoint
							'w'
			NodeStmt
				NodeID
					ID
						'C'
					Port
						':'
						CompassPoint
							'e'
			NodeStmt
				NodeID
					ID
						'D'
					Port
						':'
						CompassPoint
							's'
		'}'
`,
		},
		"NodeIDPortTwoNonCompassIDs": {
			in: `graph { A:port1:port2 }`,
			want: `File
	Graph
		'graph'
		'{'
		StmtList
			NodeStmt
				NodeID
					ID
						'A'
					Port
						':'
						ID
							'port1'
						':'
						ID
							'port2'
		'}'
`,
			wantErrors: []string{
				"1:23: expected compass point (c, e, n, ne, nw, s, se, sw, w, or _)",
			},
		},
		"NodeIDPortTwoCompassPoints": {
			in: `graph { A:n:e }`,
			want: `File
	Graph
		'graph'
		'{'
		StmtList
			NodeStmt
				NodeID
					ID
						'A'
					Port
						':'
						ID
							'n'
						':'
						CompassPoint
							'e'
		'}'
`,
		},
		"NodeIDPortDoubleColon": {
			in: `graph { A:: }`,
			want: `File
	Graph
		'graph'
		'{'
		StmtList
			NodeStmt
				NodeID
					ID
						'A'
					Port
						':'
						':'
		'}'
`,
			wantErrors: []string{
				"1:11: expected ID for port",
				"1:13: expected compass point (c, e, n, ne, nw, s, se, sw, w, or _)",
			},
		},
		"NodeIDPortFollowedByAttrList": {
			in: `graph { A:[ ] }`,
			want: `File
	Graph
		'graph'
		'{'
		StmtList
			NodeStmt
				NodeID
					ID
						'A'
					Port
						':'
				AttrList
					'['
					']'
		'}'
`,
			wantErrors: []string{
				"1:11: expected ID for port",
			},
		},
		"NodeIDPortColonFollowedByAttrList": {
			in: `graph { A:port1:[ ] }`,
			want: `File
	Graph
		'graph'
		'{'
		StmtList
			NodeStmt
				NodeID
					ID
						'A'
					Port
						':'
						ID
							'port1'
						':'
				AttrList
					'['
					']'
		'}'
`,
			wantErrors: []string{
				"1:17: expected compass point (c, e, n, ne, nw, s, se, sw, w, or _)",
			},
		},
		"NodeIDPortFollowedByUnexpectedToken": {
			in: `graph { A:= }`,
			want: `File
	Graph
		'graph'
		'{'
		StmtList
			NodeStmt
				NodeID
					ID
						'A'
					Port
						':'
			ErrorTree
				'='
		'}'
`,
			wantErrors: []string{
				"1:11: expected ID for port",
				"1:11: '=' cannot start a statement",
			},
		},
		"NodeStmtIncompleteAttrList": {
			in: `graph { A [ }`,
			want: `File
	Graph
		'graph'
		'{'
		StmtList
			NodeStmt
				NodeID
					ID
						'A'
				AttrList
					'['
		'}'
`,
			wantErrors: []string{
				"1:13: expected ] to close attribute list",
			},
		},
		"NodeStmtAttrListMissingCloseBracket": {
			in: `graph { A [color=red }`,
			want: `File
	Graph
		'graph'
		'{'
		StmtList
			NodeStmt
				NodeID
					ID
						'A'
				AttrList
					'['
					AList
						Attribute
							ID
								'color'
							'='
							ID
								'red'
		'}'
`,
			wantErrors: []string{
				"1:22: expected ] to close attribute list",
			},
		},

		// Subgraph tests - incremental construction
		"SubgraphKeyword": {
			in: `graph { subgraph }`,
			want: `File
	Graph
		'graph'
		'{'
		StmtList
			Subgraph
				'subgraph'
		'}'
`,
			wantErrors: []string{
				"1:18: expected {",
			},
		},
		"SubgraphWithLeftBrace": {
			in: `graph { subgraph { }`,
			want: `File
	Graph
		'graph'
		'{'
		StmtList
			Subgraph
				'subgraph'
				'{'
				StmtList
				'}'
`,
			wantErrors: []string{
				"1:21: expected }",
			},
		},
		"SubgraphEmpty": {
			in: `graph { subgraph {} }`,
			want: `File
	Graph
		'graph'
		'{'
		StmtList
			Subgraph
				'subgraph'
				'{'
				StmtList
				'}'
		'}'
`,
		},
		"SubgraphWithID": {
			in: `graph { subgraph foo {} }`,
			want: `File
	Graph
		'graph'
		'{'
		StmtList
			Subgraph
				'subgraph'
				ID
					'foo'
				'{'
				StmtList
				'}'
		'}'
`,
		},
		"SubgraphWithoutKeyword": {
			in: `graph { {} }`,
			want: `File
	Graph
		'graph'
		'{'
		StmtList
			Subgraph
				'{'
				StmtList
				'}'
		'}'
`,
		},
		// Verifies that { } style subgraphs don't allow an ID - any ID after { is a node statement
		"SubgraphWithoutKeywordWithID": {
			in: `graph { { A } }`,
			want: `File
	Graph
		'graph'
		'{'
		StmtList
			Subgraph
				'{'
				StmtList
					NodeStmt
						NodeID
							ID
								'A'
				'}'
		'}'
`,
		},
		"SubgraphWithoutKeywordIncomplete": {
			in: `graph { { }`,
			want: `File
	Graph
		'graph'
		'{'
		StmtList
			Subgraph
				'{'
				StmtList
				'}'
`,
			wantErrors: []string{
				"1:12: expected }",
			},
		},
		"SubgraphWithNodes": {
			in: `graph { subgraph { A B } }`,
			want: `File
	Graph
		'graph'
		'{'
		StmtList
			Subgraph
				'subgraph'
				'{'
				StmtList
					NodeStmt
						NodeID
							ID
								'A'
					NodeStmt
						NodeID
							ID
								'B'
				'}'
		'}'
`,
		},
		"SubgraphWithAttribute": {
			in: `graph { subgraph { rank=same } }`,
			want: `File
	Graph
		'graph'
		'{'
		StmtList
			Subgraph
				'subgraph'
				'{'
				StmtList
					Attribute
						ID
							'rank'
						'='
						ID
							'same'
				'}'
		'}'
`,
		},
		"SubgraphNested": {
			in: `graph { subgraph { subgraph {} } }`,
			want: `File
	Graph
		'graph'
		'{'
		StmtList
			Subgraph
				'subgraph'
				'{'
				StmtList
					Subgraph
						'subgraph'
						'{'
						StmtList
						'}'
				'}'
		'}'
`,
		},
		"SubgraphMissingCloseBrace": {
			in: `graph { subgraph { A }`,
			want: `File
	Graph
		'graph'
		'{'
		StmtList
			Subgraph
				'subgraph'
				'{'
				StmtList
					NodeStmt
						NodeID
							ID
								'A'
				'}'
`,
			wantErrors: []string{
				"1:23: expected }",
			},
		},
		"SubgraphMultiple": {
			in: `graph { subgraph { A } subgraph { B } }`,
			want: `File
	Graph
		'graph'
		'{'
		StmtList
			Subgraph
				'subgraph'
				'{'
				StmtList
					NodeStmt
						NodeID
							ID
								'A'
				'}'
			Subgraph
				'subgraph'
				'{'
				StmtList
					NodeStmt
						NodeID
							ID
								'B'
				'}'
		'}'
`,
		},
		"SubgraphRecoveryGarbageTokens": {
			in: `graph { subgraph foo bar baz { } }`,
			want: `File
	Graph
		'graph'
		'{'
		StmtList
			Subgraph
				'subgraph'
				ID
					'foo'
				ErrorTree
					'bar'
				ErrorTree
					'baz'
				'{'
				StmtList
				'}'
		'}'
`,
			wantErrors: []string{
				"1:22: unexpected token ID 'bar'",
				"1:26: unexpected token ID 'baz'",
			},
		},
		"SubgraphRecoveryOneGarbageToken": {
			in: `graph { subgraph foo bar { A } }`,
			want: `File
	Graph
		'graph'
		'{'
		StmtList
			Subgraph
				'subgraph'
				ID
					'foo'
				ErrorTree
					'bar'
				'{'
				StmtList
					NodeStmt
						NodeID
							ID
								'A'
				'}'
		'}'
`,
			wantErrors: []string{
				"1:22: unexpected token ID 'bar'",
			},
		},
		"SubgraphRecoveryWithKeywordGarbage": {
			in: `graph { subgraph foo graph { } }`,
			want: `File
	Graph
		'graph'
		'{'
		StmtList
			Subgraph
				'subgraph'
				ID
					'foo'
			AttrStmt
				'graph'
			Subgraph
				'{'
				StmtList
				'}'
		'}'
`,
			wantErrors: []string{
				"1:22: expected {",
				"1:28: expected [ to start attribute list",
			},
		},
		"SubgraphAnonymousNoRecovery": {
			in: `graph { { foo bar { } } }`,
			want: `File
	Graph
		'graph'
		'{'
		StmtList
			Subgraph
				'{'
				StmtList
					NodeStmt
						NodeID
							ID
								'foo'
					NodeStmt
						NodeID
							ID
								'bar'
					Subgraph
						'{'
						StmtList
						'}'
				'}'
		'}'
`,
		},

		// Edge statement tests with attribute lists
		"EdgeWithEmptyAttrList": {
			in: `graph { A -- B [] }`,
			want: `File
	Graph
		'graph'
		'{'
		StmtList
			EdgeStmt
				NodeID
					ID
						'A'
				'--'
				NodeID
					ID
						'B'
				AttrList
					'['
					']'
		'}'
`,
		},
		"EdgeWithSingleAttribute": {
			in: `graph { A -- B [color=red] }`,
			want: `File
	Graph
		'graph'
		'{'
		StmtList
			EdgeStmt
				NodeID
					ID
						'A'
				'--'
				NodeID
					ID
						'B'
				AttrList
					'['
					AList
						Attribute
							ID
								'color'
							'='
							ID
								'red'
					']'
		'}'
`,
		},
		"EdgeWithMultipleAttributes": {
			in: `digraph { 1 -> 2 -> 3 -> 4 [a=b, c=d] }`,
			want: `File
	Graph
		'digraph'
		'{'
		StmtList
			EdgeStmt
				NodeID
					ID
						'1'
				'->'
				NodeID
					ID
						'2'
				'->'
				NodeID
					ID
						'3'
				'->'
				NodeID
					ID
						'4'
				AttrList
					'['
					AList
						Attribute
							ID
								'a'
							'='
							ID
								'b'
						','
						Attribute
							ID
								'c'
							'='
							ID
								'd'
					']'
		'}'
`,
		},
		"EdgeWithMultipleAttrLists": {
			in: `graph { A -- B [color=red][shape=box] }`,
			want: `File
	Graph
		'graph'
		'{'
		StmtList
			EdgeStmt
				NodeID
					ID
						'A'
				'--'
				NodeID
					ID
						'B'
				AttrList
					'['
					AList
						Attribute
							ID
								'color'
							'='
							ID
								'red'
					']'
					'['
					AList
						Attribute
							ID
								'shape'
							'='
							ID
								'box'
					']'
		'}'
`,
		},

		// Edge statement tests with subgraphs
		"EdgeWithLHSSubgraph": {
			in: `digraph { {A B} -> C }`,
			want: `File
	Graph
		'digraph'
		'{'
		StmtList
			EdgeStmt
				Subgraph
					'{'
					StmtList
						NodeStmt
							NodeID
								ID
									'A'
						NodeStmt
							NodeID
								ID
									'B'
					'}'
				'->'
				NodeID
					ID
						'C'
		'}'
`,
		},
		"EdgeWithRHSSubgraph": {
			in: `digraph { A -> {B C} }`,
			want: `File
	Graph
		'digraph'
		'{'
		StmtList
			EdgeStmt
				NodeID
					ID
						'A'
				'->'
				Subgraph
					'{'
					StmtList
						NodeStmt
							NodeID
								ID
									'B'
						NodeStmt
							NodeID
								ID
									'C'
					'}'
		'}'
`,
		},
		"EdgeWithBothSubgraphs": {
			in: `digraph { {A B} -> {C D} }`,
			want: `File
	Graph
		'digraph'
		'{'
		StmtList
			EdgeStmt
				Subgraph
					'{'
					StmtList
						NodeStmt
							NodeID
								ID
									'A'
						NodeStmt
							NodeID
								ID
									'B'
					'}'
				'->'
				Subgraph
					'{'
					StmtList
						NodeStmt
							NodeID
								ID
									'C'
						NodeStmt
							NodeID
								ID
									'D'
					'}'
		'}'
`,
		},
		"EdgeWithNestedSubgraphs": {
			in: `graph { {1 2} -- {3 -- {4 5}} }`,
			want: `File
	Graph
		'graph'
		'{'
		StmtList
			EdgeStmt
				Subgraph
					'{'
					StmtList
						NodeStmt
							NodeID
								ID
									'1'
						NodeStmt
							NodeID
								ID
									'2'
					'}'
				'--'
				Subgraph
					'{'
					StmtList
						EdgeStmt
							NodeID
								ID
									'3'
							'--'
							Subgraph
								'{'
								StmtList
									NodeStmt
										NodeID
											ID
												'4'
									NodeStmt
										NodeID
											ID
												'5'
								'}'
					'}'
		'}'
`,
		},
		"EdgeWithExplicitSubgraph": {
			in: `digraph { A -> subgraph foo {B C} }`,
			want: `File
	Graph
		'digraph'
		'{'
		StmtList
			EdgeStmt
				NodeID
					ID
						'A'
				'->'
				Subgraph
					'subgraph'
					ID
						'foo'
					'{'
					StmtList
						NodeStmt
							NodeID
								ID
									'B'
						NodeStmt
							NodeID
								ID
									'C'
					'}'
		'}'
`,
		},
		"EdgeChainWithSubgraph": {
			in: `graph { A -- {B} -- C }`,
			want: `File
	Graph
		'graph'
		'{'
		StmtList
			EdgeStmt
				NodeID
					ID
						'A'
				'--'
				Subgraph
					'{'
					StmtList
						NodeStmt
							NodeID
								ID
									'B'
					'}'
				'--'
				NodeID
					ID
						'C'
		'}'
`,
		},

		// Edge statement tests with ports
		"EdgeWithPorts": {
			in: `digraph { "node4":f0:n -> node5:f1 }`,
			want: `File
	Graph
		'digraph'
		'{'
		StmtList
			EdgeStmt
				NodeID
					ID
						'"node4"'
					Port
						':'
						ID
							'f0'
						':'
						CompassPoint
							'n'
				'->'
				NodeID
					ID
						'node5'
					Port
						':'
						ID
							'f1'
		'}'
`,
		},
		"EdgeWithPortsAndAttrList": {
			in: `digraph { A:n -> B:s [color=red] }`,
			want: `File
	Graph
		'digraph'
		'{'
		StmtList
			EdgeStmt
				NodeID
					ID
						'A'
					Port
						':'
						CompassPoint
							'n'
				'->'
				NodeID
					ID
						'B'
					Port
						':'
						CompassPoint
							's'
				AttrList
					'['
					AList
						Attribute
							ID
								'color'
							'='
							ID
								'red'
					']'
		'}'
`,
		},

		// Edge statement error recovery tests
		"EdgeWithIncompleteSubgraphRHS": {
			in: `graph { A -- { }`,
			want: `File
	Graph
		'graph'
		'{'
		StmtList
			EdgeStmt
				NodeID
					ID
						'A'
				'--'
				Subgraph
					'{'
					StmtList
					'}'
`,
			wantErrors: []string{
				"1:17: expected }",
			},
		},
		"EdgeWithSubgraphMissingCloseBrace": {
			in: `graph { A -- { B }`,
			want: `File
	Graph
		'graph'
		'{'
		StmtList
			EdgeStmt
				NodeID
					ID
						'A'
				'--'
				Subgraph
					'{'
					StmtList
						NodeStmt
							NodeID
								ID
									'B'
					'}'
`,
			wantErrors: []string{
				"1:19: expected }",
			},
		},
		"EdgeWithSubgraphIncompleteAttr": {
			in: `graph { A -- B [color }`,
			want: `File
	Graph
		'graph'
		'{'
		StmtList
			EdgeStmt
				NodeID
					ID
						'A'
				'--'
				NodeID
					ID
						'B'
				AttrList
					'['
					AList
						Attribute
							ID
								'color'
		'}'
`,
			wantErrors: []string{
				"1:23: expected =",
				"1:23: expected ] to close attribute list",
			},
		},

		"EdgeOperatorMismatch": {
			in: `digraph { A -- B }
graph { C -> D }`,
			want: `File
	Graph
		'digraph'
		'{'
		StmtList
			EdgeStmt
				NodeID
					ID
						'A'
				'--'
				NodeID
					ID
						'B'
		'}'
	Graph
		'graph'
		'{'
		StmtList
			EdgeStmt
				NodeID
					ID
						'C'
				'->'
				NodeID
					ID
						'D'
		'}'
`,
			wantErrors: []string{
				"1:13: expected '->' for edge in directed graph",
				"2:11: expected '--' for edge in undirected graph",
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			p := dot.NewParser([]byte(test.in))
			g := p.Parse()

			assert.EqualValuesf(t, g.String(), test.want, "Parse(%q)", test.in)
			assert.EqualValuesf(t, errorStrings(p.Errors()), test.wantErrors, "Parse(%q) errors", test.in)

			// Verify String() matches Render(Default)
			var buf strings.Builder
			err := g.Render(&buf, dot.Default)
			assert.NoErrorf(t, err, "Render(%q, Default)", test.in)
			assert.EqualValuesf(t, g.String(), buf.String(), "String() should match Render(Default)")

			// Verify positions via Render(Scheme) when expected
			if test.wantScheme != "" {
				buf.Reset()
				err = g.Render(&buf, dot.Scheme)
				assert.NoErrorf(t, err, "Render(%q, Scheme)", test.in)
				assert.EqualValuesf(t, buf.String(), test.wantScheme, "Render(%q, Scheme)", test.in)
			}
		})
	}
}

func errorStrings(errors []dot.Error) []string {
	if len(errors) == 0 {
		return nil
	}
	result := make([]string, len(errors))
	for i, err := range errors {
		result[i] = err.Error()
	}
	return result
}
