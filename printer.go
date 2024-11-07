package dot

import (
	"fmt"
	"io"

	"github.com/teleivo/dot/internal/ast"
	"github.com/teleivo/dot/internal/token"
)

func Print(r io.Reader, w io.Writer) error {
	p, err := NewParser(r)
	if err != nil {
		return err
	}

	g, err := p.Parse()
	if err != nil {
		return err
	}

	return printNode(w, g)
}

func printNode(w io.Writer, node ast.Node) error {
	switch n := node.(type) {
	case ast.Graph:
		return printGraph(w, n)
	}
	return nil
}

func printGraph(w io.Writer, graph ast.Graph) error {
	if graph.Strict {
		fmt.Fprintf(w, "%s ", token.Strict)
	}
	if graph.Directed {
		fmt.Fprint(w, token.Digraph)
	} else {
		fmt.Fprint(w, token.Graph)
	}
	fmt.Fprint(w, " ")
	if graph.ID != "" {
		err := printID(w, graph.ID)
		if err != nil {
			return err
		}
		fmt.Fprint(w, " ")
	}
	fmt.Fprint(w, token.LeftBrace)
	for _, stmt := range graph.Stmts {
		fmt.Fprintln(w)
		printIndent(w, 1)
		err := printStatement(w, stmt)
		if err != nil {
			return err
		}
	}
	if len(graph.Stmts) > 0 { // no statements print as {}
		fmt.Fprintln(w)
	}
	fmt.Fprint(w, token.RightBrace)
	return nil
}

// TODO maybe ID should be its own thing in the ast so I can handle large IDs instead of only
// dealing with strings
func printID(w io.Writer, id string) error {
	fmt.Fprint(w, id)
	return nil
}

func printStatement(w io.Writer, stmt ast.Stmt) error {
	var err error
	switch st := stmt.(type) {
	case *ast.EdgeStmt:
		err = printEdgeStmt(w, st)
	}
	return err
}

func printEdgeStmt(w io.Writer, edgeStmt *ast.EdgeStmt) error {
	err := printEdgeOperand(w, edgeStmt.Left)
	if err != nil {
		return err
	}

	fmt.Fprint(w, " ")
	if edgeStmt.Right.Directed {
		fmt.Fprint(w, token.DirectedEgde)
	} else {
		fmt.Fprint(w, token.UndirectedEgde)
	}
	fmt.Fprint(w, " ")
	err = printEdgeOperand(w, edgeStmt.Right.Right)
	if err != nil {
		return err
	}

	for cur := edgeStmt.Right.Next; cur != nil; cur = cur.Next {
		fmt.Fprint(w, " ")
		if edgeStmt.Right.Directed {
			fmt.Fprint(w, token.DirectedEgde)
		} else {
			fmt.Fprint(w, token.UndirectedEgde)
		}
		err = printEdgeOperand(w, cur.Right)
		if err != nil {
			return err
		}
	}

	return err
}

func printEdgeOperand(w io.Writer, edgeOperand ast.EdgeOperand) error {
	var err error
	switch op := edgeOperand.(type) {
	case ast.NodeID:
		err = printNodeID(w, op)
	}
	return err
}

func printNodeID(w io.Writer, nodeID ast.NodeID) error {
	err := printID(w, nodeID.ID)
	if err != nil {
		return err
	}
	return nil
}

func printIndent(w io.Writer, level int) {
	fmt.Fprint(w, "\t")
}
