package dot

import (
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
					Type:    token.Identifier,
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
					Type:    token.Identifier,
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
					Type:    token.Identifier,
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
					Type:    token.Identifier,
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
					Type:    token.Identifier,
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
					Type:    token.Identifier,
					Literal: "t",
					Start:   token.Position{Row: 2, Column: 16},
					End:     token.Position{Row: 2, Column: 16},
				},
				{
					Type:    token.Identifier,
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
					Type:    token.Identifier,
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
					Type:    token.Identifier,
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
					Type:    token.Identifier,
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
					Type:    token.Identifier,
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
					Type:    token.Identifier,
					Literal: "none",
					Start:   token.Position{Row: 5, Column: 22},
					End:     token.Position{Row: 5, Column: 25},
				},
				{
					Type:    token.Identifier,
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
					Type:    token.Identifier,
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
					Type:    token.Identifier,
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
					Type:    token.Identifier,
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
					Type:    token.Identifier,
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
					Type:    token.Identifier,
					Literal: "B",
					Start:   token.Position{Row: 1, Column: 9},
					End:     token.Position{Row: 1, Column: 9},
				},
				{
					Type:    token.Identifier,
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
					Type:    token.Identifier,
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
					Type:    token.Identifier,
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
					Type:    token.Identifier,
					Literal: `"F"`,
					Start:   token.Position{Row: 4, Column: 5},
					End:     token.Position{Row: 4, Column: 7},
				},
				{
					Type:    token.Identifier,
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
					Type:    token.Identifier,
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
					Type:    token.Identifier,
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
					Type:    token.Identifier,
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
					Type:    token.Identifier,
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
					Type:    token.Identifier,
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
					Type:    token.Identifier,
					Literal: "2",
					Start:   token.Position{Row: 1, Column: 6},
					End:     token.Position{Row: 1, Column: 6},
				},
				{
					Type:    token.Identifier,
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
					Type:    token.Identifier,
					Literal: "4",
					Start:   token.Position{Row: 1, Column: 12},
					End:     token.Position{Row: 1, Column: 12},
				},
				{
					Type:    token.Identifier,
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
					Type:    token.Identifier,
					Literal: "6",
					Start:   token.Position{Row: 1, Column: 18},
					End:     token.Position{Row: 1, Column: 18},
				},
				{
					Type:    token.Identifier,
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
					Type:    token.Identifier,
					Literal: "8",
					Start:   token.Position{Row: 1, Column: 24},
					End:     token.Position{Row: 1, Column: 24},
				},
				{
					Type:    token.Identifier,
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
					Type:    token.Identifier,
					Literal: "10",
					Start:   token.Position{Row: 1, Column: 30},
					End:     token.Position{Row: 1, Column: 31},
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
				want token.Token
			}{
				{
					in: "_A",
					want: token.Token{
						Type:    token.Identifier,
						Literal: "_A",
						Start:   token.Position{Row: 1, Column: 1},
						End:     token.Position{Row: 1, Column: 2},
					},
				},
				{
					in: "A_cZ",
					want: token.Token{
						Type:    token.Identifier,
						Literal: "A_cZ",
						Start:   token.Position{Row: 1, Column: 1},
						End:     token.Position{Row: 1, Column: 4},
					},
				},
				{
					in: "A10",
					want: token.Token{
						Type:    token.Identifier,
						Literal: "A10",
						Start:   token.Position{Row: 1, Column: 1},
						End:     token.Position{Row: 1, Column: 3},
					},
				},
				{
					in: "\u0080ÿ  ",
					want: token.Token{
						Type:    token.Identifier,
						Literal: "\u0080ÿ",
						Start:   token.Position{Row: 1, Column: 1},
						End:     token.Position{Row: 1, Column: 2},
					},
				},
				{
					in: `Контрагенты`,
					want: token.Token{
						Type:    token.Identifier,
						Literal: `Контрагенты`,
						Start:   token.Position{Row: 1, Column: 1},
						End:     token.Position{Row: 1, Column: 11},
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
				in        string
				wantToken token.Token
				wantErr   Error
			}{
				{
					in: "  \x7f", // \177
					wantToken: token.Token{
						Type: token.ILLEGAL, Literal: "\x7f",
						Start: token.Position{Row: 1, Column: 3},
						End:   token.Position{Row: 1, Column: 3},
					},
					wantErr: Error{
						LineNr:      1,
						CharacterNr: 3,
						Character:   '\177',
						Reason:      "unquoted identifiers must start with a letter or underscore, and can only contain letters, digits, and underscores",
					},
				},
				{
					in: "  _zab\x7fx", // \177
					wantToken: token.Token{
						Type: token.ILLEGAL, Literal: "\x7f",
						Start: token.Position{Row: 1, Column: 7},
						End:   token.Position{Row: 1, Column: 7},
					},
					wantErr: Error{
						LineNr:      1,
						CharacterNr: 7,
						Character:   '\177',
						Reason:      "unquoted identifiers can only contain letters, digits, and underscores",
					},
				},
				{
					in: "A\000B", // null byte
				wantToken: token.Token{
					Type: token.ILLEGAL, Literal: "\x00",
					Start: token.Position{Row: 1, Column: 2},
					End:   token.Position{Row: 1, Column: 2},
				},
					wantErr: Error{
						LineNr:      1,
						CharacterNr: 2,
						Character:   '\000',
						Reason:      "illegal character NUL: unquoted identifiers can only contain letters, digits, and underscores",
					},
				},
			}

			for i, test := range tests {
				t.Run(strconv.Itoa(i), func(t *testing.T) {
					scanner, err := NewScanner(strings.NewReader(test.in))

					require.NoErrorf(t, err, "NewScanner(%q)", test.in)

					assertErrorNew(t, scanner, test.wantToken, test.wantErr, test.in)
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
						Type:    token.Identifier,
						Literal: "-.9",
						Start:   token.Position{Row: 1, Column: 2},
						End:     token.Position{Row: 1, Column: 4},
					},
				},
				{
					in: "-0.13",
					want: token.Token{
						Type:    token.Identifier,
						Literal: "-0.13",
						Start:   token.Position{Row: 1, Column: 1},
						End:     token.Position{Row: 1, Column: 5},
					},
				},
				{
					in: "-0.",
					want: token.Token{
						Type:    token.Identifier,
						Literal: "-0.",
						Start:   token.Position{Row: 1, Column: 1},
						End:     token.Position{Row: 1, Column: 3},
					},
				},
				{
					in: "-92.58",
					want: token.Token{
						Type:    token.Identifier,
						Literal: "-92.58",
						Start:   token.Position{Row: 1, Column: 1},
						End:     token.Position{Row: 1, Column: 6},
					},
				},
				{
					in: "-92",
					want: token.Token{
						Type:    token.Identifier,
						Literal: "-92",
						Start:   token.Position{Row: 1, Column: 1},
						End:     token.Position{Row: 1, Column: 3},
					},
				},
				{
					in: ".13",
					want: token.Token{
						Type:    token.Identifier,
						Literal: ".13",
						Start:   token.Position{Row: 1, Column: 1},
						End:     token.Position{Row: 1, Column: 3},
					},
				},
				{
					in: "0.",
					want: token.Token{
						Type:    token.Identifier,
						Literal: "0.",
						Start:   token.Position{Row: 1, Column: 1},
						End:     token.Position{Row: 1, Column: 2},
					},
				},
				{
					in: "0.13",
					want: token.Token{
						Type:    token.Identifier,
						Literal: "0.13",
						Start:   token.Position{Row: 1, Column: 1},
						End:     token.Position{Row: 1, Column: 4},
					},
				},
				{
					in: "47",
					want: token.Token{
						Type:    token.Identifier,
						Literal: "47",
						Start:   token.Position{Row: 1, Column: 1},
						End:     token.Position{Row: 1, Column: 2},
					},
				},
				{
					in: "47.58",
					want: token.Token{
						Type:    token.Identifier,
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
				want Error
			}{
				{
					in: "-.1A",
					want: Error{
						LineNr:      1,
						CharacterNr: 4,
						Character:   'A',
						Reason:      "a numeral can optionally lead with a `-`, has to have at least one digit before or after a `.` which must only be followed by digits",
					},
				},
				{
					in: "1-20",
					want: Error{
						LineNr:      1,
						CharacterNr: 2,
						Character:   '-',
						Reason:      "a numeral can only be prefixed with a `-`",
					},
				},
				{
					in: ".13.4",
					want: Error{
						LineNr:      1,
						CharacterNr: 4,
						Character:   '.',
						Reason:      "a numeral can only have one `.` that is at least preceded or followed by digits",
					},
				},
				{
					in: "-.",
					want: Error{ // TODO I point the error past the EOF
						LineNr:      1,
						CharacterNr: 3,
						// Character:   '.',
						Reason: "a numeral must have at least one digit",
					},
				},
				{
					in: "\n. 0",
					want: Error{
						LineNr:      2,
						CharacterNr: 2,
						Character:   ' ',
						Reason:      "a numeral must have at least one digit",
					},
				},
				{
					in: "100\u00A0200", // non-breaking space between 100 and 200
					want: Error{
						LineNr:      1,
						CharacterNr: 4,
						Character:   160,
						Reason:      "a numeral can optionally lead with a `-`, has to have at least one digit before or after a `.` which must only be followed by digits",
					},
				},
				{
					in: "\n\n\n\t  - F",
					want: Error{
						LineNr:      4,
						CharacterNr: 5,
						Character:   ' ',
						Reason:      "a numeral must have at least one digit",
					},
				},
			}

			for i, test := range tests {
				t.Run(strconv.Itoa(i), func(t *testing.T) {
					scanner, err := NewScanner(strings.NewReader(test.in))

					require.NoErrorf(t, err, "NewScanner(%q)", test.in)

					assertError(t, scanner, test.want, test.in)
				})
			}
		})
	})

	t.Run("QuotedIdentifiers", func(t *testing.T) {
		t.Run("Valid", func(t *testing.T) {
			tests := []struct {
				in   string
				want token.Token
			}{
				{
					in: `"graph"`,
					want: token.Token{
						Type:    token.Identifier,
						Literal: `"graph"`,
						Start:   token.Position{Row: 1, Column: 1},
						End:     token.Position{Row: 1, Column: 7},
					},
				},
				{
					in: `"strict"`,
					want: token.Token{
						Type:    token.Identifier,
						Literal: `"strict"`,
						Start:   token.Position{Row: 1, Column: 1},
						End:     token.Position{Row: 1, Column: 8},
					},
				},
				{
					in: `"\"d"`,
					want: token.Token{
						Type:    token.Identifier,
						Literal: `"\"d"`,
						Start:   token.Position{Row: 1, Column: 1},
						End:     token.Position{Row: 1, Column: 5},
					},
				},
				{
					in: `"\nd"`,
					want: token.Token{
						Type:    token.Identifier,
						Literal: `"\nd"`,
						Start:   token.Position{Row: 1, Column: 1},
						End:     token.Position{Row: 1, Column: 5},
					},
				},
				{
					in: `"\\d"`,
					want: token.Token{
						Type:    token.Identifier,
						Literal: `"\\d"`,
						Start:   token.Position{Row: 1, Column: 1},
						End:     token.Position{Row: 1, Column: 5},
					},
				},
				{
					in: `"_A"`,
					want: token.Token{
						Type:    token.Identifier,
						Literal: `"_A"`,
						Start:   token.Position{Row: 1, Column: 1},
						End:     token.Position{Row: 1, Column: 4},
					},
				},
				{
					in: `"-.9"`,
					want: token.Token{
						Type:    token.Identifier,
						Literal: `"-.9"`,
						Start:   token.Position{Row: 1, Column: 1},
						End:     token.Position{Row: 1, Column: 5},
					},
				},
				{
					in: `"A--B"`,
					want: token.Token{
						Type:    token.Identifier,
						Literal: `"A--B"`,
						Start:   token.Position{Row: 1, Column: 1},
						End:     token.Position{Row: 1, Column: 6},
					},
				},
				{
					in: `"A->B"`,
					want: token.Token{
						Type:    token.Identifier,
						Literal: `"A->B"`,
						Start:   token.Position{Row: 1, Column: 1},
						End:     token.Position{Row: 1, Column: 6},
					},
				},
				{
					in: `"A-B"`,
					want: token.Token{
						Type:    token.Identifier,
						Literal: `"A-B"`,
						Start:   token.Position{Row: 1, Column: 1},
						End:     token.Position{Row: 1, Column: 5},
					},
				},
				{
					in: `"Helvetica,Arial,sans-serif"`,
					want: token.Token{
						Type:    token.Identifier,
						Literal: `"Helvetica,Arial,sans-serif"`,
						Start:   token.Position{Row: 1, Column: 1},
						End:     token.Position{Row: 1, Column: 28},
					},
				},
				{
					in: `"#00008844"`,
					want: token.Token{
						Type:    token.Identifier,
						Literal: `"#00008844"`,
						Start:   token.Position{Row: 1, Column: 1},
						End:     token.Position{Row: 1, Column: 11},
					},
				},
				{
					in: `"color\
#00008844"`,
					want: token.Token{
						Type: token.Identifier,
						Literal: `"color\
#00008844"`,
						Start: token.Position{Row: 1, Column: 1},
						End:   token.Position{Row: 2, Column: 10},
					},
				},
				// this is not legal according to https://graphviz.org/doc/info/lang.html#ids but actually
				// supported by the dot tooling (this does not work in
				// https://magjac.com/graphviz-visual-editor maybe it uses an older version of dot. It might
				// also not an official site)
				{
					in: `"color
#00008844"`,
					want: token.Token{
						Type: token.Identifier,
						Literal: `"color
#00008844"`,
						Start: token.Position{Row: 1, Column: 1},
						End:   token.Position{Row: 2, Column: 10},
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
				want Error
			}{
				{
					in: `"asdf`,
					want: Error{
						LineNr:      1,
						CharacterNr: 6,
						Character:   0,
						Reason:      "missing closing quote",
					},
				},
				{
					in: `"asdf	
		}`,
					want: Error{
						LineNr:      2,
						CharacterNr: 4,
						Character:   0,
						Reason:      "missing closing quote",
					},
				},
				{
					in: `"` + strings.Repeat("a", 16348),
					want: Error{
						LineNr:      1,
						CharacterNr: 16349,
						Character:   'a',
						Reason:      "potentially missing closing quote, found none after max 16348 characters",
					},
				},
			}

			for i, test := range tests {
				t.Run(strconv.Itoa(i), func(t *testing.T) {
					scanner, err := NewScanner(strings.NewReader(test.in))

					require.NoErrorf(t, err, "NewScanner(%q)", test.in)

					assertError(t, scanner, test.want, test.in)
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
						Literal: `//	C++ style line comment "noidentifier" /* ignore this */ edge `,
						Start:   token.Position{Row: 2, Column: 8},
						End:     token.Position{Row: 2, Column: 71},
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
				in        string
				want      *token.Token
				wantError Error
			}{
				{
					in: "/ is not a valid comment",
					wantError: Error{
						LineNr:      1,
						CharacterNr: 1,
						Character:   '/',
						Reason:      "missing '/' for single-line or a '*' for a multi-line comment",
					},
				},
				{
					in: "A/",
					want: &token.Token{
						Type:    token.Identifier,
						Literal: "A",
						Start: token.Position{
							Row:    1,
							Column: 1,
						},
						End: token.Position{
							Row:    1,
							Column: 1,
						},
					},
					wantError: Error{
						LineNr:      1,
						CharacterNr: 2,
						Character:   '/',
						Reason:      "missing '/' for single-line or a '*' for a multi-line comment",
					},
				},
				{
					in: "/# is not a valid comment",
					wantError: Error{
						LineNr:      1,
						CharacterNr: 1,
						Character:   '/',
						Reason:      "missing '/' for single-line or a '*' for a multi-line comment",
					},
				},
				{
					in: "/* is not a valid comment",
					wantError: Error{
						LineNr:      1,
						CharacterNr: 26,
						Character:   0,
						Reason:      "missing closing marker '*/' for multi-line comment",
					},
				},
			}

			for i, test := range tests {
				t.Run(strconv.Itoa(i), func(t *testing.T) {
					scanner, err := NewScanner(strings.NewReader(test.in))

					require.NoErrorf(t, err, "NewScanner(%q)", test.in)

					if test.want != nil {
						assertNextToken(t, scanner, *test.want)
					}

					assertError(t, scanner, test.wantError, test.in)
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

func assertNextToken(t *testing.T, scanner *Scanner, wantToken token.Token) {
	t.Helper()

	assertNextTokenf(t, scanner, wantToken, "Next()")
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

func assertErrorNew(t *testing.T, scanner *Scanner, wantToken token.Token, wantErr error, input string) {
	t.Helper()

	got, err := scanner.Next()

	assert.EqualValuesf(t, got, wantToken, "Next() for input %q", input)

	if wantErr != nil {
		gotErr, ok := err.(Error)
		assert.Truef(t, ok, "Next() for input %q wanted scanner.Error, instead got %v", input, err)
		if ok {
			assert.EqualValuesf(t, gotErr, wantErr, "Next() for input %q", input)
		}

		// TODO is this so that subsequent calls will always get the same error?
		_, err = scanner.Next()
		gotErr, ok = err.(Error)
		assert.Truef(t, ok, "Next() for input %q wanted scanner.Error, instead got %v", input, err)
		if ok {
			assert.EqualValuesf(t, gotErr, wantErr, "Next() for input %q", input)
		}
	} else {
		// TODO assert we did not get an error
	}
}

func assertError(t *testing.T, scanner *Scanner, want Error, input string) {
	t.Helper()

	tok, err := scanner.Next()

	var wantTok token.Token
	assert.EqualValuesf(t, tok, wantTok, "Next() for input %q", input)
	got, ok := err.(Error)
	assert.Truef(t, ok, "Next() for input %q wanted scanner.Error, instead got %v", input, err)
	if ok {
		assert.EqualValuesf(t, got, want, "Next() for input %q", input)
	}

	// TODO is this so that subsequent calls will always get the same error?
	_, err = scanner.Next()
	got, ok = err.(Error)
	assert.Truef(t, ok, "Next() for input %q wanted scanner.Error, instead got %v", input, err)
	if ok {
		assert.EqualValuesf(t, got, want, "Next() for input %q", input)
	}
}
