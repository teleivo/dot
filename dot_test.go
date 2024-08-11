package dot_test

import (
	"strings"
	"testing"

	"github.com/teleivo/assertive/assert"
	"github.com/teleivo/assertive/require"
	"github.com/teleivo/dot"
)

func TestParser(t *testing.T) {
	t.Run("Header", func(t *testing.T) {
		tests := map[string]struct {
			in   string
			want dot.Graph
			err  error
		}{
			"Empty": {
				in: "",
				want: dot.Graph{
					Nodes: map[string]*dot.Node{},
				},
			},
			"EmptyDirectedGraph": {
				in: "digraph {}",
				want: dot.Graph{
					Directed: true,
					Nodes:    map[string]*dot.Node{},
				},
			},
			"EmptyUndirectedGraph": {
				in: "graph {}",
				want: dot.Graph{
					Nodes: map[string]*dot.Node{},
				},
			},
			"StrictDirectedUnnamedGraph": {
				in: `strict digraph {}`,
				want: dot.Graph{
					Strict:   true,
					Directed: true,
					Nodes:    map[string]*dot.Node{},
				},
			},
			"StrictDirectedNamedGraph": {
				in: `strict digraph dependencies {}`,
				want: dot.Graph{
					Strict:   true,
					Directed: true,
					ID:       "dependencies",
					Nodes:    map[string]*dot.Node{},
				},
			},
		}

		for name, test := range tests {
			t.Run(name, func(t *testing.T) {
				p, err := dot.New(strings.NewReader(test.in))

				require.NoErrorf(t, err, "New(%q)", test.in)

				g, err := p.Parse()

				assert.NoErrorf(t, err, "Parse(%q)", test.in)
				assert.EqualValuesf(t, g, &test.want, "Parse(%q)", test.in)
			})
		}

		t.Run("Invalid", func(t *testing.T) {
			tests := map[string]struct {
				in     string
				errMsg string
			}{
				"StrictMustBeFirstKeyword": {
					in:     "digraph strict {}",
					errMsg: `got "strict" instead`,
				},
				"GraphIDMustComeAfterGraphKeywords": {
					in:     "dependencies {}",
					errMsg: `got "dependencies" instead`,
				},
				"LeftBraceMustFollow": {
					in:     "graph dependencies [",
					errMsg: `got "[" instead`,
				},
			}

			for name, test := range tests {
				t.Run(name, func(t *testing.T) {
					p, err := dot.New(strings.NewReader(test.in))

					require.NoErrorf(t, err, "New(%q)", test.in)

					_, err = p.Parse()

					require.NotNilf(t, err, "Parse(%q)", test.in)
					assertContains(t, err.Error(), test.errMsg)
				})
			}
		})
	})

	t.Run("NodeStatement", func(t *testing.T) {
		tests := map[string]struct {
			in   string
			want dot.Graph
			err  error
		}{
			"OnlyNode": {
				in: "graph { foo }",
				want: dot.Graph{
					Nodes: map[string]*dot.Node{
						"foo": {ID: "foo", Attributes: map[string]dot.Attribute{}},
					},
				},
			},
			"OnlyNodes": {
				in: "graph { foo ; bar baz }",
				want: dot.Graph{
					Nodes: map[string]*dot.Node{
						"foo": {ID: "foo", Attributes: map[string]dot.Attribute{}},
						"bar": {ID: "bar", Attributes: map[string]dot.Attribute{}},
						"baz": {ID: "baz", Attributes: map[string]dot.Attribute{}},
					},
				},
			},
			"OnlyNodeWithEmptyAttributeList": {
				in: "graph { foo [] }",
				want: dot.Graph{
					Nodes: map[string]*dot.Node{
						"foo": {ID: "foo", Attributes: map[string]dot.Attribute{}},
					},
				},
			},
			"NodeWithSingleAttribute": {
				in: "graph { foo [a=b] }",
				want: dot.Graph{
					Nodes: map[string]*dot.Node{
						"foo": {
							ID: "foo",
							Attributes: map[string]dot.Attribute{
								"a": {Name: "a", Value: "b"},
							},
						},
					},
				},
			},
			"NodeWithAttributesAndTrailingComma": {
				in: "graph { foo [a=b,] }",
				want: dot.Graph{
					Nodes: map[string]*dot.Node{
						"foo": {
							ID: "foo",
							Attributes: map[string]dot.Attribute{
								"a": {Name: "a", Value: "b"},
							},
						},
					},
				},
			},
			"NodeWithAttributesAndTrailingSemicolon": {
				in: "graph { foo [a=b;] }",
				want: dot.Graph{
					Nodes: map[string]*dot.Node{
						"foo": {
							ID: "foo",
							Attributes: map[string]dot.Attribute{
								"a": {Name: "a", Value: "b"},
							},
						},
					},
				},
			},
			"NodeWithAttributeOverriding": {
				in: "graph { foo [a=b;c=d]; foo [a=e] }",
				want: dot.Graph{
					Nodes: map[string]*dot.Node{
						"foo": {
							ID: "foo",
							Attributes: map[string]dot.Attribute{
								"a": {Name: "a", Value: "e"},
								"c": {Name: "c", Value: "d"},
							},
						},
					},
				},
			},
			"NodeWithMultipleAttributesInSingleBracketPair": {
				in: "graph { foo [a=b c=d,e=f;g=h] }",
				want: dot.Graph{
					Nodes: map[string]*dot.Node{
						"foo": {
							ID: "foo",
							Attributes: map[string]dot.Attribute{
								"a": {Name: "a", Value: "b"},
								"c": {Name: "c", Value: "d"},
								"e": {Name: "e", Value: "f"},
								"g": {Name: "g", Value: "h"},
							},
						},
					},
				},
			},
			"NodeWithMultipleAttributesInMultipleBracketPairs": {
				in: "graph { foo [a=b c=d][e=f;g=h] }",
				want: dot.Graph{
					Nodes: map[string]*dot.Node{
						"foo": {
							ID: "foo",
							Attributes: map[string]dot.Attribute{
								"a": {Name: "a", Value: "b"},
								"c": {Name: "c", Value: "d"},
								"e": {Name: "e", Value: "f"},
								"g": {Name: "g", Value: "h"},
							},
						},
					},
				},
			},
		}

		for name, test := range tests {
			t.Run(name, func(t *testing.T) {
				p, err := dot.New(strings.NewReader(test.in))

				require.NoErrorf(t, err, "New(%q)", test.in)

				g, err := p.Parse()

				assert.NoErrorf(t, err, "Parse(%q)", test.in)
				assert.EqualValuesf(t, g, &test.want, "Parse(%q)", test.in)
			})
		}

		t.Run("Invalid", func(t *testing.T) {
			tests := map[string]struct {
				in     string
				errMsg string
			}{
				"AttributeListWithoutClosingBracket": {
					in:     "graph { foo [ }",
					errMsg: `expected next token to be one of ["]" "identifier"]`,
				},
				"AttributeWithoutName": {
					in:     "graph { foo [ = b ] }",
					errMsg: `expected next token to be one of ["]" "identifier"]`,
				},
				"AttributeWithoutValue": {
					in:     "graph { foo [ a = ] }",
					errMsg: `expected next token to be "identifier"`,
				},
			}

			for name, test := range tests {
				t.Run(name, func(t *testing.T) {
					p, err := dot.New(strings.NewReader(test.in))

					require.NoErrorf(t, err, "New(%q)", test.in)

					_, err = p.Parse()

					require.NotNilf(t, err, "Parse(%q)", test.in)
					assertContains(t, err.Error(), test.errMsg)
				})
			}
		})
	})
}

func assertContains(t *testing.T, got, want string) {
	if !strings.Contains(got, want) {
		t.Errorf("got %q which does not contain %q", got, want)
	}
}
