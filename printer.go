package dot

import (
	"fmt"
	"io"
	"unicode/utf8"

	"github.com/teleivo/dot/internal/ast"
	"github.com/teleivo/dot/internal/token"
)

// maxWidth is the max number of runes after which lines are broken up into multiple lines. Not
// every dot construct can be broken up though.
const maxWidth = 100

// Printer formats dot code.
type Printer struct {
	r   io.Reader // r reader to parse dot code from
	w   io.Writer // w writer to output formatted dot code to
	col int       // col is the current column in terms of runes the printer is at
}

func NewPrinter(r io.Reader, w io.Writer) *Printer {
	return &Printer{
		r: r,
		w: w,
	}
}

func (p *Printer) Print() error {
	ps, err := NewParser(p.r)
	if err != nil {
		return err
	}

	g, err := ps.Parse()
	if err != nil {
		return err
	}

	return p.printNode(g)
}

func (p *Printer) printNode(node ast.Node) error {
	switch n := node.(type) {
	case ast.Graph:
		return p.printGraph(n)
	}
	return nil
}

func (p *Printer) printGraph(graph ast.Graph) error {
	if graph.Strict {
		fmt.Fprintf(p.w, "%s ", token.Strict)
	}
	if graph.Directed {
		fmt.Fprint(p.w, token.Digraph)
	} else {
		fmt.Fprint(p.w, token.Graph)
	}
	fmt.Fprint(p.w, " ")
	if graph.ID != "" {
		err := p.printID(graph.ID)
		if err != nil {
			return err
		}
		fmt.Fprint(p.w, " ")
	}
	fmt.Fprint(p.w, token.LeftBrace)
	for _, stmt := range graph.Stmts {
		fmt.Fprintln(p.w)
		p.printIndent(1)
		err := p.printStatement(stmt)
		if err != nil {
			return err
		}
	}
	if len(graph.Stmts) > 0 { // no statements print as {}
		fmt.Fprintln(p.w)
	}
	fmt.Fprint(p.w, token.RightBrace)
	return nil
}

func (p *Printer) printID(id ast.ID) error {
	if utf8.RuneCountInString(string(id)) <= maxWidth {
		fmt.Fprint(p.w, id)
		return nil
	}

	var runeCount int
	for i, r := range id {
		if runeCount < maxWidth-2 {
			fmt.Fprintf(p.w, "%s", string(r))
		} else {
			fmt.Fprint(p.w, "\\n")
			fmt.Fprintf(p.w, "%s", id[i:])
			return nil
		}
		runeCount++
	}

	return nil
}

func (p *Printer) printStatement(stmt ast.Stmt) error {
	var err error
	switch st := stmt.(type) {
	case *ast.EdgeStmt:
		err = p.printEdgeStmt(st)
	case *ast.NodeStmt:
		err = p.printNodeStmt(st)
	}
	return err
}

func (p *Printer) printEdgeStmt(edgeStmt *ast.EdgeStmt) error {
	err := p.printEdgeOperand(edgeStmt.Left)
	if err != nil {
		return err
	}

	fmt.Fprint(p.w, " ")
	if edgeStmt.Right.Directed {
		fmt.Fprint(p.w, token.DirectedEgde)
	} else {
		fmt.Fprint(p.w, token.UndirectedEgde)
	}
	fmt.Fprint(p.w, " ")
	err = p.printEdgeOperand(edgeStmt.Right.Right)
	if err != nil {
		return err
	}

	for cur := edgeStmt.Right.Next; cur != nil; cur = cur.Next {
		fmt.Fprint(p.w, " ")
		if edgeStmt.Right.Directed {
			fmt.Fprint(p.w, token.DirectedEgde)
		} else {
			fmt.Fprint(p.w, token.UndirectedEgde)
		}
		fmt.Fprint(p.w, " ")
		err = p.printEdgeOperand(cur.Right)
		if err != nil {
			return err
		}
	}

	return err
}

func (p *Printer) printEdgeOperand(edgeOperand ast.EdgeOperand) error {
	var err error
	switch op := edgeOperand.(type) {
	case ast.NodeID:
		err = p.printNodeID(op)
	}
	return err
}

func (p *Printer) printNodeID(nodeID ast.NodeID) error {
	err := p.printID(nodeID.ID)
	if err != nil {
		return err
	}
	return nil
}

func (p *Printer) printIndent(level int) {
	fmt.Fprint(p.w, "\t")
	p.col++
}

func (p *Printer) printNodeStmt(nodeStmt *ast.NodeStmt) error {
	return p.printNodeID(nodeStmt.NodeID)
}
