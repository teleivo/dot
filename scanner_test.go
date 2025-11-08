package dot

import (
	"errors"
	"strconv"
	"strings"
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
				{Type: token.EOF},
			},
		},
		"SingleCharacter": {
			in: "a",
			want: []token.Token{
				{
					Type: token.ID, Literal: "a",
					Start: token.Position{Row: 1, Column: 1},
					End:   token.Position{Row: 1, Column: 1},
				},
			},
		},
		"OnlyWhitespace": {
			in: "\t \n \t\t   \r\n",
			want: []token.Token{
				{Type: token.EOF},
			},
		},
		"LiteralSingleCharacterTokens": {
			in: "{};=[],:",
			want: []token.Token{
				{
					Type: token.LeftBrace, Literal: "{",
					Start: token.Position{Row: 1, Column: 1},
					End:   token.Position{Row: 1, Column: 1},
				},
				{
					Type: token.RightBrace, Literal: "}",
					Start: token.Position{Row: 1, Column: 2},
					End:   token.Position{Row: 1, Column: 2},
				},
				{
					Type: token.Semicolon, Literal: ";",
					Start: token.Position{Row: 1, Column: 3},
					End:   token.Position{Row: 1, Column: 3},
				},
				{
					Type: token.Equal, Literal: "=",
					Start: token.Position{Row: 1, Column: 4},
					End:   token.Position{Row: 1, Column: 4},
				},
				{
					Type: token.LeftBracket, Literal: "[",
					Start: token.Position{Row: 1, Column: 5},
					End:   token.Position{Row: 1, Column: 5},
				},
				{
					Type: token.RightBracket, Literal: "]",
					Start: token.Position{Row: 1, Column: 6},
					End:   token.Position{Row: 1, Column: 6},
				},
				{
					Type: token.Comma, Literal: ",",
					Start: token.Position{Row: 1, Column: 7},
					End:   token.Position{Row: 1, Column: 7},
				},
				{
					Type: token.Colon, Literal: ":",
					Start: token.Position{Row: 1, Column: 8},
					End:   token.Position{Row: 1, Column: 8},
				},
				{Type: token.EOF},
			},
		},
		"KeywordsAreCaseInsensitive": {
			in: " graph Graph strict  Strict\ndigraph\tDigraph Subgraph  subgraph Node node edge Edge \n \t ",
			want: []token.Token{
				{
					Type:    token.Graph,
					Literal: "graph",
					Start:   token.Position{Row: 1, Column: 2},
					End:     token.Position{Row: 1, Column: 6},
				},
				{
					Type:    token.Graph,
					Literal: "Graph",
					Start:   token.Position{Row: 1, Column: 8},
					End:     token.Position{Row: 1, Column: 12},
				},
				{
					Type:    token.Strict,
					Literal: "strict",
					Start:   token.Position{Row: 1, Column: 14},
					End:     token.Position{Row: 1, Column: 19},
				},
				{
					Type:    token.Strict,
					Literal: "Strict",
					Start:   token.Position{Row: 1, Column: 22},
					End:     token.Position{Row: 1, Column: 27},
				},
				{
					Type:    token.Digraph,
					Literal: "digraph",
					Start:   token.Position{Row: 2, Column: 1},
					End:     token.Position{Row: 2, Column: 7},
				},
				{
					Type:    token.Digraph,
					Literal: "Digraph",
					Start:   token.Position{Row: 2, Column: 9},
					End:     token.Position{Row: 2, Column: 15},
				},
				{
					Type:    token.Subgraph,
					Literal: "Subgraph",
					Start:   token.Position{Row: 2, Column: 17},
					End:     token.Position{Row: 2, Column: 24},
				},
				{
					Type:    token.Subgraph,
					Literal: "subgraph",
					Start:   token.Position{Row: 2, Column: 27},
					End:     token.Position{Row: 2, Column: 34},
				},
				{
					Type:    token.Node,
					Literal: "Node",
					Start:   token.Position{Row: 2, Column: 36},
					End:     token.Position{Row: 2, Column: 39},
				},
				{
					Type:    token.Node,
					Literal: "node",
					Start:   token.Position{Row: 2, Column: 41},
					End:     token.Position{Row: 2, Column: 44},
				},
				{
					Type:    token.Edge,
					Literal: "edge",
					Start:   token.Position{Row: 2, Column: 46},
					End:     token.Position{Row: 2, Column: 49},
				},
				{
					Type:    token.Edge,
					Literal: "Edge",
					Start:   token.Position{Row: 2, Column: 51},
					End:     token.Position{Row: 2, Column: 54},
				},
				{Type: token.EOF},
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
					Start:   token.Position{Row: 1, Column: 1},
					End:     token.Position{Row: 1, Column: 1},
				},
				{
					Type:    token.Comment,
					Literal: `//commenting on A`,
					Start:   token.Position{Row: 1, Column: 2},
					End:     token.Position{Row: 1, Column: 18},
				},
				{
					Type:    token.ID,
					Literal: "B",
					Start:   token.Position{Row: 2, Column: 4},
					End:     token.Position{Row: 2, Column: 4},
				},
				{
					Type:    token.Comment,
					Literal: `#commenting on B`,
					Start:   token.Position{Row: 2, Column: 5},
					End:     token.Position{Row: 2, Column: 20},
				},
				{
					Type:    token.ID,
					Literal: `"C"`,
					Start:   token.Position{Row: 3, Column: 1},
					End:     token.Position{Row: 3, Column: 3},
				},
				{
					Type:    token.Comment,
					Literal: `//commenting on C`,
					Start:   token.Position{Row: 3, Column: 4},
					End:     token.Position{Row: 3, Column: 20},
				},
				{
					Type:    token.ID,
					Literal: `"D"`,
					Start:   token.Position{Row: 4, Column: 1},
					End:     token.Position{Row: 4, Column: 3},
				},
				{
					Type:    token.Comment,
					Literal: `#commenting on D`,
					Start:   token.Position{Row: 4, Column: 4},
					End:     token.Position{Row: 4, Column: 19},
				},
				{Type: token.EOF},
			},
		},
		"EmptyQuotedIdentifier": {
			in: `""`,
			want: []token.Token{
				{
					Type:    token.ID,
					Literal: `""`,
					Start:   token.Position{Row: 1, Column: 1},
					End:     token.Position{Row: 1, Column: 2},
				},
				{Type: token.EOF},
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
					Start:   token.Position{Row: 2, Column: 6},
					End:     token.Position{Row: 2, Column: 6},
				},
				{
					Type: token.Semicolon, Literal: ";",
					Start: token.Position{Row: 2, Column: 7},
					End:   token.Position{Row: 2, Column: 7},
				},
				{
					Type:    token.ID,
					Literal: "B",
					Start:   token.Position{Row: 2, Column: 8},
					End:     token.Position{Row: 2, Column: 8},
				},
				{
					Type: token.Semicolon, Literal: ";",
					Start: token.Position{Row: 2, Column: 9},
					End:   token.Position{Row: 2, Column: 9},
				},
				{
					Type:    token.ID,
					Literal: "C",
					Start:   token.Position{Row: 2, Column: 10},
					End:     token.Position{Row: 2, Column: 10},
				},
				{
					Type:    token.ID,
					Literal: `"D"`,
					Start:   token.Position{Row: 2, Column: 11},
					End:     token.Position{Row: 2, Column: 13},
				},
				{
					Type:    token.ID,
					Literal: `"E"`,
					Start:   token.Position{Row: 2, Column: 14},
					End:     token.Position{Row: 2, Column: 16},
				},
				{
					Type: token.RightBrace, Literal: "}",
					Start: token.Position{Row: 3, Column: 4},
					End:   token.Position{Row: 3, Column: 4},
				},
				{Type: token.EOF},
			},
		},
		"UnquotedQuotedUnquotedSandwich": {
			in: `A"B"C`,
			want: []token.Token{
				{
					Type:    token.ID,
					Literal: "A",
					Start:   token.Position{Row: 1, Column: 1},
					End:     token.Position{Row: 1, Column: 1},
				},
				{
					Type:    token.ID,
					Literal: `"B"`,
					Start:   token.Position{Row: 1, Column: 2},
					End:     token.Position{Row: 1, Column: 4},
				},
				{
					Type:    token.ID,
					Literal: "C",
					Start:   token.Position{Row: 1, Column: 5},
					End:     token.Position{Row: 1, Column: 5},
				},
				{Type: token.EOF},
			},
		},
		"QuotedFollowedByUnquoted": {
			in: `"A"_B`,
			want: []token.Token{
				{
					Type:    token.ID,
					Literal: `"A"`,
					Start:   token.Position{Row: 1, Column: 1},
					End:     token.Position{Row: 1, Column: 3},
				},
				{
					Type:    token.ID,
					Literal: "_B",
					Start:   token.Position{Row: 1, Column: 4},
					End:     token.Position{Row: 1, Column: 5},
				},
				{Type: token.EOF},
			},
		},
		"UnquotedFollowedByQuoted": {
			in: `A_1"B"`,
			want: []token.Token{
				{
					Type:    token.ID,
					Literal: "A_1",
					Start:   token.Position{Row: 1, Column: 1},
					End:     token.Position{Row: 1, Column: 3},
				},
				{
					Type:    token.ID,
					Literal: `"B"`,
					Start:   token.Position{Row: 1, Column: 4},
					End:     token.Position{Row: 1, Column: 6},
				},
				{Type: token.EOF},
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
					Start:   token.Position{Row: 1, Column: 2},
					End:     token.Position{Row: 1, Column: 6},
				},
				{
					Type: token.LeftBracket, Literal: "[",
					Start: token.Position{Row: 1, Column: 8},
					End:   token.Position{Row: 1, Column: 8},
				},
				{
					Type:    token.ID,
					Literal: "labelloc",
					Start:   token.Position{Row: 2, Column: 5},
					End:     token.Position{Row: 2, Column: 12},
				},
				{
					Type: token.Equal, Literal: "=",
					Start: token.Position{Row: 2, Column: 14},
					End:   token.Position{Row: 2, Column: 14},
				},
				{
					Type:    token.ID,
					Literal: "t",
					Start:   token.Position{Row: 2, Column: 16},
					End:     token.Position{Row: 2, Column: 16},
				},
				{
					Type:    token.ID,
					Literal: "fontname",
					Start:   token.Position{Row: 3, Column: 5},
					End:     token.Position{Row: 3, Column: 12},
				},
				{
					Type: token.Equal, Literal: "=",
					Start: token.Position{Row: 3, Column: 14},
					End:   token.Position{Row: 3, Column: 14},
				},
				{
					Type:    token.ID,
					Literal: `"Helvetica,Arial,sans-serif"`,
					Start:   token.Position{Row: 3, Column: 16},
					End:     token.Position{Row: 3, Column: 43},
				},
				{
					Type: token.Comma, Literal: ",",
					Start: token.Position{Row: 3, Column: 44},
					End:   token.Position{Row: 3, Column: 44},
				},
				{
					Type:    token.ID,
					Literal: "fontsize",
					Start:   token.Position{Row: 3, Column: 45},
					End:     token.Position{Row: 3, Column: 52},
				},
				{
					Type: token.Equal, Literal: "=",
					Start: token.Position{Row: 3, Column: 53},
					End:   token.Position{Row: 3, Column: 53},
				},
				{
					Type:    token.ID,
					Literal: "16",
					Start:   token.Position{Row: 3, Column: 54},
					End:     token.Position{Row: 3, Column: 55},
				},
				{
					Type: token.RightBracket, Literal: "]",
					Start: token.Position{Row: 4, Column: 4},
					End:   token.Position{Row: 4, Column: 4},
				},
				{
					Type:    token.Edge,
					Literal: "edge",
					Start:   token.Position{Row: 5, Column: 6},
					End:     token.Position{Row: 5, Column: 9},
				},
				{
					Type: token.LeftBracket, Literal: "[",
					Start: token.Position{Row: 5, Column: 11},
					End:   token.Position{Row: 5, Column: 11},
				},
				{
					Type:    token.ID,
					Literal: "arrowhead",
					Start:   token.Position{Row: 5, Column: 12},
					End:     token.Position{Row: 5, Column: 20},
				},
				{
					Type: token.Equal, Literal: "=",
					Start: token.Position{Row: 5, Column: 21},
					End:   token.Position{Row: 5, Column: 21},
				},
				{
					Type:    token.ID,
					Literal: "none",
					Start:   token.Position{Row: 5, Column: 22},
					End:     token.Position{Row: 5, Column: 25},
				},
				{
					Type:    token.ID,
					Literal: "color",
					Start:   token.Position{Row: 5, Column: 27},
					End:     token.Position{Row: 5, Column: 31},
				},
				{
					Type: token.Equal, Literal: "=",
					Start: token.Position{Row: 5, Column: 32},
					End:   token.Position{Row: 5, Column: 32},
				},
				{
					Type:    token.ID,
					Literal: `"#00008844"`,
					Start:   token.Position{Row: 5, Column: 33},
					End:     token.Position{Row: 5, Column: 43},
				},
				{
					Type: token.Comma, Literal: ",",
					Start: token.Position{Row: 5, Column: 44},
					End:   token.Position{Row: 5, Column: 44},
				},
				{
					Type:    token.ID,
					Literal: "style",
					Start:   token.Position{Row: 5, Column: 45},
					End:     token.Position{Row: 5, Column: 49},
				},
				{
					Type: token.Equal, Literal: "=",
					Start: token.Position{Row: 5, Column: 51},
					End:   token.Position{Row: 5, Column: 51},
				},
				{
					Type:    token.ID,
					Literal: "filled",
					Start:   token.Position{Row: 5, Column: 53},
					End:     token.Position{Row: 5, Column: 58},
				},
				{
					Type: token.RightBracket, Literal: "]",
					Start: token.Position{Row: 5, Column: 59},
					End:   token.Position{Row: 5, Column: 59},
				},
				{
					Type: token.Semicolon, Literal: ";",
					Start: token.Position{Row: 5, Column: 60},
					End:   token.Position{Row: 5, Column: 60},
				},
				{Type: token.EOF},
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
					Start:   token.Position{Row: 1, Column: 3},
					End:     token.Position{Row: 1, Column: 3},
				},
				{
					Type: token.DirectedEdge, Literal: "->",
					Start: token.Position{Row: 1, Column: 5},
					End:   token.Position{Row: 1, Column: 6},
				},
				{
					Type: token.LeftBrace, Literal: "{",
					Start: token.Position{Row: 1, Column: 8},
					End:   token.Position{Row: 1, Column: 8},
				},
				{
					Type:    token.ID,
					Literal: "B",
					Start:   token.Position{Row: 1, Column: 9},
					End:     token.Position{Row: 1, Column: 9},
				},
				{
					Type:    token.ID,
					Literal: "C",
					Start:   token.Position{Row: 1, Column: 11},
					End:     token.Position{Row: 1, Column: 11},
				},
				{
					Type: token.RightBrace, Literal: "}",
					Start: token.Position{Row: 1, Column: 12},
					End:   token.Position{Row: 1, Column: 12},
				},
				{
					Type:    token.ID,
					Literal: "D",
					Start:   token.Position{Row: 2, Column: 5},
					End:     token.Position{Row: 2, Column: 5},
				},
				{
					Type: token.UndirectedEdge, Literal: "--",
					Start: token.Position{Row: 2, Column: 7},
					End:   token.Position{Row: 2, Column: 8},
				},
				{
					Type:    token.ID,
					Literal: "E",
					Start:   token.Position{Row: 2, Column: 10},
					End:     token.Position{Row: 2, Column: 10},
				},
				{
					Type:    token.Subgraph,
					Literal: "subgraph",
					Start:   token.Position{Row: 3, Column: 4},
					End:     token.Position{Row: 3, Column: 11},
				},
				{
					Type: token.LeftBrace, Literal: "{",
					Start: token.Position{Row: 3, Column: 13},
					End:   token.Position{Row: 3, Column: 13},
				},
				{
					Type:    token.ID,
					Literal: `"F"`,
					Start:   token.Position{Row: 4, Column: 5},
					End:     token.Position{Row: 4, Column: 7},
				},
				{
					Type:    token.ID,
					Literal: "rank",
					Start:   token.Position{Row: 5, Column: 6},
					End:     token.Position{Row: 5, Column: 9},
				},
				{
					Type: token.Equal, Literal: "=",
					Start: token.Position{Row: 5, Column: 11},
					End:   token.Position{Row: 5, Column: 11},
				},
				{
					Type:    token.ID,
					Literal: "same",
					Start:   token.Position{Row: 5, Column: 13},
					End:     token.Position{Row: 5, Column: 16},
				},
				{
					Type: token.Semicolon, Literal: ";",
					Start: token.Position{Row: 5, Column: 17},
					End:   token.Position{Row: 5, Column: 17},
				},
				{
					Type:    token.ID,
					Literal: "A",
					Start:   token.Position{Row: 5, Column: 19},
					End:     token.Position{Row: 5, Column: 19},
				},
				{
					Type: token.Semicolon, Literal: ";",
					Start: token.Position{Row: 5, Column: 20},
					End:   token.Position{Row: 5, Column: 20},
				},
				{
					Type:    token.ID,
					Literal: "B",
					Start:   token.Position{Row: 5, Column: 21},
					End:     token.Position{Row: 5, Column: 21},
				},
				{
					Type: token.Semicolon, Literal: ";",
					Start: token.Position{Row: 5, Column: 22},
					End:   token.Position{Row: 5, Column: 22},
				},
				{
					Type:    token.ID,
					Literal: "C",
					Start:   token.Position{Row: 5, Column: 23},
					End:     token.Position{Row: 5, Column: 23},
				},
				{
					Type: token.Semicolon, Literal: ";",
					Start: token.Position{Row: 5, Column: 24},
					End:   token.Position{Row: 5, Column: 24},
				},
				{
					Type: token.RightBrace, Literal: "}",
					Start: token.Position{Row: 6, Column: 4},
					End:   token.Position{Row: 6, Column: 4},
				},
				{Type: token.EOF},
			},
		},
		"EdgesWithNumeralOperands": {
			in: `  1->2 3 ->4  5--6 7-- 8 9 --10`,
			want: []token.Token{
				{
					Type:    token.ID,
					Literal: "1",
					Start:   token.Position{Row: 1, Column: 3},
					End:     token.Position{Row: 1, Column: 3},
				},
				{
					Type:    token.DirectedEdge,
					Literal: "->",
					Start:   token.Position{Row: 1, Column: 4},
					End:     token.Position{Row: 1, Column: 5},
				},
				{
					Type:    token.ID,
					Literal: "2",
					Start:   token.Position{Row: 1, Column: 6},
					End:     token.Position{Row: 1, Column: 6},
				},
				{
					Type:    token.ID,
					Literal: "3",
					Start:   token.Position{Row: 1, Column: 8},
					End:     token.Position{Row: 1, Column: 8},
				},
				{
					Type:    token.DirectedEdge,
					Literal: "->",
					Start:   token.Position{Row: 1, Column: 10},
					End:     token.Position{Row: 1, Column: 11},
				},
				{
					Type:    token.ID,
					Literal: "4",
					Start:   token.Position{Row: 1, Column: 12},
					End:     token.Position{Row: 1, Column: 12},
				},
				{
					Type:    token.ID,
					Literal: "5",
					Start:   token.Position{Row: 1, Column: 15},
					End:     token.Position{Row: 1, Column: 15},
				},
				{
					Type:    token.UndirectedEdge,
					Literal: "--",
					Start:   token.Position{Row: 1, Column: 16},
					End:     token.Position{Row: 1, Column: 17},
				},
				{
					Type:    token.ID,
					Literal: "6",
					Start:   token.Position{Row: 1, Column: 18},
					End:     token.Position{Row: 1, Column: 18},
				},
				{
					Type:    token.ID,
					Literal: "7",
					Start:   token.Position{Row: 1, Column: 20},
					End:     token.Position{Row: 1, Column: 20},
				},
				{
					Type:    token.UndirectedEdge,
					Literal: "--",
					Start:   token.Position{Row: 1, Column: 21},
					End:     token.Position{Row: 1, Column: 22},
				},
				{
					Type:    token.ID,
					Literal: "8",
					Start:   token.Position{Row: 1, Column: 24},
					End:     token.Position{Row: 1, Column: 24},
				},
				{
					Type:    token.ID,
					Literal: "9",
					Start:   token.Position{Row: 1, Column: 26},
					End:     token.Position{Row: 1, Column: 26},
				},
				{
					Type:    token.UndirectedEdge,
					Literal: "--",
					Start:   token.Position{Row: 1, Column: 28},
					End:     token.Position{Row: 1, Column: 29},
				},
				{
					Type:    token.ID,
					Literal: "10",
					Start:   token.Position{Row: 1, Column: 30},
					End:     token.Position{Row: 1, Column: 31},
				},
			},
		},
		"NumeralFollowedByLineComment": {
			in: "123#comment",
			want: []token.Token{
				{
					Type:    token.ID,
					Literal: "123",
					Start:   token.Position{Row: 1, Column: 1},
					End:     token.Position{Row: 1, Column: 3},
				},
				{
					Type:    token.Comment,
					Literal: "#comment",
					Start:   token.Position{Row: 1, Column: 4},
					End:     token.Position{Row: 1, Column: 11},
				},
			},
		},
		"NumeralFollowedBySingleLineComment": {
			in: "456//comment",
			want: []token.Token{
				{
					Type:    token.ID,
					Literal: "456",
					Start:   token.Position{Row: 1, Column: 1},
					End:     token.Position{Row: 1, Column: 3},
				},
				{
					Type:    token.Comment,
					Literal: "//comment",
					Start:   token.Position{Row: 1, Column: 4},
					End:     token.Position{Row: 1, Column: 12},
				},
			},
		},
		"UnquotedIDFollowedByUndirectedEdgeNoWhitespace": {
			in: "ab--cd",
			want: []token.Token{
				{
					Type:    token.ID,
					Literal: "ab",
					Start:   token.Position{Row: 1, Column: 1},
					End:     token.Position{Row: 1, Column: 2},
				},
				{
					Type:    token.UndirectedEdge,
					Literal: "--",
					Start:   token.Position{Row: 1, Column: 3},
					End:     token.Position{Row: 1, Column: 4},
				},
				{
					Type:    token.ID,
					Literal: "cd",
					Start:   token.Position{Row: 1, Column: 5},
					End:     token.Position{Row: 1, Column: 6},
				},
			},
		},
		"UnquotedIDFollowedByDirectedEdgeNoWhitespace": {
			in: "ab->cd",
			want: []token.Token{
				{
					Type:    token.ID,
					Literal: "ab",
					Start:   token.Position{Row: 1, Column: 1},
					End:     token.Position{Row: 1, Column: 2},
				},
				{
					Type:    token.DirectedEdge,
					Literal: "->",
					Start:   token.Position{Row: 1, Column: 3},
					End:     token.Position{Row: 1, Column: 4},
				},
				{
					Type:    token.ID,
					Literal: "cd",
					Start:   token.Position{Row: 1, Column: 5},
					End:     token.Position{Row: 1, Column: 6},
				},
			},
		},
		"NumeralFollowedByUndirectedEdgeNoWhitespace": {
			in: "12--34",
			want: []token.Token{
				{
					Type:    token.ID,
					Literal: "12",
					Start:   token.Position{Row: 1, Column: 1},
					End:     token.Position{Row: 1, Column: 2},
				},
				{
					Type:    token.UndirectedEdge,
					Literal: "--",
					Start:   token.Position{Row: 1, Column: 3},
					End:     token.Position{Row: 1, Column: 4},
				},
				{
					Type:    token.ID,
					Literal: "34",
					Start:   token.Position{Row: 1, Column: 5},
					End:     token.Position{Row: 1, Column: 6},
				},
			},
		},
		"NumeralFollowedByDirectedEdgeNoWhitespace": {
			in: "12->34",
			want: []token.Token{
				{
					Type:    token.ID,
					Literal: "12",
					Start:   token.Position{Row: 1, Column: 1},
					End:     token.Position{Row: 1, Column: 2},
				},
				{
					Type:    token.DirectedEdge,
					Literal: "->",
					Start:   token.Position{Row: 1, Column: 3},
					End:     token.Position{Row: 1, Column: 4},
				},
				{
					Type:    token.ID,
					Literal: "34",
					Start:   token.Position{Row: 1, Column: 5},
					End:     token.Position{Row: 1, Column: 6},
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			scanner, err := NewScanner(strings.NewReader(test.in))

			require.NoErrorf(t, err, "NewScanner(%q)", test.in)

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
							Start:   token.Position{Row: 1, Column: 1},
							End:     token.Position{Row: 1, Column: 2},
						},
					},
				},
				{
					in: "A_cZ",
					want: []token.Token{
						{
							Type:    token.ID,
							Literal: "A_cZ",
							Start:   token.Position{Row: 1, Column: 1},
							End:     token.Position{Row: 1, Column: 4},
						},
					},
				},
				{
					in: "A10",
					want: []token.Token{
						{
							Type:    token.ID,
							Literal: "A10",
							Start:   token.Position{Row: 1, Column: 1},
							End:     token.Position{Row: 1, Column: 3},
						},
					},
				},
				{
					in: "\u0080ÿ  ",
					want: []token.Token{
						{
							Type:    token.ID,
							Literal: "\u0080ÿ",
							Start:   token.Position{Row: 1, Column: 1},
							End:     token.Position{Row: 1, Column: 2},
						},
					},
				},
				{
					in: "Контрагенты",
					want: []token.Token{
						{
							Type:    token.ID,
							Literal: "Контрагенты",
							Start:   token.Position{Row: 1, Column: 1},
							End:     token.Position{Row: 1, Column: 11},
						},
					},
				},
			}

			for i, test := range tests {
				t.Run(strconv.Itoa(i), func(t *testing.T) {
					scanner, err := NewScanner(strings.NewReader(test.in))

					require.NoErrorf(t, err, "NewScanner(%q)", test.in)

					assertTokens(t, scanner, test.want)
				})
			}
		})

		t.Run("Invalid", func(t *testing.T) {
			tests := []struct {
				in   string
				want []struct {
					token token.Token
					err   error
				}
			}{
				{
					in: "  \x7f", // \177 - cannot start any token
					want: []struct {
						token token.Token
						err   error
					}{
						{
							token.Token{
								Type:    token.ERROR,
								Literal: "\x7f",
								Start:   token.Position{Row: 1, Column: 3},
								End:     token.Position{Row: 1, Column: 3},
							},
							Error{
								LineNr:      1,
								CharacterNr: 3,
								Character:   '\177',
								Reason:      "unquoted IDs must start with a letter or underscore",
							},
						},
					},
				},
				{
					in: "  _zab\x7fx", // \177 in middle - ERROR token consumes until separator
					want: []struct {
						token token.Token
						err   error
					}{
						{
							token.Token{
								Type:    token.ERROR,
								Literal: "_zab\x7fx",
								Start:   token.Position{Row: 1, Column: 3},
								End:     token.Position{Row: 1, Column: 8},
							},
							Error{
								LineNr:      1,
								CharacterNr: 7,
								Character:   '\177',
								Reason:      "unquoted IDs can only contain letters, digits, and underscores",
							},
						},
					},
				},
				{
					in: "A\000\000B", // null bytes within identifier - consume entire sequence as ERROR
					want: []struct {
						token token.Token
						err   error
					}{
						{
							token.Token{
								Type:    token.ERROR,
								Literal: "A\x00\x00B",
								Start:   token.Position{Row: 1, Column: 1},
								End:     token.Position{Row: 1, Column: 4},
							},
							Error{
								LineNr:      1,
								CharacterNr: 2,
								Character:   '\000',
								Reason:      "unquoted IDs cannot contain null bytes",
							},
						},
					},
				},
				{
					in: "A;x\000y{B", // null byte with adjacent chars grouped into ERROR, separated by terminals
					want: []struct {
						token token.Token
						err   error
					}{
						{
							token.Token{
								Type:    token.ID,
								Literal: "A",
								Start:   token.Position{Row: 1, Column: 1},
								End:     token.Position{Row: 1, Column: 1},
							},
							nil,
						},
						{
							token.Token{
								Type:    token.Semicolon,
								Literal: ";",
								Start:   token.Position{Row: 1, Column: 2},
								End:     token.Position{Row: 1, Column: 2},
							},
							nil,
						},
						{
							token.Token{
								Type:    token.ERROR,
								Literal: "x\x00y",
								Start:   token.Position{Row: 1, Column: 3},
								End:     token.Position{Row: 1, Column: 5},
							},
							Error{
								LineNr:      1,
								CharacterNr: 4,
								Character:   '\000',
								Reason:      "unquoted IDs cannot contain null bytes",
							},
						},
						{
							token.Token{
								Type:    token.LeftBrace,
								Literal: "{",
								Start:   token.Position{Row: 1, Column: 6},
								End:     token.Position{Row: 1, Column: 6},
							},
							nil,
						},
						{
							token.Token{
								Type:    token.ID,
								Literal: "B",
								Start:   token.Position{Row: 1, Column: 7},
								End:     token.Position{Row: 1, Column: 7},
							},
							nil,
						},
					},
				},
				{
					in: "@@@",
					want: []struct {
						token token.Token
						err   error
					}{
						{
							token.Token{
								Type:    token.ERROR,
								Literal: "@@@",
								Start:   token.Position{Row: 1, Column: 1},
								End:     token.Position{Row: 1, Column: 3},
							},
							Error{
								LineNr:      1,
								CharacterNr: 1,
								Character:   '@',
								Reason:      "unquoted IDs must start with a letter or underscore",
							},
						},
					},
				},
				{
					in: "abc@def$ghi",
					want: []struct {
						token token.Token
						err   error
					}{
						{
							token.Token{
								Type:    token.ERROR,
								Literal: "abc@def$ghi",
								Start:   token.Position{Row: 1, Column: 1},
								End:     token.Position{Row: 1, Column: 11},
							},
							Error{
								LineNr:      1,
								CharacterNr: 4,
								Character:   '@',
								Reason:      "unquoted IDs can only contain letters, digits, and underscores",
							},
						},
					},
				},
			}

			for i, test := range tests {
				t.Run(strconv.Itoa(i), func(t *testing.T) {
					scanner, err := NewScanner(strings.NewReader(test.in))
					require.NoErrorf(t, err, "NewScanner(%q)", test.in)

					assertNext(t, scanner, test.want, test.in)
				})
			}
		})
	})

	t.Run("EdgeOperators", func(t *testing.T) {
		t.Run("Invalid", func(t *testing.T) {
			tests := []struct {
				in   string
				want []struct {
					token token.Token
					err   error
				}
			}{
				{
					in: "a-b",
					want: []struct {
						token token.Token
						err   error
					}{
						{
							token.Token{
								Type:    token.ERROR,
								Literal: "a-b",
								Start:   token.Position{Row: 1, Column: 1},
								End:     token.Position{Row: 1, Column: 3},
							},
							Error{
								LineNr:      1,
								CharacterNr: 2,
								Character:   '-',
								Reason:      "use '--' (undirected) or '->' (directed) for edges, or quote the ID",
							},
						},
					},
				},
			}

			for i, test := range tests {
				t.Run(strconv.Itoa(i), func(t *testing.T) {
					scanner, err := NewScanner(strings.NewReader(test.in))
					require.NoErrorf(t, err, "NewScanner(%q)", test.in)

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
						Start:   token.Position{Row: 1, Column: 2},
						End:     token.Position{Row: 1, Column: 4},
					},
				},
				{
					in: "-0.13",
					want: token.Token{
						Type:    token.ID,
						Literal: "-0.13",
						Start:   token.Position{Row: 1, Column: 1},
						End:     token.Position{Row: 1, Column: 5},
					},
				},
				{
					in: "-0.",
					want: token.Token{
						Type:    token.ID,
						Literal: "-0.",
						Start:   token.Position{Row: 1, Column: 1},
						End:     token.Position{Row: 1, Column: 3},
					},
				},
				{
					in: "-92.58",
					want: token.Token{
						Type:    token.ID,
						Literal: "-92.58",
						Start:   token.Position{Row: 1, Column: 1},
						End:     token.Position{Row: 1, Column: 6},
					},
				},
				{
					in: "-92",
					want: token.Token{
						Type:    token.ID,
						Literal: "-92",
						Start:   token.Position{Row: 1, Column: 1},
						End:     token.Position{Row: 1, Column: 3},
					},
				},
				{
					in: ".13",
					want: token.Token{
						Type:    token.ID,
						Literal: ".13",
						Start:   token.Position{Row: 1, Column: 1},
						End:     token.Position{Row: 1, Column: 3},
					},
				},
				{
					in: "0.",
					want: token.Token{
						Type:    token.ID,
						Literal: "0.",
						Start:   token.Position{Row: 1, Column: 1},
						End:     token.Position{Row: 1, Column: 2},
					},
				},
				{
					in: "0.13",
					want: token.Token{
						Type:    token.ID,
						Literal: "0.13",
						Start:   token.Position{Row: 1, Column: 1},
						End:     token.Position{Row: 1, Column: 4},
					},
				},
				{
					in: "47",
					want: token.Token{
						Type:    token.ID,
						Literal: "47",
						Start:   token.Position{Row: 1, Column: 1},
						End:     token.Position{Row: 1, Column: 2},
					},
				},
				{
					in: "47.58",
					want: token.Token{
						Type:    token.ID,
						Literal: "47.58",
						Start:   token.Position{Row: 1, Column: 1},
						End:     token.Position{Row: 1, Column: 5},
					},
				},
			}

			for i, test := range tests {
				t.Run(strconv.Itoa(i), func(t *testing.T) {
					scanner, err := NewScanner(strings.NewReader(test.in))

					require.NoErrorf(t, err, "NewScanner(%q)", test.in)

					assertTokens(t, scanner, []token.Token{test.want})
				})
			}
		})

		t.Run("Invalid", func(t *testing.T) {
			tests := []struct {
				in   string
				want []struct {
					token token.Token
					err   error
				}
			}{
				{
					in: "-.1A",
					want: []struct {
						token token.Token
						err   error
					}{
						{
							token.Token{
								Type:    token.ERROR,
								Literal: "-.1A",
								Start:   token.Position{Row: 1, Column: 1},
								End:     token.Position{Row: 1, Column: 4},
							},
							Error{
								LineNr:      1,
								CharacterNr: 4,
								Character:   'A',
								Reason:      "a numeral can optionally lead with a `-`, has to have at least one digit before or after a `.` which must only be followed by digits",
							},
						},
					},
				},
				{
					in: "1-20",
					want: []struct {
						token token.Token
						err   error
					}{
						{
							token.Token{
								Type:    token.ERROR,
								Literal: "1-20",
								Start:   token.Position{Row: 1, Column: 1},
								End:     token.Position{Row: 1, Column: 4},
							},
							Error{
								LineNr:      1,
								CharacterNr: 2,
								Character:   '-',
								Reason:      "a numeral can only be prefixed with a `-`",
							},
						},
					},
				},
				{
					in: ".13.4",
					want: []struct {
						token token.Token
						err   error
					}{
						{
							token.Token{
								Type:    token.ERROR,
								Literal: ".13.4",
								Start:   token.Position{Row: 1, Column: 1},
								End:     token.Position{Row: 1, Column: 5},
							},
							Error{
								LineNr:      1,
								CharacterNr: 4,
								Character:   '.',
								Reason:      "a numeral can only have one `.` that is at least preceded or followed by digits",
							},
						},
					},
				},
				{
					in: "-.",
					want: []struct {
						token token.Token
						err   error
					}{
						{
							token.Token{
								Type:    token.ERROR,
								Literal: "-.",
								Start:   token.Position{Row: 1, Column: 1},
								End:     token.Position{Row: 1, Column: 2},
							},
							Error{
								LineNr:      1,
								CharacterNr: 1,
								Character:   -1,
								Reason:      "a numeral must have at least one digit",
							},
						},
					},
				},
				{
					in: "\n. 0",
					want: []struct {
						token token.Token
						err   error
					}{
						{
							token.Token{
								Type:    token.ERROR,
								Literal: ".",
								Start:   token.Position{Row: 2, Column: 1},
								End:     token.Position{Row: 2, Column: 1},
							},
							Error{
								LineNr:      2,
								CharacterNr: 1,
								Character:   ' ',
								Reason:      "a numeral must have at least one digit",
							},
						},
						{
							token.Token{
								Type:    token.ID,
								Literal: "0",
								Start:   token.Position{Row: 2, Column: 3},
								End:     token.Position{Row: 2, Column: 3},
							},
							nil,
						},
					},
				},
				{
					in: "100\u00A0200", // non-breaking space between 100 and 200
					want: []struct {
						token token.Token
						err   error
					}{
						{
							token.Token{
								Type:    token.ERROR,
								Literal: "100\u00A0200",
								Start:   token.Position{Row: 1, Column: 1},
								End:     token.Position{Row: 1, Column: 7},
							},
							Error{
								LineNr:      1,
								CharacterNr: 4,
								Character:   160,
								Reason:      "a numeral can optionally lead with a `-`, has to have at least one digit before or after a `.` which must only be followed by digits",
							},
						},
					},
				},
				{
					in: "\n\n\n\t  - F",
					want: []struct {
						token token.Token
						err   error
					}{
						{
							token.Token{
								Type:    token.ERROR,
								Literal: "-",
								Start:   token.Position{Row: 4, Column: 4},
								End:     token.Position{Row: 4, Column: 4},
							},
							Error{
								LineNr:      4,
								CharacterNr: 4,
								Character:   ' ',
								Reason:      "a numeral must have at least one digit",
							},
						},
						{
							token.Token{
								Type:    token.ID,
								Literal: "F",
								Start:   token.Position{Row: 4, Column: 6},
								End:     token.Position{Row: 4, Column: 6},
							},
							nil,
						},
					},
				},
				{
					in: "A---B",
					want: []struct {
						token token.Token
						err   error
					}{
						{
							token.Token{
								Type:    token.ID,
								Literal: "A",
								Start:   token.Position{Row: 1, Column: 1},
								End:     token.Position{Row: 1, Column: 1},
							},
							nil,
						},
						{
							token.Token{
								Type:    token.UndirectedEdge,
								Literal: "--",
								Start:   token.Position{Row: 1, Column: 2},
								End:     token.Position{Row: 1, Column: 3},
							},
							nil,
						},
						{
							token.Token{
								Type:    token.ERROR,
								Literal: "-B",
								Start:   token.Position{Row: 1, Column: 4},
								End:     token.Position{Row: 1, Column: 5},
							},
							Error{
								LineNr:      1,
								CharacterNr: 5,
								Character:   'B',
								Reason:      "not allowed after '-' in number: only digits and '.' are allowed",
							},
						},
					},
				},
				{
					in: "1.2.3abc",
					want: []struct {
						token token.Token
						err   error
					}{
						{
							token.Token{
								Type:    token.ERROR,
								Literal: "1.2.3abc",
								Start:   token.Position{Row: 1, Column: 1},
								End:     token.Position{Row: 1, Column: 8},
							},
							Error{
								LineNr:      1,
								CharacterNr: 4,
								Character:   '.',
								Reason:      "a numeral can only have one `.` that is at least preceded or followed by digits",
							},
						},
					},
				},
			}

			for i, test := range tests {
				t.Run(strconv.Itoa(i), func(t *testing.T) {
					scanner, err := NewScanner(strings.NewReader(test.in))
					require.NoErrorf(t, err, "NewScanner(%q)", test.in)

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
							Start:   token.Position{Row: 1, Column: 1},
							End:     token.Position{Row: 1, Column: 7},
						},
						{
							Type:    token.ID,
							Literal: `"strict"`,
							Start:   token.Position{Row: 1, Column: 8},
							End:     token.Position{Row: 1, Column: 15},
						},
					},
				},
				{
					in: `"\"d"`,
					want: []token.Token{
						{
							Type:    token.ID,
							Literal: `"\"d"`,
							Start:   token.Position{Row: 1, Column: 1},
							End:     token.Position{Row: 1, Column: 5},
						},
					},
				},
				{
					in: `"\nd"`,
					want: []token.Token{
						{
							Type:    token.ID,
							Literal: `"\nd"`,
							Start:   token.Position{Row: 1, Column: 1},
							End:     token.Position{Row: 1, Column: 5},
						},
					},
				},
				{
					in: `"\\d"`,
					want: []token.Token{
						{
							Type:    token.ID,
							Literal: `"\\d"`,
							Start:   token.Position{Row: 1, Column: 1},
							End:     token.Position{Row: 1, Column: 5},
						},
					},
				},
				{
					in: `"_A"`,
					want: []token.Token{
						{
							Type:    token.ID,
							Literal: `"_A"`,
							Start:   token.Position{Row: 1, Column: 1},
							End:     token.Position{Row: 1, Column: 4},
						},
					},
				},
				{
					in: `"-.9"`,
					want: []token.Token{
						{
							Type:    token.ID,
							Literal: `"-.9"`,
							Start:   token.Position{Row: 1, Column: 1},
							End:     token.Position{Row: 1, Column: 5},
						},
					},
				},
				{
					in: `"A--B"`,
					want: []token.Token{
						{
							Type:    token.ID,
							Literal: `"A--B"`,
							Start:   token.Position{Row: 1, Column: 1},
							End:     token.Position{Row: 1, Column: 6},
						},
					},
				},
				{
					in: `"A->B"`,
					want: []token.Token{
						{
							Type:    token.ID,
							Literal: `"A->B"`,
							Start:   token.Position{Row: 1, Column: 1},
							End:     token.Position{Row: 1, Column: 6},
						},
					},
				},
				{
					in: `"A-B"`,
					want: []token.Token{
						{
							Type:    token.ID,
							Literal: `"A-B"`,
							Start:   token.Position{Row: 1, Column: 1},
							End:     token.Position{Row: 1, Column: 5},
						},
					},
				},
				{
					in: `"Helvetica,Arial,sans-serif"`,
					want: []token.Token{
						{
							Type:    token.ID,
							Literal: `"Helvetica,Arial,sans-serif"`,
							Start:   token.Position{Row: 1, Column: 1},
							End:     token.Position{Row: 1, Column: 28},
						},
					},
				},
				{
					in: `"#00008844"`,
					want: []token.Token{
						{
							Type:    token.ID,
							Literal: `"#00008844"`,
							Start:   token.Position{Row: 1, Column: 1},
							End:     token.Position{Row: 1, Column: 11},
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
							Start: token.Position{Row: 1, Column: 1},
							End:   token.Position{Row: 2, Column: 10},
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
							Start: token.Position{Row: 1, Column: 1},
							End:   token.Position{Row: 2, Column: 10},
						},
					},
				},
				{
					in: `"emoji 🎉 test"`,
					want: []token.Token{
						{
							Type:    token.ID,
							Literal: `"emoji 🎉 test"`,
							Start:   token.Position{Row: 1, Column: 1},
							End:     token.Position{Row: 1, Column: 14},
						},
					},
				},
				{
					in: `"unicode: éñ中文"`,
					want: []token.Token{
						{
							Type:    token.ID,
							Literal: `"unicode: éñ中文"`,
							Start:   token.Position{Row: 1, Column: 1},
							End:     token.Position{Row: 1, Column: 15},
						},
					},
				},
			}

			for i, test := range tests {
				t.Run(strconv.Itoa(i), func(t *testing.T) {
					scanner, err := NewScanner(strings.NewReader(test.in))

					require.NoErrorf(t, err, "NewScanner(%q)", test.in)

					assertTokens(t, scanner, test.want)
				})
			}
		})

		t.Run("Invalid", func(t *testing.T) {
			tests := []struct {
				in   string
				want []struct {
					token token.Token
					err   error
				}
			}{
				{
					in: `"asdf`,
					want: []struct {
						token token.Token
						err   error
					}{
						{
							token.Token{
								Type:    token.ERROR,
								Literal: `"asdf`,
								Start:   token.Position{Row: 1, Column: 1},
								End:     token.Position{Row: 1, Column: 5},
							},
							Error{
								LineNr:      1,
								CharacterNr: 1,
								Character:   '"',
								Reason:      "missing closing quote",
							},
						},
					},
				},
				{
					in: `"asdf
		}`,
					want: []struct {
						token token.Token
						err   error
					}{
						{
							token.Token{
								Type: token.ERROR,
								Literal: `"asdf
		}`,
								Start: token.Position{Row: 1, Column: 1},
								End:   token.Position{Row: 2, Column: 3},
							},
							Error{
								LineNr:      1,
								CharacterNr: 1,
								Character:   '"',
								Reason:      "missing closing quote",
							},
						},
					},
				},
				{
					in: "\"node\x00with\x00nul\"",
					want: []struct {
						token token.Token
						err   error
					}{
						{
							token.Token{
								Type:    token.ERROR,
								Literal: "\"node\x00with\x00nul\"",
								Start:   token.Position{Row: 1, Column: 1},
								End:     token.Position{Row: 1, Column: 15},
							},
							Error{
								LineNr:      1,
								CharacterNr: 6,
								Character:   '\x00',
								Reason:      "quoted IDs cannot contain null bytes",
							},
						},
					},
				},
			}

			for i, test := range tests {
				t.Run(strconv.Itoa(i), func(t *testing.T) {
					scanner, err := NewScanner(strings.NewReader(test.in))
					require.NoErrorf(t, err, "NewScanner(%q)", test.in)

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
						Start:   token.Position{Row: 3, Column: 8},
						End:     token.Position{Row: 3, Column: 78},
					},
				},
				{
					in: `
							//	C++ style line comment "noidentifier" /* ignore this */ edge
			`,
					want: token.Token{
						Type:    token.Comment,
						Literal: `//	C++ style line comment "noidentifier" /* ignore this */ edge`,
						Start:   token.Position{Row: 2, Column: 8},
						End:     token.Position{Row: 2, Column: 70},
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
						Start: token.Position{Row: 1, Column: 2},
						End:   token.Position{Row: 6, Column: 7},
					},
				},
				{
					in: `/* ** */`,
					want: token.Token{
						Type:    token.Comment,
						Literal: `/* ** */`,
						Start:   token.Position{Row: 1, Column: 1},
						End:     token.Position{Row: 1, Column: 8},
					},
				},
				{
					in: `/* * */`,
					want: token.Token{
						Type:    token.Comment,
						Literal: `/* * */`,
						Start:   token.Position{Row: 1, Column: 1},
						End:     token.Position{Row: 1, Column: 7},
					},
				},
				{
					in: `/* *
*/`,
					want: token.Token{
						Type: token.Comment,
						Literal: `/* *
*/`,
						Start: token.Position{Row: 1, Column: 1},
						End:   token.Position{Row: 2, Column: 2},
					},
				},
			}

			for i, test := range tests {
				t.Run(strconv.Itoa(i), func(t *testing.T) {
					scanner, err := NewScanner(strings.NewReader(test.in))

					require.NoErrorf(t, err, "NewScanner(%q)", test.in)

					assertTokens(t, scanner, []token.Token{test.want})
				})
			}
		})
		t.Run("Invalid", func(t *testing.T) {
			tests := []struct {
				in   string
				want []struct {
					token token.Token
					err   error
				}
			}{
				{
					in: "/ is not a valid comment",
					want: []struct {
						token token.Token
						err   error
					}{
						{
							token.Token{
								Type:    token.ERROR,
								Literal: "/",
								Start:   token.Position{Row: 1, Column: 1},
								End:     token.Position{Row: 1, Column: 1},
							},
							Error{
								LineNr:      1,
								CharacterNr: 1,
								Character:   '/',
								Reason:      "use '//' (line) or '/*...*/' (block) for comments",
							},
						},
						{
							token.Token{
								Type:    token.ID,
								Literal: "is",
								Start:   token.Position{Row: 1, Column: 3},
								End:     token.Position{Row: 1, Column: 4},
							},
							nil,
						},
						{
							token.Token{
								Type:    token.ID,
								Literal: "not",
								Start:   token.Position{Row: 1, Column: 6},
								End:     token.Position{Row: 1, Column: 8},
							},
							nil,
						},
						{
							token.Token{
								Type:    token.ID,
								Literal: "a",
								Start:   token.Position{Row: 1, Column: 10},
								End:     token.Position{Row: 1, Column: 10},
							},
							nil,
						},
						{
							token.Token{
								Type:    token.ID,
								Literal: "valid",
								Start:   token.Position{Row: 1, Column: 12},
								End:     token.Position{Row: 1, Column: 16},
							},
							nil,
						},
						{
							token.Token{
								Type:    token.ID,
								Literal: "comment",
								Start:   token.Position{Row: 1, Column: 18},
								End:     token.Position{Row: 1, Column: 24},
							},
							nil,
						},
					},
				},
				{
					in: "A/",
					want: []struct {
						token token.Token
						err   error
					}{
						{
							token.Token{
								Type:    token.ID,
								Literal: "A",
								Start:   token.Position{Row: 1, Column: 1},
								End:     token.Position{Row: 1, Column: 1},
							},
							nil,
						},
						{
							token.Token{
								Type:    token.ERROR,
								Literal: "/",
								Start:   token.Position{Row: 1, Column: 2},
								End:     token.Position{Row: 1, Column: 2},
							},
							Error{
								LineNr:      1,
								CharacterNr: 2,
								Character:   '/',
								Reason:      "use '//' (line) or '/*...*/' (block) for comments",
							},
						},
					},
				},
				{
					in: "/# is not a valid comment",
					want: []struct {
						token token.Token
						err   error
					}{
						{
							token.Token{
								Type:    token.ERROR,
								Literal: "/",
								Start:   token.Position{Row: 1, Column: 1},
								End:     token.Position{Row: 1, Column: 1},
							},
							Error{
								LineNr:      1,
								CharacterNr: 1,
								Character:   '/',
								Reason:      "use '//' (line) or '/*...*/' (block) for comments",
							},
						},
						{
							token.Token{
								Type:    token.Comment,
								Literal: "# is not a valid comment",
								Start:   token.Position{Row: 1, Column: 2},
								End:     token.Position{Row: 1, Column: 25},
							},
							nil,
						},
					},
				},
				{
					in: "/* is not a valid comment",
					want: []struct {
						token token.Token
						err   error
					}{
						{
							token.Token{
								Type:    token.ERROR,
								Literal: "/* is not a valid comment",
								Start:   token.Position{Row: 1, Column: 1},
								End:     token.Position{Row: 1, Column: 25},
							},
							Error{
								LineNr:      1,
								CharacterNr: 1,
								Character:   '/',
								Reason:      "unclosed comment: missing '*/'",
							},
						},
					},
				},
			}

			for i, test := range tests {
				t.Run(strconv.Itoa(i), func(t *testing.T) {
					scanner, err := NewScanner(strings.NewReader(test.in))
					require.NoErrorf(t, err, "NewScanner(%q)", test.in)

					assertNext(t, scanner, test.want, test.in)
				})
			}
		})
	})
}

