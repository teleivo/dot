package dot

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"unicode"

	"github.com/teleivo/dot/internal/token"
)

type Lexer struct {
	r         *bufio.Reader
	cur       rune
	curLineNr int
	curCharNr int
	next      rune
	eof       bool
	err       error
}

func NewLexer(r io.Reader) (*Lexer, error) {
	l := Lexer{
		r:         bufio.NewReader(r),
		curLineNr: 1,
	}

	// initialize current and next runes
	err := l.readRune()
	if err != nil {
		return nil, err
	}
	err = l.readRune()
	if err != nil {
		return nil, err
	}
	// 2 readRune calls are needed to fill the cur and next runes
	l.curCharNr = 1

	return &l, nil
}

const maxUnquotedStringLen = 16347 // adjusted https://gitlab.com/graphviz/graphviz/-/issues/1261 to be zero based
const unquotedStringErr = `unquoted string identifiers can contain alphabetic ([a-zA-Z\200-\377]) characters, underscores ('_') or digits([0-9]), but not begin with a digit`

// Next advances the lexers position by one token and returns it. True is returned if the lexer was
// able to get another token and false otherwise.
func (l *Lexer) Next() (token.Token, error) {
	var tok token.Token

	err := l.skipInsignificant()
	if err != nil {
		return tok, l.err
	}
	if l.isEOF() {
		tok.Type = token.EOF
		return tok, nil
	}

	switch l.cur {
	case '{':
		tok, err = l.tokenizeRuneAs(token.LeftBrace)
	case '}':
		tok, err = l.tokenizeRuneAs(token.RightBrace)
	case '[':
		tok, err = l.tokenizeRuneAs(token.LeftBracket)
	case ']':
		tok, err = l.tokenizeRuneAs(token.RightBracket)
	case ':':
		tok, err = l.tokenizeRuneAs(token.Colon)
	case ',':
		tok, err = l.tokenizeRuneAs(token.Comma)
	case ';':
		tok, err = l.tokenizeRuneAs(token.Semicolon)
	case '=':
		tok, err = l.tokenizeRuneAs(token.Equal)
	default:
		if isEdgeOperator(l.cur, l.next) {
			tok, err = l.tokenizeEdgeOperator()
		} else if isStartofIdentifier(l.cur) {
			tok, err = l.tokenizeIdentifier()
			// we already advance in tokenizeIdentifier so we dont want to at the end of the loop
			if err != nil {
				l.err = err
				return tok, err
			}
			return tok, err
		} else {
			err = l.lexError(unquotedStringErr)
		}
	}

	if err != nil {
		l.err = err
		return tok, err
	}

	err = l.readRune()
	if err != nil {
		return tok, err
	}
	return tok, err
}

// readRune reads one rune and advances the lexers position markers depending on the read rune.
func (l *Lexer) readRune() error {
	if l.isEOF() {

	}
	if l.isDone() {
		return l.err
	}

	r, _, err := l.r.ReadRune()
	if err != nil {
		if !errors.Is(err, io.EOF) {
			// fmt.Printf("%d:%d: l.cur %q, l.next %q, eof %v, err %v\n", l.curLineNr, l.curCharNr, l.cur, l.next, l.eof, err)
			l.err = fmt.Errorf("failed to read rune due to: %v", err)
			return l.err
		}

		l.eof = true
	}

	if l.cur == '\n' {
		l.curLineNr++
		l.curCharNr = 1
	} else {
		l.curCharNr++
	}
	l.cur = l.next
	l.next = r
	// fmt.Printf("%d:%d: l.cur %q, l.next %q, eof %v, err %v\n", l.curLineNr, l.curCharNr, l.cur, l.next, l.eof, err)
	return nil
}

// skipInsignificant skips whitespace and comments.
func (l *Lexer) skipInsignificant() error {
	l.skipWhitespace()
	if l.err != nil {
		return l.err
	}
	if l.isEOF() {
		return nil
	}

	// C preprocessor output # and C++ style single line comments
	if l.cur != '#' && !(l.cur == '/' && l.next == '/') {
		return nil
	}

	l.skipLine()
	if l.err != nil {
		return l.err
	}

	l.skipWhitespace()
	if l.err != nil {
		return l.err
	}

	return nil
}

func (l *Lexer) skipWhitespace() {
	for isWhitespace(l.cur) {
		err := l.readRune()
		if err != nil {
			return
		}
	}
}

// isWhitespace determines if the rune is considered whitespace. It does not include non-breaking
// whitespace \240 which is considered whitespace by [unicode.isWhitespace].
func isWhitespace(r rune) bool {
	switch r {
	case ' ', '\t', '\n':
		return true
	}
	return false
}

func (l *Lexer) hasNext() bool {
	return !(l.eof && l.cur == 0)
}

func (l *Lexer) isDone() bool {
	return l.isEOF() || l.err != nil
}

func (l *Lexer) isEOF() bool {
	return !l.hasNext()
}

func (l *Lexer) skipLine() {
	for l.cur != '\n' {
		err := l.readRune()
		if err != nil {
			return
		}
	}

	err := l.readRune()
	if err != nil {
		return
	}
}

func (l *Lexer) tokenizeRuneAs(tokenType token.TokenType) (token.Token, error) {
	return token.Token{Type: tokenType, Literal: string(l.cur)}, nil
}

func isEdgeOperator(first, second rune) bool {
	return first == '-' && (second == '>' || second == '-')
}

func (l *Lexer) tokenizeEdgeOperator() (token.Token, error) {
	err := l.readRune()
	if err != nil {
		var tok token.Token
		return tok, err
	}

	if l.cur == '-' {
		return token.Token{Type: token.UndirectedEgde, Literal: token.UndirectedEgde.String()}, err
	}
	return token.Token{Type: token.DirectedEgde, Literal: token.DirectedEgde.String()}, err
}

