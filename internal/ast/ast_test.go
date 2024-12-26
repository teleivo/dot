package ast

import (
	"testing"

	"github.com/teleivo/assertive/assert"
	// "github.com/teleivo/dot/internal/token"
)

func TestStringer(t *testing.T) {
	tests := []struct {
		in   Node
		want string
	}{
		{
			in: &NodeStmt{
				NodeID: NodeID{ID: ID{Literal: "foo"}},
				AttrList: &AttrList{
					AList: &AList{
						Attribute: Attribute{Name: ID{Literal: "a"}, Value: ID{Literal: "b"}},
						Next: &AList{
							Attribute: Attribute{Name: ID{Literal: "c"}, Value: ID{Literal: "d"}},
						},
					},
					Next: &AttrList{
						AList: &AList{
							Attribute: Attribute{Name: ID{Literal: "e"}, Value: ID{Literal: "f"}},
						},
					},
				},
			},
			want: `foo [a=b,c=d] [e=f]`,
		},
		{
			in: &NodeStmt{
				NodeID: NodeID{ID: ID{Literal: "foo"}, Port: &Port{Name: &ID{Literal: `"f0"`}}},
			},
			want: `foo:"f0":_`,
		},
		{
			in: &NodeStmt{
				NodeID: NodeID{
					ID: ID{Literal: "foo"},
					Port: &Port{
						Name:         &ID{Literal: `"f0"`},
						CompassPoint: CompassPoint{Type: CompassPointNorthWest},
					},
				},
			},
			want: `foo:"f0":nw`,
		},
		{
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
		{
			in: Graph{
				Strict:   true,
				Directed: true,
				ID:       &ID{Literal: `"wonder"`},
			},
			want: `strict digraph "wonder" {}`,
		},
		{
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

	for _, tc := range tests {
		got := tc.in.String()

		assert.EqualValuesf(t, got, tc.want, "String()")
	}
}
