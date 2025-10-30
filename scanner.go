package dot

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"unicode"

	"github.com/teleivo/dot/token"
)

// Scanner tokenizes DOT language source code into a stream of tokens.
type Scanner struct {
	r         *bufio.Reader
	cur       rune
	curRow    int
	curColumn int
	next      rune
	eof       bool
}

// NewScanner creates a new scanner that reads DOT source code from r. Returns an error if the
// scanner cannot be initialized.
func NewScanner(r io.Reader) (*Scanner, error) {
	scanner := Scanner{
		r:      bufio.NewReader(r),
		curRow: 1,
	}

	// initialize current and next runes
	err := scanner.readRune()
	if err != nil {
		return nil, err
	}
	err = scanner.readRune()
	if err != nil {
		return nil, err
	}
	// 2 readRune calls are needed to fill the cur and next runes
	scanner.curColumn = 1

	return &scanner, nil
}

const (
	maxUnquotedStringLen   = 16347 // adjusted https://gitlab.com/graphviz/graphviz/-/issues/1261 to be zero based
	unquotedStringStartErr = "unquoted identifiers must start with a letter or underscore, and can only contain letters, digits, and underscores"
	unquotedStringErr      = "unquoted identifiers can only contain letters, digits, and underscores"
	unquotedStringNulErr   = "illegal character NUL: unquoted identifiers can only contain letters, digits, and underscores"
)

// Next advances the scanners position by one token and returns it. The scanner will stop trying to
// tokenize more tokens on the first error it encounters. A token of typen [token.EOF] is returned
// once the underlying reader returns [io.EOF] and the peek token has been consumed.
func (sc *Scanner) Next() (token.Token, error) {
	var tok token.Token
	var err error

	sc.skipWhitespace()
	if !sc.hasNext() {
		tok.Type = token.EOF
		return tok, nil
	}

	switch sc.cur {
	case '{':
		tok, err = sc.tokenizeRuneAs(token.LeftBrace)
	case '}':
		tok, err = sc.tokenizeRuneAs(token.RightBrace)
	case '[':
		tok, err = sc.tokenizeRuneAs(token.LeftBracket)
	case ']':
		tok, err = sc.tokenizeRuneAs(token.RightBracket)
	case ':':
		tok, err = sc.tokenizeRuneAs(token.Colon)
	case ',':
		tok, err = sc.tokenizeRuneAs(token.Comma)
	case ';':
		tok, err = sc.tokenizeRuneAs(token.Semicolon)
	case '=':
		tok, err = sc.tokenizeRuneAs(token.Equal)
	case '#', '/':
		tok, err = sc.tokenizeComment()
	default:
		if isEdgeOperator(sc.cur, sc.next) {
			tok, err = sc.tokenizeEdgeOperator()
		} else if isStartofIdentifier(sc.cur) {
			tok, err = sc.tokenizeIdentifier()
		} else {
			err = sc.error(unquotedStringStartErr)
			pos := token.Position{Row: sc.curRow, Column: sc.curColumn}
			tok = token.Token{Type: token.ILLEGAL, Literal: string(sc.cur), Start: pos, End: pos}
			if advanceErr := sc.readRune(); advanceErr != nil {
				return tok, advanceErr
			}
		}
	}

	return tok, err
}

// readRune reads one rune and advances the scanner's position markers depending on the read rune.
// It returns an error only for non-EOF I/O errors (such as disk read failures or network errors).
// Any non-EOF error is considered terminal and will cause scanning to stop. [io.EOF] is not considered
// an error and causes the scanner to enter EOF state (sc.eof = true). After any error, subsequent
// calls to readRune are no-ops.
func (sc *Scanner) readRune() error {
	if sc.isDone() {
		// Already at EOF, advance position and clear cur to signal no more chars
		// Only skip position advancement if cur was already cleared (is 0)
		if sc.cur == '\n' {
			sc.curRow++
			sc.curColumn = 1
		} else if sc.cur != 0 {
			sc.curColumn++
		}
		sc.cur = 0
		return nil
	}

	r, _, err := sc.r.ReadRune()
	if err != nil {
		sc.eof = true // set EOF state for any error to prevent retry loops
		// Still advance position even at EOF (including for null bytes)
		if sc.cur == '\n' {
			sc.curRow++
			sc.curColumn = 1
		} else {
			sc.curColumn++
		}
		sc.cur = sc.next
		sc.next = 0
		if errors.Is(err, io.EOF) {
			return nil // EOF is not an error
		}
		return fmt.Errorf("failed to read rune: %v", err)
	}

	if sc.cur == '\n' {
		sc.curRow++
		sc.curColumn = 1
	} else {
		sc.curColumn++
	}
	sc.cur = sc.next
	sc.next = r
	return nil
}

