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
	r           io.Reader // r reader to parse dot code from
	w           io.Writer // w writer to output formatted dot code to
	column      int       // column is the current column in terms of runes the printer is at
	indentLevel int       // indentLevel is the current level of indentation to be applied when indenting
}

func NewPrinter(r io.Reader, w io.Writer) *Printer {
	return &Printer{
		r:           r,
		w:           w,
		indentLevel: 1,
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
	err := p.printStmts(graph.Stmts)
	if err != nil {
		return err
	}
	p.print(token.RightBrace)
	return nil
}

func (p *Printer) printStmts(stmts []ast.Stmt) error {
	for _, stmt := range stmts {
		p.printNewline()
		p.printIndent()
		err := p.printStmt(stmt)
		if err != nil {
			return err
		}
	}
	// no statements print as {}
	if len(stmts) > 0 {
		p.printNewline()
	}
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

func (p *Printer) printStmt(stmt ast.Stmt) error {
	var err error
	switch st := stmt.(type) {
	case *ast.NodeStmt:
		err = p.printNodeStmt(st)
	case *ast.AttrStmt:
		err = p.printAttrStmt(st)
	case ast.Attribute:
		err = p.printAttribute(st)
	case *ast.EdgeStmt:
		err = p.printEdgeStmt(st)
	case ast.Subgraph:
		err = p.printSubgraph(st)
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

	if nodeID.Port == nil {
		return nil
	}

	if nodeID.Port.Name != "" {
		p.print(token.Colon)
		err := p.printID(nodeID.Port.Name)
		if err != nil {
			return err
		}
	}
	if nodeID.Port.CompassPoint != ast.CompassPointUnderscore {
		p.print(token.Colon)
		p.print(nodeID.Port.CompassPoint)
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
	p.increaseIndentation()
	for cur := attrList; cur != nil; cur = cur.Next {
		split, err := p.printAList(cur.AList, hasMultipleAttrs)
		if err != nil {
			return err
		}
		if split {
			hasMultipleAttrs = true
		}
	}
	p.decreaseIndentation()
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
		}
		err := p.printAttribute(cur.Attribute)
		if err != nil {
			return hasMultipleAttrs, err
		}
		if !hasMultipleAttrs && cur.Next != nil {
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
	if attrStmt.AttrList == nil {
		return nil
	}

	err := p.printID(attrStmt.ID)
	if err != nil {
		return err
	}
	return p.printAttrList(attrStmt.AttrList)
}

func (p *Printer) printAttribute(attribute ast.Attribute) error {
	err := p.printID(attribute.Name)
	if err != nil {
		return err
	}
	p.print(token.Equal)
	return p.printID(attribute.Value)
}

func (p *Printer) printSubgraph(subraph ast.Subgraph) error {
	p.print(token.Subgraph)
	p.printSpace()
	if subraph.ID != "" {
		err := p.printID(subraph.ID)
		if err != nil {
			return err
		}
		p.printSpace()
	}

	p.print(token.LeftBrace)
	p.increaseIndentation()
	err := p.printStmts(subraph.Stmts)
	if err != nil {
		return err
	}
	p.decreaseIndentation()
	p.printIndent()
	p.print(token.RightBrace)
	return nil
}

func (p *Printer) increaseIndentation() {
	p.indentLevel++
}

func (p *Printer) decreaseIndentation() {
	p.indentLevel--
}

func (p *Printer) printIndent() {
	for range p.indentLevel {
		p.print("\t")
	}
}

func (p *Printer) print(a ...any) {
	fmt.Fprint(p.w, a...)
	p.column++
}
