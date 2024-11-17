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
	row         int       // row is the current zero-indexed row the printer is at i.e. how many newlines it has printed
	column      int       // column is the current zero-indexed column in terms of runes the printer is at
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
		p.printToken(token.Strict)
		p.printSpace()
	}

	if graph.Directed {
		p.printToken(token.Digraph)
	} else {
		p.printToken(token.Graph)
	}

	p.printSpace()
	if graph.ID != "" {
		err := p.printID(graph.ID)
		if err != nil {
			return err
		}
		p.printSpace()
	}

	p.printToken(token.LeftBrace)
	err := p.printStmts(graph.Stmts)
	if err != nil {
		return err
	}
	p.printToken(token.RightBrace)
	return nil
}

func (p *Printer) printStmts(stmts []ast.Stmt) error {
	var hasPrinted bool
	rowStart := p.row
	colStart := p.column

	for _, stmt := range stmts {
		err := p.printStmt(stmt)
		if err != nil {
			return err
		}
		if !hasPrinted && (rowStart != p.row || colStart != p.column) {
			hasPrinted = true
		}
	}

	// no statements are printed as {}
	if hasPrinted {
		p.printNewline()
	}
	return nil
}

func (p *Printer) printNewline() {
	fmt.Fprintln(p.w)
	p.column = 0
	p.row++
}

func (p *Printer) printSpace() {
	p.printRune(' ')
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
		p.printRune('"')
	}
	p.print(id[:breakPointBytes])
	// standard C convention of a backslash immediately preceding a newline character
	p.printRune('\\')
	p.printNewline()
	p.print(id[breakPointBytes:])
	if isUnquoted { // closing quote
		p.printRune('"')
	}

	return nil
}

func (p *Printer) printStmt(stmt ast.Stmt) error {
	var err error
	switch st := stmt.(type) {
	case *ast.NodeStmt:
		err = p.printNodeStmt(st)
	case *ast.EdgeStmt:
		p.printNewline()
		p.printIndent()
		err = p.printEdgeStmt(st)
	case *ast.AttrStmt:
		err = p.printAttrStmt(st)
	case ast.Attribute:
		p.printNewline()
		p.printIndent()
		err = p.printAttribute(st)
	case ast.Subgraph:
		p.printNewline()
		p.printIndent()
		err = p.printSubgraph(st)
	case ast.Comment:
		err = p.printComment(st)
	}
	return err
}