func (sc *Scanner) skipWhitespace() {
	for sc.hasNext() && isWhitespace(sc.cur) {
		err := sc.readRune()
		if err != nil {
			return
		}
	}
}

// isWhitespace determines if the rune is considered whitespace. It does not include non-breaking
// whitespace \240 which is considered whitespace by [unicode.isWhitespace].
func isWhitespace(r rune) bool {
	switch r {
	case ' ', '\t', '\r', '\n':
		return true
	}
	return false
}

func (sc *Scanner) hasNext() bool {
	return !sc.eof || sc.cur != 0
}

func (sc *Scanner) isDone() bool {
	return sc.eof
}

func (sc *Scanner) tokenizeRuneAs(tokenType token.TokenType) (token.Token, error) {
	pos := token.Position{Row: sc.curRow, Column: sc.curColumn}
	tok := token.Token{Type: tokenType, Literal: string(sc.cur), Start: pos, End: pos}
	err := sc.readRune()
	return tok, err
}

func (sc *Scanner) tokenizeComment() (token.Token, error) {
	var tok token.Token
	var err error
	var comment []rune
	var hasClosingMarker bool

	if sc.cur == '/' && sc.hasNext() && sc.next != '/' && sc.next != '*' {
		pos := token.Position{Row: sc.curRow, Column: sc.curColumn}
		tok = token.Token{Type: token.ILLEGAL, Literal: string(sc.cur), Start: pos, End: pos}
		err := sc.error("missing '/' for single-line or a '*' for a multi-line comment")
		if advanceErr := sc.readRune(); advanceErr != nil {
			return tok, advanceErr
		}
		return tok, err
	}

	start := token.Position{Row: sc.curRow, Column: sc.curColumn}
	var end token.Position
	isMultiLine := sc.cur == '/' && sc.hasNext() && sc.next == '*'
	for ; sc.hasNext() && err == nil && (isMultiLine || sc.cur != '\n'); err = sc.readRune() {
		end = token.Position{Row: sc.curRow, Column: sc.curColumn}
		comment = append(comment, sc.cur)

		if isMultiLine && sc.cur == '*' && sc.hasNext() && sc.next == '/' {
			hasClosingMarker = true
			comment = append(comment, sc.next)
			err = sc.readRune() // consume last rune '/' of closing marker
			if err != nil {
				break
			}
			end = token.Position{Row: sc.curRow, Column: sc.curColumn}
			err = sc.readRune() // advance past the closing '/' to next char
			break
		}
	}

	if isMultiLine && !hasClosingMarker {
		pos := token.Position{Row: sc.curRow, Column: sc.curColumn}
		tok = token.Token{Type: token.ILLEGAL, Literal: string(sc.cur), Start: pos, End: pos}
		err = sc.error("missing closing marker '*/' for multi-line comment")
		if advanceErr := sc.readRune(); advanceErr != nil {
			return tok, advanceErr
		}
	}
	if err != nil {
		return tok, err
	}

	return token.Token{
		Type:    token.Comment,
		Literal: string(comment),
		Start:   start,
		End:     end,
	}, nil
}

func isEdgeOperator(first, second rune) bool {
	return first == '-' && (second == '>' || second == '-')
}

func (sc *Scanner) tokenizeEdgeOperator() (token.Token, error) {
	var tok token.Token
	start := token.Position{Row: sc.curRow, Column: sc.curColumn}
	err := sc.readRune()
	if err != nil {
		return tok, err
	}

	end := token.Position{Row: sc.curRow, Column: sc.curColumn}
	if sc.cur == '-' {
		tok = token.Token{
			Type:    token.UndirectedEdge,
			Literal: token.UndirectedEdge.String(),
			Start:   start,
			End:     end,
		}
	} else {
		tok = token.Token{
			Type:    token.DirectedEdge,
			Literal: token.DirectedEdge.String(),
			Start:   start,
			End:     end,
		}
	}
	err = sc.readRune()
	return tok, err
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

// isAlphabetic determines if the rune is part of the allowed alphabetic characters of an
// [unquoted identifier].
//
// The Graphviz spec mentions \200-\377 which refers to UTF-8 bytes with the high bit set.
// In practice, this means any UTF-8 encoded character (rune >= 0x80) is accepted.
//
// [unquoted identifier]: https://graphviz.org/doc/info/lang.html#ids
func isAlphabetic(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '\200')
}

