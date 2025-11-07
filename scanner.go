package dot

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"unicode"

	"github.com/teleivo/dot/token"
)

const (
	eof = -1 // end of file
)

// Scanner tokenizes DOT language source code into a stream of tokens.
type Scanner struct {
	r         *bufio.Reader
	cur       rune
	curRow    int
	curColumn int
	peek      rune
	eof       bool
}

// NewScanner creates a new scanner that reads DOT source code from r. Returns an error if the
// scanner cannot be initialized.
func NewScanner(r io.Reader) (*Scanner, error) {
	sc := Scanner{
		r:      bufio.NewReader(r),
		cur:    eof,
		peek:   eof,
		curRow: 1,
	}

	// initialize current and peek runes
	err := sc.next()
	if err != nil {
		return nil, err
	}
	err = sc.next()
	if err != nil {
		return nil, err
	}
	sc.curColumn = 1

	return &sc, nil
}

const (
	unquotedStringStartErr = "unquoted identifiers must start with a letter or underscore, and can only contain letters, digits, and underscores"
	unquotedStringNulErr   = "illegal character NUL: unquoted identifiers can only contain letters, digits, and underscores"
	quotedStringNulErr     = "illegal character NUL: quoted identifiers cannot contain null bytes"
)

// Next advances the scanners position by one token and returns it. When encountering invalid input,
// the scanner continues scanning and returns both a token and an error. Invalid input results in a
// token of type [token.ERROR] that greedily consumes characters until a separator is encountered.
// I/O errors (other than [io.EOF]) stop scanning immediately. A token of type [token.EOF] is returned
// once the underlying reader returns [io.EOF] and the peek token has been consumed.
func (sc *Scanner) Next() (token.Token, error) {
	var tok token.Token
	var err error

	sc.skipWhitespace()
	if sc.cur < 0 {
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
		if isEdgeOperator(sc.cur, sc.peek) {
			tok, err = sc.tokenizeEdgeOperator()
		} else {
			tok, err = sc.tokenizeIdentifier()
		}
	}

	return tok, err
}

// next reads one rune and advances the scanner's position markers depending on the read rune.
// It returns an error only for non-EOF I/O errors (such as disk read failures or network errors).
// Any non-EOF error is considered terminal and will cause scanning to stop. [io.EOF] is not considered
// an error and causes the scanner to enter EOF state (sc.cur = eof). After any error, subsequent
// calls to next are no-ops.
func (sc *Scanner) next() error {
	// advance position based on current rune
	if sc.cur == '\n' {
		sc.curRow++
		sc.curColumn = 1
	} else if sc.cur >= 0 {
		sc.curColumn++
	}

	// already at EOF
	if sc.eof {
		sc.cur = eof
		return nil
	}

	r, _, err := sc.r.ReadRune()
	if err != nil {
		sc.eof = true
		sc.cur = sc.peek
		sc.peek = eof
		if errors.Is(err, io.EOF) {
			return nil
		}
		return fmt.Errorf("failed to read character: %v", err)
	}

	sc.cur = sc.peek
	sc.peek = r
	return nil
}

// pos returns the current position as a token.Position.
func (sc *Scanner) pos() token.Position {
	return token.Position{Row: sc.curRow, Column: sc.curColumn}
}

