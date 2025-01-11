package dot_test

import (
	"strings"
	"testing"

	"github.com/teleivo/assertive/assert"
	"github.com/teleivo/assertive/require"
	"github.com/teleivo/dot"
	"github.com/teleivo/dot/ast"
	"github.com/teleivo/dot/token"
)

// TODO add GraphStart to tests
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
					GraphStart: token.Position{Row: 1, Column: 1},
					Directed:   true,
					LeftBrace:  token.Position{Row: 1, Column: 9},
					RightBrace: token.Position{Row: 1, Column: 10},
				},
			},
			"GraphWithComments": {
				in: `/** header explaining
				the graph */
graph {
} // trailing comment`,
				want: ast.Graph{
					GraphStart: token.Position{Row: 3, Column: 1},
					LeftBrace:  token.Position{Row: 3, Column: 7},
					RightBrace: token.Position{Row: 4, Column: 1},
					Comments: []ast.Comment{
						{
							Text: `/** header explaining
				the graph */`,
							StartPos: token.Position{Row: 1, Column: 1},
							EndPos:   token.Position{Row: 2, Column: 16},
						},
						{
							Text:     "// trailing comment",
							StartPos: token.Position{Row: 4, Column: 3},
							EndPos:   token.Position{Row: 4, Column: 21},
						},
					},
				},
			},
			"EmptyUndirectedGraph": {
				in: "graph {}",
				want: ast.Graph{
					GraphStart: token.Position{Row: 1, Column: 1},
					LeftBrace:  token.Position{Row: 1, Column: 7},
					RightBrace: token.Position{Row: 1, Column: 8},
				},
			},
			"StrictDirectedUnnamedGraph": {
				in: ` strict digraph {}`,
				want: ast.Graph{
					StrictStart: &token.Position{Row: 1, Column: 2},
					GraphStart:  token.Position{Row: 1, Column: 9},
					Directed:    true,
					LeftBrace:   token.Position{Row: 1, Column: 17},
					RightBrace:  token.Position{Row: 1, Column: 18},
				},
			},
			"StrictDirectedNamedGraph": {
				in: `strict digraph dependencies {}`,
				want: ast.Graph{
					StrictStart: &token.Position{Row: 1, Column: 1},
					GraphStart:  token.Position{Row: 1, Column: 8},
					Directed:    true,
					ID: &ast.ID{
						Literal:  "dependencies",
						StartPos: token.Position{Row: 1, Column: 16},
						EndPos:   token.Position{Row: 1, Column: 27},
					},
					LeftBrace:  token.Position{Row: 1, Column: 29},
					RightBrace: token.Position{Row: 1, Column: 30},
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

	t.Run("NodeStmt", func(t *testing.T) {
		tests := map[string]struct {
			in   string
			want ast.Graph
			err  error
		}{
			"OnlyNode": {
				in: "graph { foo }",
				want: ast.Graph{
					GraphStart: token.Position{Row: 1, Column: 1},
					Stmts: []ast.Stmt{
						&ast.NodeStmt{
							NodeID: ast.NodeID{
								ID: ast.ID{
									Literal:  "foo",
									StartPos: token.Position{Row: 1, Column: 9},
									EndPos:   token.Position{Row: 1, Column: 11},
								},
							},
						},
					},
					LeftBrace:  token.Position{Row: 1, Column: 7},
					RightBrace: token.Position{Row: 1, Column: 13},
				},
			},
			"OnlyNodes": {
				in: `graph { foo ; bar baz
					trash
				}`,
				want: ast.Graph{
					GraphStart: token.Position{Row: 1, Column: 1},
					Stmts: []ast.Stmt{
						&ast.NodeStmt{
							NodeID: ast.NodeID{
								ID: ast.ID{
									Literal:  "foo",
									StartPos: token.Position{Row: 1, Column: 9},
									EndPos:   token.Position{Row: 1, Column: 11},
								},
							},
						},
						&ast.NodeStmt{
							NodeID: ast.NodeID{
								ID: ast.ID{
									Literal:  "bar",
									StartPos: token.Position{Row: 1, Column: 15},
									EndPos:   token.Position{Row: 1, Column: 17},
								},
							},
						},
						&ast.NodeStmt{
							NodeID: ast.NodeID{
								ID: ast.ID{
									Literal:  "baz",
									StartPos: token.Position{Row: 1, Column: 19},
									EndPos:   token.Position{Row: 1, Column: 21},
								},
							},
						},
						&ast.NodeStmt{
							NodeID: ast.NodeID{
								ID: ast.ID{
									Literal:  "trash",
									StartPos: token.Position{Row: 2, Column: 6},
									EndPos:   token.Position{Row: 2, Column: 10},
								},
							},
						},
					},
					LeftBrace:  token.Position{Row: 1, Column: 7},
					RightBrace: token.Position{Row: 3, Column: 5},
				},
			},
			"NodeWithPortName": {
				in: "graph { foo:f0 }",
				want: ast.Graph{
					GraphStart: token.Position{Row: 1, Column: 1},
					Stmts: []ast.Stmt{
						&ast.NodeStmt{
							NodeID: ast.NodeID{
								ID: ast.ID{
									Literal:  "foo",
									StartPos: token.Position{Row: 1, Column: 9},
									EndPos:   token.Position{Row: 1, Column: 11},
								},
								Port: &ast.Port{
									Name: &ast.ID{
										Literal:  "f0",
										StartPos: token.Position{Row: 1, Column: 13},
										EndPos:   token.Position{Row: 1, Column: 14},
									},
								},
							},
						},
					},
					LeftBrace:  token.Position{Row: 1, Column: 7},
					RightBrace: token.Position{Row: 1, Column: 16},
				},
			},
			"NodeWithPortNameAndCompassPointUnderscore": {
				in: `graph { foo:"f0":_ }`,
				want: ast.Graph{
					GraphStart: token.Position{Row: 1, Column: 1},
					Stmts: []ast.Stmt{
						&ast.NodeStmt{
							NodeID: ast.NodeID{
								ID: ast.ID{
									Literal:  "foo",
									StartPos: token.Position{Row: 1, Column: 9},
									EndPos:   token.Position{Row: 1, Column: 11},
								}, Port: &ast.Port{
									Name: &ast.ID{
										Literal:  `"f0"`,
										StartPos: token.Position{Row: 1, Column: 13},
										EndPos:   token.Position{Row: 1, Column: 16},
									},
									CompassPoint: &ast.CompassPoint{
										Type:     ast.CompassPointUnderscore,
										StartPos: token.Position{Row: 1, Column: 18},
										EndPos:   token.Position{Row: 1, Column: 18},
									},
								},
							},
						},
					},
					LeftBrace:  token.Position{Row: 1, Column: 7},
					RightBrace: token.Position{Row: 1, Column: 20},
				},
			},
			"NodeWithPortNameAndCompassPointNorth": {
				in: `graph { foo:"f0":n }`,
				want: ast.Graph{
					GraphStart: token.Position{Row: 1, Column: 1},
					Stmts: []ast.Stmt{
						&ast.NodeStmt{
							NodeID: ast.NodeID{
								ID: ast.ID{
									Literal:  "foo",
									StartPos: token.Position{Row: 1, Column: 9},
									EndPos:   token.Position{Row: 1, Column: 11},
								}, Port: &ast.Port{
									Name: &ast.ID{
										Literal:  `"f0"`,
										StartPos: token.Position{Row: 1, Column: 13},
										EndPos:   token.Position{Row: 1, Column: 16},
									},
									CompassPoint: &ast.CompassPoint{
										Type:     ast.CompassPointNorth,
										StartPos: token.Position{Row: 1, Column: 18},
										EndPos:   token.Position{Row: 1, Column: 18},
									},
								},
							},
						},
					},
					LeftBrace:  token.Position{Row: 1, Column: 7},
					RightBrace: token.Position{Row: 1, Column: 20},
				},
			},
			"NodeWithPortNameAndCompassPointNorthEast": {
				in: `graph { foo:f0:ne }`,
				want: ast.Graph{
					GraphStart: token.Position{Row: 1, Column: 1},
					Stmts: []ast.Stmt{
						&ast.NodeStmt{
							NodeID: ast.NodeID{
								ID: ast.ID{
									Literal:  "foo",
									StartPos: token.Position{Row: 1, Column: 9},
									EndPos:   token.Position{Row: 1, Column: 11},
								}, Port: &ast.Port{
									Name: &ast.ID{
										Literal:  "f0",
										StartPos: token.Position{Row: 1, Column: 13},
										EndPos:   token.Position{Row: 1, Column: 14},
									},
									CompassPoint: &ast.CompassPoint{
										Type:     ast.CompassPointNorthEast,
										StartPos: token.Position{Row: 1, Column: 16},
										EndPos:   token.Position{Row: 1, Column: 17},
									},
								},
							},
						},
					},
					LeftBrace:  token.Position{Row: 1, Column: 7},
					RightBrace: token.Position{Row: 1, Column: 19},
				},
			},
			"NodeWithPortNameAndCompassPointEast": {
				in: `graph { foo:f0:e }`,
				want: ast.Graph{
					GraphStart: token.Position{Row: 1, Column: 1},
					Stmts: []ast.Stmt{
						&ast.NodeStmt{
							NodeID: ast.NodeID{
								ID: ast.ID{
									Literal:  "foo",
									StartPos: token.Position{Row: 1, Column: 9},
									EndPos:   token.Position{Row: 1, Column: 11},
								}, Port: &ast.Port{
									Name: &ast.ID{
										Literal:  "f0",
										StartPos: token.Position{Row: 1, Column: 13},
										EndPos:   token.Position{Row: 1, Column: 14},
									},
									CompassPoint: &ast.CompassPoint{
										Type:     ast.CompassPointEast,
										StartPos: token.Position{Row: 1, Column: 16},
										EndPos:   token.Position{Row: 1, Column: 16},
									},
								},
							},
						},
					},
					LeftBrace:  token.Position{Row: 1, Column: 7},
					RightBrace: token.Position{Row: 1, Column: 18},
				},
			},
			"NodeWithPortNameAndCompassPointSouthEast": {
				in: `graph { foo:f0:se }`,
				want: ast.Graph{
					GraphStart: token.Position{Row: 1, Column: 1},
					Stmts: []ast.Stmt{
						&ast.NodeStmt{
							NodeID: ast.NodeID{
								ID: ast.ID{
									Literal:  "foo",
									StartPos: token.Position{Row: 1, Column: 9},
									EndPos:   token.Position{Row: 1, Column: 11},
								}, Port: &ast.Port{
									Name: &ast.ID{
										Literal:  "f0",
										StartPos: token.Position{Row: 1, Column: 13},
										EndPos:   token.Position{Row: 1, Column: 14},
									},
									CompassPoint: &ast.CompassPoint{
										Type:     ast.CompassPointSouthEast,
										StartPos: token.Position{Row: 1, Column: 16},
										EndPos:   token.Position{Row: 1, Column: 17},
									},
								},
							},
						},
					},
					LeftBrace:  token.Position{Row: 1, Column: 7},
					RightBrace: token.Position{Row: 1, Column: 19},
				},
			},
			"NodeWithPortNameAndCompassPointSouth": {
				in: `graph { foo:f0:s }`,
				want: ast.Graph{
					GraphStart: token.Position{Row: 1, Column: 1},
					Stmts: []ast.Stmt{
						&ast.NodeStmt{
							NodeID: ast.NodeID{
								ID: ast.ID{
									Literal:  "foo",
									StartPos: token.Position{Row: 1, Column: 9},
									EndPos:   token.Position{Row: 1, Column: 11},
								}, Port: &ast.Port{
									Name: &ast.ID{
										Literal:  "f0",
										StartPos: token.Position{Row: 1, Column: 13},
										EndPos:   token.Position{Row: 1, Column: 14},
									},
									CompassPoint: &ast.CompassPoint{
										Type:     ast.CompassPointSouth,
										StartPos: token.Position{Row: 1, Column: 16},
										EndPos:   token.Position{Row: 1, Column: 16},
									},
								},
							},
						},
					},
					LeftBrace:  token.Position{Row: 1, Column: 7},
					RightBrace: token.Position{Row: 1, Column: 18},
				},
			},
			"NodeWithPortNameAndCompassPointSouthWest": {
				in: `graph { foo:f0:sw }`,
				want: ast.Graph{
					GraphStart: token.Position{Row: 1, Column: 1},
					Stmts: []ast.Stmt{
						&ast.NodeStmt{
							NodeID: ast.NodeID{
								ID: ast.ID{
									Literal:  "foo",
									StartPos: token.Position{Row: 1, Column: 9},
									EndPos:   token.Position{Row: 1, Column: 11},
								}, Port: &ast.Port{
									Name: &ast.ID{
										Literal:  "f0",
										StartPos: token.Position{Row: 1, Column: 13},
										EndPos:   token.Position{Row: 1, Column: 14},
									},
									CompassPoint: &ast.CompassPoint{
										Type:     ast.CompassPointSouthWest,
										StartPos: token.Position{Row: 1, Column: 16},
										EndPos:   token.Position{Row: 1, Column: 17},
									},
								},
							},
						},
					},
					LeftBrace:  token.Position{Row: 1, Column: 7},
					RightBrace: token.Position{Row: 1, Column: 19},
				},
			},
			"NodeWithPortNameAndCompassPointWest": {
				in: `graph { foo:f0:w }`,
				want: ast.Graph{
					GraphStart: token.Position{Row: 1, Column: 1},
					Stmts: []ast.Stmt{
						&ast.NodeStmt{
							NodeID: ast.NodeID{
								ID: ast.ID{
									Literal:  "foo",
									StartPos: token.Position{Row: 1, Column: 9},
									EndPos:   token.Position{Row: 1, Column: 11},
								}, Port: &ast.Port{
									Name: &ast.ID{
										Literal:  "f0",
										StartPos: token.Position{Row: 1, Column: 13},
										EndPos:   token.Position{Row: 1, Column: 14},
									},
									CompassPoint: &ast.CompassPoint{
										Type:     ast.CompassPointWest,
										StartPos: token.Position{Row: 1, Column: 16},
										EndPos:   token.Position{Row: 1, Column: 16},
									},
								},
							},
						},
					},
					LeftBrace:  token.Position{Row: 1, Column: 7},
					RightBrace: token.Position{Row: 1, Column: 18},
				},
			},
			"NodeWithPortNameAndCompassPointNorthWest": {
				in: `graph { foo:f0:nw }`,
				want: ast.Graph{
					GraphStart: token.Position{Row: 1, Column: 1},
					Stmts: []ast.Stmt{
						&ast.NodeStmt{
							NodeID: ast.NodeID{
								ID: ast.ID{
									Literal:  "foo",
									StartPos: token.Position{Row: 1, Column: 9},
									EndPos:   token.Position{Row: 1, Column: 11},
								}, Port: &ast.Port{
									Name: &ast.ID{
										Literal:  "f0",
										StartPos: token.Position{Row: 1, Column: 13},
										EndPos:   token.Position{Row: 1, Column: 14},
									},
									CompassPoint: &ast.CompassPoint{
										Type:     ast.CompassPointNorthWest,
										StartPos: token.Position{Row: 1, Column: 16},
										EndPos:   token.Position{Row: 1, Column: 17},
									},
								},
							},
						},
					},
					LeftBrace:  token.Position{Row: 1, Column: 7},
					RightBrace: token.Position{Row: 1, Column: 19},
				},
			},
			"NodeWithPortNameAndCompassPointCenter": {
				in: `graph { foo:f0:c }`,
				want: ast.Graph{
					GraphStart: token.Position{Row: 1, Column: 1},
					Stmts: []ast.Stmt{
						&ast.NodeStmt{
							NodeID: ast.NodeID{
								ID: ast.ID{
									Literal:  "foo",
									StartPos: token.Position{Row: 1, Column: 9},
									EndPos:   token.Position{Row: 1, Column: 11},
								}, Port: &ast.Port{
									Name: &ast.ID{
										Literal:  "f0",
										StartPos: token.Position{Row: 1, Column: 13},
										EndPos:   token.Position{Row: 1, Column: 14},
									},
									CompassPoint: &ast.CompassPoint{
										Type:     ast.CompassPointCenter,
										StartPos: token.Position{Row: 1, Column: 16},
										EndPos:   token.Position{Row: 1, Column: 16},
									},
								},
							},
						},
					},
					LeftBrace:  token.Position{Row: 1, Column: 7},
					RightBrace: token.Position{Row: 1, Column: 18},
				},
			},
			"NodeWithCompassPointNorth": {
				in: `graph { foo:n }`,
				want: ast.Graph{
					GraphStart: token.Position{Row: 1, Column: 1},
					Stmts: []ast.Stmt{
						&ast.NodeStmt{
							NodeID: ast.NodeID{
								ID: ast.ID{
									Literal:  "foo",
									StartPos: token.Position{Row: 1, Column: 9},
									EndPos:   token.Position{Row: 1, Column: 11},
								}, Port: &ast.Port{
									CompassPoint: &ast.CompassPoint{
										Type:     ast.CompassPointNorth,
										StartPos: token.Position{Row: 1, Column: 13},
										EndPos:   token.Position{Row: 1, Column: 13},
									},
								},
							},
						},
					},
					LeftBrace:  token.Position{Row: 1, Column: 7},
					RightBrace: token.Position{Row: 1, Column: 15},
				},
			},
			"NodeWithPortNameEqualToACompassPoint": { // https://graphviz.org/docs/attr-types/portPos
				in: `graph { foo:n:n }`,
				want: ast.Graph{
					GraphStart: token.Position{Row: 1, Column: 1},
					Stmts: []ast.Stmt{
						&ast.NodeStmt{
							NodeID: ast.NodeID{
								ID: ast.ID{
									Literal:  "foo",
									StartPos: token.Position{Row: 1, Column: 9},
									EndPos:   token.Position{Row: 1, Column: 11},
								}, Port: &ast.Port{
									Name: &ast.ID{
										Literal:  "n",
										StartPos: token.Position{Row: 1, Column: 13},
										EndPos:   token.Position{Row: 1, Column: 13},
									},
									CompassPoint: &ast.CompassPoint{
										Type:     ast.CompassPointNorth,
										StartPos: token.Position{Row: 1, Column: 15},
										EndPos:   token.Position{Row: 1, Column: 15},
									},
								},
							},
						},
					},
					LeftBrace:  token.Position{Row: 1, Column: 7},
					RightBrace: token.Position{Row: 1, Column: 17},
				},
			},
			"OnlyNodeWithEmptyAttributeList": {
				in: "graph { foo [] }",
				want: ast.Graph{
					GraphStart: token.Position{Row: 1, Column: 1},
					Stmts: []ast.Stmt{
						&ast.NodeStmt{
							NodeID: ast.NodeID{
								ID: ast.ID{
									Literal:  "foo",
									StartPos: token.Position{Row: 1, Column: 9},
									EndPos:   token.Position{Row: 1, Column: 11},
								},
							},
							AttrList: &ast.AttrList{
								LeftBracket:  token.Position{Row: 1, Column: 13},
								RightBracket: token.Position{Row: 1, Column: 14},
							},
						},
					},
					LeftBrace:  token.Position{Row: 1, Column: 7},
					RightBrace: token.Position{Row: 1, Column: 16},
				},
			},
			"NodeWithSingleAttributeAndEmptyAttributeList": {
				in: "graph { foo [] [a=b] }",
				want: ast.Graph{
					GraphStart: token.Position{Row: 1, Column: 1},
					Stmts: []ast.Stmt{
						&ast.NodeStmt{
							NodeID: ast.NodeID{
								ID: ast.ID{
									Literal:  "foo",
									StartPos: token.Position{Row: 1, Column: 9},
									EndPos:   token.Position{Row: 1, Column: 11},
								},
							},
							AttrList: &ast.AttrList{
								Next: &ast.AttrList{
									AList: &ast.AList{
										Attribute: ast.Attribute{
											Name:  ast.ID{Literal: "a", StartPos: token.Position{Row: 1, Column: 17}, EndPos: token.Position{Row: 1, Column: 17}},
											Value: ast.ID{Literal: "b", StartPos: token.Position{Row: 1, Column: 19}, EndPos: token.Position{Row: 1, Column: 19}},
										},
									},
									LeftBracket:  token.Position{Row: 1, Column: 16},
									RightBracket: token.Position{Row: 1, Column: 20},
								},
								LeftBracket:  token.Position{Row: 1, Column: 13},
								RightBracket: token.Position{Row: 1, Column: 14},
							},
						},
					},
					LeftBrace:  token.Position{Row: 1, Column: 7},
					RightBrace: token.Position{Row: 1, Column: 22},
				},
			},
			"NodeWithSingleAttribute": {
				in: "graph { foo [a=b] }",
				want: ast.Graph{
					GraphStart: token.Position{Row: 1, Column: 1},
					Stmts: []ast.Stmt{
						&ast.NodeStmt{
							NodeID: ast.NodeID{
								ID: ast.ID{
									Literal:  "foo",
									StartPos: token.Position{Row: 1, Column: 9},
									EndPos:   token.Position{Row: 1, Column: 11},
								},
							},
							AttrList: &ast.AttrList{
								AList: &ast.AList{
									Attribute: ast.Attribute{
										Name: ast.ID{
											Literal:  "a",
											StartPos: token.Position{Row: 1, Column: 14},
											EndPos:   token.Position{Row: 1, Column: 14},
										},
										Value: ast.ID{
											Literal:  "b",
											StartPos: token.Position{Row: 1, Column: 16},
											EndPos:   token.Position{Row: 1, Column: 16},
										},
									},
								},
								LeftBracket:  token.Position{Row: 1, Column: 13},
								RightBracket: token.Position{Row: 1, Column: 17},
							},
						},
					},
					LeftBrace:  token.Position{Row: 1, Column: 7},
					RightBrace: token.Position{Row: 1, Column: 19},
				},
			},
			"NodeWithAttributesAndTrailingComma": {
				in: "graph { foo [a=b,] }",
				want: ast.Graph{
					GraphStart: token.Position{Row: 1, Column: 1},
					Stmts: []ast.Stmt{
						&ast.NodeStmt{
							NodeID: ast.NodeID{
								ID: ast.ID{
									Literal:  "foo",
									StartPos: token.Position{Row: 1, Column: 9},
									EndPos:   token.Position{Row: 1, Column: 11},
								},
							},
							AttrList: &ast.AttrList{
								AList:        &ast.AList{Attribute: ast.Attribute{Name: ast.ID{Literal: "a", StartPos: token.Position{Row: 1, Column: 14}, EndPos: token.Position{Row: 1, Column: 14}}, Value: ast.ID{Literal: "b", StartPos: token.Position{Row: 1, Column: 16}, EndPos: token.Position{Row: 1, Column: 16}}}},
								LeftBracket:  token.Position{Row: 1, Column: 13},
								RightBracket: token.Position{Row: 1, Column: 18},
							},
						},
					},
					LeftBrace:  token.Position{Row: 1, Column: 7},
					RightBrace: token.Position{Row: 1, Column: 20},
				},
			},
			"NodeWithAttributesAndTrailingSemicolon": {
				in: "graph { foo [a=b;] }",
				want: ast.Graph{
					GraphStart: token.Position{Row: 1, Column: 1},
					Stmts: []ast.Stmt{
						&ast.NodeStmt{
							NodeID: ast.NodeID{
								ID: ast.ID{
									Literal:  "foo",
									StartPos: token.Position{Row: 1, Column: 9},
									EndPos:   token.Position{Row: 1, Column: 11},
								},
							},
							AttrList: &ast.AttrList{
								AList: &ast.AList{
									Attribute: ast.Attribute{
										Name: ast.ID{
											Literal:  "a",
											StartPos: token.Position{Row: 1, Column: 14},
											EndPos:   token.Position{Row: 1, Column: 14},
										},
										Value: ast.ID{
											Literal:  "b",
											StartPos: token.Position{Row: 1, Column: 16},
											EndPos:   token.Position{Row: 1, Column: 16},
										},
									},
								},
								LeftBracket:  token.Position{Row: 1, Column: 13},
								RightBracket: token.Position{Row: 1, Column: 18},
							},
						},
					},
					LeftBrace:  token.Position{Row: 1, Column: 7},
					RightBrace: token.Position{Row: 1, Column: 20},
				},
			},
			"NodeWithAttributeOverriding": {
				in: "graph { foo [a=b;c=d]; foo [a=e] }",
				want: ast.Graph{
					GraphStart: token.Position{Row: 1, Column: 1},
					Stmts: []ast.Stmt{
						&ast.NodeStmt{
							NodeID: ast.NodeID{
								ID: ast.ID{
									Literal:  "foo",
									StartPos: token.Position{Row: 1, Column: 9},
									EndPos:   token.Position{Row: 1, Column: 11},
								},
							},
							AttrList: &ast.AttrList{
								AList: &ast.AList{
									Attribute: ast.Attribute{
										Name: ast.ID{
											Literal:  "a",
											StartPos: token.Position{Row: 1, Column: 14},
											EndPos:   token.Position{Row: 1, Column: 14},
										},
										Value: ast.ID{
											Literal:  "b",
											StartPos: token.Position{Row: 1, Column: 16},
											EndPos:   token.Position{Row: 1, Column: 16},
										},
									},
									Next: &ast.AList{
										Attribute: ast.Attribute{
											Name: ast.ID{
												Literal:  "c",
												StartPos: token.Position{Row: 1, Column: 18},
												EndPos:   token.Position{Row: 1, Column: 18},
											},
											Value: ast.ID{
												Literal:  "d",
												StartPos: token.Position{Row: 1, Column: 20},
												EndPos:   token.Position{Row: 1, Column: 20},
											},
										},
									},
								},
								LeftBracket:  token.Position{Row: 1, Column: 13},
								RightBracket: token.Position{Row: 1, Column: 21},
							},
						},
						&ast.NodeStmt{
							NodeID: ast.NodeID{
								ID: ast.ID{
									Literal:  "foo",
									StartPos: token.Position{Row: 1, Column: 24},
									EndPos:   token.Position{Row: 1, Column: 26},
								},
							},
							AttrList: &ast.AttrList{
								AList: &ast.AList{
									Attribute: ast.Attribute{
										Name: ast.ID{
											Literal:  "a",
											StartPos: token.Position{Row: 1, Column: 29},
											EndPos:   token.Position{Row: 1, Column: 29},
										},
										Value: ast.ID{
											Literal:  "e",
											StartPos: token.Position{Row: 1, Column: 31},
											EndPos:   token.Position{Row: 1, Column: 31},
										},
									},
								},
								LeftBracket:  token.Position{Row: 1, Column: 28},
								RightBracket: token.Position{Row: 1, Column: 32},
							},
						},
					},
					LeftBrace:  token.Position{Row: 1, Column: 7},
					RightBrace: token.Position{Row: 1, Column: 34},
				},
			},
			"NodeWithMultipleAttributesInSingleBracketPair": {
				in: "graph { foo [a=b c=d,e=f;g=h] }",
				want: ast.Graph{
					GraphStart: token.Position{Row: 1, Column: 1},
					Stmts: []ast.Stmt{
						&ast.NodeStmt{
							NodeID: ast.NodeID{
								ID: ast.ID{
									Literal:  "foo",
									StartPos: token.Position{Row: 1, Column: 9},
									EndPos:   token.Position{Row: 1, Column: 11},
								},
							},
							AttrList: &ast.AttrList{
								AList: &ast.AList{
									Attribute: ast.Attribute{
										Name: ast.ID{
											Literal:  "a",
											StartPos: token.Position{Row: 1, Column: 14},
											EndPos:   token.Position{Row: 1, Column: 14},
										}, Value: ast.ID{
											Literal:  "b",
											StartPos: token.Position{Row: 1, Column: 16},
											EndPos:   token.Position{Row: 1, Column: 16},
										},
									},
									Next: &ast.AList{
										Attribute: ast.Attribute{
											Name: ast.ID{
												Literal:  "c",
												StartPos: token.Position{Row: 1, Column: 18},
												EndPos:   token.Position{Row: 1, Column: 18},
											}, Value: ast.ID{
												Literal:  "d",
												StartPos: token.Position{Row: 1, Column: 20},
												EndPos:   token.Position{Row: 1, Column: 20},
											},
										},
										Next: &ast.AList{
											Attribute: ast.Attribute{
												Name: ast.ID{
													Literal:  "e",
													StartPos: token.Position{Row: 1, Column: 22},
													EndPos:   token.Position{Row: 1, Column: 22},
												}, Value: ast.ID{
													Literal:  "f",
													StartPos: token.Position{Row: 1, Column: 24},
													EndPos:   token.Position{Row: 1, Column: 24},
												},
											},
											Next: &ast.AList{
												Attribute: ast.Attribute{
													Name: ast.ID{
														Literal:  "g",
														StartPos: token.Position{Row: 1, Column: 26},
														EndPos:   token.Position{Row: 1, Column: 26},
													},
													Value: ast.ID{
														Literal:  "h",
														StartPos: token.Position{Row: 1, Column: 28},
														EndPos:   token.Position{Row: 1, Column: 28},
													},
												},
											},
										},
									},
								},
								LeftBracket:  token.Position{Row: 1, Column: 13},
								RightBracket: token.Position{Row: 1, Column: 29},
							},
						},
					},
					LeftBrace:  token.Position{Row: 1, Column: 7},
					RightBrace: token.Position{Row: 1, Column: 31},
				},
			},
			"NodeWithMultipleAttributesInMultipleBracketPairs": {
				in: "graph { foo [a=b c=d][e=f;g=h] }",
				want: ast.Graph{
					GraphStart: token.Position{Row: 1, Column: 1},
					Stmts: []ast.Stmt{
						&ast.NodeStmt{
							NodeID: ast.NodeID{
								ID: ast.ID{
									Literal:  "foo",
									StartPos: token.Position{Row: 1, Column: 9},
									EndPos:   token.Position{Row: 1, Column: 11},
								},
							},
							AttrList: &ast.AttrList{
								AList: &ast.AList{
									Attribute: ast.Attribute{
										Name: ast.ID{
											Literal:  "a",
											StartPos: token.Position{Row: 1, Column: 14},
											EndPos:   token.Position{Row: 1, Column: 14},
										},
										Value: ast.ID{
											Literal:  "b",
											StartPos: token.Position{Row: 1, Column: 16},
											EndPos:   token.Position{Row: 1, Column: 16},
										},
									},
									Next: &ast.AList{
										Attribute: ast.Attribute{
											Name:  ast.ID{Literal: "c", StartPos: token.Position{Row: 1, Column: 18}, EndPos: token.Position{Row: 1, Column: 18}},
											Value: ast.ID{Literal: "d", StartPos: token.Position{Row: 1, Column: 20}, EndPos: token.Position{Row: 1, Column: 20}},
										},
									},
								},
								Next: &ast.AttrList{
									AList: &ast.AList{
										Attribute: ast.Attribute{
											Name:  ast.ID{Literal: "e", StartPos: token.Position{Row: 1, Column: 23}, EndPos: token.Position{Row: 1, Column: 23}},
											Value: ast.ID{Literal: "f", StartPos: token.Position{Row: 1, Column: 25}, EndPos: token.Position{Row: 1, Column: 25}},
										},
										Next: &ast.AList{
											Attribute: ast.Attribute{
												Name:  ast.ID{Literal: "g", StartPos: token.Position{Row: 1, Column: 27}, EndPos: token.Position{Row: 1, Column: 27}},
												Value: ast.ID{Literal: "h", StartPos: token.Position{Row: 1, Column: 29}, EndPos: token.Position{Row: 1, Column: 29}},
											},
										},
									},
									LeftBracket:  token.Position{Row: 1, Column: 22},
									RightBracket: token.Position{Row: 1, Column: 30},
								},
								LeftBracket:  token.Position{Row: 1, Column: 13},
								RightBracket: token.Position{Row: 1, Column: 21},
							},
						},
					},
					LeftBrace:  token.Position{Row: 1, Column: 7},
					RightBrace: token.Position{Row: 1, Column: 32},
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

	t.Run("EdgeStmt", func(t *testing.T) {
		tests := map[string]struct {
			in   string
			want ast.Graph
			err  error
		}{
			"SingleUndirectedEdge": {
				in: "graph { 1 -- 2 }",
				want: ast.Graph{
					GraphStart: token.Position{Row: 1, Column: 1},
					Stmts: []ast.Stmt{
						&ast.EdgeStmt{
							Left: ast.NodeID{
								ID: ast.ID{
									Literal:  "1",
									StartPos: token.Position{Row: 1, Column: 9},
									EndPos:   token.Position{Row: 1, Column: 9},
								},
							},
							Right: ast.EdgeRHS{
								Right: ast.NodeID{
									ID: ast.ID{
										Literal:  "2",
										StartPos: token.Position{Row: 1, Column: 14},
										EndPos:   token.Position{Row: 1, Column: 14},
									},
								},
								StartPos: token.Position{Row: 1, Column: 11},
							},
						},
					},
					LeftBrace:  token.Position{Row: 1, Column: 7},
					RightBrace: token.Position{Row: 1, Column: 16},
				},
			},
			"SingleDirectedEdge": {
				in: "digraph { 1 -> 2 }",
				want: ast.Graph{
					GraphStart: token.Position{Row: 1, Column: 1},
					Directed:   true,
					Stmts: []ast.Stmt{
						&ast.EdgeStmt{
							Left: ast.NodeID{
								ID: ast.ID{
									Literal:  "1",
									StartPos: token.Position{Row: 1, Column: 11},
									EndPos:   token.Position{Row: 1, Column: 11},
								},
							},
							Right: ast.EdgeRHS{
								Directed: true,
								Right: ast.NodeID{
									ID: ast.ID{
										Literal:  "2",
										StartPos: token.Position{Row: 1, Column: 16},
										EndPos:   token.Position{Row: 1, Column: 16},
									},
								},
								StartPos: token.Position{Row: 1, Column: 13},
							},
						},
					},
					LeftBrace:  token.Position{Row: 1, Column: 9},
					RightBrace: token.Position{Row: 1, Column: 18},
				},
			},
			"MultipleDirectedEdgesWithAttributeList": {
				in: "digraph { 1 -> 2 -> 3 -> 4 [a=b] }",
				want: ast.Graph{
					GraphStart: token.Position{Row: 1, Column: 1},
					Directed:   true,
					Stmts: []ast.Stmt{
						&ast.EdgeStmt{
							Left: ast.NodeID{
								ID: ast.ID{
									Literal:  "1",
									StartPos: token.Position{Row: 1, Column: 11},
									EndPos:   token.Position{Row: 1, Column: 11},
								},
							},
							Right: ast.EdgeRHS{
								Directed: true,
								Right: ast.NodeID{
									ID: ast.ID{
										Literal:  "2",
										StartPos: token.Position{Row: 1, Column: 16},
										EndPos:   token.Position{Row: 1, Column: 16},
									},
								},
								Next: &ast.EdgeRHS{
									Directed: true,
									Right: ast.NodeID{
										ID: ast.ID{
											Literal:  "3",
											StartPos: token.Position{Row: 1, Column: 21},
											EndPos:   token.Position{Row: 1, Column: 21},
										},
									},
									Next: &ast.EdgeRHS{
										Directed: true,
										Right: ast.NodeID{
											ID: ast.ID{
												Literal:  "4",
												StartPos: token.Position{Row: 1, Column: 26},
												EndPos:   token.Position{Row: 1, Column: 26},
											},
										},
										StartPos: token.Position{Row: 1, Column: 23},
									},
									StartPos: token.Position{Row: 1, Column: 18},
								},
								StartPos: token.Position{Row: 1, Column: 13},
							},
							AttrList: &ast.AttrList{
								AList: &ast.AList{
									Attribute: ast.Attribute{
										Name: ast.ID{
											Literal:  "a",
											StartPos: token.Position{Row: 1, Column: 29},
											EndPos:   token.Position{Row: 1, Column: 29},
										}, Value: ast.ID{
											Literal:  "b",
											StartPos: token.Position{Row: 1, Column: 31},
											EndPos:   token.Position{Row: 1, Column: 31},
										},
									},
								},
								LeftBracket:  token.Position{Row: 1, Column: 28},
								RightBracket: token.Position{Row: 1, Column: 32},
							},
						},
					},
					LeftBrace:  token.Position{Row: 1, Column: 9},
					RightBrace: token.Position{Row: 1, Column: 34},
				},
			},
			"EdgeWithLHSShortSubgraph": {
				in: "digraph { {A B} -> C }",
				want: ast.Graph{
					GraphStart: token.Position{Row: 1, Column: 1},
					Directed:   true,
					Stmts: []ast.Stmt{
						&ast.EdgeStmt{
							Left: ast.Subgraph{
								Stmts: []ast.Stmt{
									&ast.NodeStmt{
										NodeID: ast.NodeID{
											ID: ast.ID{
												Literal:  "A",
												StartPos: token.Position{Row: 1, Column: 12},
												EndPos:   token.Position{Row: 1, Column: 12},
											},
										},
									},
									&ast.NodeStmt{
										NodeID: ast.NodeID{
											ID: ast.ID{
												Literal:  "B",
												StartPos: token.Position{Row: 1, Column: 14},
												EndPos:   token.Position{Row: 1, Column: 14},
											},
										},
									},
								},
								LeftBrace:  token.Position{Row: 1, Column: 11},
								RightBrace: token.Position{Row: 1, Column: 15},
							},
							Right: ast.EdgeRHS{
								Directed: true,
								Right: ast.NodeID{
									ID: ast.ID{
										Literal:  "C",
										StartPos: token.Position{Row: 1, Column: 20},
										EndPos:   token.Position{Row: 1, Column: 20},
									},
								},
								StartPos: token.Position{Row: 1, Column: 17},
							},
						},
					},
					LeftBrace:  token.Position{Row: 1, Column: 9},
					RightBrace: token.Position{Row: 1, Column: 22},
				},
			},
			"EdgeWithRHSShortSubgraph": {
				in: "digraph { A -> {B C} }",
				want: ast.Graph{
					GraphStart: token.Position{Row: 1, Column: 1},
					Directed:   true,
					Stmts: []ast.Stmt{
						&ast.EdgeStmt{
							Left: ast.NodeID{
								ID: ast.ID{
									Literal:  "A",
									StartPos: token.Position{Row: 1, Column: 11},
									EndPos:   token.Position{Row: 1, Column: 11},
								},
							},
							Right: ast.EdgeRHS{
								Directed: true,
								Right: ast.Subgraph{
									Stmts: []ast.Stmt{
										&ast.NodeStmt{
											NodeID: ast.NodeID{
												ID: ast.ID{
													Literal:  "B",
													StartPos: token.Position{Row: 1, Column: 17},
													EndPos:   token.Position{Row: 1, Column: 17},
												},
											},
										},
										&ast.NodeStmt{
											NodeID: ast.NodeID{
												ID: ast.ID{
													Literal:  "C",
													StartPos: token.Position{Row: 1, Column: 19},
													EndPos:   token.Position{Row: 1, Column: 19},
												},
											},
										},
									},
									LeftBrace:  token.Position{Row: 1, Column: 16},
									RightBrace: token.Position{Row: 1, Column: 20},
								},
								StartPos: token.Position{Row: 1, Column: 13},
							},
						},
					},
					LeftBrace:  token.Position{Row: 1, Column: 9},
					RightBrace: token.Position{Row: 1, Column: 22},
				},
			},
			"EdgeWithNestedSubraphs": {
				in: "graph { {1 2} -- {3 -- {4 5}} }",
				want: ast.Graph{
					GraphStart: token.Position{Row: 1, Column: 1},
					Stmts: []ast.Stmt{
						&ast.EdgeStmt{
							Left: ast.Subgraph{
								Stmts: []ast.Stmt{
									&ast.NodeStmt{
										NodeID: ast.NodeID{
											ID: ast.ID{
												Literal:  "1",
												StartPos: token.Position{Row: 1, Column: 10},
												EndPos:   token.Position{Row: 1, Column: 10},
											},
										},
									},
									&ast.NodeStmt{
										NodeID: ast.NodeID{
											ID: ast.ID{
												Literal:  "2",
												StartPos: token.Position{Row: 1, Column: 12},
												EndPos:   token.Position{Row: 1, Column: 12},
											},
										},
									},
								},
								LeftBrace:  token.Position{Row: 1, Column: 9},
								RightBrace: token.Position{Row: 1, Column: 13},
							},
							Right: ast.EdgeRHS{
								Right: ast.Subgraph{
									Stmts: []ast.Stmt{
										&ast.EdgeStmt{
											Left: ast.NodeID{
												ID: ast.ID{
													Literal:  "3",
													StartPos: token.Position{Row: 1, Column: 19},
													EndPos:   token.Position{Row: 1, Column: 19},
												},
											},
											Right: ast.EdgeRHS{
												Right: ast.Subgraph{
													Stmts: []ast.Stmt{
														&ast.NodeStmt{
															NodeID: ast.NodeID{
																ID: ast.ID{
																	Literal:  "4",
																	StartPos: token.Position{Row: 1, Column: 25},
																	EndPos:   token.Position{Row: 1, Column: 25},
																},
															},
														},
														&ast.NodeStmt{
															NodeID: ast.NodeID{
																ID: ast.ID{
																	Literal:  "5",
																	StartPos: token.Position{Row: 1, Column: 27},
																	EndPos:   token.Position{Row: 1, Column: 27},
																},
															},
														},
													},
													LeftBrace:  token.Position{Row: 1, Column: 24},
													RightBrace: token.Position{Row: 1, Column: 28},
												},
												StartPos: token.Position{Row: 1, Column: 21},
											},
										},
									},
									LeftBrace:  token.Position{Row: 1, Column: 18},
									RightBrace: token.Position{Row: 1, Column: 29},
								},
								StartPos: token.Position{Row: 1, Column: 15},
							},
						},
					},
					LeftBrace:  token.Position{Row: 1, Column: 7},
					RightBrace: token.Position{Row: 1, Column: 31},
				},
			},
			"EdgeWithRHSExplicitSubraph": {
				in: "digraph { A -> subgraph foo {B C} }",
				want: ast.Graph{
					GraphStart: token.Position{Row: 1, Column: 1},
					Directed:   true,
					Stmts: []ast.Stmt{
						&ast.EdgeStmt{
							Left: ast.NodeID{
								ID: ast.ID{
									Literal:  "A",
									StartPos: token.Position{Row: 1, Column: 11},
									EndPos:   token.Position{Row: 1, Column: 11},
								},
							},
							Right: ast.EdgeRHS{
								Directed: true,
								Right: ast.Subgraph{
									ID: &ast.ID{
										Literal:  "foo",
										StartPos: token.Position{Row: 1, Column: 25},
										EndPos:   token.Position{Row: 1, Column: 27},
									},
									Stmts: []ast.Stmt{
										&ast.NodeStmt{
											NodeID: ast.NodeID{
												ID: ast.ID{
													Literal:  "B",
													StartPos: token.Position{Row: 1, Column: 30},
													EndPos:   token.Position{Row: 1, Column: 30},
												},
											},
										},
										&ast.NodeStmt{
											NodeID: ast.NodeID{
												ID: ast.ID{
													Literal:  "C",
													StartPos: token.Position{Row: 1, Column: 32},
													EndPos:   token.Position{Row: 1, Column: 32},
												},
											},
										},
									},
									SubgraphStart: &token.Position{Row: 1, Column: 16},
									LeftBrace:     token.Position{Row: 1, Column: 29},
									RightBrace:    token.Position{Row: 1, Column: 33},
								},
								StartPos: token.Position{Row: 1, Column: 13},
							},
						},
					},
					LeftBrace:  token.Position{Row: 1, Column: 9},
					RightBrace: token.Position{Row: 1, Column: 35},
				},
			},
			"EdgeWithPorts": {
				in: `digraph {
			"node4":f0:n -> node5:f1;
}`,
				want: ast.Graph{
					GraphStart: token.Position{Row: 1, Column: 1},
					Directed:   true,
					Stmts: []ast.Stmt{
						&ast.EdgeStmt{
							Left: ast.NodeID{
								ID: ast.ID{
									Literal:  `"node4"`,
									StartPos: token.Position{Row: 2, Column: 4},
									EndPos:   token.Position{Row: 2, Column: 10},
								},
								Port: &ast.Port{
									Name: &ast.ID{
										Literal:  "f0",
										StartPos: token.Position{Row: 2, Column: 12},
										EndPos:   token.Position{Row: 2, Column: 13},
									},
									CompassPoint: &ast.CompassPoint{
										Type:     ast.CompassPointNorth,
										StartPos: token.Position{Row: 2, Column: 15},
										EndPos:   token.Position{Row: 2, Column: 15},
									},
								},
							},
							Right: ast.EdgeRHS{
								Directed: true,
								Right: ast.NodeID{
									ID: ast.ID{
										Literal:  "node5",
										StartPos: token.Position{Row: 2, Column: 20},
										EndPos:   token.Position{Row: 2, Column: 24},
									},
									Port: &ast.Port{
										Name: &ast.ID{
											Literal:  "f1",
											StartPos: token.Position{Row: 2, Column: 26},
											EndPos:   token.Position{Row: 2, Column: 27},
										},
									},
								},
								StartPos: token.Position{Row: 2, Column: 17},
							},
						},
					},
					LeftBrace:  token.Position{Row: 1, Column: 9},
					RightBrace: token.Position{Row: 3, Column: 1},
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

	t.Run("AttrStmt", func(t *testing.T) {
		tests := map[string]struct {
			in   string
			want ast.Graph
			err  error
		}{
			"OnlyGraph": {
				in: "graph { graph [] }",
				want: ast.Graph{
					GraphStart: token.Position{Row: 1, Column: 1},
					Stmts: []ast.Stmt{
						&ast.AttrStmt{
							ID: ast.ID{
								Literal:  "graph",
								StartPos: token.Position{Row: 1, Column: 9},
								EndPos:   token.Position{Row: 1, Column: 13},
							},
							AttrList: ast.AttrList{
								LeftBracket:  token.Position{Row: 1, Column: 15},
								RightBracket: token.Position{Row: 1, Column: 16},
							},
						},
					},
					LeftBrace:  token.Position{Row: 1, Column: 7},
					RightBrace: token.Position{Row: 1, Column: 18},
				},
			},
			"OnlyNode": {
				in: "graph { node [] }",
				want: ast.Graph{
					GraphStart: token.Position{Row: 1, Column: 1},
					Stmts: []ast.Stmt{
						&ast.AttrStmt{
							ID: ast.ID{
								Literal:  "node",
								StartPos: token.Position{Row: 1, Column: 9},
								EndPos:   token.Position{Row: 1, Column: 12},
							},
							AttrList: ast.AttrList{
								LeftBracket:  token.Position{Row: 1, Column: 14},
								RightBracket: token.Position{Row: 1, Column: 15},
							},
						},
					},
					LeftBrace:  token.Position{Row: 1, Column: 7},
					RightBrace: token.Position{Row: 1, Column: 17},
				},
			},
			"OnlyEdge": {
				in: "graph { edge [] }",
				want: ast.Graph{
					GraphStart: token.Position{Row: 1, Column: 1},
					Stmts: []ast.Stmt{
						&ast.AttrStmt{
							ID: ast.ID{
								Literal:  "edge",
								StartPos: token.Position{Row: 1, Column: 9},
								EndPos:   token.Position{Row: 1, Column: 12},
							},
							AttrList: ast.AttrList{
								LeftBracket:  token.Position{Row: 1, Column: 14},
								RightBracket: token.Position{Row: 1, Column: 15},
							},
						},
					},
					LeftBrace:  token.Position{Row: 1, Column: 7},
					RightBrace: token.Position{Row: 1, Column: 17},
				},
			},
			"GraphWithAttribute": {
				in: "graph { graph [a=b] }",
				want: ast.Graph{
					GraphStart: token.Position{Row: 1, Column: 1},
					Stmts: []ast.Stmt{
						&ast.AttrStmt{
							ID: ast.ID{
								Literal:  "graph",
								StartPos: token.Position{Row: 1, Column: 9},
								EndPos:   token.Position{Row: 1, Column: 13},
							},
							AttrList: ast.AttrList{
								AList: &ast.AList{
									Attribute: ast.Attribute{
										Name: ast.ID{
											Literal:  "a",
											StartPos: token.Position{Row: 1, Column: 16},
											EndPos:   token.Position{Row: 1, Column: 16},
										}, Value: ast.ID{
											Literal:  "b",
											StartPos: token.Position{Row: 1, Column: 18},
											EndPos:   token.Position{Row: 1, Column: 18},
										},
									},
								},
								LeftBracket:  token.Position{Row: 1, Column: 15},
								RightBracket: token.Position{Row: 1, Column: 19},
							},
						},
					},
					LeftBrace:  token.Position{Row: 1, Column: 7},
					RightBrace: token.Position{Row: 1, Column: 21},
				},
			},
			"NodeWithAttribute": {
				in: "graph { node [a=b] }",
				want: ast.Graph{
					GraphStart: token.Position{Row: 1, Column: 1},
					Stmts: []ast.Stmt{
						&ast.AttrStmt{
							ID: ast.ID{
								Literal:  "node",
								StartPos: token.Position{Row: 1, Column: 9},
								EndPos:   token.Position{Row: 1, Column: 12},
							},
							AttrList: ast.AttrList{
								AList: &ast.AList{
									Attribute: ast.Attribute{
										Name: ast.ID{
											Literal:  "a",
											StartPos: token.Position{Row: 1, Column: 15},
											EndPos:   token.Position{Row: 1, Column: 15},
										}, Value: ast.ID{
											Literal:  "b",
											StartPos: token.Position{Row: 1, Column: 17},
											EndPos:   token.Position{Row: 1, Column: 17},
										},
									},
								},
								LeftBracket:  token.Position{Row: 1, Column: 14},
								RightBracket: token.Position{Row: 1, Column: 18},
							},
						},
					},
					LeftBrace:  token.Position{Row: 1, Column: 7},
					RightBrace: token.Position{Row: 1, Column: 20},
				},
			},
			"EdgeWithAttribute": {
				in: "graph { edge [a=b] }",
				want: ast.Graph{
					GraphStart: token.Position{Row: 1, Column: 1},
					Stmts: []ast.Stmt{
						&ast.AttrStmt{
							ID: ast.ID{
								Literal:  "edge",
								StartPos: token.Position{Row: 1, Column: 9},
								EndPos:   token.Position{Row: 1, Column: 12},
							},
							AttrList: ast.AttrList{
								AList: &ast.AList{
									Attribute: ast.Attribute{
										Name: ast.ID{
											Literal:  "a",
											StartPos: token.Position{Row: 1, Column: 15},
											EndPos:   token.Position{Row: 1, Column: 15},
										}, Value: ast.ID{
											Literal:  "b",
											StartPos: token.Position{Row: 1, Column: 17},
											EndPos:   token.Position{Row: 1, Column: 17},
										},
									},
								},
								LeftBracket:  token.Position{Row: 1, Column: 14},
								RightBracket: token.Position{Row: 1, Column: 18},
							},
						},
					},
					LeftBrace:  token.Position{Row: 1, Column: 7},
					RightBrace: token.Position{Row: 1, Column: 20},
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
					GraphStart: token.Position{Row: 1, Column: 1},
					Stmts: []ast.Stmt{
						ast.Attribute{
							Name: ast.ID{
								Literal:  "rank",
								StartPos: token.Position{Row: 1, Column: 9},
								EndPos:   token.Position{Row: 1, Column: 12},
							}, Value: ast.ID{
								Literal:  "same",
								StartPos: token.Position{Row: 1, Column: 16},
								EndPos:   token.Position{Row: 1, Column: 19},
							},
						},
					},
					LeftBrace:  token.Position{Row: 1, Column: 7},
					RightBrace: token.Position{Row: 1, Column: 22},
				},
			},
			"QuotedAttributeValueSpanningMultipleLines": {
				in: `graph { 	label="Rainy days
				in summer"
}`,
				want: ast.Graph{
					GraphStart: token.Position{Row: 1, Column: 1},
					Stmts: []ast.Stmt{
						ast.Attribute{
							Name: ast.ID{
								Literal:  "label",
								StartPos: token.Position{Row: 1, Column: 10},
								EndPos:   token.Position{Row: 1, Column: 14},
							}, Value: ast.ID{
								Literal: `"Rainy days
				in summer"`,
								StartPos: token.Position{Row: 1, Column: 16},
								EndPos:   token.Position{Row: 2, Column: 14},
							},
						},
					},
					LeftBrace:  token.Position{Row: 1, Column: 7},
					RightBrace: token.Position{Row: 3, Column: 1},
				},
			},
			// https://graphviz.org/doc/info/lang.html#comments-and-optional-formatting
			"QuotedAttributeValueSpanningMultipleLinesWithBackslashFollowedByNewline": {
				in: `graph { 	label="Rainy days\
				in summer"
}`,
				want: ast.Graph{
					GraphStart: token.Position{Row: 1, Column: 1},
					Stmts: []ast.Stmt{
						ast.Attribute{
							Name: ast.ID{
								Literal:  "label",
								StartPos: token.Position{Row: 1, Column: 10},
								EndPos:   token.Position{Row: 1, Column: 14},
							}, Value: ast.ID{
								Literal: `"Rainy days\
				in summer"`,
								StartPos: token.Position{Row: 1, Column: 16},
								EndPos:   token.Position{Row: 2, Column: 14},
							},
						},
					},
					LeftBrace:  token.Position{Row: 1, Column: 7},
					RightBrace: token.Position{Row: 3, Column: 1},
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
					GraphStart: token.Position{Row: 1, Column: 1},
					Stmts: []ast.Stmt{
						ast.Subgraph{
							SubgraphStart: &token.Position{Row: 1, Column: 9},
							LeftBrace:     token.Position{Row: 1, Column: 18},
							RightBrace:    token.Position{Row: 1, Column: 19},
						},
					},
					LeftBrace:  token.Position{Row: 1, Column: 7},
					RightBrace: token.Position{Row: 1, Column: 21},
				},
			},
			"EmptyWithoutKeyword": {
				in: "graph { {} }",
				want: ast.Graph{
					GraphStart: token.Position{Row: 1, Column: 1},
					Stmts: []ast.Stmt{
						ast.Subgraph{
							LeftBrace:  token.Position{Row: 1, Column: 9},
							RightBrace: token.Position{Row: 1, Column: 10},
						},
					},
					LeftBrace:  token.Position{Row: 1, Column: 7},
					RightBrace: token.Position{Row: 1, Column: 12},
				},
			},
			"SubgraphWithID": {
				in: "graph { subgraph foo {} }",
				want: ast.Graph{
					GraphStart: token.Position{Row: 1, Column: 1},
					Stmts: []ast.Stmt{
						ast.Subgraph{
							ID: &ast.ID{
								Literal:  "foo",
								StartPos: token.Position{Row: 1, Column: 18},
								EndPos:   token.Position{Row: 1, Column: 20},
							},
							SubgraphStart: &token.Position{Row: 1, Column: 9},
							LeftBrace:     token.Position{Row: 1, Column: 22},
							RightBrace:    token.Position{Row: 1, Column: 23},
						},
					},
					LeftBrace:  token.Position{Row: 1, Column: 7},
					RightBrace: token.Position{Row: 1, Column: 25},
				},
			},
			"SubgraphWithAttributesAndNodes": {
				in: `graph {
					subgraph {
						rank = same; A; B;
					}
				}`,
				want: ast.Graph{
					GraphStart: token.Position{Row: 1, Column: 1},
					Stmts: []ast.Stmt{
						ast.Subgraph{
							Stmts: []ast.Stmt{
								ast.Attribute{
									Name: ast.ID{
										Literal:  "rank",
										StartPos: token.Position{Row: 3, Column: 7},
										EndPos:   token.Position{Row: 3, Column: 10},
									}, Value: ast.ID{
										Literal:  "same",
										StartPos: token.Position{Row: 3, Column: 14},
										EndPos:   token.Position{Row: 3, Column: 17},
									},
								},
								&ast.NodeStmt{
									NodeID: ast.NodeID{
										ID: ast.ID{
											Literal:  "A",
											StartPos: token.Position{Row: 3, Column: 20},
											EndPos:   token.Position{Row: 3, Column: 20},
										},
									},
								},
								&ast.NodeStmt{
									NodeID: ast.NodeID{
										ID: ast.ID{
											Literal:  "B",
											StartPos: token.Position{Row: 3, Column: 23},
											EndPos:   token.Position{Row: 3, Column: 23},
										},
									},
								},
							},
							SubgraphStart: &token.Position{Row: 2, Column: 6},
							LeftBrace:     token.Position{Row: 2, Column: 15},
							RightBrace:    token.Position{Row: 4, Column: 6},
						},
					},
					LeftBrace:  token.Position{Row: 1, Column: 7},
					RightBrace: token.Position{Row: 5, Column: 5},
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
					GraphStart: token.Position{Row: 1, Column: 1},
					Comments: []ast.Comment{
						{
							Text:     "# ok",
							StartPos: token.Position{Row: 1, Column: 10},
							EndPos:   token.Position{Row: 1, Column: 13},
						},
					},
					LeftBrace:  token.Position{Row: 1, Column: 7},
					RightBrace: token.Position{Row: 2, Column: 5},
				},
			},
			"Single": {
				in: `graph { 
				// ok
				}`,
				want: ast.Graph{
					GraphStart: token.Position{Row: 1, Column: 1},
					Comments: []ast.Comment{
						{
							Text:     "// ok",
							StartPos: token.Position{Row: 2, Column: 5},
							EndPos:   token.Position{Row: 2, Column: 9},
						},
					},
					LeftBrace:  token.Position{Row: 1, Column: 7},
					RightBrace: token.Position{Row: 3, Column: 5},
				},
			},
			"MultiLine": {
				in: `graph { /* ok
				then */
				}`,
				want: ast.Graph{
					GraphStart: token.Position{Row: 1, Column: 1},
					Comments: []ast.Comment{
						{
							Text: `/* ok
				then */`,
							StartPos: token.Position{Row: 1, Column: 9},
							EndPos:   token.Position{Row: 2, Column: 11},
						},
					},
					LeftBrace:  token.Position{Row: 1, Column: 7},
					RightBrace: token.Position{Row: 3, Column: 5},
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
