package ast

import (
	"testing"

	"github.com/teleivo/assertive/assert"
	"github.com/teleivo/dot/internal/token"
)

func TestStringer(t *testing.T) {
	tests := map[string]struct {
		in   Node
		want string
	}{
		"NodeStmtWithAttrLists": {
			in: &NodeStmt{
				NodeID: NodeID{
					ID: ID{Literal: "foo"},
				},
				AttrList: &AttrList{
					AList: &AList{
						Attribute: Attribute{
							Name:  ID{Literal: "a"},
							Value: ID{Literal: "b"},
						},
						Next: &AList{
							Attribute: Attribute{
								Name:  ID{Literal: "c"},
								Value: ID{Literal: "d"},
							},
						},
					},
					Next: &AttrList{
						AList: &AList{
							Attribute: Attribute{
								Name:  ID{Literal: "e"},
								Value: ID{Literal: "f"},
							},
						},
					},
				},
			},
			want: `foo [a=b,c=d] [e=f]`,
		},
		"NodeStmtWithPortWithName": {
			in: &NodeStmt{
				NodeID: NodeID{ID: ID{Literal: "foo"}, Port: &Port{Name: &ID{Literal: `"f0"`}}},
			},
			want: `foo:"f0"`,
		},
		"NodeStmtWithPortWithNameAndCompassPoint": {
			in: &NodeStmt{
				NodeID: NodeID{
					ID: ID{Literal: "foo"},
					Port: &Port{
						Name:         &ID{Literal: `"f0"`},
						CompassPoint: &CompassPoint{Type: CompassPointNorthWest},
					},
				},
			},
			want: `foo:"f0":nw`,
		},
		"EdgeStmtWithSubgraph": {
			in: &EdgeStmt{
				Left: NodeID{ID: ID{Literal: "1"}},
				Right: EdgeRHS{
					Directed: true,
					Right: Subgraph{
						ID: &ID{Literal: "internal"},
						Stmts: []Stmt{
							&NodeStmt{NodeID: NodeID{ID: ID{Literal: "2"}}},
						},
					},
					Next: &EdgeRHS{
						Directed: true,
						Right:    NodeID{ID: ID{Literal: "3"}},
						Next: &EdgeRHS{
							Directed: true,
							Right: Subgraph{
								Stmts: []Stmt{
									&NodeStmt{NodeID: NodeID{ID: ID{Literal: "4"}}},
									&NodeStmt{NodeID: NodeID{ID: ID{Literal: "5"}}},
								},
							},
						},
					},
				},
				AttrList: &AttrList{
					AList: &AList{
						Attribute: Attribute{Name: ID{Literal: "a"}, Value: ID{Literal: "b"}},
					},
				},
			},
			want: `1 -> subgraph internal {2} -> 3 -> subgraph {4 5} [a=b]`,
		},
		"DigraphWithID": {
			in: Graph{
				Strict:   true,
				Directed: true,
				ID:       &ID{Literal: `"wonder"`},
			},
			want: `strict digraph "wonder" {}`,
		},
		"Attribute": {
			in: Graph{
				Stmts: []Stmt{
					Attribute{Name: ID{Literal: "foo"}, Value: ID{Literal: "bar"}},
				},
			},
			want: `graph {
	foo=bar
}`,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := test.in.String()

			assert.EqualValuesf(t, got, test.want, "String()")
		})
	}
}

