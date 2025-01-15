// Package printer prints dot ASTs formatted in the spirit of https://github.com/mvdan/gofumpt.
package printer

import (
	"fmt"
	"io"

	"github.com/teleivo/dot"
	"github.com/teleivo/dot/ast"
	"github.com/teleivo/dot/token"
)

// maxColumn is the max number of runes after which lines are broken up into multiple lines. Not
// every dot construct can be broken up though.
const maxColumn = 100

// Printer formats dot code.
type Printer struct {
	r            io.Reader       // r reader to parse dot code from
	w            io.Writer       // w writer to output formatted dot code to
	row          int             // row is the current one-indexed row the printer is at i.e. how many newlines it has printed. 0 means nothing has been printed
	column       int             // column is the current one-indexed column in terms of runes the printer is at. 0 means no rune has been printed on the current row
	indentLevel  int             // indentLevel is the current level of indentation to be applied when indenting
	prevToken    token.TokenType // prevToken is the type of the last printed token
	prevPosition token.Position  // prevPosition is the position of the last printed token
	newline      bool            // newline indicates a buffered newline that should be printed
	commentIndex int             // commentIndex points to the next comment to be printed
	comments     []ast.Comment   // comments lists all comments in the Graph to be printed
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
	pr.comments = g.Comments

	err = pr.printNode(g)
	if err != nil {
		return err
	}
	pr.printRemainingComments()

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
		p.printToken(token.Strict, *graph.StrictStart)
		p.printSpace()
	}

	if graph.Directed {
		p.printToken(token.Digraph, graph.GraphStart)
	} else {
		p.printToken(token.Graph, graph.GraphStart)
	}
	p.printSpace()

	if graph.ID != nil {
		err := p.printID(*graph.ID)
		if err != nil {
			return err
		}
		p.printSpace()
	}

	p.printToken(token.LeftBrace, graph.LeftBrace)
	p.increaseIndentation()

	err := p.printStmts(graph.Stmts)
	if err != nil {
		return err
	}

	p.decreaseIndentation()
	p.printNewline()
	p.printToken(token.RightBrace, graph.RightBrace)
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

func (p *Printer) printID(id ast.ID) error {
	p.printComments(id.StartPos)

	p.prevToken = token.Identifier
	p.prevPosition = id.EndPos

	if id.Literal[0] != '"' { // print unquoted identifiers as is
		p.print(id)
		return nil
	}

	// print opening " to start the ID with the correct indentation
	p.printRune('"')

	const offset = 1 // as opening " was printed
	runeCount := 1
	start := offset
	var prevRune rune
	for curRuneIdx, curRune := range id.Literal[offset:] {
		curRuneIdx += offset // adjust for the opening "

		// newlines without preceding '\' are not mentioned as legal in
		// https://graphviz.org/doc/info/lang.html#ids but are supported by the dot tooling. Support
		// such newlines and write them where the user intended them to be.
		if prevRune != '\\' && curRune == '\n' {
			p.printStringWithoutIndent(id.Literal[start:curRuneIdx]) // print everything up to the newline
			p.forceNewline()
			start = curRuneIdx + 1 // start again after the newline
			runeCount = 0
			// TODO this is where I need to add some logic to skip any existing ID continuation
		} else if prevRune == '\\' && curRune == '\n' {
			// does all up to \ fit?
			runeCount -= 2
			if p.column+runeCount+1 > maxColumn { // the word and '\' do not fit on the current line
				p.printLineContinuation()
			}

			// print everything up to the line continuation in the id.Literal
			end := curRuneIdx - 1

			// print word (and whitespace if it fits as well)
			p.printStringWithoutIndent(id.Literal[start:end])

			runeCount = 0
			start = end + 2 // skip the line continuation in id.Literal
		} else if isWhitespace(curRune) {
			if p.column+runeCount+1 > maxColumn { // the word and '\' do not fit on the current line
				p.printLineContinuation()
			}

			// TODO improve this scandalous indexing :joy:
			end := curRuneIdx
			if p.column+runeCount+1 < maxColumn { // the word and whitespace fit on the current line
				end++
				runeCount = -1
			} else {
				runeCount = 0 // for the whitespace that was not printed
			}

			// print word (and whitespace if it fits as well)
			p.printStringWithoutIndent(id.Literal[start:end])
			start = end
		} else if /* closing quote */ curRune == '"' && curRuneIdx+1 == len(id.Literal) {
			if p.column+runeCount+1 > maxColumn { // the word and " do not fit on the current line
				p.printLineContinuation()
			}
			p.printStringWithoutIndent(id.Literal[start:])
		}
		prevRune = curRune
		runeCount++
	}

	return nil
}

