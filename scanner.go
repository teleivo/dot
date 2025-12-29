package dot

import (
	"fmt"
	"unicode"
	"unicode/utf8"

	"github.com/teleivo/dot/internal/assert"
	"github.com/teleivo/dot/token"
)

const (
	eof = -1 // end of file
)

// Scanner tokenizes DOT language source code into a stream of tokens.
type Scanner struct {
	src       []byte
	offset    int
	cur       rune
	curLine   int
	curColumn int
	peek      rune
	eof       bool
}

// NewScanner creates a new scanner that tokenizes the given DOT source code.
func NewScanner(src []byte) *Scanner {
	sc := Scanner{
		src:     src,
		cur:     eof,
		peek:    eof,
		curLine: 1,
	}

	// initialize current and peek runes
	sc.next()
	sc.next()
	sc.curColumn = 1

	return &sc
}

// Next advances the scanners position by one token and returns it. When encountering invalid input,
// the scanner continues scanning. Invalid input results in a token of type [token.ERROR] with the
// error message in [token.Token.Error] that greedily consumes characters until a separator is encountered.
// A token of type [token.EOF] is returned once the end of input is reached.
func (sc *Scanner) Next() token.Token {
	var tok token.Token

	sc.skipWhitespace()
	if sc.cur < 0 {
		tok.Type = token.EOF
		pos := sc.pos()
		tok.Start = pos
		tok.End = pos
		return tok
	}

	switch sc.cur {
	case '{':
		tok = sc.tokenizeRuneAs(token.LeftBrace)
	case '}':
		tok = sc.tokenizeRuneAs(token.RightBrace)
	case '[':
		tok = sc.tokenizeRuneAs(token.LeftBracket)
	case ']':
		tok = sc.tokenizeRuneAs(token.RightBracket)
	case ':':
		tok = sc.tokenizeRuneAs(token.Colon)
	case ',':
		tok = sc.tokenizeRuneAs(token.Comma)
	case ';':
		tok = sc.tokenizeRuneAs(token.Semicolon)
	case '=':
		tok = sc.tokenizeRuneAs(token.Equal)
	case '#', '/':
		tok = sc.tokenizeComment()
	default:
		if isEdgeOperator(sc.cur, sc.peek) {
			tok = sc.tokenizeEdgeOperator()
		} else {
			tok = sc.tokenizeIdentifier()
		}
	}

	assert.That(tok.Type != token.ERROR || tok.Error != "", "ERROR token must have an error message")

	return tok
}

// next reads one rune and advances the scanner's position markers depending on the read rune.
func (sc *Scanner) next() {
	// advance position based on current rune
	if sc.cur == '\n' {
		sc.curLine++
		sc.curColumn = 1
	} else if sc.cur >= 0 {
		sc.curColumn++
	}

	// already at EOF
	if sc.eof {
		sc.cur = eof
		return
	}

	sc.cur = sc.peek

	if sc.offset < len(sc.src) {
		r, size := utf8.DecodeRune(sc.src[sc.offset:])
		sc.offset += size
		sc.peek = r
		// RuneError with size 1 means invalid UTF-8 byte sequence
		// continue scanning and let tokenization produce an ERROR token
	} else {
		sc.eof = true
		sc.peek = eof
	}
}

// pos returns the current position as a token.Position.
func (sc *Scanner) pos() token.Position {
	return token.Position{Line: sc.curLine, Column: sc.curColumn}
}

