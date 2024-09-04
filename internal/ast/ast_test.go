package ast

import (
	"testing"

	"github.com/teleivo/assertive/assert"
)

func TestStringer(t *testing.T) {
	// TODO implement String() on graph
	// TODO implement String() on subgraph

	tests := []struct {
		in   Node
		want string
	}{
		{
			in: &NodeStmt{
				ID: NodeID{ID: "foo"},
				AttrList: &AttrList{
					AList: &AList{
						Attribute: Attribute{Name: "a", Value: "b"},
						Next: &AList{
							Attribute: Attribute{Name: "c", Value: "d"},
						},
					},
					Next: &AttrList{
						AList: &AList{
							Attribute: Attribute{Name: "e", Value: "f"},
						},
					},
				},
			},
			want: `foo [a=b,c=d] [e=f]`,
		},
		{
			in: &NodeStmt{
				ID: NodeID{ID: "foo", Port: &Port{Name: `"f0"`}},
			},
			want: `foo:"f0":_`,
		},
		{
			in: &NodeStmt{
				ID: NodeID{ID: "foo", Port: &Port{Name: `"f0"`, CompassPoint: NorthWest}},
			},
			want: `foo:"f0":nw`,
		},
		{
			in: &EdgeStmt{
				Left: NodeID{ID: "1"},
				Right: EdgeRHS{
					Directed: true,
					Right: Subgraph{
						ID: "internal",
						Stmts: []Stmt{
							&NodeStmt{ID: NodeID{ID: "2"}},
						},
					},
					Next: &EdgeRHS{
						Directed: true,
						Right:    NodeID{ID: "3"},
						Next: &EdgeRHS{
							Directed: true,
							Right: Subgraph{
								Stmts: []Stmt{
									&NodeStmt{ID: NodeID{ID: "4"}},
									&NodeStmt{ID: NodeID{ID: "5"}},
								},
							},
						},
					},
				},
				AttrList: &AttrList{
					AList: &AList{
						Attribute: Attribute{Name: "a", Value: "b"},
					},
				},
			},
			want: `1 -> subgraph internal {2} -> 3 -> subgraph {4 5} [a=b]`,
		},
	}

	for _, tc := range tests {
		got := tc.in.String()

		assert.EqualValuesf(t, got, tc.want, "String()")
	}
}