func (p *Printer) printStmt(stmt ast.Stmt) error {
	var err error
	switch st := stmt.(type) {
	case *ast.NodeStmt:
		err = p.printNodeStmt(st)
	case *ast.EdgeStmt:
		err = p.printEdgeStmt(st)
	case *ast.AttrStmt:
		err = p.printAttrStmt(st)
	case ast.Attribute:
		p.printNewline()
		err = p.printAttribute(st)
	case ast.Subgraph:
		p.printNewline()
		err = p.printSubgraph(st)
	}
	return err
}

func (p *Printer) printNodeStmt(nodeStmt *ast.NodeStmt) error {
	p.printNewline()
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
		p.printToken(token.Colon, withColumnOffset(nodeID.Port.Name.StartPos, -1))
		err = p.printID(*nodeID.Port.Name)
		if err != nil {
			return err
		}
	}
	if nodeID.Port.CompassPoint != nil && nodeID.Port.CompassPoint.Type != ast.CompassPointUnderscore {
		p.printToken(token.Colon, withColumnOffset(nodeID.Port.CompassPoint.StartPos, -1))
		p.print(nodeID.Port.CompassPoint)
	}

	return nil
}

func (p *Printer) printAttrList(attrList *ast.AttrList) error {
	if attrList == nil {
		return nil
	}

	// TODO that is not 100% true as an attrList can solely be a chain of []
	var hasMultipleAttrs bool
	if attrList.Next != nil {
		hasMultipleAttrs = true
	}

	p.printSpace()
	p.printToken(token.LeftBracket, attrList.LeftBracket)
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
	}
	// TODO if I remember correctly I am merging A [color=blue] [style=filled] into A [color=blue,
	// style=filled]. How does me taking out '[]' affect printing of comments? Add to the test case.
	p.printToken(token.RightBracket, attrList.End())

	return nil
}

