package dot_test

import (
	"strings"
	"testing"

	"github.com/teleivo/assertive/assert"
	"github.com/teleivo/assertive/require"
	"github.com/teleivo/dot"
	"github.com/teleivo/dot/internal/ast"
)

func TestParser(t *testing.T) {
	t.Run("Header", func(t *testing.T) {
		tests := map[string]struct {
			in   string
			want ast.Graph
			err  error
		}{
			"Empty": {
				in:   "",
				want: ast.Graph{},
			},
			"EmptyDirectedGraph": {
				in: "digraph {}",
				want: ast.Graph{
					Directed: true,
				},
			},
			"EmptyUndirectedGraph": {
				in:   "graph {}",
				want: ast.Graph{},
			},
			"StrictDirectedUnnamedGraph": {
				in: `strict digraph {}`,
				want: ast.Graph{
					Strict:   true,
					Directed: true,
				},
			},
			"StrictDirectedNamedGraph": {
				in: `strict digraph dependencies {}`,
				want: ast.Graph{
					Strict:   true,
					Directed: true,
					ID:       "dependencies",
				},
			},
		}

		for name, test := range tests {
			t.Run(name, func(t *testing.T) {
				p, err := dot.New(strings.NewReader(test.in))

				require.NoErrorf(t, err, "New(%q)", test.in)

				g, err := p.Parse()

				assert.NoErrorf(t, err, "Parse(%q)", test.in)
				assert.EqualValuesf(t, g, test.want, "Parse(%q)", test.in)
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
			want ast.Graph
			err  error
		}{
			"OnlyNode": {
				in: "graph { foo }",
				want: ast.Graph{
					Stmts: []ast.Stmt{
						&ast.NodeStmt{ID: "foo"},
					},
				},
			},
			"OnlyNodes": {
				in: `graph { foo ; bar baz
					trash
				}`,
				want: ast.Graph{
					Stmts: []ast.Stmt{
						&ast.NodeStmt{ID: "foo"},
						&ast.NodeStmt{ID: "bar"},
						&ast.NodeStmt{ID: "baz"},
						&ast.NodeStmt{ID: "trash"},
					},
				},
			},
			"OnlyNodeWithEmptyAttributeList": {
				in: "graph { foo [] }",
				want: ast.Graph{
					Stmts: []ast.Stmt{
						&ast.NodeStmt{ID: "foo"},
					},
				},
			},
			"NodeWithSingleAttributeAndEmptyAttributeList": {
				in: "graph { foo [] [a=b] }",
				want: ast.Graph{
					Stmts: []ast.Stmt{
						&ast.NodeStmt{
							ID: "foo",
							AttrList: &ast.AttrList{
								AList: &ast.AList{
									Attribute: ast.Attribute{Name: "a", Value: "b"},
								},
							},
						},
					},
				},
			},
			"NodeWithSingleAttribute": {
				in: "graph { foo [a=b] }",
				want: ast.Graph{
					Stmts: []ast.Stmt{
						&ast.NodeStmt{
							ID: "foo",
							AttrList: &ast.AttrList{
								AList: &ast.AList{
									Attribute: ast.Attribute{Name: "a", Value: "b"},
								},
							},
						},
					},
				},
			},
			"NodeWithAttributesAndTrailingComma": {
				in: "graph { foo [a=b,] }",
				want: ast.Graph{
					Stmts: []ast.Stmt{
						&ast.NodeStmt{
							ID: "foo",
							AttrList: &ast.AttrList{
								AList: &ast.AList{
									Attribute: ast.Attribute{Name: "a", Value: "b"},
								},
							},
						},
					},
				},
			},
			"NodeWithAttributesAndTrailingSemicolon": {
				in: "graph { foo [a=b;] }",
				want: ast.Graph{
					Stmts: []ast.Stmt{
						&ast.NodeStmt{
							ID: "foo",
							AttrList: &ast.AttrList{
								AList: &ast.AList{
									Attribute: ast.Attribute{Name: "a", Value: "b"},
								},
							},
						},
					},
				},
			},
			"NodeWithAttributeOverriding": {
				in: "graph { foo [a=b;c=d]; foo [a=e] }",
				want: ast.Graph{
					Stmts: []ast.Stmt{
						&ast.NodeStmt{
							ID: "foo",
							AttrList: &ast.AttrList{
								AList: &ast.AList{
									Attribute: ast.Attribute{Name: "a", Value: "b"},
									Next: &ast.AList{
										Attribute: ast.Attribute{Name: "c", Value: "d"},
									},
								},
							},
						},
						&ast.NodeStmt{
							ID: "foo",
							AttrList: &ast.AttrList{
								AList: &ast.AList{
									Attribute: ast.Attribute{Name: "a", Value: "e"},
								},
							},
						},
					},
				},
			},
			"NodeWithMultipleAttributesInSingleBracketPair": {
				in: "graph { foo [a=b c=d,e=f;g=h] }",
				want: ast.Graph{
					Stmts: []ast.Stmt{
						&ast.NodeStmt{
							ID: "foo",
							AttrList: &ast.AttrList{
								AList: &ast.AList{
									Attribute: ast.Attribute{Name: "a", Value: "b"},
									Next: &ast.AList{
										Attribute: ast.Attribute{Name: "c", Value: "d"},
										Next: &ast.AList{
											Attribute: ast.Attribute{Name: "e", Value: "f"},
											Next: &ast.AList{
												Attribute: ast.Attribute{Name: "g", Value: "h"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			"NodeWithMultipleAttributesInMultipleBracketPairs": {
				in: "graph { foo [a=b c=d][e=f;g=h] }",
				want: ast.Graph{
					Stmts: []ast.Stmt{
						&ast.NodeStmt{
							ID: "foo",
							AttrList: &ast.AttrList{
								AList: &ast.AList{
									Attribute: ast.Attribute{Name: "a", Value: "b"},
									Next: &ast.AList{
										Attribute: ast.Attribute{Name: "c", Value: "d"},
									},
								},
								Next: &ast.AttrList{
									AList: &ast.AList{
										Attribute: ast.Attribute{Name: "e", Value: "f"},
										Next: &ast.AList{
											Attribute: ast.Attribute{Name: "g", Value: "h"},
										},
									},
								},
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
				assert.EqualValuesf(t, g, test.want, "Parse(%q)", test.in)
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

	t.Run("EdgeStatement", func(t *testing.T) {
		tests := map[string]struct {
			in   string
			want ast.Graph
			err  error
		}{
			"SingleUndirectedEdge": {
				in: "graph { 1 -- 2 }",
				want: ast.Graph{
					Stmts: []ast.Stmt{
						&ast.EdgeStmt{
							Left:  "1",
							Right: ast.EdgeRHS{Right: "2"},
						},
					},
				},
			},
			"SingleDirectedEdge": {
				in: "graph { 1 -> 2 }",
				want: ast.Graph{
					Stmts: []ast.Stmt{
						&ast.EdgeStmt{
							Left:  "1",
							Right: ast.EdgeRHS{Directed: true, Right: "2"},
						},
					},
				},
			},
			"MultipleDirectedEdges": {
				in: "graph { 1 -> 2 -> 3 -> 4 }",
				want: ast.Graph{
					Stmts: []ast.Stmt{
						&ast.EdgeStmt{
							Left: "1",
							Right: ast.EdgeRHS{
								Directed: true,
								Right:    "2",
								Next: &ast.EdgeRHS{
									Directed: true,
									Right:    "3",
									Next: &ast.EdgeRHS{
										Directed: true,
										Right:    "4",
									},
								},
							},
						},
					},
				},
			},
			// TODO with attr_list
		}

		for name, test := range tests {
			t.Run(name, func(t *testing.T) {
				p, err := dot.New(strings.NewReader(test.in))

				require.NoErrorf(t, err, "New(%q)", test.in)

				g, err := p.Parse()

				assert.NoErrorf(t, err, "Parse(%q)", test.in)
				assert.EqualValuesf(t, g, test.want, "Parse(%q)", test.in)
			})
		}

		t.Run("Invalid", func(t *testing.T) {
			tests := map[string]struct {
				in     string
				errMsg string
			}{
				"GraphWithoutAttributeList": {
					in:     "graph { 1 -> 2 }",
					errMsg: "undirected graph cannot contain directed edges",
				},
				"NodeWithoutAttributeList": {
					in:     "digraph { 1 -- 2  }",
					errMsg: "directed graph cannot contain undirected edges",
				},
				// TODO test more cases
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

	t.Run("AttributeStatement", func(t *testing.T) {
		tests := map[string]struct {
			in   string
			want ast.Graph
			err  error
		}{
			"OnlyGraph": {
				in: "graph { graph [] }",
				want: ast.Graph{
					Stmts: []ast.Stmt{
						&ast.AttrStmt{ID: "graph"},
					},
				},
			},
			"OnlyNode": {
				in: "graph { node [] }",
				want: ast.Graph{
					Stmts: []ast.Stmt{
						&ast.AttrStmt{ID: "node"},
					},
				},
			},
			"OnlyEdge": {
				in: "graph { edge [] }",
				want: ast.Graph{
					Stmts: []ast.Stmt{
						&ast.AttrStmt{ID: "edge"},
					},
				},
			},
			"GraphWithAttribute": {
				in: "graph { graph [a=b] }",
				want: ast.Graph{
					Stmts: []ast.Stmt{
						&ast.AttrStmt{
							ID: "graph",
							AttrList: &ast.AttrList{
								AList: &ast.AList{
									Attribute: ast.Attribute{Name: "a", Value: "b"},
								},
							},
						},
					},
				},
			},
			"NodeWithAttribute": {
				in: "graph { node [a=b] }",
				want: ast.Graph{
					Stmts: []ast.Stmt{
						&ast.AttrStmt{
							ID: "node",
							AttrList: &ast.AttrList{
								AList: &ast.AList{
									Attribute: ast.Attribute{Name: "a", Value: "b"},
								},
							},
						},
					},
				},
			},
			"EdgeWithAttribute": {
				in: "graph { edge [a=b] }",
				want: ast.Graph{
					Stmts: []ast.Stmt{
						&ast.AttrStmt{
							ID: "edge",
							AttrList: &ast.AttrList{
								AList: &ast.AList{
									Attribute: ast.Attribute{Name: "a", Value: "b"},
								},
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
				assert.EqualValuesf(t, g, test.want, "Parse(%q)", test.in)
			})
		}

		t.Run("Invalid", func(t *testing.T) {
			tests := map[string]struct {
				in     string
				errMsg string
			}{
				"GraphWithoutAttributeList": {
					in:     "graph { graph }",
					errMsg: `expected next token to be "["`,
				},
				"NodeWithoutAttributeList": {
					in:     "graph { node }",
					errMsg: `expected next token to be "["`,
				},
				"EdgeWithoutAttributeList": {
					in:     "graph { edge }",
					errMsg: `expected next token to be "["`,
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
