package dot

import (
	"strconv"
	"strings"
	"testing"

	"github.com/teleivo/assertive/assert"
	"github.com/teleivo/assertive/require"
	"github.com/teleivo/dot/internal/token"
)

func TestLexer(t *testing.T) {
	tests := map[string]struct {
		in   string
		want []token.Token
		err  error
	}{
		"Empty": {
			in: "",
			want: []token.Token{
				{Type: token.EOF},
			},
		},
		"OnlyWhitespace": {
			in: "\t \n \t\t   ",
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
					Type: token.DirectedEgde, Literal: "->",
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
					Type: token.UndirectedEgde, Literal: "--",
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
					Type:    token.DirectedEgde,
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
					Type:    token.DirectedEgde,
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
					Type:    token.UndirectedEgde,
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
					Type:    token.UndirectedEgde,
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
					Type:    token.UndirectedEgde,
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
			lexer, err := NewLexer(strings.NewReader(test.in))

			require.NoErrorf(t, err, "NewLexer(%q)", test.in)

			assertTokens(t, lexer, test.want)
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
					in: `ÿ  `,
					want: token.Token{
						Type:    token.Identifier,
						Literal: `ÿ`,
						Start:   token.Position{Row: 1, Column: 1},
						End:     token.Position{Row: 1, Column: 2},
					},
				},
			}

			for i, test := range tests {
				t.Run(strconv.Itoa(i), func(t *testing.T) {
					lexer, err := NewLexer(strings.NewReader(test.in))

					require.NoErrorf(t, err, "NewLexer(%q)", test.in)

					assertTokens(t, lexer, []token.Token{test.want})
				})
			}
		})

		t.Run("Invalid", func(t *testing.T) {
			tests := []struct {
				in   string
				want LexError
			}{
				{
					in: "  ", // \177
					want: LexError{
						LineNr:      1,
						CharacterNr: 3,
						Character:   '',
						Reason:      `unquoted string identifiers can contain alphabetic ([a-zA-Z\200-\377]) characters, underscores ('_') or digits([0-9]), but not begin with a digit`,
					},
				},
				{
					in: "  _zabx", // \177
					want: LexError{
						LineNr:      1,
						CharacterNr: 7,
						Character:   '',
						Reason:      `unquoted string identifiers can contain alphabetic ([a-zA-Z\200-\377]) characters, underscores ('_') or digits([0-9]), but not begin with a digit`,
					},
				},
				{
					in: `Ā`, // Unicode character U+0100 = \400 which cannot be written as rune(\400) as its outside of Gos valid octal range
					want: LexError{
						LineNr:      1,
						CharacterNr: 1,
						Character:   'Ā',
						Reason:      `unquoted string identifiers can contain alphabetic ([a-zA-Z\200-\377]) characters, underscores ('_') or digits([0-9]), but not begin with a digit`,
					},
				},
				{
					in: `_Ā`, // Unicode character U+0100 = \400 which cannot be written as rune(\400) as its outside of Gos valid octal range
					want: LexError{
						LineNr:      1,
						CharacterNr: 2,
						Character:   'Ā',
						Reason:      `unquoted string identifiers can contain alphabetic ([a-zA-Z\200-\377]) characters, underscores ('_') or digits([0-9]), but not begin with a digit`,
					},
				},
				{
					in: "A\000B", // null byte
					want: LexError{
						LineNr:      1,
						CharacterNr: 2,
						Character:   '\000',
						Reason:      `unquoted string identifiers can contain alphabetic ([a-zA-Z\200-\377]) characters, underscores ('_') or digits([0-9]), but not begin with a digit`,
					},
				},
			}

			for i, test := range tests {
				t.Run(strconv.Itoa(i), func(t *testing.T) {
					lexer, err := NewLexer(strings.NewReader(test.in))

					require.NoErrorf(t, err, "NewLexer(%q)", test.in)

					assertLexError(t, lexer, test.want)
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
					lexer, err := NewLexer(strings.NewReader(test.in))

					require.NoErrorf(t, err, "NewLexer(%q)", test.in)

					assertTokens(t, lexer, []token.Token{test.want})
				})
			}
		})

		t.Run("Invalid", func(t *testing.T) {
			tests := []struct {
				in   string
				want LexError
			}{
				{
					in: "-.1A",
					want: LexError{
						LineNr:      1,
						CharacterNr: 4,
						Character:   'A',
						Reason:      "a numeral can optionally lead with a `-`, has to have at least one digit before or after a `.` which must only be followed by digits",
					},
				},
				{
					in: "1-20",
					want: LexError{
						LineNr:      1,
						CharacterNr: 2,
						Character:   '-',
						Reason:      "a numeral can only be prefixed with a `-`",
					},
				},
				{
					in: ".13.4",
					want: LexError{
						LineNr:      1,
						CharacterNr: 4,
						Character:   '.',
						Reason:      "a numeral can only have one `.` that is at least preceded or followed by digits",
					},
				},
				{
					in: "-.",
					want: LexError{ // TODO I point the error past the EOF
						LineNr:      1,
						CharacterNr: 3,
						// Character:   '.',
						Reason: "a numeral must have at least one digit",
					},
				},
				{
					in: "\n. 0",
					want: LexError{
						LineNr:      2,
						CharacterNr: 2,
						Character:   ' ',
						Reason:      "a numeral must have at least one digit",
					},
				},
				{
					in: `100 200 `, // non-breakig space \240 between 100 and 200
					want: LexError{
						LineNr:      1,
						CharacterNr: 4,
						Character:   ' ',
						Reason:      "a numeral can optionally lead with a `-`, has to have at least one digit before or after a `.` which must only be followed by digits",
					},
				},
				{
					in: "\n\n\n\t  - F",
					want: LexError{
						LineNr:      4,
						CharacterNr: 5,
						Character:   ' ',
						Reason:      "a numeral must have at least one digit",
					},
				},
			}

			for i, test := range tests {
				t.Run(strconv.Itoa(i), func(t *testing.T) {
					lexer, err := NewLexer(strings.NewReader(test.in))

					require.NoErrorf(t, err, "NewLexer(%q)", test.in)

					assertLexError(t, lexer, test.want)
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
			}

			for i, test := range tests {
				t.Run(strconv.Itoa(i), func(t *testing.T) {
					lexer, err := NewLexer(strings.NewReader(test.in))

					require.NoErrorf(t, err, "NewLexer(%q)", test.in)

					assertTokens(t, lexer, []token.Token{test.want})
				})
			}
		})

		t.Run("Invalid", func(t *testing.T) {
			tests := []struct {
				in   string
				want LexError
			}{
				{
					in: `"asdf`,
					want: LexError{
						LineNr:      1,
						CharacterNr: 6,
						Character:   0,
						Reason:      "missing closing quote",
					},
				},
				{
					in: `"asdf	
		}`,
					want: LexError{
						LineNr:      2,
						CharacterNr: 4,
						Character:   0,
						Reason:      "missing closing quote",
					},
				},
				{
					in: `"` + strings.Repeat("a", 16348),
					want: LexError{
						LineNr:      1,
						CharacterNr: 16349,
						Character:   'a',
						Reason:      "potentially missing closing quote, found none after max 16348 characters",
					},
				},
			}

			for i, test := range tests {
				t.Run(strconv.Itoa(i), func(t *testing.T) {
					lexer, err := NewLexer(strings.NewReader(test.in))

					require.NoErrorf(t, err, "NewLexer(%q)", test.in)

					assertLexError(t, lexer, test.want)
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
					lexer, err := NewLexer(strings.NewReader(test.in))

					require.NoErrorf(t, err, "NewLexer(%q)", test.in)

					assertTokens(t, lexer, []token.Token{test.want})
				})
			}
		})
		t.Run("Invalid", func(t *testing.T) {
			tests := []struct {
				in   string
				want LexError
			}{
				{
					in: "/ is not a valid comment",
					want: LexError{
						LineNr:      1,
						CharacterNr: 1,
						Character:   '/',
						Reason:      "missing '/' for single-line or a '*' for a multi-line comment",
					},
				},
				{
					in: "/# is not a valid comment",
					want: LexError{
						LineNr:      1,
						CharacterNr: 1,
						Character:   '/',
						Reason:      "missing '/' for single-line or a '*' for a multi-line comment",
					},
				},
				{
					in: "/* is not a valid comment",
					want: LexError{
						LineNr:      1,
						CharacterNr: 26,
						Character:   0,
						Reason:      "missing closing marker '*/' for multi-line comment",
					},
				},
			}

			for i, test := range tests {
				t.Run(strconv.Itoa(i), func(t *testing.T) {
					lexer, err := NewLexer(strings.NewReader(test.in))

					require.NoErrorf(t, err, "NewLexer(%q)", test.in)

					assertLexError(t, lexer, test.want)
				})
			}
		})
	})
}