func TestPosition(t *testing.T) {
	tests := map[string]struct {
		in        Positioner
		wantStart token.Position
		wantEnd   token.Position
	}{
		"NodeStmt": {
			in: &NodeStmt{
				NodeID: NodeID{
					ID: ID{
						Literal: `f1`,
						StartPos: token.Position{
							Row:    1,
							Column: 1,
						},
						EndPos: token.Position{
							Row:    1,
							Column: 2,
						},
					},
				},
			},
			wantStart: token.Position{
				Row:    1,
				Column: 1,
			},
			wantEnd: token.Position{
				Row:    1,
				Column: 2,
			},
		},
		"NodeStmtWithAttrList": {
			in: &NodeStmt{
				NodeID: NodeID{
					ID: ID{
						Literal: `f1`,
						StartPos: token.Position{
							Row:    1,
							Column: 1,
						},
						EndPos: token.Position{
							Row:    1,
							Column: 2,
						},
					},
				},
				AttrList: &AttrList{
					StartPos: token.Position{
						Row:    1,
						Column: 3,
					},
					EndPos: token.Position{
						Row:    1,
						Column: 5,
					},
				},
			},
			wantStart: token.Position{
				Row:    1,
				Column: 1,
			},
			wantEnd: token.Position{
				Row:    1,
				Column: 5,
			},
		},
		"NodeID": {
			in: NodeID{
				ID: ID{
					Literal: "pc",
					StartPos: token.Position{
						Row:    1,
						Column: 1,
					},
					EndPos: token.Position{
						Row:    1,
						Column: 2,
					},
				},
			},
			wantStart: token.Position{
				Row:    1,
				Column: 1,
			},
			wantEnd: token.Position{
				Row:    1,
				Column: 2,
			},
		},
		"NodeIDWithPort": {
			in: NodeID{
				ID: ID{
					Literal: "pc",
					StartPos: token.Position{
						Row:    1,
						Column: 1,
					},
					EndPos: token.Position{
						Row:    1,
						Column: 2,
					},
				},
				Port: &Port{
					Name: &ID{
						Literal: `"f0"`,
						StartPos: token.Position{
							Row:    1,
							Column: 3,
						},
						EndPos: token.Position{
							Row:    1,
							Column: 6,
						},
					},
				},
			},
			wantStart: token.Position{
				Row:    1,
				Column: 1,
			},
			wantEnd: token.Position{
				Row:    1,
				Column: 6,
			},
		},
		"PortWithName": {
			in: Port{
				Name: &ID{
					Literal: `"f0"`,
					StartPos: token.Position{
						Row:    1,
						Column: 2,
					},
					EndPos: token.Position{
						Row:    1,
						Column: 5,
					},
				},
			},
			wantStart: token.Position{
				Row:    1,
				Column: 1,
			},
			wantEnd: token.Position{
				Row:    1,
				Column: 5,
			},
		},
		"PortWithCompassPoint": {
			in: Port{
				CompassPoint: &CompassPoint{
					Type: CompassPointSouth,
					StartPos: token.Position{
						Row:    1,
						Column: 2,
					},
					EndPos: token.Position{
						Row:    1,
						Column: 3,
					},
				},
			},
			wantStart: token.Position{
				Row:    1,
				Column: 1,
			},
			wantEnd: token.Position{
				Row:    1,
				Column: 3,
			},
		},
		"PortWithNameAndCompassPoint": {
			in: Port{
				Name: &ID{
					Literal: `"f0"`,
					StartPos: token.Position{
						Row:    1,
						Column: 2,
					},
					EndPos: token.Position{
						Row:    1,
						Column: 5,
					},
				},
				CompassPoint: &CompassPoint{
					Type: CompassPointSouthWest,
					StartPos: token.Position{
						Row:    1,
						Column: 7,
					},
					EndPos: token.Position{
						Row:    1,
						Column: 8,
					},
				},
			},
			wantStart: token.Position{
				Row:    1,
				Column: 1,
			},
			wantEnd: token.Position{
				Row:    1,
				Column: 8,
			},
		},
		"AttrStmt": {
			in: AttrStmt{
				ID: ID{
					Literal: `f1`,
					StartPos: token.Position{
						Row:    1,
						Column: 1,
					},
					EndPos: token.Position{
						Row:    1,
						Column: 2,
					},
				},
				AttrList: AttrList{
					StartPos: token.Position{
						Row:    1,
						Column: 3,
					},
					EndPos: token.Position{
						Row:    1,
						Column: 5,
					},
				},
			},
			wantStart: token.Position{
				Row:    1,
				Column: 1,
			},
			wantEnd: token.Position{
				Row:    1,
				Column: 5,
			},
		},
		"AttrListEmpty": {
			in: &AttrList{
				StartPos: token.Position{
					Row:    1,
					Column: 1,
				},
				EndPos: token.Position{
					Row:    1,
					Column: 4,
				},
			},
			wantStart: token.Position{
				Row:    1,
				Column: 1,
			},
			wantEnd: token.Position{
				Row:    1,
				Column: 4,
			},
		},
		"AttrListWithAList": {
			in: &AttrList{
				StartPos: token.Position{
					Row:    1,
					Column: 1,
				},
				EndPos: token.Position{
					Row:    1,
					Column: 8,
				},
				AList: &AList{
					Attribute: Attribute{
						Name: ID{
							Literal: "pc",
							StartPos: token.Position{
								Row:    1,
								Column: 1,
							},
							EndPos: token.Position{
								Row:    1,
								Column: 2,
							},
						},
						Value: ID{
							Literal: "2",
							StartPos: token.Position{
								Row:    1,
								Column: 6,
							},
							EndPos: token.Position{
								Row:    1,
								Column: 6,
							},
						},
					},
				},
			},
			wantStart: token.Position{
				Row:    1,
				Column: 1,
			},
			wantEnd: token.Position{
				Row:    1,
				Column: 8,
			},
		},
		"AttrListWithAListAndNextWithAList": {
			in: &AttrList{
				StartPos: token.Position{
					Row:    1,
					Column: 1,
				},
				EndPos: token.Position{
					Row:    1,
					Column: 8,
				},
				AList: &AList{
					Attribute: Attribute{
						Name: ID{
							Literal: "pc",
							StartPos: token.Position{
								Row:    1,
								Column: 1,
							},
							EndPos: token.Position{
								Row:    1,
								Column: 2,
							},
						},
						Value: ID{
							Literal: "2",
							StartPos: token.Position{
								Row:    1,
								Column: 6,
							},
							EndPos: token.Position{
								Row:    1,
								Column: 6,
							},
						},
					},
				},
				Next: &AttrList{
					StartPos: token.Position{
						Row:    1,
						Column: 10,
					},
					EndPos: token.Position{
						Row:    1,
						Column: 15,
					},
					AList: &AList{
						Attribute: Attribute{
							Name: ID{
								Literal: "pc",
								StartPos: token.Position{
									Row:    1,
									Column: 11,
								},
								EndPos: token.Position{
									Row:    1,
									Column: 12,
								},
							},
							Value: ID{
								Literal: "2",
								StartPos: token.Position{
									Row:    1,
									Column: 14,
								},
								EndPos: token.Position{
									Row:    1,
									Column: 14,
								},
							},
						},
					},
				},
			},
			wantStart: token.Position{
				Row:    1,
				Column: 1,
			},
			wantEnd: token.Position{
				Row:    1,
				Column: 15,
			},
		},
		"AttrListWithAListAndNextWithEmptyAList": {
			in: &AttrList{
				StartPos: token.Position{
					Row:    1,
					Column: 1,
				},
				EndPos: token.Position{
					Row:    1,
					Column: 8,
				},
				AList: &AList{
					Attribute: Attribute{
						Name: ID{
							Literal: "pc",
							StartPos: token.Position{
								Row:    1,
								Column: 1,
							},
							EndPos: token.Position{
								Row:    1,
								Column: 2,
							},
						},
						Value: ID{
							Literal: "2",
							StartPos: token.Position{
								Row:    1,
								Column: 6,
							},
							EndPos: token.Position{
								Row:    1,
								Column: 6,
							},
						},
					},
				},
				Next: &AttrList{
					StartPos: token.Position{
						Row:    1,
						Column: 10,
					},
					EndPos: token.Position{
						Row:    1,
						Column: 11,
					},
				},
			},
			wantStart: token.Position{
				Row:    1,
				Column: 1,
			},
			wantEnd: token.Position{
				Row:    1,
				Column: 11,
			},
		},
		"AttrListWithEmptyAListAndNextWithAList": {
			in: &AttrList{
				StartPos: token.Position{
					Row:    1,
					Column: 1,
				},
				EndPos: token.Position{
					Row:    1,
					Column: 2,
				},
				Next: &AttrList{
					StartPos: token.Position{
						Row:    1,
						Column: 4,
					},
					EndPos: token.Position{
						Row:    1,
						Column: 15,
					},
					AList: &AList{
						Attribute: Attribute{
							Name: ID{
								Literal: "pc",
								StartPos: token.Position{
									Row:    1,
									Column: 11,
								},
								EndPos: token.Position{
									Row:    1,
									Column: 12,
								},
							},
							Value: ID{
								Literal: "2",
								StartPos: token.Position{
									Row:    1,
									Column: 14,
								},
								EndPos: token.Position{
									Row:    1,
									Column: 14,
								},
							},
						},
					},
				},
			},
			wantStart: token.Position{
				Row:    1,
				Column: 1,
			},
			wantEnd: token.Position{
				Row:    1,
				Column: 15,
			},
		},
		"AList": {
			in: &AList{
				Attribute: Attribute{
					Name: ID{
						Literal: "pc",
						StartPos: token.Position{
							Row:    1,
							Column: 1,
						},
						EndPos: token.Position{
							Row:    1,
							Column: 2,
						},
					},
					Value: ID{
						Literal: "2",
						StartPos: token.Position{
							Row:    1,
							Column: 6,
						},
						EndPos: token.Position{
							Row:    1,
							Column: 6,
						},
					},
				},
			},
			wantStart: token.Position{
				Row:    1,
				Column: 1,
			},
			wantEnd: token.Position{
				Row:    1,
				Column: 6,
			},
		},
		"AListWithNext": {
			in: &AList{
				Attribute: Attribute{
					Name: ID{
						Literal: "pc",
						StartPos: token.Position{
							Row:    1,
							Column: 1,
						},
						EndPos: token.Position{
							Row:    1,
							Column: 2,
						},
					},
					Value: ID{
						Literal: "2",
						StartPos: token.Position{
							Row:    1,
							Column: 6,
						},
						EndPos: token.Position{
							Row:    1,
							Column: 6,
						},
					},
				},
				Next: &AList{
					Attribute: Attribute{
						Name: ID{
							Literal: "int",
							StartPos: token.Position{
								Row:    1,
								Column: 8,
							},
							EndPos: token.Position{
								Row:    1,
								Column: 10,
							},
						},
						Value: ID{
							Literal: "3",
							StartPos: token.Position{
								Row:    1,
								Column: 13,
							},
							EndPos: token.Position{
								Row:    1,
								Column: 13,
							},
						},
					},
				},
			},
			wantStart: token.Position{
				Row:    1,
				Column: 1,
			},
			wantEnd: token.Position{
				Row:    1,
				Column: 13,
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := test.in.Start()

			assert.EqualValuesf(t, got, test.wantStart, "Start()")

			got = test.in.End()
			assert.EqualValuesf(t, got, test.wantEnd, "End()")
		})
	}
}