type errorReader struct {
	err error
}

func (r *errorReader) Read(p []byte) (n int, err error) {
	return 0, r.err
}

func TestNewScanner(t *testing.T) {
	t.Run("ReaderError", func(t *testing.T) {
		expectedErr := errors.New("disk read failure")
		reader := &errorReader{err: expectedErr}

		scanner, err := NewScanner(reader)

		require.Nil(t, scanner)
		require.NotNil(t, err)
		assert.True(t, strings.Contains(err.Error(), "failed to read character"))
		assert.Truef(t, strings.Contains(err.Error(), expectedErr.Error()), "error message should include underlying error, got: %v", err)
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

	tok, err := scanner.Next()

	require.NoErrorf(t, err, format, args...)
	require.EqualValuesf(t, tok, wantToken, format, args)
}

func assertEOF(t *testing.T, scanner *Scanner) {
	t.Helper()

	tok, err := scanner.Next()

	assert.NoErrorf(t, err, "Next()")
	assert.EqualValuesf(t, tok, token.Token{Type: token.EOF}, "Next()")
}

func assertNext(t *testing.T, scanner *Scanner, want []struct {
	token token.Token
	err   error
}, input string,
) {
	t.Helper()

	for i, wantPair := range want {
		gotToken, gotErr := scanner.Next()

		assert.EqualValuesf(t, gotToken, wantPair.token, "token at index %d for input %q", i, input)

		if wantPair.err != nil {
			gotScanErr, ok := gotErr.(Error)
			assert.Truef(t, ok, "Next() at index %d for input %q wanted scanner.Error, instead got %v", i, input, gotErr)
			if ok {
				assert.EqualValuesf(t, gotScanErr, wantPair.err, "error at index %d for input %q", i, input)
			}
		} else {
			assert.NoErrorf(t, gotErr, "Next() at index %d for input %q should not return an error", i, input)
		}
	}

	// Verify EOF after all expected tokens
	eofToken, eofErr := scanner.Next()
	assert.NoErrorf(t, eofErr, "EOF for input %q", input)
	assert.EqualValuesf(t, token.Token{Type: token.EOF}, eofToken, "EOF for input %q", input)
}

func TestError(t *testing.T) {
	tests := map[string]struct {
		err  Error
		want string
	}{
		"CommonPrintableAsciiCharacter": {
			err: Error{
				LineNr:      1,
				CharacterNr: 8,
				Character:   '@',
				Reason:      "unquoted IDs must start with a letter or underscore",
			},
			want: "1:8: invalid character '@': unquoted IDs must start with a letter or underscore",
		},
		"NullByte": {
			err: Error{
				LineNr:      2,
				CharacterNr: 5,
				Character:   '\x00',
				Reason:      "unquoted IDs cannot contain null bytes",
			},
			want: "2:5: invalid character U+0000: unquoted IDs cannot contain null bytes",
		},
		"TabControlCharacter": {
			err: Error{
				LineNr:      1,
				CharacterNr: 10,
				Character:   '\t',
				Reason:      "unexpected character in ID",
			},
			want: "1:10: invalid character U+0009: unexpected character in ID",
		},
		"DelCharacter": {
			err: Error{
				LineNr:      1,
				CharacterNr: 3,
				Character:   '\x7f',
				Reason:      "unquoted IDs must start with a letter or underscore",
			},
			want: "1:3: invalid character U+007F: unquoted IDs must start with a letter or underscore",
		},
		"NonBreakingSpace": {
			err: Error{
				LineNr:      5,
				CharacterNr: 12,
				Character:   '\u00A0',
				Reason:      "unexpected whitespace character",
			},
			want: "5:12: invalid character U+00A0: unexpected whitespace character",
		},
		"CyrillicALooksLikeLatin": {
			err: Error{
				LineNr:      3,
				CharacterNr: 7,
				Character:   '\u0410',
				Reason:      "test ambiguous character",
			},
			want: "3:7: invalid character U+0410 'А': test ambiguous character",
		},
		"RegularDash": {
			err: Error{
				LineNr:      1,
				CharacterNr: 9,
				Character:   '-',
				Reason:      "must be followed by '-' or '>'",
			},
			want: "1:9: invalid character '-': must be followed by '-' or '>'",
		},
		"NegativeCharacterNoFormatting": {
			err: Error{
				LineNr:      1,
				CharacterNr: 5,
				Character:   -1,
				Reason:      "unexpected EOF",
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
