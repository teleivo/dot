// Package printer prints dot ASTs formatted in the spirit of https://github.com/mvdan/gofumpt.
package printer

import (
	"io"

	"github.com/teleivo/dot"
	"github.com/teleivo/dot/ast"
)

const (
	// maxColumn is the max number of runes after which lines are broken up into multiple lines. Not
	// every dot construct can be broken up though.
	maxColumn = 100
	// tabWidth represents the number of columns a tab takes up
	tabWidth = 4
)

// Printer formats dot code.
type Printer struct {
	r       io.Reader // r reader to parse dot code from
	w       io.Writer // w writer to output formatted dot code to
	row     int       // row is the current one-indexed row the printer is at i.e. how many newlines it has printed. 0 means nothing has been printed
	column  int       // column is the current one-indexed column in terms of runes the printer is at. A tab counts as [tabWidth] columns. 0 means no rune has been printed on the current row
	newline bool      // newline indicates a buffered newline that should be printed
}

func NewPrinter(r io.Reader, w io.Writer) *Printer {
	return &Printer{
		r: r,
		w: w,
	}
}

func (pr *Printer) Print() error {
	ps, err := dot.NewParser(pr.r)
	if err != nil {
		return err
	}

	g, err := ps.Parse()
	if err != nil {
		return err
	}

	err = pr.printNode(g)
	if err != nil {
		return err
	}

	return nil
}

func (p *Printer) printNode(node ast.Node) error {
	switch n := node.(type) {
	case ast.Graph:
		return p.printGraph(n)
	}
	return nil
}

func (p *Printer) printGraph(graph ast.Graph) error {
	if graph.IsStrict() {
		// p.printToken(token.Strict, *graph.StrictStart)
		// p.printSpace()
	}

	if graph.Directed {
		// p.printToken(token.Digraph, graph.GraphStart)
	} else {
		// p.printToken(token.Graph, graph.GraphStart)
	}
	// p.printSpace()

	if graph.ID != nil {
		err := p.printID(*graph.ID)
		if err != nil {
			return err
		}
		// p.printSpace()
	}

	// p.printToken(token.LeftBrace, graph.LeftBrace)
	// p.increaseIndentation()

	err := p.printStmts(graph.Stmts)
	if err != nil {
		return err
	}

	// p.decreaseIndentation()
	// p.printNewline()
	// p.printToken(token.RightBrace, graph.RightBrace)
	return nil
}

func (p *Printer) printStmts(stmts []ast.Stmt) error {
	for _, stmt := range stmts {
		err := p.printStmt(stmt)
		if err != nil {
			return err
		}
	}
	return nil
}

// printID prints a DOT [identifier]. newlines without preceding '\' are not mentioned as legal but
// are supported by the DOT tooling. Such newlines are normalized to line continuations.
//
// [identifier:] https://graphviz.org/doc/info/lang.html#ids
func (p *Printer) printID(id ast.ID) error {
	return nil
}

func (p *Printer) printStmt(stmt ast.Stmt) error {
	return nil
}

func (p *Printer) printNodeStmt(nodeStmt *ast.NodeStmt) error {
	// p.printNewline()
	err := p.printNodeID(nodeStmt.NodeID)
	if err != nil {
		return err
	}
	return p.printAttrList(nodeStmt.AttrList)
}

func (p *Printer) printNodeID(nodeID ast.NodeID) error {
	err := p.printID(nodeID.ID)
	if err != nil {
		return err
	}

	if nodeID.Port == nil {
		return nil
	}

	if nodeID.Port.Name != nil {
		// p.printToken(token.Colon, withColumnOffset(nodeID.Port.Name.StartPos, -1))
		err = p.printID(*nodeID.Port.Name)
		if err != nil {
			return err
		}
	}
	if nodeID.Port.CompassPoint != nil && nodeID.Port.CompassPoint.Type != ast.CompassPointUnderscore {
		// p.printToken(token.Colon, withColumnOffset(nodeID.Port.CompassPoint.StartPos, -1))
		// p.print(nodeID.Port.CompassPoint)
	}

	return nil
}