func (sc *Scanner) skipWhitespace() {
	for sc.cur >= 0 && isWhitespace(sc.cur) {
		sc.next()
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

func (sc *Scanner) tokenizeRuneAs(tokenType token.Kind) token.Token {
	pos := sc.pos()
	tok := token.Token{Type: tokenType, Literal: string(sc.cur), Start: pos, End: pos}
	sc.next()
	return tok
}

func (sc *Scanner) tokenizeComment() token.Token {
	if sc.cur == '/' && (sc.peek < 0 || (sc.peek != '/' && sc.peek != '*')) {
		pos := sc.pos()
		tok := token.Token{
			Type:    token.ERROR,
			Literal: string(sc.cur),
			Error:   sc.error("use '//' (line) or '/*...*/' (block) for comments"),
			Start:   pos,
			End:     pos,
		}
		sc.next()
		return tok
	}

	start := sc.pos()
	isMultiLine := sc.cur == '/' && sc.peek == '*'
	var end token.Position
	var comment []rune
	var hasClosingMarker bool
	for prev := rune(eof); sc.cur >= 0 && (isMultiLine || sc.cur != '\n'); sc.next() {
		end = sc.pos()
		comment = append(comment, sc.cur)

		if isMultiLine && prev == '*' && sc.cur == '/' {
			hasClosingMarker = true
			sc.next() // advance past the closing '/'
			break
		}
		prev = sc.cur
	}

	tok := token.Token{
		Type:    token.Comment,
		Literal: string(comment),
		Start:   start,
		End:     end,
	}
	if isMultiLine && !hasClosingMarker {
		tok.Type = token.ERROR
		tok.Error = "invalid character '/': unclosed comment: missing '*/'"
	}

	return tok
}

func isEdgeOperator(first, second rune) bool {
	return first == '-' && (second == '>' || second == '-')
}

func (sc *Scanner) tokenizeEdgeOperator() token.Token {
	var tok token.Token
	start := sc.pos()
	sc.next()

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
	sc.next()
	return tok
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

func (sc *Scanner) tokenizeIdentifier() token.Token {
	if isStartOfNumeral(sc.cur) {
		return sc.tokenizeNumeral()
	} else if isStartOfQuotedString(sc.cur) {
		return sc.tokenizeQuotedID()
	} else {
		return sc.tokenizeUnquotedID()
	}
}

func (sc *Scanner) error(reason string) string {
	switch {
	case sc.cur < 0:
		return reason
	case sc.cur >= 0x20 && sc.cur < 0x7F:
		return fmt.Sprintf("invalid character %q: %s", sc.cur, reason)
	case sc.cur >= 0x80:
		return fmt.Sprintf("invalid character U+%04X '%c': %s", sc.cur, sc.cur, reason)
	default:
		return fmt.Sprintf("invalid character U+%04X: %s", sc.cur, reason)
	}
}

// tokenizeUnquotedID considers the current rune(s) as an identifier that might be a DOT
// keyword.
func (sc *Scanner) tokenizeUnquotedID() token.Token {
	var firstErrMsg string
	var id []rune
	start := sc.pos()
	var end token.Position

	for ; sc.cur >= 0 && !sc.isTokenSeparator(); sc.next() {
		if firstErrMsg == "" && !isLegalInUnquotedID(sc.cur) {
			switch sc.cur {
			case 0:
				firstErrMsg = sc.error("unquoted IDs cannot contain null bytes")
			case '-':
				firstErrMsg = sc.error("use '--' (undirected) or '->' (directed) for edges, or quote the ID")
			default:
				if len(id) == 0 {
					firstErrMsg = sc.error("unquoted IDs must start with a letter or underscore")
				} else {
					firstErrMsg = sc.error("unquoted IDs can only contain letters, digits, and underscores")
				}
			}
		}

		id = append(id, sc.cur)
		end = sc.pos()
	}

	literal := string(id)

	if firstErrMsg != "" {
		return token.Token{
			Type:    token.ERROR,
			Literal: literal,
			Error:   firstErrMsg,
			Start:   start,
			End:     end,
		}
	}

	return token.Token{
		Type:    token.Lookup(literal),
		Literal: literal,
		Start:   start,
		End:     end,
	}
}

// isTerminal determines if the rune is considered a terminal token in the dot language. This does
// only checks for single rune terminals. Edge operators are thus not considered.
func isTerminal(r rune) bool {
	switch r {
	case '{', '}', '[', ']', ':', ';', '=', ',':
		return true
	default:
		return false
	}
}

func isLegalInUnquotedID(r rune) bool {
	return r != utf8.RuneError && (isStartOfUnquotedString(r) || unicode.IsDigit(r))
}

func (sc *Scanner) tokenizeNumeral() token.Token {
	var firstErrMsg string
	var id []rune
	var hasDigit bool
	start := sc.pos()
	var end token.Position

	for pos, prev, hasDot := 0, rune(eof), false; sc.cur >= 0 && !sc.isTokenSeparator(); sc.next() {
		end = sc.pos()
		if firstErrMsg == "" && sc.cur == '-' && pos != 0 {
			firstErrMsg = sc.error("ambiguous: quote for ID containing '-', use space for separate IDs, or '--'/'->' for edges")
		} else if firstErrMsg == "" && sc.cur == '.' && hasDot {
			firstErrMsg = sc.error("ambiguous: quote for ID containing multiple '.', or use one decimal point for number")
		} else if firstErrMsg == "" && sc.cur != '-' && sc.cur != '.' && !unicode.IsDigit(sc.cur) { // otherwise only digits are allowed
			if prev == '-' {
				firstErrMsg = sc.error("invalid character in number: only digits and decimal point can follow '-'")
			} else {
				firstErrMsg = sc.error("invalid character in number: valid forms are '1', '-1', '1.2', '-.1', '.1'")
			}
		}

		if sc.cur == '.' {
			hasDot = true
		} else if unicode.IsDigit(sc.cur) {
			hasDigit = true
		}

		id = append(id, sc.cur)
		prev = sc.cur
		pos++
	}

	literal := string(id)

	if firstErrMsg == "" && !hasDigit {
		firstErrMsg = sc.error("ambiguous: quote for ID, or add digit for number like '-.1' or '-0.'")
	}

	if firstErrMsg != "" {
		return token.Token{
			Type:    token.ERROR,
			Literal: literal,
			Error:   firstErrMsg,
			Start:   start,
			End:     end,
		}
	}

	return token.Token{
		Type:    token.ID,
		Literal: literal,
		Start:   start,
		End:     end,
	}
}

// isTokenSeparator determines if the rune separates tokens.
func (sc *Scanner) isTokenSeparator() bool {
	return isTerminal(sc.cur) || isWhitespace(sc.cur) || isEdgeOperator(sc.cur, sc.peek) || sc.cur == '/' || sc.cur == '#' || sc.cur == '"'
}

func (sc *Scanner) tokenizeQuotedID() token.Token {
	var nulByteErrMsg string
	var id []rune
	var pendingEscape bool
	var hasClosingQuote bool
	start := sc.pos()
	var end token.Position

	for pos := 0; sc.cur >= 0; sc.next() {
		end = sc.pos()
		id = append(id, sc.cur)

		if sc.cur == 0 && nulByteErrMsg == "" {
			nulByteErrMsg = sc.error("quoted IDs cannot contain null bytes")
		}

		if sc.cur == '\\' && !pendingEscape {
			pendingEscape = true
		} else if pos != 0 && sc.cur == '"' && !pendingEscape {
			hasClosingQuote = true
			sc.next() // advance past the closing '"'
			break
		} else {
			pendingEscape = false
		}
		pos++
	}

	literal := string(id)

	tok := token.Token{
		Type:    token.ID,
		Literal: literal,
		Start:   start,
		End:     end,
	}
	if nulByteErrMsg != "" {
		tok.Type = token.ERROR
		tok.Error = nulByteErrMsg
	} else if !hasClosingQuote {
		tok.Type = token.ERROR
		tok.Error = `invalid character '"': unclosed ID: missing closing '"'`
	}

	return tok
}