func (p *Printer) printNodeStmt(nodeStmt *ast.NodeStmt) error {
	p.printNewline()
	p.printIndent()
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
		p.printToken(token.Colon)
		err := p.printID(nodeID.Port.Name)
		if err != nil {
			return err
		}
	}
	if nodeID.Port.CompassPoint != ast.CompassPointUnderscore {
		p.printToken(token.Colon)
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
	p.printToken(token.LeftBracket)
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
	p.printToken(token.RightBracket)
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
		p.printToken(token.DirectedEgde)
	} else {
		p.printToken(token.UndirectedEgde)
	}

	p.printSpace()
	err = p.printEdgeOperand(edgeStmt.Right.Right)
	if err != nil {
		return err
	}

	for cur := edgeStmt.Right.Next; cur != nil; cur = cur.Next {
		p.printSpace()
		if edgeStmt.Right.Directed {
			p.printToken(token.DirectedEgde)
		} else {
			p.printToken(token.UndirectedEgde)
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
	if !hasAttr(attrStmt.AttrList) {
		return nil
	}

	p.printNewline()
	p.printIndent()
	err := p.printID(attrStmt.ID)
	if err != nil {
		return err
	}
	return p.printAttrList(attrStmt.AttrList)
}

func hasAttr(attrList *ast.AttrList) bool {
	if attrList == nil {
		return false
	}

	for cur := attrList.AList; cur != nil; cur = cur.Next {
		return true
	}
	return false
}

func (p *Printer) printAttribute(attribute ast.Attribute) error {
	err := p.printID(attribute.Name)
	if err != nil {
		return err
	}
	p.printToken(token.Equal)
	return p.printID(attribute.Value)
}

func (p *Printer) printSubgraph(subraph ast.Subgraph) error {
	p.printToken(token.Subgraph)
	p.printSpace()
	if subraph.ID != "" {
		err := p.printID(subraph.ID)
		if err != nil {
			return err
		}
		p.printSpace()
	}

	p.printToken(token.LeftBrace)
	p.increaseIndentation()
	err := p.printStmts(subraph.Stmts)
	if err != nil {
		return err
	}
	p.decreaseIndentation()
	p.printIndent()
	p.printToken(token.RightBrace)
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

	// TODO use a true buffer on the printer instead of the rune slice; I can also use that for the
	// multi-line IDs
	var words []rune
	var runeCountInWords int
	var printedMultiLineMarker bool
	var hasNonWhitespace bool
	var waitingOnNonWhitespace bool
	for _, r := range text {
		if !hasNonWhitespace && (r == ' ' || r == '\t' || r == '\n') {
			// discard whitespace before any non-whitespace runes
			continue
		} else if r == ' ' || r == '\t' || r == '\n' {
			// discard any whitespace between words so words are separated by exactly one
			waitingOnNonWhitespace = true
			continue
		}

		if waitingOnNonWhitespace {
			words = append(words, ' ')
			waitingOnNonWhitespace = false
		}
		words = append(words, r)
		hasNonWhitespace = true
		runeCountInWords++

		// TODO does this already need to capture an entire word even if already > maxColumn?
		// TODO test edge case properly, how many runes do we need to know that this is a multi-line comment?
		// is the addition of the indentLevel correct?

		// TODO might also have to deal with single line comments that are broken up as they are
		// next to a statement
		// deals with multi-line comments
		if p.column+p.indentLevel+runeCountInWords >= maxColumn {
			// fmt.Fprintf(os.Stderr, "\np.column=%d, p.indentLevel=%d, runeCountInWords=%d, %s\n", p.column, p.indentLevel, runeCountInWords, string(words))
			// return nil
			if !printedMultiLineMarker {
				p.printNewline()
				p.printIndent()
				p.printRune('/')
				p.printRune('*')
				p.increaseIndentation()
				p.printNewline()
				p.printIndent()
				printedMultiLineMarker = true
			}

			// print words from buffer
			var inWord bool
			var start int
			var runeCount int
			isFirstWord := true
			for i, b := range words {
				if !inWord {
					inWord = true
					start = i
					runeCount = 1
					continue
				}

				if b == ' ' {
					// print word without space if it fits on line
					if p.column+runeCount <= maxColumn {
						if !isFirstWord {
							p.printSpace()
						}
					} else {
						p.printNewline()
						p.printIndent()
					}
					for _, c := range words[start:i] {
						p.printRune(c)
					}
					inWord = false
					isFirstWord = false

					continue
				}

				runeCount++
			}

			// TODO test with a word equal to or greater than the maxColumn, it should not be broken
			// up
			// https://github.com/teleivo/dot/blob/fake/27b6dbfe4b99f67df74bfb7323e19d6c547f68fd/parser_test.go#L13
			if inWord {
				// print word without space if it fits on line
				if p.column+runeCount <= maxColumn {
					if !isFirstWord {
						p.printSpace()
					}
				} else {
					p.printNewline()
					p.printIndent()
				}
				for _, c := range words[start:] {
					p.printRune(c)
				}
			}

			words = nil
			runeCountInWords = 0
			hasNonWhitespace = false
			waitingOnNonWhitespace = false
		}
	}

	if len(words) > 0 && !printedMultiLineMarker { // means comment fits onto single line
		p.printNewline()
		p.printIndent()
		p.printRune('/')
		p.printRune('/')
		p.printSpace()
		p.printRunes(words)
		return nil
	}

	if printedMultiLineMarker {
		p.printRunes(words)
		p.decreaseIndentation()
		p.printNewline()
		p.printIndent()
		p.printRune('*')
		p.printRune('/')
	}
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
		p.printRune('\t')
	}
}

func (p *Printer) print(a fmt.Stringer) {
	for _, r := range a.String() {
		p.printRune(r)
	}
}

func (p *Printer) printRunes(a []rune) {
	for _, r := range a {
		p.printRune(r)
	}
}

func (p *Printer) printRune(a rune) {
	fmt.Fprintf(p.w, "%c", a)
	p.column++
}

func (p *Printer) printToken(a token.TokenType) {
	token := a.String()
	fmt.Fprint(p.w, token)
	// tokens are single byte runes i.e. byte count = rune count
	p.column += len(token)
}