func isStartofIdentifier(r rune) bool {
	if isStartOfUnquotedString(r) ||
		isStartOfNumeral(r) ||
		isStartOfQuotedString(r) {
		return true
	}

	return false
}

func isStartOfUnquotedString(r rune) bool {
	return r == '_' || isAlphabetic(r)
}

// isAlphabetic determines if the rune is part of the allowed alphabetic characters of an unquoted
// identifier as defined in https://graphviz.org/doc/info/lang.html#ids.
func isAlphabetic(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '\200' && r <= '\377')
}

func isStartOfNumeral(r rune) bool {
	return r == '-' || r == '.' || unicode.IsDigit(r)
}

func isStartOfQuotedString(r rune) bool {
	return r == '"'
}

func isStartOfHTMLString(r rune) bool {
	return r == '<'
}

func (l *Lexer) tokenizeIdentifier() (token.Token, error) {
	if isStartOfUnquotedString(l.cur) {
		return l.tokenizeUnquotedString()
	} else if isStartOfNumeral(l.cur) {
		return l.tokenizeNumeral()
	} else if isStartOfQuotedString(l.cur) {
		return l.tokenizeQuotedString()
	}

	var tok token.Token
	return tok, l.lexError("invalid token")
}

func (l *Lexer) lexError(reason string) LexError {
	return LexError{
		LineNr:      l.curLineNr,
		CharacterNr: l.curCharNr,
		Character:   l.cur,
		Reason:      reason,
	}
}

// tokenizeUnquotedString considers the current rune(s) as an identifier that might be a dot
// keyword.
func (l *Lexer) tokenizeUnquotedString() (token.Token, error) {
	var tok token.Token
	var err error
	var id []rune

	for ; l.hasNext() && err == nil && !isUnquotedStringSeparator(l.cur); err = l.readRune() {
		if !isLegalInUnquotedString(l.cur) {
			return tok, l.lexError(unquotedStringErr)
		}

		id = append(id, l.cur)
	}

	if err != nil {
		return tok, err
	}

	literal := string(id)
	tok = token.Token{Type: token.LookupKeyword(literal), Literal: literal}

	return tok, nil
}

// isUnquotedStringSeparator determines if the rune separates tokens.
func isUnquotedStringSeparator(r rune) bool {
	return isTerminal(r) || isWhitespace(r) || r == '-' // potential edge operator
}

// isTerminal determines if the rune is considered a terminal token in the dot language. This does
// not contain edge operators
func isTerminal(r rune) bool {
	tok, ok := token.Type(string(r))
	if !ok {
		return false
	}

	switch tok {
	case token.LeftBrace, token.RightBrace, token.LeftBracket, token.RightBracket, token.Colon, token.Semicolon, token.Equal, token.Comma:
		return true
	}
	return false
}

func isLegalInUnquotedString(r rune) bool {
	return isStartOfUnquotedString(r) || unicode.IsDigit(r)
}

func (l *Lexer) tokenizeNumeral() (token.Token, error) {
	var tok token.Token
	var err error
	var id []rune
	var hasDigit bool

	for pos, hasDot := 0, false; l.hasNext() && err == nil && !isNumeralSeparator(l.cur); err, pos = l.readRune(), pos+1 {
		if l.cur == '-' && pos != 0 {
			return tok, l.lexError("a numeral can only be prefixed with a `-`")
		}

		if l.cur == '.' && hasDot {
			return tok, l.lexError("a numeral can only have one `.` that is at least preceded or followed by digits")
		}

		if l.cur != '-' && l.cur != '.' && !unicode.IsDigit(l.cur) { // otherwise only digits are allowed
			return tok, l.lexError("a numeral can optionally lead with a `-`, has to have at least one digit before or after a `.` which must only be followed by digits")
		}

		if l.cur == '.' {
			hasDot = true
		} else if unicode.IsDigit(l.cur) {
			hasDigit = true
		}

		id = append(id, l.cur)
	}

	if !hasDigit {
		err = l.lexError("a numeral must have at least one digit")
	}
	if err != nil {
		return tok, err
	}

	return token.Token{Type: token.Identifier, Literal: string(id)}, nil
}

func isNumeralSeparator(r rune) bool {
	return isTerminal(r) || isWhitespace(r)
}

func (l *Lexer) tokenizeQuotedString() (token.Token, error) {
	var tok token.Token
	var err error
	var id []rune
	var hasClosingQuote bool

	for pos, prev := 0, rune(0); l.hasNext() && err == nil; err, pos = l.readRune(), pos+1 {
		id = append(id, l.cur)

		if pos != 0 && l.cur == '"' && prev != '\\' { // assuming a non-escaped quote after pos 0 closes the string
			hasClosingQuote = true
			err = l.readRune() // consume closing quote
			break
		}
		if pos > maxUnquotedStringLen {
			return tok, l.lexError(fmt.Sprintf("potentially missing closing quote, found none after max %d characters", maxUnquotedStringLen+1))
		}
		prev = l.cur
	}

	if !hasClosingQuote {
		err = l.lexError("missing closing quote")
	}
	if err != nil {
		return tok, err
	}

	return token.Token{Type: token.Identifier, Literal: string(id)}, nil
}

type LexError struct {
	LineNr      int    // Line number the error was found.
	CharacterNr int    // Character number the error was found.
	Character   rune   // Character that caused the error.
	Reason      string // Reason for the error.
}

func (le LexError) Error() string {
	return fmt.Sprintf("%d:%d: %s", le.LineNr, le.CharacterNr, le.Reason)
}