func (p *Printer) printAttrList(attrList *ast.AttrList) error {
	if attrList == nil {
		return nil
	}

	_, split := hasMultipleAttributes(attrList)

	// p.printSpace()
	// p.printToken(token.LeftBracket, attrList.LeftBracket)
	// p.increaseIndentation()

	for cur := attrList; cur != nil; cur = cur.Next {
		err := p.printAList(cur.AList, split)
		if err != nil {
			return err
		}
	}

	// p.decreaseIndentation()
	if split {
		// p.printNewline()
	}
	// TODO if I remember correctly I am merging A [color=blue] [style=filled] into A [color=blue,
	// style=filled]. How does me taking out '[]' affect printing of comments? Add to the test case.
	// p.printToken(token.RightBracket, attrList.End())

	return nil
}

func (p *Printer) printAList(aList *ast.AList, split bool) error {
	for cur := aList; cur != nil; cur = cur.Next {
		if split {
			// p.printNewline()
		}
		err := p.printAttribute(cur.Attribute)
		if err != nil {
			return err
		}
		if !split && cur.Next != nil {
			// p.printSpace()
		}
	}

	return nil
}

// hasMultipleAttributes traverses the AttrLists and ALists counting up to two ALists. This can be
// used to omit empty brackets or split attributes onto multiple lines.
func hasMultipleAttributes(attrList *ast.AttrList) (int, bool) {
	if attrList == nil {
		return 0, false
	}

	var cnt int
	for cur := attrList; cur != nil; cur = cur.Next {
		for curAList := cur.AList; curAList != nil; curAList = curAList.Next {
			cnt++
			if cnt > 1 {
				return cnt, true
			}
		}
	}

	return cnt, false
}

func (p *Printer) printEdgeStmt(edgeStmt *ast.EdgeStmt) error {
	// // p.printNewline()

	err := p.printEdgeOperand(edgeStmt.Left)
	if err != nil {
		return err
	}

	// // p.printSpace()
	if edgeStmt.Right.Directed {
		// // p.printToken(token.DirectedEgde, edgeStmt.Right.StartPos)
	} else {
		// // p.printToken(token.UndirectedEgde, edgeStmt.Right.StartPos)
	}

	// // p.printSpace()
	err = p.printEdgeOperand(edgeStmt.Right.Right)
	if err != nil {
		return err
	}

	for cur := edgeStmt.Right.Next; cur != nil; cur = cur.Next {
		// // p.printSpace()
		if edgeStmt.Right.Directed {
			// // p.printToken(token.DirectedEgde, cur.StartPos)
		} else {
			// // p.printToken(token.UndirectedEgde, cur.StartPos)
		}
		// // p.printSpace()
		err = p.printEdgeOperand(cur.Right)
		if err != nil {
			return err
		}
	}

	return p.printAttrList(edgeStmt.AttrList)
}

func (p *Printer) printEdgeOperand(edgeOperand ast.EdgeOperand) error {
	var err error
	switch op := edgeOperand.(type) {
	case ast.NodeID:
		err = p.printNodeID(op)
	case ast.Subgraph:
		err = p.printSubgraph(op)
	}
	return err
}

func (p *Printer) printAttrStmt(attrStmt *ast.AttrStmt) error {
	cnt, _ := hasMultipleAttributes(&attrStmt.AttrList)
	if cnt == 0 {
		return nil
	}

	// // p.printNewline()
	err := p.printID(attrStmt.ID)
	if err != nil {
		return err
	}
	return p.printAttrList(&attrStmt.AttrList)
}

func (p *Printer) printAttribute(attribute ast.Attribute) error {
	err := p.printID(attribute.Name)
	if err != nil {
		return err
	}
	// TODO fix this using the correct position of the '=' which I need to know the position of equal
	// to support a comment before it. Add the position info to the ast
	// // p.printToken(token.Equal, attribute.Name.EndPos)
	return p.printID(attribute.Value)
}

func (p *Printer) printSubgraph(subraph ast.Subgraph) error {
	// // p.printToken(token.Subgraph, subraph.Start())
	// // p.printSpace()
	if subraph.ID != nil {
		err := p.printID(*subraph.ID)
		if err != nil {
			return err
		}
		// // p.printSpace()
	}

	// // p.printToken(token.LeftBrace, subraph.LeftBrace)

	err := p.printStmts(subraph.Stmts)
	if err != nil {
		return err
	}

	return nil
}