func assertTokens(t *testing.T, lexer *Lexer, want []token.Token) {
	t.Helper()

	for i, wantTok := range want {
		tok, err := lexer.NextToken()

		require.NoErrorf(t, err, "NextToken() at i=%d", i)
		require.EqualValuesf(t, tok, wantTok, "NextToken() at i=%d", i)
	}
	assertEOF(t, lexer)
}

func assertEOF(t *testing.T, lexer *Lexer) {
	t.Helper()

	tok, err := lexer.NextToken()

	assert.NoErrorf(t, err, "NextToken()")
	assert.EqualValuesf(t, tok, token.Token{Type: token.EOF}, "NextToken()")
}

func assertLexError(t *testing.T, lexer *Lexer, want LexError) {
	t.Helper()

	tok, err := lexer.NextToken()

	var wantTok token.Token
	assert.EqualValuesf(t, tok, wantTok, "NextToken()")
	got, ok := err.(LexError)
	assert.Truef(t, ok, "NextToken() wanted LexError, instead got %v", err)
	if ok {
		assert.EqualValuesf(t, got, want, "NextToken()")
	}

	// TODO is this so that subsequent calls will always get the same error?
	_, err = lexer.NextToken()
	got, ok = err.(LexError)
	assert.Truef(t, ok, "NextToken() wanted LexError, instead got %v", err)
	if ok {
		assert.EqualValuesf(t, got, want, "NextToken()")
	}
}