func (p *Printer) printAList(aList *ast.AList, hasMultipleAttrs bool) (bool, error) {
	for cur := aList; cur != nil; cur = cur.Next {
		if aList.Next != nil {
			hasMultipleAttrs = true
		}

		if hasMultipleAttrs {
			p.printNewline()
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
	p.printNewline()

	err := p.printEdgeOperand(edgeStmt.Left)
	if err != nil {
		return err
	}

	p.printSpace()
	if edgeStmt.Right.Directed {
		p.printToken(token.DirectedEgde, edgeStmt.Right.StartPos)
	} else {
		p.printToken(token.UndirectedEgde, edgeStmt.Right.StartPos)
	}

	p.printSpace()
	err = p.printEdgeOperand(edgeStmt.Right.Right)
	if err != nil {
		return err
	}

	for cur := edgeStmt.Right.Next; cur != nil; cur = cur.Next {
		p.printSpace()
		if edgeStmt.Right.Directed {
			p.printToken(token.DirectedEgde, cur.StartPos)
		} else {
			p.printToken(token.UndirectedEgde, cur.StartPos)
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
	p.printNewline()
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
	p.printToken(token.Equal, attribute.Name.EndPos)
	return p.printID(attribute.Value)
}

func (p *Printer) printSubgraph(subraph ast.Subgraph) error {
	// TODO reconsider always printing subraph as I now know whether the user wanted it
	p.printToken(token.Subgraph, subraph.Start())
	p.printSpace()
	if subraph.ID != nil {
		err := p.printID(*subraph.ID)
		if err != nil {
			return err
		}
		p.printSpace()
	}

	p.printToken(token.LeftBrace, subraph.LeftBrace)
	p.increaseIndentation()

	err := p.printStmts(subraph.Stmts)
	if err != nil {
		return err
	}

	p.decreaseIndentation()
	p.printNewline()
	p.printToken(token.RightBrace, subraph.RightBrace)
	return nil
}

func (p *Printer) printComment(comment ast.Comment) error {
	text := comment.Text
	// discard markers
	if text[0] == '#' {
		text = text[1:]
	} else if text[1] == '/' {
		text = text[2:]
	} else { // discard multi-line markers
		text = text[2 : len(text)-2]
	}

	// put a comment only on a new line if that was the intent! a comment starting on the same
	// line as the previous token is seen as the intent of keeping them together
	putOnNewLine := p.prevPosition.Row > 0 && p.prevPosition.Row != comment.StartPos.Row
	isFirstWord := true
	var inWord bool
	var start, runeCount int
	for i, r := range text {
		if !inWord && !isWhitespace(r) {
			inWord = true
			start = i
			runeCount = 1
		} else if inWord && !isWhitespace(r) {
			runeCount++
		} else if inWord && isWhitespace(r) { // word boundary
			col := p.column + 1 + runeCount // 1 for the space separating words

			// breakup long comment or start new one with the intent to be on a new line
			if col > maxColumn || (isFirstWord && putOnNewLine) {
				p.forceNewline()
			}
			// separate comment from previous token on the same line except for comments at the start of a
			// file
			if isFirstWord && !putOnNewLine && p.row > 0 {
				p.printSpace()
			}
			// start comment
			if col > maxColumn || isFirstWord {
				p.printRune('/')
				p.printRune('/')
			}
			// separate word from marker and separate words
			p.printSpace()

			for _, c := range text[start:i] {
				p.printRune(c)
			}

			isFirstWord = false
			inWord = false
		}
	}

	if inWord {
		col := p.column + 1 + runeCount // 1 for the space separating words

		// breakup long comment or start new one with the intent to be on a new line
		if col > maxColumn || (isFirstWord && putOnNewLine) {
			p.forceNewline()
		}
		// separate comment from previous token on the same line except for comments at the start of a
		// file
		if isFirstWord && !putOnNewLine && p.row > 0 {
			p.printSpace()
		}
		// start comment
		if col > maxColumn || isFirstWord {
			p.printRune('/')
			p.printRune('/')
		}
		// separate word from marker and separate words
		p.printSpace()

		for _, c := range text[start:] {
			p.printRune(c)
		}
	}

	p.prevToken = token.Comment
	p.prevPosition = comment.EndPos

	return nil
}

func isWhitespace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n'
}

func (p *Printer) increaseIndentation() {
	p.indentLevel++
}

func (p *Printer) decreaseIndentation() {
	p.indentLevel--
}

func (p *Printer) printString(a string) {
	for _, r := range a {
		p.printRune(r)
	}
}

func (p *Printer) printStringWithoutIndent(a string) {
	for _, r := range a {
		p.printRuneWithoutIndent(r)
	}
}

func (p *Printer) print(a fmt.Stringer) {
	for _, r := range a.String() {
		p.printRune(r)
	}
}

func (p *Printer) printTab() {
	p.printRune('\t')
}

func (p *Printer) printSpace() {
	p.printRune(' ')
}

// TODO should this be aware of r being a newline?
func (p *Printer) printRune(r rune) {
	for p.column < p.indentLevel {
		fmt.Fprintf(p.w, "%c", '\t')
		p.column++
	}

	p.printRuneWithoutIndent(r)
}

func (p *Printer) printRuneWithoutIndent(a rune) {
	fmt.Fprintf(p.w, "%c", a)
	if p.row == 0 {
		p.row = 1
	}
	p.column++
}

func (p *Printer) printToken(tokenType token.TokenType, pos token.Position) {
	p.printComments(pos)

	tok := tokenType.String()
	p.printString(tok)

	p.prevToken = tokenType
	p.prevPosition = withColumnOffset(pos, len(tok))
}

// printComments print all comments preceding the next token to be printed.
func (p *Printer) printComments(nextTokenPos token.Position) {
	// TODO replace all print with positional print funcs
	// TODO bring back block comment to support a comment in between tokens
	// TODO handle errors
	var printed bool
	var err error
	for ; err == nil && p.commentIndex < len(p.comments) && p.comments[p.commentIndex].StartPos.Before(nextTokenPos); p.commentIndex++ {
		comment := p.comments[p.commentIndex]
		err = p.printComment(comment)
		printed = true
	}

	// TODO I might not want the newline once I bring block comments back
	if printed || p.newline {
		p.printNewline()
		p.flushNewline()
	} else {
		p.newline = false
	}
}

func (p *Printer) printRemainingComments() {
	// TODO handle errors
	var err error
	for ; err == nil && p.commentIndex < len(p.comments); p.commentIndex++ {
		comment := p.comments[p.commentIndex]
		err = p.printComment(comment)
	}
}

// printNewline queues a newline to be printed. Printing an ID or a token can trigger the newline to
// be written if appropriate. Use forceNewline to immediately write a newline.
func (p *Printer) printNewline() {
	p.newline = true
}

// flushNewline writes a newline if it has previously been queued by [Printer.printNewline].
func (p *Printer) flushNewline() bool {
	if !p.newline {
		return false
	}

	p.forceNewline()
	return true
}

// forceNewline immediately writes a newline to [Printer.w] and clears a newline queued by
// [Printer.printNewline].
func (p *Printer) forceNewline() {
	fmt.Fprintln(p.w)
	p.column = 0
	p.row++
	p.newline = false
}

// printLineContinuation prints the standard C convention of a backslash immediately preceding a
// newline character.
func (p *Printer) printLineContinuation() {
	p.printRuneWithoutIndent('\\')
	p.forceNewline() // immediately print the newline as there cannot be any interspersed comment
}

// withColumnOffset returns a new position with the added offset to the given positions column.
func withColumnOffset(pos token.Position, columnOffset int) token.Position {
	return token.Position{
		Row:    pos.Row,
		Column: pos.Column + columnOffset,
	}
}