func (sc *Scanner) skipWhitespace() {
	for sc.cur >= 0 && isWhitespace(sc.cur) {
		err := sc.next()
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

func (sc *Scanner) tokenizeRuneAs(tokenType token.TokenType) (token.Token, error) {
	pos := sc.pos()
	tok := token.Token{Type: tokenType, Literal: string(sc.cur), Start: pos, End: pos}
	err := sc.next()
	return tok, err
}

func (sc *Scanner) tokenizeComment() (token.Token, error) {
	var tok token.Token
	var err error
	var comment []rune
	var hasClosingMarker bool

	if sc.cur == '/' && (sc.peek < 0 || (sc.peek != '/' && sc.peek != '*')) {
		pos := sc.pos()
		tok = token.Token{Type: token.ERROR, Literal: string(sc.cur), Start: pos, End: pos}
		err := sc.error("missing '/' for single-line or a '*' for a multi-line comment")
		if advanceErr := sc.next(); advanceErr != nil {
			return tok, advanceErr
		}
		return tok, err
	}

	start := sc.pos()
	var end token.Position
	isMultiLine := sc.cur == '/' && sc.peek == '*'
	for ; sc.cur >= 0 && err == nil && (isMultiLine || sc.cur != '\n'); err = sc.next() {
		end = sc.pos()
		comment = append(comment, sc.cur)

		if isMultiLine && sc.cur == '*' && sc.peek == '/' {
			hasClosingMarker = true
			comment = append(comment, sc.peek)
			err = sc.next() // consume last rune '/' of closing marker
			if err != nil {
				break
			}
			end = sc.pos()
			err = sc.next() // advance past the closing '/' to next char
			break
		}
	}

	tType := token.Comment
	if isMultiLine && !hasClosingMarker {
		err = Error{
			LineNr:      start.Row,
			CharacterNr: start.Column,
			Character:   '/',
			Reason:      "missing closing marker '*/' for multi-line comment",
		}
		tType = token.ERROR
		if advanceErr := sc.next(); advanceErr != nil {
			return token.Token{
				Type:    token.ERROR,
				Literal: string(comment),
				Start:   start,
				End:     end,
			}, advanceErr
		}
	}

	return token.Token{
		Type:    tType,
		Literal: string(comment),
		Start:   start,
		End:     end,
	}, err
}

func isEdgeOperator(first, second rune) bool {
	return first == '-' && (second == '>' || second == '-')
}

func (sc *Scanner) tokenizeEdgeOperator() (token.Token, error) {
	var tok token.Token
	start := sc.pos()
	err := sc.next()
	if err != nil {
		return tok, err
	}

	end := sc.pos()
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
	err = sc.next()
	return tok, err
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
	if isStartOfNumeral(sc.cur) {
		return sc.tokenizeNumeral()
	} else if isStartOfQuotedString(sc.cur) {
		return sc.tokenizeQuotedString()
	} else {
		return sc.tokenizeUnquotedString()
	}
}

func (sc *Scanner) error(reason string) Error {
	return Error{
		LineNr:      sc.curRow,
		CharacterNr: sc.curColumn,
		Character:   sc.cur,
		Reason:      reason,
	}
}

// tokenizeUnquotedString considers the current rune(s) as an identifier that might be a DOT
// keyword.
func (sc *Scanner) tokenizeUnquotedString() (token.Token, error) {
	var firstErr error
	var err error
	var id []rune
	start := sc.pos()
	var end token.Position

	for ; sc.cur >= 0 && err == nil && !isUnquotedStringSeparator(sc.cur); err = sc.next() {
		if firstErr == nil && !isLegalInUnquotedString(sc.cur) {
			if sc.cur == 0 {
				firstErr = sc.error(unquotedStringNulErr)
			} else {
				firstErr = sc.error(unquotedStringStartErr)
			}
		}

		id = append(id, sc.cur)
		end = sc.pos()
	}

	literal := string(id)

	// Prioritize terminal I/O errors from sc.next()
	if err != nil {
		return token.Token{
			Type:    token.ERROR,
			Literal: literal,
			Start:   start,
			End:     end,
		}, err
	}

	if firstErr != nil {
		return token.Token{
			Type:    token.ERROR,
			Literal: literal,
			Start:   start,
			End:     end,
		}, firstErr
	}

	return token.Token{
		Type:    token.Lookup(literal),
		Literal: literal,
		Start:   start,
		End:     end,
	}, nil
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
	var firstErr error
	var err error
	var id []rune
	var hasDigit bool
	start := sc.pos()
	var end token.Position

	for pos, hasDot := 0, false; sc.cur >= 0 && err == nil && !sc.isNumeralSeparator(); err, pos = sc.next(), pos+1 {
		end = sc.pos()
		if firstErr == nil && sc.cur == '-' && pos != 0 {
			firstErr = sc.error("a numeral can only be prefixed with a `-`")
		} else if firstErr == nil && sc.cur == '.' && hasDot {
			firstErr = sc.error("a numeral can only have one `.` that is at least preceded or followed by digits")
		} else if firstErr == nil && sc.cur != '-' && sc.cur != '.' && !unicode.IsDigit(sc.cur) { // otherwise only digits are allowed
			firstErr = sc.error("a numeral can optionally lead with a `-`, has to have at least one digit before or after a `.` which must only be followed by digits")
		}

		if sc.cur == '.' {
			hasDot = true
		} else if unicode.IsDigit(sc.cur) {
			hasDigit = true
		}

		id = append(id, sc.cur)
	}

	literal := string(id)

	// Prioritize terminal I/O errors from sc.next()
	if err != nil {
		return token.Token{
			Type:    token.ERROR,
			Literal: literal,
			Start:   start,
			End:     end,
		}, err
	}

	if firstErr == nil && !hasDigit {
		firstErr = Error{
			LineNr:      start.Row,
			CharacterNr: start.Column,
			Character:   sc.cur,
			Reason:      "a numeral must have at least one digit",
		}
		if advanceErr := sc.next(); advanceErr != nil {
			return token.Token{
				Type:    token.ERROR,
				Literal: literal,
				Start:   start,
				End:     end,
			}, advanceErr
		}
	}

	if firstErr != nil {
		return token.Token{
			Type:    token.ERROR,
			Literal: literal,
			Start:   start,
			End:     end,
		}, firstErr
	}

	return token.Token{
		Type:    token.Lookup(literal),
		Literal: literal,
		Start:   start,
		End:     end,
	}, nil
}

func (sc *Scanner) isNumeralSeparator() bool {
	return isTerminal(sc.cur) || isWhitespace(sc.cur) || isEdgeOperator(sc.cur, sc.peek)
}

func (sc *Scanner) tokenizeQuotedString() (token.Token, error) {
	var err error
	var nulByteErr error
	var id []rune
	var hasClosingQuote bool
	start := sc.pos()
	var end token.Position

	for pos, prev := 0, rune(0); sc.cur >= 0 && err == nil; err, pos = sc.next(), pos+1 {
		end = sc.pos()
		id = append(id, sc.cur)

		if sc.cur == 0 && nulByteErr == nil {
			nulByteErr = sc.error(quotedStringNulErr)
		}

		if pos != 0 && sc.cur == '"' && prev != '\\' {
			hasClosingQuote = true
			err = sc.next()
			break
		}
		prev = sc.cur
	}

	literal := string(id)

	// Prioritize terminal I/O errors from sc.next()
	if err != nil {
		return token.Token{
			Type:    token.ERROR,
			Literal: literal,
			Start:   start,
			End:     end,
		}, err
	}

	tType := token.Identifier
	if nulByteErr != nil {
		err = nulByteErr
		tType = token.ERROR
	} else if !hasClosingQuote {
		err = Error{
			LineNr:      start.Row,
			CharacterNr: start.Column,
			Character:   '"',
			Reason:      "missing closing quote",
		}
		tType = token.ERROR
		if advanceErr := sc.next(); advanceErr != nil {
			return token.Token{
				Type:    token.ERROR,
				Literal: literal,
				Start:   start,
				End:     end,
			}, advanceErr
		}
	}

	return token.Token{
		Type:    tType,
		Literal: literal,
		Start:   start,
		End:     end,
	}, err
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
	if e.Character < 0 {
		return fmt.Sprintf("%d:%d: %s", e.LineNr, e.CharacterNr, e.Reason)
	}
	return fmt.Sprintf("%d:%d: illegal character %#U: %s", e.LineNr, e.CharacterNr, e.Character, e.Reason)
}
