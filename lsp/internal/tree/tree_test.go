package tree

import (
	"testing"

	"github.com/teleivo/assertive/assert"
)

func TestComponentString(t *testing.T) {
	tests := []struct {
		comp Component
		want string
	}{
		{0, ""},
		{Graph, "Graph"},
		{Subgraph, "Subgraph"},
		{Cluster, "Cluster"},
		{Node, "Node"},
		{Edge, "Edge"},
		{Node | Edge, "Node, Edge"},
		{Graph | Node | Edge, "Graph, Node, Edge"},
		{Graph | Cluster | Node | Edge, "Graph, Cluster, Node, Edge"},
		{Graph | Subgraph | Cluster | Node | Edge, "Graph, Subgraph, Cluster, Node, Edge"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.comp.String()
			assert.EqualValues(t, got, tt.want, "unexpected string")
		})
	}
}
