package dot

import (
	"fmt"
	"io"
	"unicode/utf8"

	"github.com/teleivo/dot/internal/ast"
	"github.com/teleivo/dot/internal/token"
)

// maxColumn is the max number of runes after which lines are broken up into multiple lines. Not
// every dot construct can be broken up though.
const maxColumn = 100

// Printer formats dot code.
type Printer struct {
	r      io.Reader // r reader to parse dot code from
	w      io.Writer // w writer to output formatted dot code to
	column int       // column is the current column in terms of runes the printer is at
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
		p.print(token.Strict)
		p.printSpace()
	}
	if graph.Directed {
		p.print(token.Digraph)
	} else {
		p.print(token.Graph)
	}
	p.printSpace()
	if graph.ID != "" {
		err := p.printID(graph.ID)
		if err != nil {
			return err
		}
		p.printSpace()
	}
	p.print(token.LeftBrace)
	for _, stmt := range graph.Stmts {
		p.printNewline()
		p.printIndent()
		err := p.printStatement(stmt)
		if err != nil {
			return err
		}
	}
	if len(graph.Stmts) > 0 { // no statements print as {}
		p.printNewline()
	}
	p.print(token.RightBrace)
	return nil
}

func (p *Printer) printNewline() {
	fmt.Fprintln(p.w)
	p.column = 0
}

func (p *Printer) printSpace() {
	p.print(" ")
	p.column++
}

func (p *Printer) printID(id ast.ID) error {
	runeCount := utf8.RuneCountInString(string(id))
	if p.column+runeCount <= maxColumn {
		p.print(id)
		return nil
	}

	var isUnquoted bool
	runeIndex := p.column
	breakPointCol := maxColumn - 2 // 2 = "\\n"
	if id[0] != '"' {
		isUnquoted = true
		// accounting for the added quote
		runeIndex++
		breakPointCol++
	}

	// find the starting byte of the rune that will end up on the next line
	var breakPointBytes int
	for i := range id {
		runeIndex++
		if runeIndex > breakPointCol {
			breakPointBytes = i
			break
		}
	}

	if isUnquoted { // opening quote
		p.print(`"`)
	}
	p.print(id[:breakPointBytes])
	// standard C convention of a backslash immediately preceding a newline character
	p.print(`\`)
	p.printNewline()
	p.print(id[breakPointBytes:])
	if isUnquoted { // closing quote
		p.print(`"`)
	}

	return nil
}

func (p *Printer) printStatement(stmt ast.Stmt) error {
	var err error
	switch st := stmt.(type) {
	case *ast.NodeStmt:
		err = p.printNodeStmt(st)
	case *ast.AttrStmt:
		err = p.printAttrStmt(st)
	case *ast.EdgeStmt:
		err = p.printEdgeStmt(st)
	}
	return err
}

func (p *Printer) printNodeStmt(nodeStmt *ast.NodeStmt) error {
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
	if nodeID.Port != nil {
		p.print(token.Colon)
		err := p.printID(nodeID.Port.Name)
		if err != nil {
			return err
		}
		// TODO do not print default CompassPoint
		if nodeID.Port.CompassPoint != ast.CompassPointUnderscore {
			p.print(token.Colon)
			p.print(nodeID.Port.CompassPoint)
		}
	}
	return nil
}

func (p *Printer) printAttrList(attrList *ast.AttrList) error {
	if attrList == nil {
		return nil
	}

	var hasMultipleAttrs bool
	if attrList.Next != nil {
		hasMultipleAttrs = true
	}

	p.printSpace()
	p.print(token.LeftBracket)
	for cur := attrList; cur != nil; cur = cur.Next {
		split, err := p.printAList(cur.AList, hasMultipleAttrs)
		if err != nil {
			return err
		}
		if split {
			hasMultipleAttrs = true
		}
	}
	if hasMultipleAttrs {
		p.printNewline()
		p.printIndent()
	}
	p.print(token.RightBracket)
	return nil
}

func (p *Printer) printAList(aList *ast.AList, hasMultipleAttrs bool) (bool, error) {
	if aList.Next != nil {
		hasMultipleAttrs = true
	}

	for cur := aList; cur != nil; cur = cur.Next {
		if hasMultipleAttrs {
			p.printNewline()
			p.printIndent()
			p.printIndent()
		}
		err := p.printID(cur.Attribute.Name)
		if err != nil {
			return hasMultipleAttrs, err
		}
		p.print(token.Equal)
		p.print(cur.Attribute.Value)
		if hasMultipleAttrs {
			p.print(token.Comma)
		} else if cur.Next != nil {
			p.printSpace()
		}
	}

	return hasMultipleAttrs, nil
}

func (p *Printer) printEdgeStmt(edgeStmt *ast.EdgeStmt) error {
	err := p.printEdgeOperand(edgeStmt.Left)
	if err != nil {
		return err
	}

	p.printSpace()
	if edgeStmt.Right.Directed {
		p.print(token.DirectedEgde)
	} else {
		p.print(token.UndirectedEgde)
	}
	p.printSpace()
	err = p.printEdgeOperand(edgeStmt.Right.Right)
	if err != nil {
		return err
	}

	for cur := edgeStmt.Right.Next; cur != nil; cur = cur.Next {
		p.printSpace()
		if edgeStmt.Right.Directed {
			p.print(token.DirectedEgde)
		} else {
			p.print(token.UndirectedEgde)
		}
		p.printSpace()
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

func (p *Printer) printAttrStmt(attrStmt *ast.AttrStmt) error {
	err := p.printID(attrStmt.ID)
	if err != nil {
		return err
	}
	return p.printAttrList(attrStmt.AttrList)
}

func (p *Printer) printIndent() {
	p.print("\t")
}

func (p *Printer) print(a ...any) {
	fmt.Fprint(p.w, a...)
	p.column++
}
