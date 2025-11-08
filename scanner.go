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

	if sc.cur == '/' && (sc.peek < 0 || (sc.peek != '/' && sc.peek != '*')) {
		pos := sc.pos()
		tok = token.Token{Type: token.ERROR, Literal: string(sc.cur), Start: pos, End: pos}
		err := sc.error("use '//' (line) or '/*...*/' (block) for comments")
		if advanceErr := sc.next(); advanceErr != nil {
			return tok, advanceErr
		}
		return tok, err
	}

	start := sc.pos()
	isMultiLine := sc.cur == '/' && sc.peek == '*'
	var err error
	var end token.Position
	var comment []rune
	var hasClosingMarker bool
	for prev := rune(eof); sc.cur >= 0 && err == nil && (isMultiLine || sc.cur != '\n'); err = sc.next() {
		end = sc.pos()
		comment = append(comment, sc.cur)

		if isMultiLine && prev == '*' && sc.cur == '/' {
			hasClosingMarker = true
			err = sc.next() // advance past the closing '/'
			break
		}
		prev = sc.cur
	}

	tType := token.Comment
	if isMultiLine && !hasClosingMarker {
		err = Error{
			LineNr:      start.Row,
			CharacterNr: start.Column,
			Character:   '/',
			Reason:      "unclosed comment: missing '*/'",
		}
		tType = token.ERROR
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
		return sc.tokenizeQuotedID()
	} else {
		return sc.tokenizeUnquotedID()
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

// tokenizeUnquotedID considers the current rune(s) as an identifier that might be a DOT
// keyword.
func (sc *Scanner) tokenizeUnquotedID() (token.Token, error) {
	var firstErr error
	var err error
	var id []rune
	start := sc.pos()
	var end token.Position

	for ; sc.cur >= 0 && err == nil && !sc.isTokenSeparator(); err = sc.next() {
		if firstErr == nil && !isLegalInUnquotedID(sc.cur) {
			switch sc.cur {
			case 0:
				firstErr = sc.error("unquoted IDs cannot contain null bytes")
			case '-':
				firstErr = sc.error("use '--' (undirected) or '->' (directed) for edges, or quote the ID")
			default:
				if len(id) == 0 {
					firstErr = sc.error("unquoted IDs must start with a letter or underscore")
				} else {
					firstErr = sc.error("unquoted IDs can only contain letters, digits, and underscores")
				}
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

// isTerminal determines if the rune is considered a terminal token in the dot language. This does
// only checks for single rune terminals. Edge operators are thus not considered.
func isTerminal(r rune) bool {
	tok, ok := token.Type(string(r))
	if !ok {
		return false
	}

	return tok.IsTerminal()
}

func isLegalInUnquotedID(r rune) bool {
	return isStartOfUnquotedString(r) || unicode.IsDigit(r)
}

func (sc *Scanner) tokenizeNumeral() (token.Token, error) {
	var firstErr error
	var err error
	var id []rune
	var hasDigit bool
	start := sc.pos()
	var end token.Position

	for pos, prev, hasDot := 0, rune(eof), false; sc.cur >= 0 && err == nil && !sc.isTokenSeparator(); err, pos = sc.next(), pos+1 {
		end = sc.pos()
		if firstErr == nil && sc.cur == '-' && pos != 0 {
			firstErr = sc.error("ambiguous: quote for ID containing '-', use space for separate IDs, or '--'/'->' for edges")
		} else if firstErr == nil && sc.cur == '.' && hasDot {
			firstErr = sc.error("ambiguous: quote for ID containing multiple '.', or use one decimal point for number")
		} else if firstErr == nil && sc.cur != '-' && sc.cur != '.' && !unicode.IsDigit(sc.cur) { // otherwise only digits are allowed
			if prev == '-' {
				firstErr = sc.error("invalid character in number: only digits and decimal point can follow '-'")
			} else {
				firstErr = sc.error("invalid character in number: valid forms are '1', '-1', '1.2', '-.1', '.1'")
			}
		}

		if sc.cur == '.' {
			hasDot = true
		} else if unicode.IsDigit(sc.cur) {
			hasDigit = true
		}

		id = append(id, sc.cur)
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

	if firstErr == nil && !hasDigit {
		firstErr = Error{
			LineNr:      start.Row,
			CharacterNr: start.Column,
			Character:   sc.cur,
			Reason:      "ambiguous: quote for ID, or add digit for number like '-.1' or '-0.'",
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

// isTokenSeparator determines if the rune separates tokens.
func (sc *Scanner) isTokenSeparator() bool {
	return isTerminal(sc.cur) || isWhitespace(sc.cur) || isEdgeOperator(sc.cur, sc.peek) || sc.cur == '/' || sc.cur == '#' || sc.cur == '"'
}

func (sc *Scanner) tokenizeQuotedID() (token.Token, error) {
	var err error
	var nulByteErr error
	var id []rune
	var hasClosingQuote bool
	start := sc.pos()
	var end token.Position

	for pos, prev := 0, rune(eof); sc.cur >= 0 && err == nil; err, pos = sc.next(), pos+1 {
		end = sc.pos()
		id = append(id, sc.cur)

		if sc.cur == 0 && nulByteErr == nil {
			nulByteErr = sc.error("quoted IDs cannot contain null bytes")
		}

		if pos != 0 && sc.cur == '"' && prev != '\\' {
			hasClosingQuote = true
			err = sc.next() // advance past the closing '"'
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

	tType := token.ID
	if nulByteErr != nil {
		err = nulByteErr
		tType = token.ERROR
	} else if !hasClosingQuote {
		err = Error{
			LineNr:      start.Row,
			CharacterNr: start.Column,
			Character:   '"',
			Reason:      "unclosed ID: missing closing '\"'",
		}
		tType = token.ERROR
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

func (e Error) Error() string {
	switch {
	case e.Character < 0:
		return fmt.Sprintf("%d:%d: %s", e.LineNr, e.CharacterNr, e.Reason)
	case e.Character >= 0x20 && e.Character < 0x7F:
		return fmt.Sprintf("%d:%d: invalid character %q: %s", e.LineNr, e.CharacterNr, e.Character, e.Reason)
	case e.Character >= 0x80 && unicode.IsPrint(e.Character):
		return fmt.Sprintf("%d:%d: invalid character U+%04X %q: %s", e.LineNr, e.CharacterNr, e.Character, e.Character, e.Reason)
	default:
		return fmt.Sprintf("%d:%d: invalid character U+%04X: %s", e.LineNr, e.CharacterNr, e.Character, e.Reason)
	}
}
