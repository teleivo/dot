package dot

import (
	"iter"
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
			in:   "",
			want: []token.Token{},
		},
		"OnlyWhitespace": {
			in:   "\t \n \t\t   ",
			want: []token.Token{},
		},
		"LiteralSingleCharacterTokens": {
			in: "{};=[],:",
			want: []token.Token{
				{Type: token.LeftBrace, Literal: "{"},
				{Type: token.RightBrace, Literal: "}"},
				{Type: token.Semicolon, Literal: ";"},
				{Type: token.Equal, Literal: "="},
				{Type: token.LeftBracket, Literal: "["},
				{Type: token.RightBracket, Literal: "]"},
				{Type: token.Comma, Literal: ","},
				{Type: token.Colon, Literal: ":"},
			},
		},
		"KeywordsAreCaseInsensitive": {
			in: " graph Graph strict  Strict\ndigraph\tDigraph Subgraph  subgraph Node node edge Edge \n \t ",
			want: []token.Token{
				{Type: token.Graph, Literal: "graph"},
				{Type: token.Graph, Literal: "Graph"},
				{Type: token.Strict, Literal: "strict"},
				{Type: token.Strict, Literal: "Strict"},
				{Type: token.Digraph, Literal: "digraph"},
				{Type: token.Digraph, Literal: "Digraph"},
				{Type: token.Subgraph, Literal: "Subgraph"},
				{Type: token.Subgraph, Literal: "subgraph"},
				{Type: token.Node, Literal: "Node"},
				{Type: token.Node, Literal: "node"},
				{Type: token.Edge, Literal: "edge"},
				{Type: token.Edge, Literal: "Edge"},
			},
		},
		"IdentifiersQuoted": { // https://graphviz.org/doc/info/lang.html#ids
			in: `"graph" "strict" "\"d" "_A" "-.9" "A--B" "A-B" "A->B" "Helvetica,Arial,sans-serif" "#00008844"`,
			want: []token.Token{
				{Type: token.Identifier, Literal: `"graph"`},
				{Type: token.Identifier, Literal: `"strict"`},
				{Type: token.Identifier, Literal: `"\"d"`},
				{Type: token.Identifier, Literal: `"_A"`},
				{Type: token.Identifier, Literal: `"-.9"`},
				{Type: token.Identifier, Literal: `"A--B"`},
				{Type: token.Identifier, Literal: `"A-B"`},
				{Type: token.Identifier, Literal: `"A->B"`},
				{Type: token.Identifier, Literal: `"Helvetica,Arial,sans-serif"`},
				{Type: token.Identifier, Literal: `"#00008844"`},
			},
		},
		"AttributeList": {
			in: `	graph [
				labelloc = t
				fontname = "Helvetica,Arial,sans-serif"
			]
						edge [arrowhead=none color="#00008844"]  `,
			want: []token.Token{
				{Type: token.Graph, Literal: "graph"},
				{Type: token.LeftBracket, Literal: "["},
				{Type: token.Identifier, Literal: "labelloc"},
				{Type: token.Equal, Literal: "="},
				{Type: token.Identifier, Literal: "t"},
				{Type: token.Identifier, Literal: "fontname"},
				{Type: token.Equal, Literal: "="},
				{Type: token.Identifier, Literal: `"Helvetica,Arial,sans-serif"`},
				{Type: token.RightBracket, Literal: "]"},
				{Type: token.Edge, Literal: "edge"},
				{Type: token.LeftBracket, Literal: "["},
				{Type: token.Identifier, Literal: "arrowhead"},
				{Type: token.Equal, Literal: "="},
				{Type: token.Identifier, Literal: "none"},
				{Type: token.Identifier, Literal: "color"},
				{Type: token.Equal, Literal: "="},
				{Type: token.Identifier, Literal: `"#00008844"`},
				{Type: token.RightBracket, Literal: "]"},
			},
		},
		"Subgraphs": {
			in: `  A -> {B C}
				D -- E
			subgraph {
			  rank = same; A; B; C;
			}`,
			want: []token.Token{
				{Type: token.Identifier, Literal: "A"},
				{Type: token.DirectedEgde, Literal: "->"},
				{Type: token.LeftBrace, Literal: "{"},
				{Type: token.Identifier, Literal: "B"},
				{Type: token.Identifier, Literal: "C"},
				{Type: token.RightBrace, Literal: "}"},
				{Type: token.Identifier, Literal: "D"},
				{Type: token.UndirectedEgde, Literal: "--"},
				{Type: token.Identifier, Literal: "E"},
				{Type: token.Subgraph, Literal: "subgraph"},
				{Type: token.LeftBrace, Literal: "{"},
				{Type: token.Identifier, Literal: "rank"},
				{Type: token.Equal, Literal: "="},
				{Type: token.Identifier, Literal: "same"},
				{Type: token.Semicolon, Literal: ";"},
				{Type: token.Identifier, Literal: "A"},
				{Type: token.Semicolon, Literal: ";"},
				{Type: token.Identifier, Literal: "B"},
				{Type: token.Semicolon, Literal: ";"},
				{Type: token.Identifier, Literal: "C"},
				{Type: token.Semicolon, Literal: ";"},
				{Type: token.RightBrace, Literal: "}"},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			lexer := New(strings.NewReader(test.in))

			got := make([]token.Token, 0, len(tests))
			for token, err := range lexer.All() {
				assert.NoError(t, err)
				got = append(got, token)
			}
			assert.EqualValuesf(t, got, test.want, "All(%q)", test.in)
		})
	}

	// TODO is there some other error case I would want to test?
	errorTests := map[string]struct {
		in   string
		errs []*LexError
	}{}

	for name, test := range errorTests {
		t.Run(name, func(t *testing.T) {
			lexer := New(strings.NewReader(test.in))

			var i int
			for _, err := range lexer.All() {
				if test.errs[i] == nil {
					assert.NoErrorf(t, err, "All(%q) at index %d", test.in, i)
				} else {
					got, ok := err.(LexError)
					require.Truef(t, ok, "All(%q) at index %d wanted LexError, instead got %q", test.in, i, err)
					assert.EqualValuesf(t, got, *test.errs[i], "All(%q) at index %d", test.in, i)
				}
				i++
			}
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
					in:   "_A",
					want: token.Token{Type: token.Identifier, Literal: "_A"},
				},
				{
					in:   "A_cZ",
					want: token.Token{Type: token.Identifier, Literal: "A_cZ"},
				},
				{
					in:   "A10",
					want: token.Token{Type: token.Identifier, Literal: "A10"},
				},
				{
					in:   `ÿ  `,
					want: token.Token{Type: token.Identifier, Literal: `ÿ`},
				},
			}

			for i, test := range tests {
				t.Run(strconv.Itoa(i), func(t *testing.T) {
					lexer := New(strings.NewReader(test.in))
					next, stop := iter.Pull2(lexer.All())
					defer stop()

					got, err, ok := next()

					assert.EqualValuesf(t, got, test.want, "All(%q)", test.in)
					assert.NoErrorf(t, err, "All(%q)", test.in)
					assert.Truef(t, ok, "All(%q)", test.in)

					_, _, ok = next()

					assert.Falsef(t, ok, "All(%q) want only one token", test.in)
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
			}

			for i, test := range tests {
				t.Run(strconv.Itoa(i), func(t *testing.T) {
					lexer := New(strings.NewReader(test.in))
					next, stop := iter.Pull2(lexer.All())
					defer stop()

					_, err, ok := next()

					got, ok := err.(LexError)
					require.Truef(t, ok, "All(%q) wanted LexError, instead got %q", test.in, err)
					assert.EqualValuesf(t, got, test.want, "All(%q)", test.in)
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
					in:   " -.9\t\n",
					want: token.Token{Type: token.Identifier, Literal: "-.9"},
				},
				{
					in:   "-0.13",
					want: token.Token{Type: token.Identifier, Literal: "-0.13"},
				},
				{
					in:   "-0.",
					want: token.Token{Type: token.Identifier, Literal: "-0."},
				},
				{
					in:   "-92.58",
					want: token.Token{Type: token.Identifier, Literal: "-92.58"},
				},
				{
					in:   "-92",
					want: token.Token{Type: token.Identifier, Literal: "-92"},
				},
				{
					in:   ".13",
					want: token.Token{Type: token.Identifier, Literal: ".13"},
				},
				{
					in:   "0.",
					want: token.Token{Type: token.Identifier, Literal: "0."},
				},
				{
					in:   "0.13",
					want: token.Token{Type: token.Identifier, Literal: "0.13"},
				},
				{
					in:   "47",
					want: token.Token{Type: token.Identifier, Literal: "47"},
				},
				{
					in:   "47.58",
					want: token.Token{Type: token.Identifier, Literal: "47.58"},
				},
			}

			for i, test := range tests {
				t.Run(strconv.Itoa(i), func(t *testing.T) {
					lexer := New(strings.NewReader(test.in))
					next, stop := iter.Pull2(lexer.All())
					defer stop()

					got, err, ok := next()

					assert.EqualValuesf(t, got, test.want, "All(%q)", test.in)
					assert.NoErrorf(t, err, "All(%q)", test.in)
					assert.Truef(t, ok, "All(%q)", test.in)

					_, _, ok = next()

					assert.Falsef(t, ok, "All(%q) want only one token", test.in)
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
					lexer := New(strings.NewReader(test.in))
					next, stop := iter.Pull2(lexer.All())
					defer stop()

					_, err, ok := next()

					got, ok := err.(LexError)
					require.Truef(t, ok, "All(%q) wanted LexError, instead got %q", test.in, err)
					assert.EqualValuesf(t, got, test.want, "All(%q)", test.in)
				})
			}
		})
	})
}
