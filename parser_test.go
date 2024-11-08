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
				p, err := dot.NewParser(strings.NewReader(test.in))

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
					p, err := dot.NewParser(strings.NewReader(test.in))

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
						&ast.NodeStmt{NodeID: ast.NodeID{ID: "foo"}},
					},
				},
			},
			"OnlyNodes": {
				in: `graph { foo ; bar baz
					trash
				}`,
				want: ast.Graph{
					Stmts: []ast.Stmt{
						&ast.NodeStmt{NodeID: ast.NodeID{ID: "foo"}},
						&ast.NodeStmt{NodeID: ast.NodeID{ID: "bar"}},
						&ast.NodeStmt{NodeID: ast.NodeID{ID: "baz"}},
						&ast.NodeStmt{NodeID: ast.NodeID{ID: "trash"}},
					},
				},
			},
			"NodeWithPortName": {
				in: "graph { foo:f0 }",
				want: ast.Graph{
					Stmts: []ast.Stmt{
						&ast.NodeStmt{NodeID: ast.NodeID{ID: "foo", Port: &ast.Port{Name: "f0", CompassPoint: ast.Underscore}}},
					},
				},
			},
			"NodeWithPortNameAndCompassPointUnderscore": {
				in: `graph { foo:"f0":_ }`,
				want: ast.Graph{
					Stmts: []ast.Stmt{
						&ast.NodeStmt{NodeID: ast.NodeID{ID: "foo", Port: &ast.Port{Name: `"f0"`, CompassPoint: ast.Underscore}}},
					},
				},
			},
			"NodeWithPortNameAndCompassPointNorth": {
				in: `graph { foo:"f0":n }`,
				want: ast.Graph{
					Stmts: []ast.Stmt{
						&ast.NodeStmt{NodeID: ast.NodeID{ID: "foo", Port: &ast.Port{Name: `"f0"`, CompassPoint: ast.North}}},
					},
				},
			},
			"NodeWithPortNameAndCompassPointNorthEast": {
				in: `graph { foo:f0:ne }`,
				want: ast.Graph{
					Stmts: []ast.Stmt{
						&ast.NodeStmt{NodeID: ast.NodeID{ID: "foo", Port: &ast.Port{Name: "f0", CompassPoint: ast.NorthEast}}},
					},
				},
			},
			"NodeWithPortNameAndCompassPointEast": {
				in: `graph { foo:f0:e }`,
				want: ast.Graph{
					Stmts: []ast.Stmt{
						&ast.NodeStmt{NodeID: ast.NodeID{ID: "foo", Port: &ast.Port{Name: "f0", CompassPoint: ast.East}}},
					},
				},
			},
			"NodeWithPortNameAndCompassPointSouthEast": {
				in: `graph { foo:f0:se }`,
				want: ast.Graph{
					Stmts: []ast.Stmt{
						&ast.NodeStmt{NodeID: ast.NodeID{ID: "foo", Port: &ast.Port{Name: "f0", CompassPoint: ast.SouthEast}}},
					},
				},
			},
			"NodeWithPortNameAndCompassPointSouth": {
				in: `graph { foo:f0:s }`,
				want: ast.Graph{
					Stmts: []ast.Stmt{
						&ast.NodeStmt{NodeID: ast.NodeID{ID: "foo", Port: &ast.Port{Name: "f0", CompassPoint: ast.South}}},
					},
				},
			},
			"NodeWithPortNameAndCompassPointSouthWest": {
				in: `graph { foo:f0:sw }`,
				want: ast.Graph{
					Stmts: []ast.Stmt{
						&ast.NodeStmt{NodeID: ast.NodeID{ID: "foo", Port: &ast.Port{Name: "f0", CompassPoint: ast.SouthWest}}},
					},
				},
			},
			"NodeWithPortNameAndCompassPointWest": {
				in: `graph { foo:f0:w }`,
				want: ast.Graph{
					Stmts: []ast.Stmt{
						&ast.NodeStmt{NodeID: ast.NodeID{ID: "foo", Port: &ast.Port{Name: "f0", CompassPoint: ast.West}}},
					},
				},
			},
			"NodeWithPortNameAndCompassPointNorthWest": {
				in: `graph { foo:f0:nw }`,
				want: ast.Graph{
					Stmts: []ast.Stmt{
						&ast.NodeStmt{NodeID: ast.NodeID{ID: "foo", Port: &ast.Port{Name: "f0", CompassPoint: ast.NorthWest}}},
					},
				},
			},
			"NodeWithPortNameAndCompassPointCenter": {
				in: `graph { foo:f0:c }`,
				want: ast.Graph{
					Stmts: []ast.Stmt{
						&ast.NodeStmt{NodeID: ast.NodeID{ID: "foo", Port: &ast.Port{Name: "f0", CompassPoint: ast.Center}}},
					},
				},
			},
			"NodeWithCompassPointNorth": {
				in: `graph { foo:n }`,
				want: ast.Graph{
					Stmts: []ast.Stmt{
						&ast.NodeStmt{NodeID: ast.NodeID{ID: "foo", Port: &ast.Port{CompassPoint: ast.North}}},
					},
				},
			},
			"NodeWithPortNameEqualToACompassPoint": { // https://graphviz.org/docs/attr-types/portPos
				in: `graph { foo:n:n }`,
				want: ast.Graph{
					Stmts: []ast.Stmt{
						&ast.NodeStmt{NodeID: ast.NodeID{ID: "foo", Port: &ast.Port{Name: "n", CompassPoint: ast.North}}},
					},
				},
			},
			"OnlyNodeWithEmptyAttributeList": {
				in: "graph { foo [] }",
				want: ast.Graph{
					Stmts: []ast.Stmt{
						&ast.NodeStmt{NodeID: ast.NodeID{ID: "foo"}},
					},
				},
			},
			"NodeWithSingleAttributeAndEmptyAttributeList": {
				in: "graph { foo [] [a=b] }",
				want: ast.Graph{
					Stmts: []ast.Stmt{
						&ast.NodeStmt{
							NodeID: ast.NodeID{ID: "foo"},
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
							NodeID: ast.NodeID{ID: "foo"},
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
							NodeID: ast.NodeID{ID: "foo"},
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
							NodeID: ast.NodeID{ID: "foo"},
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
							NodeID: ast.NodeID{ID: "foo"},
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
							NodeID: ast.NodeID{ID: "foo"},
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
							NodeID: ast.NodeID{ID: "foo"},
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
							NodeID: ast.NodeID{ID: "foo"},
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
				p, err := dot.NewParser(strings.NewReader(test.in))

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
				"NodeWithPortWithoutName": {
					in:     "graph { foo: }",
					errMsg: `expected next token to be "identifier"`,
				},
				"NodeWithPortWithoutCompassPoint": {
					in:     "graph { foo:f: }",
					errMsg: `expected next token to be "identifier"`,
				},
				"NodeWithPortWithInvalidCompassPoint": {
					in:     "graph { foo:n:bottom }",
					errMsg: `expected a compass point [_ n ne`,
				},
			}

			for name, test := range tests {
				t.Run(name, func(t *testing.T) {
					p, err := dot.NewParser(strings.NewReader(test.in))

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
							Left:  ast.NodeID{ID: "1"},
							Right: ast.EdgeRHS{Right: ast.NodeID{ID: "2"}},
						},
					},
				},
			},
			"SingleDirectedEdge": {
				in: "digraph { 1 -> 2 }",
				want: ast.Graph{
					Directed: true,
					Stmts: []ast.Stmt{
						&ast.EdgeStmt{
							Left:  ast.NodeID{ID: "1"},
							Right: ast.EdgeRHS{Directed: true, Right: ast.NodeID{ID: "2"}},
						},
					},
				},
			},
			"MultipleDirectedEdgesWithAttributeList": {
				in: "digraph { 1 -> 2 -> 3 -> 4 [a=b] }",
				want: ast.Graph{
					Directed: true,
					Stmts: []ast.Stmt{
						&ast.EdgeStmt{
							Left: ast.NodeID{ID: "1"},
							Right: ast.EdgeRHS{
								Directed: true,
								Right:    ast.NodeID{ID: "2"},
								Next: &ast.EdgeRHS{
									Directed: true,
									Right:    ast.NodeID{ID: "3"},
									Next: &ast.EdgeRHS{
										Directed: true,
										Right:    ast.NodeID{ID: "4"},
									},
								},
							},
							AttrList: &ast.AttrList{
								AList: &ast.AList{
									Attribute: ast.Attribute{Name: "a", Value: "b"},
								},
							},
						},
					},
				},
			},
			"EdgeWithLHSShortSubgraph": {
				in: "digraph { {A B} -> C }",
				want: ast.Graph{
					Directed: true,
					Stmts: []ast.Stmt{
						&ast.EdgeStmt{
							Left: ast.Subgraph{
								Stmts: []ast.Stmt{
									&ast.NodeStmt{NodeID: ast.NodeID{ID: "A"}},
									&ast.NodeStmt{NodeID: ast.NodeID{ID: "B"}},
								},
							},
							Right: ast.EdgeRHS{
								Directed: true,
								Right:    ast.NodeID{ID: "C"},
							},
						},
					},
				},
			},
			"EdgeWithRHSShortSubgraph": {
				in: "digraph { A -> {B C} }",
				want: ast.Graph{
					Directed: true,
					Stmts: []ast.Stmt{
						&ast.EdgeStmt{
							Left: ast.NodeID{ID: "A"},
							Right: ast.EdgeRHS{
								Directed: true,
								Right: ast.Subgraph{
									Stmts: []ast.Stmt{
										&ast.NodeStmt{NodeID: ast.NodeID{ID: "B"}},
										&ast.NodeStmt{NodeID: ast.NodeID{ID: "C"}},
									},
								},
							},
						},
					},
				},
			},
			"EdgeWithNestedSubraphs": {
				in: "graph { {1 2} -- {3 -- {4 5}} }",
				want: ast.Graph{
					Stmts: []ast.Stmt{
						&ast.EdgeStmt{
							Left: ast.Subgraph{
								Stmts: []ast.Stmt{
									&ast.NodeStmt{NodeID: ast.NodeID{ID: "1"}},
									&ast.NodeStmt{NodeID: ast.NodeID{ID: "2"}},
								},
							},
							Right: ast.EdgeRHS{
								Right: ast.Subgraph{
									Stmts: []ast.Stmt{
										&ast.EdgeStmt{
											Left: ast.NodeID{ID: "3"},
											Right: ast.EdgeRHS{
												Right: ast.Subgraph{
													Stmts: []ast.Stmt{
														&ast.NodeStmt{NodeID: ast.NodeID{ID: "4"}},
														&ast.NodeStmt{NodeID: ast.NodeID{ID: "5"}},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			"EdgeWithRHSExplicitSubraph": {
				in: "digraph { A -> subgraph foo {B C} }",
				want: ast.Graph{
					Directed: true,
					Stmts: []ast.Stmt{
						&ast.EdgeStmt{
							Left: ast.NodeID{ID: "A"},
							Right: ast.EdgeRHS{
								Directed: true,
								Right: ast.Subgraph{
									ID: "foo",
									Stmts: []ast.Stmt{
										&ast.NodeStmt{NodeID: ast.NodeID{ID: "B"}},
										&ast.NodeStmt{NodeID: ast.NodeID{ID: "C"}},
									},
								},
							},
						},
					},
				},
			},
			"EdgeWithPorts": {
				in: `digraph {
			"node4":f0:n -> node5:f1;
}`,
				want: ast.Graph{
					Directed: true,
					Stmts: []ast.Stmt{
						&ast.EdgeStmt{
							Left:  ast.NodeID{ID: `"node4"`, Port: &ast.Port{Name: "f0", CompassPoint: ast.North}},
							Right: ast.EdgeRHS{Directed: true, Right: ast.NodeID{ID: "node5", Port: &ast.Port{Name: "f1"}}},
						},
					},
				},
			},
		}

		for name, test := range tests {
			t.Run(name, func(t *testing.T) {
				p, err := dot.NewParser(strings.NewReader(test.in))

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
				"UndirectedGraphWithDirectedEdge": {
					in:     "graph { 1 -> 2 }",
					errMsg: "undirected graph cannot contain directed edges",
				},
				"DirectedGraphWithUndirectedEdge": {
					in:     "digraph { 1 -- 2  }",
					errMsg: "directed graph cannot contain undirected edges",
				},
				"MissingRHSOperand": {
					in:     "graph { 1 -- [style=filled] }",
					errMsg: `expected next token to be one of ["identifier" "subgraph" "{"]`,
				},
			}

			for name, test := range tests {
				t.Run(name, func(t *testing.T) {
					p, err := dot.NewParser(strings.NewReader(test.in))

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
				p, err := dot.NewParser(strings.NewReader(test.in))

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
					p, err := dot.NewParser(strings.NewReader(test.in))

					require.NoErrorf(t, err, "New(%q)", test.in)

					_, err = p.Parse()

					require.NotNilf(t, err, "Parse(%q)", test.in)
					assertContains(t, err.Error(), test.errMsg)
				})
			}
		})
	})

	t.Run("AttributeAssignment", func(t *testing.T) {
		tests := map[string]struct {
			in   string
			want ast.Graph
			err  error
		}{
			"Single": {
				in: "graph { rank = same; }",
				want: ast.Graph{
					Stmts: []ast.Stmt{
						ast.Attribute{Name: "rank", Value: "same"},
					},
				},
			},
			"QuotedAttributeValueSpanningMultipleLines": {
				in: `graph { 	label="Rainy days
				in summer"
}`,
				want: ast.Graph{
					Stmts: []ast.Stmt{
						ast.Attribute{Name: "label", Value: `"Rainy days
				in summer"`},
					},
				},
			},
			// https://graphviz.org/doc/info/lang.html#comments-and-optional-formatting
			"QuotedAttributeValueSpanningMultipleLinesWithBackslashFollowedByNewline": {
				in: `graph { 	label="Rainy days\
				in summer"
}`,
				want: ast.Graph{
					Stmts: []ast.Stmt{
						ast.Attribute{Name: "label", Value: `"Rainy days\
				in summer"`},
					},
				},
			},
		}

		for name, test := range tests {
			t.Run(name, func(t *testing.T) {
				p, err := dot.NewParser(strings.NewReader(test.in))

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
				"MissingName": {
					in:     "graph { = b }",
					errMsg: `expected an "identifier" before the '='`,
				},
				"MissingValue": {
					in:     "graph { a = }",
					errMsg: `expected next token to be "identifier"`,
				},
			}

			for name, test := range tests {
				t.Run(name, func(t *testing.T) {
					p, err := dot.NewParser(strings.NewReader(test.in))

					require.NoErrorf(t, err, "New(%q)", test.in)

					_, err = p.Parse()

					require.NotNilf(t, err, "Parse(%q)", test.in)
					assertContains(t, err.Error(), test.errMsg)
				})
			}
		})
	})

	t.Run("Subgraph", func(t *testing.T) {
		tests := map[string]struct {
			in   string
			want ast.Graph
			err  error
		}{
			"EmptyWithKeyword": {
				in: "graph { subgraph {} }",
				want: ast.Graph{
					Stmts: []ast.Stmt{
						ast.Subgraph{},
					},
				},
			},
			"EmptyWithoutKeyword": {
				in: "graph { {} }",
				want: ast.Graph{
					Stmts: []ast.Stmt{
						ast.Subgraph{},
					},
				},
			},
			"SubgraphWithID": {
				in: "graph { subgraph foo {} }",
				want: ast.Graph{
					Stmts: []ast.Stmt{
						ast.Subgraph{ID: "foo"},
					},
				},
			},
			"SubgraphWithAttributesAndNodes": {
				in: `graph {
					subgraph {
						rank = same; A; B;
					}
				}`,
				want: ast.Graph{
					Stmts: []ast.Stmt{
						ast.Subgraph{
							Stmts: []ast.Stmt{
								ast.Attribute{Name: "rank", Value: "same"},
								&ast.NodeStmt{NodeID: ast.NodeID{ID: "A"}},
								&ast.NodeStmt{NodeID: ast.NodeID{ID: "B"}},
							},
						},
					},
				},
			},
		}

		for name, test := range tests {
			t.Run(name, func(t *testing.T) {
				p, err := dot.NewParser(strings.NewReader(test.in))

				require.NoErrorf(t, err, "New(%q)", test.in)

				g, err := p.Parse()

				assert.NoErrorf(t, err, "Parse(%q)", test.in)
				assert.EqualValuesf(t, g, test.want, "Parse(%q)", test.in)
			})
		}

		t.Run("Invalid", func(t *testing.T) {
			t.Skip()

			tests := map[string]struct {
				in     string
				errMsg string
			}{
				"MissingClosingBrace": {
					in:     "graph { { }",
					errMsg: `expected next token to be one of ["}" "identifier"]`,
				},
			}

			for name, test := range tests {
				t.Run(name, func(t *testing.T) {
					p, err := dot.NewParser(strings.NewReader(test.in))

					require.NoErrorf(t, err, "New(%q)", test.in)

					_, err = p.Parse()

					require.NotNilf(t, err, "Parse(%q)", test.in)
					assertContains(t, err.Error(), test.errMsg)
				})
			}
		})
	})

	t.Run("Comment", func(t *testing.T) {
		tests := map[string]struct {
			in   string
			want ast.Graph
			err  error
		}{
			"CPreprocessorStyle": {
				in: `graph {	 # ok
				}`,
				want: ast.Graph{
					Stmts: []ast.Stmt{
						ast.Comment{
							Text: "# ok",
						},
					},
				},
			},
			"Single": {
				in: `graph { 
				// ok
				}`,
				want: ast.Graph{
					Stmts: []ast.Stmt{
						ast.Comment{
							Text: "// ok",
						},
					},
				},
			},
			"MultiLine": {
				in: `graph { /* ok
				then */
				}`,
				want: ast.Graph{
					Stmts: []ast.Stmt{
						ast.Comment{
							Text: `/* ok
				then */`,
						},
					},
				},
			},
		}

		for name, test := range tests {
			t.Run(name, func(t *testing.T) {
				p, err := dot.NewParser(strings.NewReader(test.in))

				require.NoErrorf(t, err, "New(%q)", test.in)

				g, err := p.Parse()

				assert.NoErrorf(t, err, "Parse(%q)", test.in)
				assert.EqualValuesf(t, g, test.want, "Parse(%q)", test.in)
			})
		}

		t.Run("Invalid", func(t *testing.T) {
			t.Skip()

			tests := map[string]struct {
				in     string
				errMsg string
			}{
				"CPreprocessorStyleEatsClosingBrace": {
					in:     "graph { # ok }",
					errMsg: `expected next token to be one of ["}" "identifier"]`,
				},
			}

			for name, test := range tests {
				t.Run(name, func(t *testing.T) {
					p, err := dot.NewParser(strings.NewReader(test.in))

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
