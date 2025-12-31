package lsp

import (
	"testing"

	"github.com/teleivo/assertive/assert"
)

func TestAttributeContextString(t *testing.T) {
	tests := []struct {
		ctx  attributeContext
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
			got := tt.ctx.String()
			assert.EqualValuesf(t, got, tt.want, "unexpected string")
		})
	}
}
