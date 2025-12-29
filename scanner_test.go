package dot

import (
	"strconv"
	"testing"

	"github.com/teleivo/assertive/assert"
	"github.com/teleivo/assertive/require"
	"github.com/teleivo/dot/token"
)

func TestScanner(t *testing.T) {
	tests := map[string]struct {
		in   string
		want []token.Token
	}{
		"Empty": {
			in: "",
			want: []token.Token{
				{
					Type:  token.EOF,
					Start: token.Position{Line: 1, Column: 1},
					End:   token.Position{Line: 1, Column: 1},
				},
			},
		},
		"SingleCharacter": {
			in: "a",
			want: []token.Token{
				{
					Type: token.ID, Literal: "a",
					Start: token.Position{Line: 1, Column: 1},
					End:   token.Position{Line: 1, Column: 1},
				},
				{
					Type:  token.EOF,
					Start: token.Position{Line: 1, Column: 2},
					End:   token.Position{Line: 1, Column: 2},
				},
			},
		},
		"OnlyWhitespace": {
			in: "\t \n \t\t   \r\n",
			want: []token.Token{
				{
					Type:  token.EOF,
					Start: token.Position{Line: 3, Column: 1},
					End:   token.Position{Line: 3, Column: 1},
				},
			},
		},
		"LiteralSingleCharacterTokens": {
			in: "{};=[],:",
			want: []token.Token{
				{
					Type: token.LeftBrace, Literal: "{",
					Start: token.Position{Line: 1, Column: 1},
					End:   token.Position{Line: 1, Column: 1},
				},
				{
					Type: token.RightBrace, Literal: "}",
					Start: token.Position{Line: 1, Column: 2},
					End:   token.Position{Line: 1, Column: 2},
				},
				{
					Type: token.Semicolon, Literal: ";",
					Start: token.Position{Line: 1, Column: 3},
					End:   token.Position{Line: 1, Column: 3},
				},
				{
					Type: token.Equal, Literal: "=",
					Start: token.Position{Line: 1, Column: 4},
					End:   token.Position{Line: 1, Column: 4},
				},
				{
					Type: token.LeftBracket, Literal: "[",
					Start: token.Position{Line: 1, Column: 5},
					End:   token.Position{Line: 1, Column: 5},
				},
				{
					Type: token.RightBracket, Literal: "]",
					Start: token.Position{Line: 1, Column: 6},
					End:   token.Position{Line: 1, Column: 6},
				},
				{
					Type: token.Comma, Literal: ",",
					Start: token.Position{Line: 1, Column: 7},
					End:   token.Position{Line: 1, Column: 7},
				},
				{
					Type: token.Colon, Literal: ":",
					Start: token.Position{Line: 1, Column: 8},
					End:   token.Position{Line: 1, Column: 8},
				},
				{
					Type:  token.EOF,
					Start: token.Position{Line: 1, Column: 9},
					End:   token.Position{Line: 1, Column: 9},
				},
			},
		},
		"KeywordsAreCaseInsensitive": {
			in: " graph Graph strict  Strict\ndigraph\tDigraph Subgraph  subgraph Node node edge Edge \n \t ",
			want: []token.Token{
				{
					Type:    token.Graph,
					Literal: "graph",
					Start:   token.Position{Line: 1, Column: 2},
					End:     token.Position{Line: 1, Column: 6},
				},
				{
					Type:    token.Graph,
					Literal: "Graph",
					Start:   token.Position{Line: 1, Column: 8},
					End:     token.Position{Line: 1, Column: 12},
				},
				{
					Type:    token.Strict,
					Literal: "strict",
					Start:   token.Position{Line: 1, Column: 14},
					End:     token.Position{Line: 1, Column: 19},
				},
				{
					Type:    token.Strict,
					Literal: "Strict",
					Start:   token.Position{Line: 1, Column: 22},
					End:     token.Position{Line: 1, Column: 27},
				},
				{
					Type:    token.Digraph,
					Literal: "digraph",
					Start:   token.Position{Line: 2, Column: 1},
					End:     token.Position{Line: 2, Column: 7},
				},
				{
					Type:    token.Digraph,
					Literal: "Digraph",
					Start:   token.Position{Line: 2, Column: 9},
					End:     token.Position{Line: 2, Column: 15},
				},
				{
					Type:    token.Subgraph,
					Literal: "Subgraph",
					Start:   token.Position{Line: 2, Column: 17},
					End:     token.Position{Line: 2, Column: 24},
				},
				{
					Type:    token.Subgraph,
					Literal: "subgraph",
					Start:   token.Position{Line: 2, Column: 27},
					End:     token.Position{Line: 2, Column: 34},
				},
				{
					Type:    token.Node,
					Literal: "Node",
					Start:   token.Position{Line: 2, Column: 36},
					End:     token.Position{Line: 2, Column: 39},
				},
				{
					Type:    token.Node,
					Literal: "node",
					Start:   token.Position{Line: 2, Column: 41},
					End:     token.Position{Line: 2, Column: 44},
				},
				{
					Type:    token.Edge,
					Literal: "edge",
					Start:   token.Position{Line: 2, Column: 46},
					End:     token.Position{Line: 2, Column: 49},
				},
				{
					Type:    token.Edge,
					Literal: "Edge",
					Start:   token.Position{Line: 2, Column: 51},
					End:     token.Position{Line: 2, Column: 54},
				},
				{
					Type:  token.EOF,
					Start: token.Position{Line: 3, Column: 4},
					End:   token.Position{Line: 3, Column: 4},
				},
			},
		},
		"CommentsCanHugIdentifiers": {
			in: `A//commenting on A
			B#commenting on B
"C"//commenting on C
"D"#commenting on D
`,
			want: []token.Token{
				{
					Type:    token.ID,
					Literal: "A",
					Start:   token.Position{Line: 1, Column: 1},
					End:     token.Position{Line: 1, Column: 1},
				},
				{
					Type:    token.Comment,
					Literal: `//commenting on A`,
					Start:   token.Position{Line: 1, Column: 2},
					End:     token.Position{Line: 1, Column: 18},
				},
				{
					Type:    token.ID,
					Literal: "B",
					Start:   token.Position{Line: 2, Column: 4},
					End:     token.Position{Line: 2, Column: 4},
				},
				{
					Type:    token.Comment,
					Literal: `#commenting on B`,
					Start:   token.Position{Line: 2, Column: 5},
					End:     token.Position{Line: 2, Column: 20},
				},
				{
					Type:    token.ID,
					Literal: `"C"`,
					Start:   token.Position{Line: 3, Column: 1},
					End:     token.Position{Line: 3, Column: 3},
				},
				{
					Type:    token.Comment,
					Literal: `//commenting on C`,
					Start:   token.Position{Line: 3, Column: 4},
					End:     token.Position{Line: 3, Column: 20},
				},
				{
					Type:    token.ID,
					Literal: `"D"`,
					Start:   token.Position{Line: 4, Column: 1},
					End:     token.Position{Line: 4, Column: 3},
				},
				{
					Type:    token.Comment,
					Literal: `#commenting on D`,
					Start:   token.Position{Line: 4, Column: 4},
					End:     token.Position{Line: 4, Column: 19},
				},
				{
					Type:  token.EOF,
					Start: token.Position{Line: 5, Column: 1},
					End:   token.Position{Line: 5, Column: 1},
				},
			},
		},
		"EmptyQuotedIdentifier": {
			in: `""`,
			want: []token.Token{
				{
					Type:    token.ID,
					Literal: `""`,
					Start:   token.Position{Line: 1, Column: 1},
					End:     token.Position{Line: 1, Column: 2},
				},
				{
					Type:  token.EOF,
					Start: token.Position{Line: 1, Column: 3},
					End:   token.Position{Line: 1, Column: 3},
				},
			},
		},
		"Identifiers": {
			in: `
			  A;B;C"D""E"
			}`,
			want: []token.Token{
				{
					Type:    token.ID,
					Literal: "A",
					Start:   token.Position{Line: 2, Column: 6},
					End:     token.Position{Line: 2, Column: 6},
				},
				{
					Type: token.Semicolon, Literal: ";",
					Start: token.Position{Line: 2, Column: 7},
					End:   token.Position{Line: 2, Column: 7},
				},
				{
					Type:    token.ID,
					Literal: "B",
					Start:   token.Position{Line: 2, Column: 8},
					End:     token.Position{Line: 2, Column: 8},
				},
				{
					Type: token.Semicolon, Literal: ";",
					Start: token.Position{Line: 2, Column: 9},
					End:   token.Position{Line: 2, Column: 9},
				},
				{
					Type:    token.ID,
					Literal: "C",
					Start:   token.Position{Line: 2, Column: 10},
					End:     token.Position{Line: 2, Column: 10},
				},
				{
					Type:    token.ID,
					Literal: `"D"`,
					Start:   token.Position{Line: 2, Column: 11},
					End:     token.Position{Line: 2, Column: 13},
				},
				{
					Type:    token.ID,
					Literal: `"E"`,
					Start:   token.Position{Line: 2, Column: 14},
					End:     token.Position{Line: 2, Column: 16},
				},
				{
					Type: token.RightBrace, Literal: "}",
					Start: token.Position{Line: 3, Column: 4},
					End:   token.Position{Line: 3, Column: 4},
				},
				{
					Type:  token.EOF,
					Start: token.Position{Line: 3, Column: 5},
					End:   token.Position{Line: 3, Column: 5},
				},
			},
		},
		"UnquotedQuotedUnquotedSandwich": {
			in: `A"B"C`,
			want: []token.Token{
				{
					Type:    token.ID,
					Literal: "A",
					Start:   token.Position{Line: 1, Column: 1},
					End:     token.Position{Line: 1, Column: 1},
				},
				{
					Type:    token.ID,
					Literal: `"B"`,
					Start:   token.Position{Line: 1, Column: 2},
					End:     token.Position{Line: 1, Column: 4},
				},
				{
					Type:    token.ID,
					Literal: "C",
					Start:   token.Position{Line: 1, Column: 5},
					End:     token.Position{Line: 1, Column: 5},
				},
				{
					Type:  token.EOF,
					Start: token.Position{Line: 1, Column: 6},
					End:   token.Position{Line: 1, Column: 6},
				},
			},
		},
		"QuotedFollowedByUnquoted": {
			in: `"A"_B`,
			want: []token.Token{
				{
					Type:    token.ID,
					Literal: `"A"`,
					Start:   token.Position{Line: 1, Column: 1},
					End:     token.Position{Line: 1, Column: 3},
				},
				{
					Type:    token.ID,
					Literal: "_B",
					Start:   token.Position{Line: 1, Column: 4},
					End:     token.Position{Line: 1, Column: 5},
				},
				{
					Type:  token.EOF,
					Start: token.Position{Line: 1, Column: 6},
					End:   token.Position{Line: 1, Column: 6},
				},
			},
		},
		"UnquotedFollowedByQuoted": {
			in: `A_1"B"`,
			want: []token.Token{
				{
					Type:    token.ID,
					Literal: "A_1",
					Start:   token.Position{Line: 1, Column: 1},
					End:     token.Position{Line: 1, Column: 3},
				},
				{
					Type:    token.ID,
					Literal: `"B"`,
					Start:   token.Position{Line: 1, Column: 4},
					End:     token.Position{Line: 1, Column: 6},
				},
				{
					Type:  token.EOF,
					Start: token.Position{Line: 1, Column: 7},
					End:   token.Position{Line: 1, Column: 7},
				},
			},
		},
		"AttributeList": {
			in: `	graph [
				labelloc = t
				fontname = "Helvetica,Arial,sans-serif",fontsize=16
			]
					edge [arrowhead=none color="#00008844",style = filled];  `,
			want: []token.Token{
				{
					Type:    token.Graph,
					Literal: "graph",
					Start:   token.Position{Line: 1, Column: 2},
					End:     token.Position{Line: 1, Column: 6},
				},
				{
					Type: token.LeftBracket, Literal: "[",
					Start: token.Position{Line: 1, Column: 8},
					End:   token.Position{Line: 1, Column: 8},
				},
				{
					Type:    token.ID,
					Literal: "labelloc",
					Start:   token.Position{Line: 2, Column: 5},
					End:     token.Position{Line: 2, Column: 12},
				},
				{
					Type: token.Equal, Literal: "=",
					Start: token.Position{Line: 2, Column: 14},
					End:   token.Position{Line: 2, Column: 14},
				},
				{
					Type:    token.ID,
					Literal: "t",
					Start:   token.Position{Line: 2, Column: 16},
					End:     token.Position{Line: 2, Column: 16},
				},
				{
					Type:    token.ID,
					Literal: "fontname",
					Start:   token.Position{Line: 3, Column: 5},
					End:     token.Position{Line: 3, Column: 12},
				},
				{
					Type: token.Equal, Literal: "=",
					Start: token.Position{Line: 3, Column: 14},
					End:   token.Position{Line: 3, Column: 14},
				},
				{
					Type:    token.ID,
					Literal: `"Helvetica,Arial,sans-serif"`,
					Start:   token.Position{Line: 3, Column: 16},
					End:     token.Position{Line: 3, Column: 43},
				},
				{
					Type: token.Comma, Literal: ",",
					Start: token.Position{Line: 3, Column: 44},
					End:   token.Position{Line: 3, Column: 44},
				},
				{
					Type:    token.ID,
					Literal: "fontsize",
					Start:   token.Position{Line: 3, Column: 45},
					End:     token.Position{Line: 3, Column: 52},
				},
				{
					Type: token.Equal, Literal: "=",
					Start: token.Position{Line: 3, Column: 53},
					End:   token.Position{Line: 3, Column: 53},
				},
				{
					Type:    token.ID,
					Literal: "16",
					Start:   token.Position{Line: 3, Column: 54},
					End:     token.Position{Line: 3, Column: 55},
				},
				{
					Type: token.RightBracket, Literal: "]",
					Start: token.Position{Line: 4, Column: 4},
					End:   token.Position{Line: 4, Column: 4},
				},
				{
					Type:    token.Edge,
					Literal: "edge",
					Start:   token.Position{Line: 5, Column: 6},
					End:     token.Position{Line: 5, Column: 9},
				},
				{
					Type: token.LeftBracket, Literal: "[",
					Start: token.Position{Line: 5, Column: 11},
					End:   token.Position{Line: 5, Column: 11},
				},
				{
					Type:    token.ID,
					Literal: "arrowhead",
					Start:   token.Position{Line: 5, Column: 12},
					End:     token.Position{Line: 5, Column: 20},
				},
				{
					Type: token.Equal, Literal: "=",
					Start: token.Position{Line: 5, Column: 21},
					End:   token.Position{Line: 5, Column: 21},
				},
				{
					Type:    token.ID,
					Literal: "none",
					Start:   token.Position{Line: 5, Column: 22},
					End:     token.Position{Line: 5, Column: 25},
				},
				{
					Type:    token.ID,
					Literal: "color",
					Start:   token.Position{Line: 5, Column: 27},
					End:     token.Position{Line: 5, Column: 31},
				},
				{
					Type: token.Equal, Literal: "=",
					Start: token.Position{Line: 5, Column: 32},
					End:   token.Position{Line: 5, Column: 32},
				},
				{
					Type:    token.ID,
					Literal: `"#00008844"`,
					Start:   token.Position{Line: 5, Column: 33},
					End:     token.Position{Line: 5, Column: 43},
				},
				{
					Type: token.Comma, Literal: ",",
					Start: token.Position{Line: 5, Column: 44},
					End:   token.Position{Line: 5, Column: 44},
				},
				{
					Type:    token.ID,
					Literal: "style",
					Start:   token.Position{Line: 5, Column: 45},
					End:     token.Position{Line: 5, Column: 49},
				},
				{
					Type: token.Equal, Literal: "=",
					Start: token.Position{Line: 5, Column: 51},
					End:   token.Position{Line: 5, Column: 51},
				},
				{
					Type:    token.ID,
					Literal: "filled",
					Start:   token.Position{Line: 5, Column: 53},
					End:     token.Position{Line: 5, Column: 58},
				},
				{
					Type: token.RightBracket, Literal: "]",
					Start: token.Position{Line: 5, Column: 59},
					End:   token.Position{Line: 5, Column: 59},
				},
				{
					Type: token.Semicolon, Literal: ";",
					Start: token.Position{Line: 5, Column: 60},
					End:   token.Position{Line: 5, Column: 60},
				},
				{
					Type:  token.EOF,
					Start: token.Position{Line: 5, Column: 63},
					End:   token.Position{Line: 5, Column: 63},
				},
			},
		},
		"Subgraphs": {
			in: `  A -> {B C}
				D -- E
			subgraph {
				"F"
			  rank = same; A;B;C;
			}`,
			want: []token.Token{
				{
					Type:    token.ID,
					Literal: "A",
					Start:   token.Position{Line: 1, Column: 3},
					End:     token.Position{Line: 1, Column: 3},
				},
				{
					Type: token.DirectedEdge, Literal: "->",
					Start: token.Position{Line: 1, Column: 5},
					End:   token.Position{Line: 1, Column: 6},
				},
				{
					Type: token.LeftBrace, Literal: "{",
					Start: token.Position{Line: 1, Column: 8},
					End:   token.Position{Line: 1, Column: 8},
				},
				{
					Type:    token.ID,
					Literal: "B",
					Start:   token.Position{Line: 1, Column: 9},
					End:     token.Position{Line: 1, Column: 9},
				},
				{
					Type:    token.ID,
					Literal: "C",
					Start:   token.Position{Line: 1, Column: 11},
					End:     token.Position{Line: 1, Column: 11},
				},
				{
					Type: token.RightBrace, Literal: "}",
					Start: token.Position{Line: 1, Column: 12},
					End:   token.Position{Line: 1, Column: 12},
				},
				{
					Type:    token.ID,
					Literal: "D",
					Start:   token.Position{Line: 2, Column: 5},
					End:     token.Position{Line: 2, Column: 5},
				},
				{
					Type: token.UndirectedEdge, Literal: "--",
					Start: token.Position{Line: 2, Column: 7},
					End:   token.Position{Line: 2, Column: 8},
				},
				{
					Type:    token.ID,
					Literal: "E",
					Start:   token.Position{Line: 2, Column: 10},
					End:     token.Position{Line: 2, Column: 10},
				},
				{
					Type:    token.Subgraph,
					Literal: "subgraph",
					Start:   token.Position{Line: 3, Column: 4},
					End:     token.Position{Line: 3, Column: 11},
				},
				{
					Type: token.LeftBrace, Literal: "{",
					Start: token.Position{Line: 3, Column: 13},
					End:   token.Position{Line: 3, Column: 13},
				},
				{
					Type:    token.ID,
					Literal: `"F"`,
					Start:   token.Position{Line: 4, Column: 5},
					End:     token.Position{Line: 4, Column: 7},
				},
				{
					Type:    token.ID,
					Literal: "rank",
					Start:   token.Position{Line: 5, Column: 6},
					End:     token.Position{Line: 5, Column: 9},
				},
				{
					Type: token.Equal, Literal: "=",
					Start: token.Position{Line: 5, Column: 11},
					End:   token.Position{Line: 5, Column: 11},
				},
				{
					Type:    token.ID,
					Literal: "same",
					Start:   token.Position{Line: 5, Column: 13},
					End:     token.Position{Line: 5, Column: 16},
				},
				{
					Type: token.Semicolon, Literal: ";",
					Start: token.Position{Line: 5, Column: 17},
					End:   token.Position{Line: 5, Column: 17},
				},
				{
					Type:    token.ID,
					Literal: "A",
					Start:   token.Position{Line: 5, Column: 19},
					End:     token.Position{Line: 5, Column: 19},
				},
				{
					Type: token.Semicolon, Literal: ";",
					Start: token.Position{Line: 5, Column: 20},
					End:   token.Position{Line: 5, Column: 20},
				},
				{
					Type:    token.ID,
					Literal: "B",
					Start:   token.Position{Line: 5, Column: 21},
					End:     token.Position{Line: 5, Column: 21},
				},
				{
					Type: token.Semicolon, Literal: ";",
					Start: token.Position{Line: 5, Column: 22},
					End:   token.Position{Line: 5, Column: 22},
				},
				{
					Type:    token.ID,
					Literal: "C",
					Start:   token.Position{Line: 5, Column: 23},
					End:     token.Position{Line: 5, Column: 23},
				},
				{
					Type: token.Semicolon, Literal: ";",
					Start: token.Position{Line: 5, Column: 24},
					End:   token.Position{Line: 5, Column: 24},
				},
				{
					Type: token.RightBrace, Literal: "}",
					Start: token.Position{Line: 6, Column: 4},
					End:   token.Position{Line: 6, Column: 4},
				},
				{
					Type:  token.EOF,
					Start: token.Position{Line: 6, Column: 5},
					End:   token.Position{Line: 6, Column: 5},
				},
			},
		},
		"EdgesWithNumeralOperands": {
			in: `  1->2 3 ->4  5--6 7-- 8 9 --10`,
			want: []token.Token{
				{
					Type:    token.ID,
					Literal: "1",
					Start:   token.Position{Line: 1, Column: 3},
					End:     token.Position{Line: 1, Column: 3},
				},
				{
					Type:    token.DirectedEdge,
					Literal: "->",
					Start:   token.Position{Line: 1, Column: 4},
					End:     token.Position{Line: 1, Column: 5},
				},
				{
					Type:    token.ID,
					Literal: "2",
					Start:   token.Position{Line: 1, Column: 6},
					End:     token.Position{Line: 1, Column: 6},
				},
				{
					Type:    token.ID,
					Literal: "3",
					Start:   token.Position{Line: 1, Column: 8},
					End:     token.Position{Line: 1, Column: 8},
				},
				{
					Type:    token.DirectedEdge,
					Literal: "->",
					Start:   token.Position{Line: 1, Column: 10},
					End:     token.Position{Line: 1, Column: 11},
				},
				{
					Type:    token.ID,
					Literal: "4",
					Start:   token.Position{Line: 1, Column: 12},
					End:     token.Position{Line: 1, Column: 12},
				},
				{
					Type:    token.ID,
					Literal: "5",
					Start:   token.Position{Line: 1, Column: 15},
					End:     token.Position{Line: 1, Column: 15},
				},
				{
					Type:    token.UndirectedEdge,
					Literal: "--",
					Start:   token.Position{Line: 1, Column: 16},
					End:     token.Position{Line: 1, Column: 17},
				},
				{
					Type:    token.ID,
					Literal: "6",
					Start:   token.Position{Line: 1, Column: 18},
					End:     token.Position{Line: 1, Column: 18},
				},
				{
					Type:    token.ID,
					Literal: "7",
					Start:   token.Position{Line: 1, Column: 20},
					End:     token.Position{Line: 1, Column: 20},
				},
				{
					Type:    token.UndirectedEdge,
					Literal: "--",
					Start:   token.Position{Line: 1, Column: 21},
					End:     token.Position{Line: 1, Column: 22},
				},
				{
					Type:    token.ID,
					Literal: "8",
					Start:   token.Position{Line: 1, Column: 24},
					End:     token.Position{Line: 1, Column: 24},
				},
				{
					Type:    token.ID,
					Literal: "9",
					Start:   token.Position{Line: 1, Column: 26},
					End:     token.Position{Line: 1, Column: 26},
				},
				{
					Type:    token.UndirectedEdge,
					Literal: "--",
					Start:   token.Position{Line: 1, Column: 28},
					End:     token.Position{Line: 1, Column: 29},
				},
				{
					Type:    token.ID,
					Literal: "10",
					Start:   token.Position{Line: 1, Column: 30},
					End:     token.Position{Line: 1, Column: 31},
				},
			},
		},
		"NumeralFollowedByLineComment": {
			in: "123#comment",
			want: []token.Token{
				{
					Type:    token.ID,
					Literal: "123",
					Start:   token.Position{Line: 1, Column: 1},
					End:     token.Position{Line: 1, Column: 3},
				},
				{
					Type:    token.Comment,
					Literal: "#comment",
					Start:   token.Position{Line: 1, Column: 4},
					End:     token.Position{Line: 1, Column: 11},
				},
			},
		},
		"NumeralFollowedBySingleLineComment": {
			in: "456//comment",
			want: []token.Token{
				{
					Type:    token.ID,
					Literal: "456",
					Start:   token.Position{Line: 1, Column: 1},
					End:     token.Position{Line: 1, Column: 3},
				},
				{
					Type:    token.Comment,
					Literal: "//comment",
					Start:   token.Position{Line: 1, Column: 4},
					End:     token.Position{Line: 1, Column: 12},
				},
			},
		},
		"UnquotedIDFollowedByUndirectedEdgeNoWhitespace": {
			in: "ab--cd",
			want: []token.Token{
				{
					Type:    token.ID,
					Literal: "ab",
					Start:   token.Position{Line: 1, Column: 1},
					End:     token.Position{Line: 1, Column: 2},
				},
				{
					Type:    token.UndirectedEdge,
					Literal: "--",
					Start:   token.Position{Line: 1, Column: 3},
					End:     token.Position{Line: 1, Column: 4},
				},
				{
					Type:    token.ID,
					Literal: "cd",
					Start:   token.Position{Line: 1, Column: 5},
					End:     token.Position{Line: 1, Column: 6},
				},
			},
		},
		"UnquotedIDFollowedByDirectedEdgeNoWhitespace": {
			in: "ab->cd",
			want: []token.Token{
				{
					Type:    token.ID,
					Literal: "ab",
					Start:   token.Position{Line: 1, Column: 1},
					End:     token.Position{Line: 1, Column: 2},
				},
				{
					Type:    token.DirectedEdge,
					Literal: "->",
					Start:   token.Position{Line: 1, Column: 3},
					End:     token.Position{Line: 1, Column: 4},
				},
				{
					Type:    token.ID,
					Literal: "cd",
					Start:   token.Position{Line: 1, Column: 5},
					End:     token.Position{Line: 1, Column: 6},
				},
			},
		},
		"NumeralFollowedByUndirectedEdgeNoWhitespace": {
			in: "12--34",
			want: []token.Token{
				{
					Type:    token.ID,
					Literal: "12",
					Start:   token.Position{Line: 1, Column: 1},
					End:     token.Position{Line: 1, Column: 2},
				},
				{
					Type:    token.UndirectedEdge,
					Literal: "--",
					Start:   token.Position{Line: 1, Column: 3},
					End:     token.Position{Line: 1, Column: 4},
				},
				{
					Type:    token.ID,
					Literal: "34",
					Start:   token.Position{Line: 1, Column: 5},
					End:     token.Position{Line: 1, Column: 6},
				},
			},
		},
		"NumeralFollowedByDirectedEdgeNoWhitespace": {
			in: "12->34",
			want: []token.Token{
				{
					Type:    token.ID,
					Literal: "12",
					Start:   token.Position{Line: 1, Column: 1},
					End:     token.Position{Line: 1, Column: 2},
				},
				{
					Type:    token.DirectedEdge,
					Literal: "->",
					Start:   token.Position{Line: 1, Column: 3},
					End:     token.Position{Line: 1, Column: 4},
				},
				{
					Type:    token.ID,
					Literal: "34",
					Start:   token.Position{Line: 1, Column: 5},
					End:     token.Position{Line: 1, Column: 6},
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			scanner := NewScanner([]byte(test.in))
			assertTokens(t, scanner, test.want)
		})
	}

	// https://graphviz.org/doc/info/lang.html#ids
	t.Run("UnquotedIdentifiers", func(t *testing.T) {
		t.Run("Valid", func(t *testing.T) {
			tests := []struct {
				in   string
				want []token.Token
			}{
				{
					in: "_A",
					want: []token.Token{
						{
							Type:    token.ID,
							Literal: "_A",
							Start:   token.Position{Line: 1, Column: 1},
							End:     token.Position{Line: 1, Column: 2},
						},
					},
				},
				{
					in: "A_cZ",
					want: []token.Token{
						{
							Type:    token.ID,
							Literal: "A_cZ",
							Start:   token.Position{Line: 1, Column: 1},
							End:     token.Position{Line: 1, Column: 4},
						},
					},
				},
				{
					in: "A10",
					want: []token.Token{
						{
							Type:    token.ID,
							Literal: "A10",
							Start:   token.Position{Line: 1, Column: 1},
							End:     token.Position{Line: 1, Column: 3},
						},
					},
				},
				{
					in: "\u0080ÿ  ",
					want: []token.Token{
						{
							Type:    token.ID,
							Literal: "\u0080ÿ",
							Start:   token.Position{Line: 1, Column: 1},
							End:     token.Position{Line: 1, Column: 3}, // last char ÿ starts at byte offset 2 (1-based: 3)
						},
					},
				},
				{
					in: "Контрагенты",
					want: []token.Token{
						{
							Type:    token.ID,
							Literal: "Контрагенты",
							Start:   token.Position{Line: 1, Column: 1},
							End:     token.Position{Line: 1, Column: 21}, // last char starts at byte 20 (1-based: 21)
						},
					},
				},
			}

			for i, test := range tests {
				t.Run(strconv.Itoa(i), func(t *testing.T) {
					scanner := NewScanner([]byte(test.in))
					assertTokens(t, scanner, test.want)
				})
			}
		})

		t.Run("Invalid", func(t *testing.T) {
			tests := []struct {
				in   string
				want []token.Token
			}{
				{
					in: "  \x7f", // \177 - cannot start any token
					want: []token.Token{
						{
							Type:    token.ERROR,
							Literal: "\x7f",
							Error:   "invalid character U+007F: unquoted IDs must start with a letter or underscore",
							Start:   token.Position{Line: 1, Column: 3},
							End:     token.Position{Line: 1, Column: 3},
						},
					},
				},
				{
					in: "  _zab\x7fx", // \177 in middle - ERROR token consumes until separator
					want: []token.Token{
						{
							Type:    token.ERROR,
							Literal: "_zab\x7fx",
							Error:   "invalid character U+007F: unquoted IDs can only contain letters, digits, and underscores",
							Start:   token.Position{Line: 1, Column: 3},
							End:     token.Position{Line: 1, Column: 8},
						},
					},
				},
				{
					in: "A\000\000B", // null bytes within identifier - consume entire sequence as ERROR
					want: []token.Token{
						{
							Type:    token.ERROR,
							Literal: "A\x00\x00B",
							Error:   "invalid character U+0000: unquoted IDs cannot contain null bytes",
							Start:   token.Position{Line: 1, Column: 1},
							End:     token.Position{Line: 1, Column: 4},
						},
					},
				},
				{
					in: "A;x\000y{B", // null byte with adjacent chars grouped into ERROR, separated by terminals
					want: []token.Token{
						{
							Type:    token.ID,
							Literal: "A",
							Start:   token.Position{Line: 1, Column: 1},
							End:     token.Position{Line: 1, Column: 1},
						},
						{
							Type:    token.Semicolon,
							Literal: ";",
							Start:   token.Position{Line: 1, Column: 2},
							End:     token.Position{Line: 1, Column: 2},
						},
						{
							Type:    token.ERROR,
							Literal: "x\x00y",
							Error:   "invalid character U+0000: unquoted IDs cannot contain null bytes",
							Start:   token.Position{Line: 1, Column: 3},
							End:     token.Position{Line: 1, Column: 5},
						},
						{
							Type:    token.LeftBrace,
							Literal: "{",
							Start:   token.Position{Line: 1, Column: 6},
							End:     token.Position{Line: 1, Column: 6},
						},
						{
							Type:    token.ID,
							Literal: "B",
							Start:   token.Position{Line: 1, Column: 7},
							End:     token.Position{Line: 1, Column: 7},
						},
					},
				},
				{
					in: "@@@",
					want: []token.Token{
						{
							Type:    token.ERROR,
							Literal: "@@@",
							Error:   "invalid character '@': unquoted IDs must start with a letter or underscore",
							Start:   token.Position{Line: 1, Column: 1},
							End:     token.Position{Line: 1, Column: 3},
						},
					},
				},
				{
					in: "abc@def$ghi",
					want: []token.Token{
						{
							Type:    token.ERROR,
							Literal: "abc@def$ghi",
							Error:   "invalid character '@': unquoted IDs can only contain letters, digits, and underscores",
							Start:   token.Position{Line: 1, Column: 1},
							End:     token.Position{Line: 1, Column: 11},
						},
					},
				},
				{
					in: "a\x80b", // invalid UTF-8 byte (0x80 is a continuation byte without a start byte)
					want: []token.Token{
						{
							Type:    token.ERROR,
							Literal: "a\ufffdb", // ERROR consumes until separator
							Error:   "invalid character U+FFFD '�': unquoted IDs can only contain letters, digits, and underscores",
							Start:   token.Position{Line: 1, Column: 1},
							End:     token.Position{Line: 1, Column: 3},
						},
					},
				},
				{
					in: "\xfe\xff", // UTF-16 BOM (invalid UTF-8)
					want: []token.Token{
						{
							Type:    token.ERROR,
							Literal: "\ufffd\ufffd",
							Error:   "invalid character U+FFFD '�': unquoted IDs must start with a letter or underscore",
							Start:   token.Position{Line: 1, Column: 1},
							End:     token.Position{Line: 1, Column: 2},
						},
					},
				},
			}

			for i, test := range tests {
				t.Run(strconv.Itoa(i), func(t *testing.T) {
					scanner := NewScanner([]byte(test.in))
					assertNext(t, scanner, test.want, test.in)
				})
			}
		})
	})

	t.Run("EdgeOperators", func(t *testing.T) {
		t.Run("Invalid", func(t *testing.T) {
			tests := []struct {
				in   string
				want []token.Token
			}{
				{
					in: "a-b",
					want: []token.Token{
						{
							Type:    token.ERROR,
							Literal: "a-b",
							Error:   "invalid character '-': use '--' (undirected) or '->' (directed) for edges, or quote the ID",
							Start:   token.Position{Line: 1, Column: 1},
							End:     token.Position{Line: 1, Column: 3},
						},
					},
				},
			}

			for i, test := range tests {
				t.Run(strconv.Itoa(i), func(t *testing.T) {
					scanner := NewScanner([]byte(test.in))
					assertNext(t, scanner, test.want, test.in)
				})
			}
		})
	})

	t.Run("NumeralIdentifiers", func(t *testing.T) {
		t.Run("Valid", func(t *testing.T) {
			tests := []struct {
				in   string
				want token.Token
			}{
				{
					in: " -.9\t\n",
					want: token.Token{
						Type:    token.ID,
						Literal: "-.9",
						Start:   token.Position{Line: 1, Column: 2},
						End:     token.Position{Line: 1, Column: 4},
					},
				},
				{
					in: "-0.13",
					want: token.Token{
						Type:    token.ID,
						Literal: "-0.13",
						Start:   token.Position{Line: 1, Column: 1},
						End:     token.Position{Line: 1, Column: 5},
					},
				},
				{
					in: "-0.",
					want: token.Token{
						Type:    token.ID,
						Literal: "-0.",
						Start:   token.Position{Line: 1, Column: 1},
						End:     token.Position{Line: 1, Column: 3},
					},
				},
				{
					in: "-92.58",
					want: token.Token{
						Type:    token.ID,
						Literal: "-92.58",
						Start:   token.Position{Line: 1, Column: 1},
						End:     token.Position{Line: 1, Column: 6},
					},
				},
				{
					in: "-92",
					want: token.Token{
						Type:    token.ID,
						Literal: "-92",
						Start:   token.Position{Line: 1, Column: 1},
						End:     token.Position{Line: 1, Column: 3},
					},
				},
				{
					in: ".13",
					want: token.Token{
						Type:    token.ID,
						Literal: ".13",
						Start:   token.Position{Line: 1, Column: 1},
						End:     token.Position{Line: 1, Column: 3},
					},
				},
				{
					in: "0.",
					want: token.Token{
						Type:    token.ID,
						Literal: "0.",
						Start:   token.Position{Line: 1, Column: 1},
						End:     token.Position{Line: 1, Column: 2},
					},
				},
				{
					in: "0.13",
					want: token.Token{
						Type:    token.ID,
						Literal: "0.13",
						Start:   token.Position{Line: 1, Column: 1},
						End:     token.Position{Line: 1, Column: 4},
					},
				},
				{
					in: "47",
					want: token.Token{
						Type:    token.ID,
						Literal: "47",
						Start:   token.Position{Line: 1, Column: 1},
						End:     token.Position{Line: 1, Column: 2},
					},
				},
				{
					in: "47.58",
					want: token.Token{
						Type:    token.ID,
						Literal: "47.58",
						Start:   token.Position{Line: 1, Column: 1},
						End:     token.Position{Line: 1, Column: 5},
					},
				},
			}

			for i, test := range tests {
				t.Run(strconv.Itoa(i), func(t *testing.T) {
					scanner := NewScanner([]byte(test.in))
					assertTokens(t, scanner, []token.Token{test.want})
				})
			}
		})

		t.Run("Invalid", func(t *testing.T) {
			tests := []struct {
				in   string
				want []token.Token
			}{
				{
					in: "-.1A",
					want: []token.Token{
						{
							Type:    token.ERROR,
							Literal: "-.1A",
							Error:   "invalid character 'A': invalid character in number: valid forms are '1', '-1', '1.2', '-.1', '.1'",
							Start:   token.Position{Line: 1, Column: 1},
							End:     token.Position{Line: 1, Column: 4},
						},
					},
				},
				{
					in: "1-20",
					want: []token.Token{
						{
							Type:    token.ERROR,
							Literal: "1-20",
							Error:   "invalid character '-': ambiguous: quote for ID containing '-', use space for separate IDs, or '--'/'->' for edges",
							Start:   token.Position{Line: 1, Column: 1},
							End:     token.Position{Line: 1, Column: 4},
						},
					},
				},
				{
					in: ".13.4",
					want: []token.Token{
						{
							Type:    token.ERROR,
							Literal: ".13.4",
							Error:   "invalid character '.': ambiguous: quote for ID containing multiple '.', or use one decimal point for number",
							Start:   token.Position{Line: 1, Column: 1},
							End:     token.Position{Line: 1, Column: 5},
						},
					},
				},
				{
					in: "-.",
					want: []token.Token{
						{
							Type:    token.ERROR,
							Literal: "-.",
							Error:   "ambiguous: quote for ID, or add digit for number like '-.1' or '-0.'",
							Start:   token.Position{Line: 1, Column: 1},
							End:     token.Position{Line: 1, Column: 2},
						},
					},
				},
				{
					in: "\n. 0",
					want: []token.Token{
						{
							Type:    token.ERROR,
							Literal: ".",
							Error:   "invalid character ' ': ambiguous: quote for ID, or add digit for number like '-.1' or '-0.'",
							Start:   token.Position{Line: 2, Column: 1},
							End:     token.Position{Line: 2, Column: 1},
						},
						{
							Type:    token.ID,
							Literal: "0",
							Start:   token.Position{Line: 2, Column: 3},
							End:     token.Position{Line: 2, Column: 3},
						},
					},
				},
				{
					in: "100\u00A0200", // non-breaking space between 100 and 200
					want: []token.Token{
						{
							Type:    token.ERROR,
							Literal: "100\u00A0200",
							Error:   "invalid character U+00A0 '\u00a0': invalid character in number: valid forms are '1', '-1', '1.2', '-.1', '.1'",
							Start:   token.Position{Line: 1, Column: 1},
							End:     token.Position{Line: 1, Column: 8}, // 100 (3) + \u00A0 (2 bytes) + 200 (3) = 8
						},
					},
				},
				{
					in: "\n\n\n\t  - F",
					want: []token.Token{
						{
							Type:    token.ERROR,
							Literal: "-",
							Error:   "invalid character ' ': ambiguous: quote for ID, or add digit for number like '-.1' or '-0.'",
							Start:   token.Position{Line: 4, Column: 4},
							End:     token.Position{Line: 4, Column: 4},
						},
						{
							Type:    token.ID,
							Literal: "F",
							Start:   token.Position{Line: 4, Column: 6},
							End:     token.Position{Line: 4, Column: 6},
						},
					},
				},
				{
					in: "A---B",
					want: []token.Token{
						{
							Type:    token.ID,
							Literal: "A",
							Start:   token.Position{Line: 1, Column: 1},
							End:     token.Position{Line: 1, Column: 1},
						},
						{
							Type:    token.UndirectedEdge,
							Literal: "--",
							Start:   token.Position{Line: 1, Column: 2},
							End:     token.Position{Line: 1, Column: 3},
						},
						{
							Type:    token.ERROR,
							Literal: "-B",
							Error:   "invalid character 'B': invalid character in number: only digits and decimal point can follow '-'",
							Start:   token.Position{Line: 1, Column: 4},
							End:     token.Position{Line: 1, Column: 5},
						},
					},
				},
				{
					in: "1.2.3abc",
					want: []token.Token{
						{
							Type:    token.ERROR,
							Literal: "1.2.3abc",
							Error:   "invalid character '.': ambiguous: quote for ID containing multiple '.', or use one decimal point for number",
							Start:   token.Position{Line: 1, Column: 1},
							End:     token.Position{Line: 1, Column: 8},
						},
					},
				},
			}

			for i, test := range tests {
				t.Run(strconv.Itoa(i), func(t *testing.T) {
					scanner := NewScanner([]byte(test.in))
					assertNext(t, scanner, test.want, test.in)
				})
			}
		})
	})

	t.Run("QuotedIdentifiers", func(t *testing.T) {
		t.Run("Valid", func(t *testing.T) {
			tests := []struct {
				in   string
				want []token.Token
			}{
				{
					in: `"graph""strict"`,
					want: []token.Token{
						{
							Type:    token.ID,
							Literal: `"graph"`,
							Start:   token.Position{Line: 1, Column: 1},
							End:     token.Position{Line: 1, Column: 7},
						},
						{
							Type:    token.ID,
							Literal: `"strict"`,
							Start:   token.Position{Line: 1, Column: 8},
							End:     token.Position{Line: 1, Column: 15},
						},
					},
				},
				{
					in: `"\"d"`,
					want: []token.Token{
						{
							Type:    token.ID,
							Literal: `"\"d"`,
							Start:   token.Position{Line: 1, Column: 1},
							End:     token.Position{Line: 1, Column: 5},
						},
					},
				},
				{
					in: `"\nd"`,
					want: []token.Token{
						{
							Type:    token.ID,
							Literal: `"\nd"`,
							Start:   token.Position{Line: 1, Column: 1},
							End:     token.Position{Line: 1, Column: 5},
						},
					},
				},
				{
					in: `"\\d"`,
					want: []token.Token{
						{
							Type:    token.ID,
							Literal: `"\\d"`,
							Start:   token.Position{Line: 1, Column: 1},
							End:     token.Position{Line: 1, Column: 5},
						},
					},
				},
				{
					in: `"a\\"`,
					want: []token.Token{
						{
							Type:    token.ID,
							Literal: `"a\\"`,
							Start:   token.Position{Line: 1, Column: 1},
							End:     token.Position{Line: 1, Column: 5},
						},
					},
				},
				{
					in: `"_A"`,
					want: []token.Token{
						{
							Type:    token.ID,
							Literal: `"_A"`,
							Start:   token.Position{Line: 1, Column: 1},
							End:     token.Position{Line: 1, Column: 4},
						},
					},
				},
				{
					in: `"-.9"`,
					want: []token.Token{
						{
							Type:    token.ID,
							Literal: `"-.9"`,
							Start:   token.Position{Line: 1, Column: 1},
							End:     token.Position{Line: 1, Column: 5},
						},
					},
				},
				{
					in: `"A--B"`,
					want: []token.Token{
						{
							Type:    token.ID,
							Literal: `"A--B"`,
							Start:   token.Position{Line: 1, Column: 1},
							End:     token.Position{Line: 1, Column: 6},
						},
					},
				},
				{
					in: `"A->B"`,
					want: []token.Token{
						{
							Type:    token.ID,
							Literal: `"A->B"`,
							Start:   token.Position{Line: 1, Column: 1},
							End:     token.Position{Line: 1, Column: 6},
						},
					},
				},
				{
					in: `"A-B"`,
					want: []token.Token{
						{
							Type:    token.ID,
							Literal: `"A-B"`,
							Start:   token.Position{Line: 1, Column: 1},
							End:     token.Position{Line: 1, Column: 5},
						},
					},
				},
				{
					in: `"Helvetica,Arial,sans-serif"`,
					want: []token.Token{
						{
							Type:    token.ID,
							Literal: `"Helvetica,Arial,sans-serif"`,
							Start:   token.Position{Line: 1, Column: 1},
							End:     token.Position{Line: 1, Column: 28},
						},
					},
				},
				{
					in: `"#00008844"`,
					want: []token.Token{
						{
							Type:    token.ID,
							Literal: `"#00008844"`,
							Start:   token.Position{Line: 1, Column: 1},
							End:     token.Position{Line: 1, Column: 11},
						},
					},
				},
				{
					in: `"color\
#00008844"`,
					want: []token.Token{
						{
							Type: token.ID,
							Literal: `"color\
#00008844"`,
							Start: token.Position{Line: 1, Column: 1},
							End:   token.Position{Line: 2, Column: 10},
						},
					},
				},
				// this is not legal according to https://graphviz.org/doc/info/lang.html#ids but actually
				// supported by the dot tooling (this does not work in
				// https://magjac.com/graphviz-visual-editor maybe it uses an older version of dot. It might
				// also not an official site)
				{
					in: `"color
#00008844"`,
					want: []token.Token{
						{
							Type: token.ID,
							Literal: `"color
#00008844"`,
							Start: token.Position{Line: 1, Column: 1},
							End:   token.Position{Line: 2, Column: 10},
						},
					},
				},
				{
					in: `"emoji 🎉 test"`,
					want: []token.Token{
						{
							Type:    token.ID,
							Literal: `"emoji 🎉 test"`,
							Start:   token.Position{Line: 1, Column: 1},
							End:     token.Position{Line: 1, Column: 17}, // " (1) + emoji (5) + space (1) + 🎉 (4) + space (1) + test (4) + " (1) = 17
						},
					},
				},
				{
					in: `"unicode: éñ中文"`,
					want: []token.Token{
						{
							Type:    token.ID,
							Literal: `"unicode: éñ中文"`,
							Start:   token.Position{Line: 1, Column: 1},
							End:     token.Position{Line: 1, Column: 21}, // " (1) + unicode: (9) + space (1) + é (2) + ñ (2) + 中 (3) + 文 (3) + " (1) = 21? Let me recalc
						},
					},
				},
			}

			for i, test := range tests {
				t.Run(strconv.Itoa(i), func(t *testing.T) {
					scanner := NewScanner([]byte(test.in))
					assertTokens(t, scanner, test.want)
				})
			}
		})

		t.Run("Invalid", func(t *testing.T) {
			tests := []struct {
				in   string
				want []token.Token
			}{
				{
					in: `"asdf`,
					want: []token.Token{
						{
							Type:    token.ERROR,
							Literal: `"asdf`,
							Error:   "invalid character '\"': unclosed ID: missing closing '\"'",
							Start:   token.Position{Line: 1, Column: 1},
							End:     token.Position{Line: 1, Column: 5},
						},
					},
				},
				{
					in: `"asdf
		}`,
					want: []token.Token{
						{
							Type: token.ERROR,
							Literal: `"asdf
		}`,
							Error: "invalid character '\"': unclosed ID: missing closing '\"'",
							Start: token.Position{Line: 1, Column: 1},
							End:   token.Position{Line: 2, Column: 3},
						},
					},
				},
				{
					in: "\"node\x00with\x00nul\"",
					want: []token.Token{
						{
							Type:    token.ERROR,
							Literal: "\"node\x00with\x00nul\"",
							Error:   "invalid character U+0000: quoted IDs cannot contain null bytes",
							Start:   token.Position{Line: 1, Column: 1},
							End:     token.Position{Line: 1, Column: 15},
						},
					},
				},
				{
					in: `"a\"`,
					want: []token.Token{
						{
							Type:    token.ERROR,
							Literal: `"a\"`,
							Error:   "invalid character '\"': unclosed ID: missing closing '\"'",
							Start:   token.Position{Line: 1, Column: 1},
							End:     token.Position{Line: 1, Column: 4},
						},
					},
				},
			}

			for i, test := range tests {
				t.Run(strconv.Itoa(i), func(t *testing.T) {
					scanner := NewScanner([]byte(test.in))
					assertNext(t, scanner, test.want, test.in)
				})
			}
		})
	})

	// https://graphviz.org/doc/info/lang.html#comments-and-optional-formatting
	t.Run("Comments", func(t *testing.T) {
		t.Run("Valid", func(t *testing.T) {
			tests := []struct {
				in   string
				want token.Token
			}{
				{
					in: `

							#  C preprocessor style comment "noidentifier" /* ignore this */ edge  `,
					want: token.Token{
						Type:    token.Comment,
						Literal: `#  C preprocessor style comment "noidentifier" /* ignore this */ edge  `,
						Start:   token.Position{Line: 3, Column: 8},
						End:     token.Position{Line: 3, Column: 78},
					},
				},
				{
					in: `
							//	C++ style line comment "noidentifier" /* ignore this */ edge
			`,
					want: token.Token{
						Type:    token.Comment,
						Literal: `//	C++ style line comment "noidentifier" /* ignore this */ edge`,
						Start:   token.Position{Line: 2, Column: 8},
						End:     token.Position{Line: 2, Column: 70},
					},
				},
				{
					in: ` /* C++ style multi-line comment "noidentifier" edge
					# don't treat this as a separate comment
					# don't treat this as a separate comment
					*\ sneaky
spacious
					*/
			`,
					want: token.Token{
						Type: token.Comment,
						Literal: `/* C++ style multi-line comment "noidentifier" edge
					# don't treat this as a separate comment
					# don't treat this as a separate comment
					*\ sneaky
spacious
					*/`,
						Start: token.Position{Line: 1, Column: 2},
						End:   token.Position{Line: 6, Column: 7},
					},
				},
				{
					in: `/* ** */`,
					want: token.Token{
						Type:    token.Comment,
						Literal: `/* ** */`,
						Start:   token.Position{Line: 1, Column: 1},
						End:     token.Position{Line: 1, Column: 8},
					},
				},
				{
					in: `/* * */`,
					want: token.Token{
						Type:    token.Comment,
						Literal: `/* * */`,
						Start:   token.Position{Line: 1, Column: 1},
						End:     token.Position{Line: 1, Column: 7},
					},
				},
				{
					in: `/* *
*/`,
					want: token.Token{
						Type: token.Comment,
						Literal: `/* *
*/`,
						Start: token.Position{Line: 1, Column: 1},
						End:   token.Position{Line: 2, Column: 2},
					},
				},
			}

			for i, test := range tests {
				t.Run(strconv.Itoa(i), func(t *testing.T) {
					scanner := NewScanner([]byte(test.in))
					assertTokens(t, scanner, []token.Token{test.want})
				})
			}
		})
		t.Run("Invalid", func(t *testing.T) {
			tests := []struct {
				in   string
				want []token.Token
			}{
				{
					in: "/ is not a valid comment",
					want: []token.Token{
						{
							Type:    token.ERROR,
							Literal: "/",
							Error:   "invalid character '/': use '//' (line) or '/*...*/' (block) for comments",
							Start:   token.Position{Line: 1, Column: 1},
							End:     token.Position{Line: 1, Column: 1},
						},
						{
							Type:    token.ID,
							Literal: "is",
							Start:   token.Position{Line: 1, Column: 3},
							End:     token.Position{Line: 1, Column: 4},
						},
						{
							Type:    token.ID,
							Literal: "not",
							Start:   token.Position{Line: 1, Column: 6},
							End:     token.Position{Line: 1, Column: 8},
						},
						{
							Type:    token.ID,
							Literal: "a",
							Start:   token.Position{Line: 1, Column: 10},
							End:     token.Position{Line: 1, Column: 10},
						},
						{
							Type:    token.ID,
							Literal: "valid",
							Start:   token.Position{Line: 1, Column: 12},
							End:     token.Position{Line: 1, Column: 16},
						},
						{
							Type:    token.ID,
							Literal: "comment",
							Start:   token.Position{Line: 1, Column: 18},
							End:     token.Position{Line: 1, Column: 24},
						},
					},
				},
				{
					in: "A/",
					want: []token.Token{
						{
							Type:    token.ID,
							Literal: "A",
							Start:   token.Position{Line: 1, Column: 1},
							End:     token.Position{Line: 1, Column: 1},
						},
						{
							Type:    token.ERROR,
							Literal: "/",
							Error:   "invalid character '/': use '//' (line) or '/*...*/' (block) for comments",
							Start:   token.Position{Line: 1, Column: 2},
							End:     token.Position{Line: 1, Column: 2},
						},
					},
				},
				{
					in: "/# is not a valid comment",
					want: []token.Token{
						{
							Type:    token.ERROR,
							Literal: "/",
							Error:   "invalid character '/': use '//' (line) or '/*...*/' (block) for comments",
							Start:   token.Position{Line: 1, Column: 1},
							End:     token.Position{Line: 1, Column: 1},
						},
						{
							Type:    token.Comment,
							Literal: "# is not a valid comment",
							Start:   token.Position{Line: 1, Column: 2},
							End:     token.Position{Line: 1, Column: 25},
						},
					},
				},
				{
					in: "/* is not a valid comment",
					want: []token.Token{
						{
							Type:    token.ERROR,
							Literal: "/* is not a valid comment",
							Error:   "invalid character '/': unclosed comment: missing '*/'",
							Start:   token.Position{Line: 1, Column: 1},
							End:     token.Position{Line: 1, Column: 25},
						},
					},
				},
			}

			for i, test := range tests {
				t.Run(strconv.Itoa(i), func(t *testing.T) {
					scanner := NewScanner([]byte(test.in))
					assertNext(t, scanner, test.want, test.in)
				})
			}
		})
	})
}

func assertTokens(t *testing.T, scanner *Scanner, want []token.Token) {
	t.Helper()

	for i, wantToken := range want {
		assertNextTokenf(t, scanner, wantToken, "Next() at i=%d", i)
	}
	assertEOF(t, scanner)
}

func assertNextTokenf(t *testing.T, scanner *Scanner, wantToken token.Token, format string, args ...any) {
	t.Helper()

	tok := scanner.Next()
	require.EqualValuesf(t, tok, wantToken, format, args)
}

func assertEOF(t *testing.T, scanner *Scanner) {
	t.Helper()

	tok := scanner.Next()
	assert.EqualValuesf(t, token.EOF, tok.Type, "Next()")
}

func assertNext(t *testing.T, scanner *Scanner, want []token.Token, input string) {
	t.Helper()

	for i, wantToken := range want {
		gotToken := scanner.Next()
		assert.EqualValuesf(t, gotToken, wantToken, "token at index %d for input %q", i, input)
	}

	// Verify EOF after all expected tokens
	eofToken := scanner.Next()
	assert.EqualValuesf(t, token.EOF, eofToken.Type, "EOF for input %q", input)
}

func TestError(t *testing.T) {
	tests := map[string]struct {
		err  Error
		want string
	}{
		"CommonPrintableAsciiCharacter": {
			err: Error{
				Pos: token.Position{Line: 1, Column: 8},
				Msg: "invalid character '@': unquoted IDs must start with a letter or underscore",
			},
			want: "1:8: invalid character '@': unquoted IDs must start with a letter or underscore",
		},
		"NullByte": {
			err: Error{
				Pos: token.Position{Line: 2, Column: 5},
				Msg: "invalid character U+0000: unquoted IDs cannot contain null bytes",
			},
			want: "2:5: invalid character U+0000: unquoted IDs cannot contain null bytes",
		},
		"TabControlCharacter": {
			err: Error{
				Pos: token.Position{Line: 1, Column: 10},
				Msg: "invalid character U+0009: unexpected character in ID",
			},
			want: "1:10: invalid character U+0009: unexpected character in ID",
		},
		"DelCharacter": {
			err: Error{
				Pos: token.Position{Line: 1, Column: 3},
				Msg: "invalid character U+007F: unquoted IDs must start with a letter or underscore",
			},
			want: "1:3: invalid character U+007F: unquoted IDs must start with a letter or underscore",
		},
		"NonBreakingSpace": {
			err: Error{
				Pos: token.Position{Line: 5, Column: 12},
				Msg: "invalid character U+00A0 '\u00a0': unexpected whitespace character",
			},
			want: "5:12: invalid character U+00A0 '\u00a0': unexpected whitespace character",
		},
		"CyrillicALooksLikeLatin": {
			err: Error{
				Pos: token.Position{Line: 3, Column: 7},
				Msg: "invalid character U+0410 'А': test ambiguous character",
			},
			want: "3:7: invalid character U+0410 'А': test ambiguous character",
		},
		"RegularDash": {
			err: Error{
				Pos: token.Position{Line: 1, Column: 9},
				Msg: "invalid character '-': must be followed by '-' or '>'",
			},
			want: "1:9: invalid character '-': must be followed by '-' or '>'",
		},
		"NegativeCharacterNoFormatting": {
			err: Error{
				Pos: token.Position{Line: 1, Column: 5},
				Msg: "unexpected EOF",
			},
			want: "1:5: unexpected EOF",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := tt.err.Error()
			assert.EqualValuesf(t, tt.want, got, "Error message for %s", name)
		})
	}
}