func isStartOfNumeral(r rune) bool {
	return r == '-' || r == '.' || unicode.IsDigit(r)
}

func isStartOfQuotedString(r rune) bool {
	return r == '"'
}

func (sc *Scanner) tokenizeIdentifier() (token.Token, error) {
	if isStartOfUnquotedString(sc.cur) {
		return sc.tokenizeUnquotedString()
	} else if isStartOfNumeral(sc.cur) {
		return sc.tokenizeNumeral()
	} else if isStartOfQuotedString(sc.cur) {
		return sc.tokenizeQuotedString()
	}

	// TODO is this dead code?
	var tok token.Token
	return tok, sc.error("invalid token")
}

func (sc *Scanner) error(reason string) Error {
	return Error{
		LineNr:      sc.curRow,
		CharacterNr: sc.curColumn,
		Character:   sc.cur,
		Reason:      reason,
	}
}

// tokenizeUnquotedString considers the current rune(s) as an identifier that might be a dot
// keyword.
func (sc *Scanner) tokenizeUnquotedString() (token.Token, error) {
	var tok token.Token
	var err error
	var id []rune
	start := token.Position{Row: sc.curRow, Column: sc.curColumn}
	var end token.Position

	for ; sc.hasNext() && err == nil && !isUnquotedStringSeparator(sc.cur); err = sc.readRune() {
		end = token.Position{Row: sc.curRow, Column: sc.curColumn}
		if !isLegalInUnquotedString(sc.cur) {
			pos := token.Position{Row: sc.curRow, Column: sc.curColumn}
			tok = token.Token{Type: token.ILLEGAL, Literal: string(sc.cur), Start: pos, End: pos}
			var err error
			if sc.cur == 0 {
				err = sc.error(unquotedStringNulErr)
			} else {
				err = sc.error(unquotedStringErr)
			}
			if advanceErr := sc.readRune(); advanceErr != nil {
				return tok, advanceErr
			}
			return tok, err
		}

		id = append(id, sc.cur)
	}

	if err != nil {
		return tok, err
	}

	literal := string(id)
	tok = token.Token{
		Type:    token.Lookup(literal),
		Literal: literal,
		Start:   start,
		End:     end,
	}

	return tok, nil
}

// isUnquotedStringSeparator determines if the rune separates tokens.
func isUnquotedStringSeparator(r rune) bool {
	// - potential edge operator
	// / potential single- or multi-line comment
	// # potential line comment
	// " potential quoted identifier
	return isTerminal(r) || isWhitespace(r) || r == '-' || r == '/' || r == '#' || r == '"'
}

// isTerminal determines if the rune is considered a terminal token in the dot language. This does
// only checks for single rune terminals. Edge operators are thus not considered.
func isTerminal(r rune) bool {
	tok, ok := token.Type(string(r))
	if !ok {
		return false
	}

	return tok.IsTerminal()
}

func isLegalInUnquotedString(r rune) bool {
	return isStartOfUnquotedString(r) || unicode.IsDigit(r)
}

func (sc *Scanner) tokenizeNumeral() (token.Token, error) {
	var tok token.Token
	var err error
	var id []rune
	var hasDigit bool
	start := token.Position{Row: sc.curRow, Column: sc.curColumn}
	var end token.Position

	for pos, hasDot := 0, false; sc.hasNext() && err == nil && !sc.isNumeralSeparator(); err, pos = sc.readRune(), pos+1 {
		end = token.Position{Row: sc.curRow, Column: sc.curColumn}
		if sc.cur == '-' && pos != 0 {
			pos := token.Position{Row: sc.curRow, Column: sc.curColumn}
			tok = token.Token{Type: token.ILLEGAL, Literal: string(sc.cur), Start: pos, End: pos}
			err := sc.error("a numeral can only be prefixed with a `-`")
			if advanceErr := sc.readRune(); advanceErr != nil {
				return tok, advanceErr
			}
			return tok, err
		}

		if sc.cur == '.' && hasDot {
			pos := token.Position{Row: sc.curRow, Column: sc.curColumn}
			tok = token.Token{Type: token.ILLEGAL, Literal: string(sc.cur), Start: pos, End: pos}
			err := sc.error("a numeral can only have one `.` that is at least preceded or followed by digits")
			if advanceErr := sc.readRune(); advanceErr != nil {
				return tok, advanceErr
			}
			return tok, err
		}

		if sc.cur != '-' && sc.cur != '.' && !unicode.IsDigit(sc.cur) { // otherwise only digits are allowed
			pos := token.Position{Row: sc.curRow, Column: sc.curColumn}
			tok = token.Token{Type: token.ILLEGAL, Literal: string(sc.cur), Start: pos, End: pos}
			err := sc.error("a numeral can optionally lead with a `-`, has to have at least one digit before or after a `.` which must only be followed by digits")
			if advanceErr := sc.readRune(); advanceErr != nil {
				return tok, advanceErr
			}
			return tok, err
		}

		if sc.cur == '.' {
			hasDot = true
		} else if unicode.IsDigit(sc.cur) {
			hasDigit = true
		}

		id = append(id, sc.cur)
	}

	if !hasDigit {
		pos := token.Position{Row: sc.curRow, Column: sc.curColumn}
		tok = token.Token{Type: token.ILLEGAL, Literal: string(sc.cur), Start: pos, End: pos}
		err = sc.error("a numeral must have at least one digit")
		if advanceErr := sc.readRune(); advanceErr != nil {
			return tok, advanceErr
		}
	}
	if err != nil {
		return tok, err
	}

	return token.Token{
		Type:    token.Identifier,
		Literal: string(id),
		Start:   start,
		End:     end,
	}, nil
}

func (sc *Scanner) isNumeralSeparator() bool {
	return isTerminal(sc.cur) || isWhitespace(sc.cur) || isEdgeOperator(sc.cur, sc.next)
}

func (sc *Scanner) tokenizeQuotedString() (token.Token, error) {
	var tok token.Token
	var err error
	var id []rune
	var hasClosingQuote bool
	start := token.Position{Row: sc.curRow, Column: sc.curColumn}
	var end token.Position

	for pos, prev := 0, rune(0); sc.hasNext() && err == nil; err, pos = sc.readRune(), pos+1 {
		end = token.Position{Row: sc.curRow, Column: sc.curColumn}
		id = append(id, sc.cur)

		if pos != 0 && sc.cur == '"' && prev != '\\' { // assuming a non-escaped quote after pos 0 closes the string
			hasClosingQuote = true
			err = sc.readRune() // consume closing quote
			break
		}
		if pos > maxUnquotedStringLen {
			pos := token.Position{Row: sc.curRow, Column: sc.curColumn}
			tok = token.Token{Type: token.ILLEGAL, Literal: string(sc.cur), Start: pos, End: pos}
			err := sc.error(fmt.Sprintf("potentially missing closing quote, found none after max %d characters", maxUnquotedStringLen+1))
			if advanceErr := sc.readRune(); advanceErr != nil {
				return tok, advanceErr
			}
			return tok, err
		}
		prev = sc.cur
	}

	if !hasClosingQuote {
		pos := token.Position{Row: sc.curRow, Column: sc.curColumn}
		tok = token.Token{Type: token.ILLEGAL, Literal: string(sc.cur), Start: pos, End: pos}
		err = sc.error("missing closing quote")
		if advanceErr := sc.readRune(); advanceErr != nil {
			return tok, advanceErr
		}
	}
	if err != nil {
		return tok, err
	}

	return token.Token{
		Type:    token.Identifier,
		Literal: string(id),
		Start:   start,
		End:     end,
	}, nil
}

// Error represents a scanning or parsing error in DOT source code.
type Error struct {
	LineNr      int    // Line number the error was found.
	CharacterNr int    // Character number the error was found.
	Character   rune   // Character that caused the error.
	Reason      string // Reason for the error.
}

// Error returns a formatted error message with line and character position.
func (e Error) Error() string {
	if e.Character == 0 {
		return fmt.Sprintf("%d:%d: %s", e.LineNr, e.CharacterNr, e.Reason)
	}
	return fmt.Sprintf("%d:%d: illegal character %#U: %s", e.LineNr, e.CharacterNr, e.Character, e.Reason)
}
