package dot

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"iter"
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
}

func New(r io.Reader) *Lexer {
	lexer := Lexer{
		r:         bufio.NewReader(r),
		curLineNr: 1,
		// TODO is there a better way I can do this? That this is not within readChar feels like a
		// hack/but I do not want every place that calls readRune() to also have to
		// incrementPosition
		curCharNr: -1,
	}
	return &lexer
}

// TODO support comments (by discarding them)
// All returns an iterator over all dot tokens in the given reader.
func (l *Lexer) All() iter.Seq2[token.Token, error] {
	return func(yield func(token.Token, error) bool) {
		// initialize current and next runes
		err := l.readRune()
		if errors.Is(err, io.EOF) {
			return
		}
		err = l.readRune()
		if errors.Is(err, io.EOF) {
			return
		}

		for {
			var tok token.Token

			err := l.skipWhitespace()
			if err != nil {
				return
			} else if !l.hasNext() {
				return
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
					if !yield(tok, err) || l.eof {
						return
					}
					// TODO could I not advance past in tokenizeIdentifier to get rid of the
					// continue here?
					continue // as we do advance in tokenizeIdentifier we want to skip advancing at the end of the loop
				} else {
					err = l.lexError("invalid token")
				}
			}

			if !yield(tok, err) || l.eof {
				return
			}

			fmt.Println("before advance")
			fmt.Printf("l.cur %q, l.next %q, err %v\n", l.cur, l.next, err)
			err = l.readRune()
			fmt.Println("after advance")
			fmt.Printf("l.cur %q, l.next %q, err %v\n", l.cur, l.next, err)
		}
		// TODO handle illegal runes
		// TODO handle error that is not io.EOF
	}
}

func (l *Lexer) readRune() error {
	r, _, err := l.r.ReadRune()
	if err != nil {
		if !errors.Is(err, io.EOF) {
			fmt.Printf("%d:%d: l.cur %q, l.next %q, err %v\n", l.curLineNr, l.curCharNr, l.cur, l.next, err)
			return err
		}

		l.eof = true
		if l.cur == '\n' {
			l.curLineNr++
			l.curCharNr = 1
		} else {
			l.curCharNr++
		}
		l.cur = l.next
		l.next = 0
		fmt.Printf("%d:%d: l.cur %q, l.next %q, err %v\n", l.curLineNr, l.curCharNr, l.cur, l.next, err)
		return nil
	}

	if l.cur == '\n' {
		l.curLineNr++
		l.curCharNr = 1
	} else {
		l.curCharNr++
	}
	l.cur = l.next
	l.next = r
	fmt.Printf("%d:%d: l.cur %q, l.next %q, err %v\n", l.curLineNr, l.curCharNr, l.cur, l.next, err)
	return nil
}

func (l *Lexer) hasNext() bool {
	return !(l.eof && l.cur == 0)
}

func (l *Lexer) skipWhitespace() (err error) {
	for isWhitespace(l.cur) {
		err := l.readRune()
		if err != nil {
			return err
		}
	}

	return nil
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
		return token.Token{Type: token.UndirectedEgde, Literal: token.UndirectedEgde}, err
	}
	return token.Token{Type: token.DirectedEgde, Literal: token.DirectedEgde}, err
}

func isStartofIdentifier(r rune) bool {
	if isStartOfQuotedString(r) ||
		isStartOfHTMLString(r) ||
		isStartOfNumeral(r) ||
		isStartOfAnyString(r) {
		return true
	}

	return false
}

func isStartOfQuotedString(r rune) bool {
	return r == '"'
}

func isStartOfHTMLString(r rune) bool {
	return r == '<'
}

func isStartOfNumeral(r rune) bool {
	return r == '-' || r == '.' || unicode.IsDigit(r)
}

func isStartOfAnyString(r rune) bool {
	return r == '_' || isAlphabetic(r)
}

func (l *Lexer) tokenizeIdentifier() (token.Token, error) {
	if isStartOfQuotedString(l.cur) {
		return l.tokenizeQuotedString()
	} else if isStartOfHTMLString(l.cur) {
		return l.tokenizeHTMLString()
	} else if isStartOfNumeral(l.cur) {
		return l.tokenizeNumeral()
	} else if isStartOfAnyString(l.cur) {
		return l.tokenizeUnquotedString()
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

func isIdentifier(r rune) bool {
	return isAlphabetic(r) || r == '_' || r == '-' || r == '.' || r == '"' || r == '\\' || unicode.IsDigit(r)
}

func isAlphabetic(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '\200' && r <= '\377')
}

// TODO is this also dependent on the context? as in - is not a separator inside of a quoted string
// isSeparator determines if the rune separates tokens. This can be terminal tokens or whitespace.
func isSeparator(r rune) bool {
	return isTerminal(r) || r == '-' || isWhitespace(r)
}

// isTerminal determines if the rune is considered a terminal token in the dot language. This does
// not contain edge operators
func isTerminal(r rune) bool {
	switch token.TokenType(r) {
	case token.LeftBrace, token.RightBrace, token.LeftBracket, token.RightBracket, token.Colon, token.Semicolon, token.Equal, token.Comma:
		return true
	}
	return false
}

func (l *Lexer) tokenizeQuotedString() (token.Token, error) {
	var tok token.Token
	var err error

	// TODO validate the quote is closed
	// TODO cap looking for missing quote at 16384 runes https://gitlab.com/graphviz/graphviz/-/issues/1261
	// TODO how to validate any quotes inside the string are quoted?
	prev := l.cur
	id := []rune{l.cur}
	for err = l.readRune(); err == nil && (l.cur != '"' || (prev == '\\' && l.cur == '"')); err = l.readRune() {
		id = append(id, l.cur)
		prev = l.cur
	}

	if err != nil {
		return tok, err
	}

	// consume closing quote
	id = append(id, l.cur)
	// TODO error handling
	err = l.readRune()

	return token.Token{Type: token.Identifier, Literal: string(id)}, err
}

func (l *Lexer) tokenizeHTMLString() (token.Token, error) {
	var tok token.Token
	var err error

	id := []rune{l.cur}
	for err = l.readRune(); err == nil && !isSeparator(l.cur); err = l.readRune() {
		id = append(id, l.cur)
	}

	if err != nil {
		return tok, err
	}

	return token.Token{Type: token.Identifier, Literal: string(id)}, nil
}

func (l *Lexer) tokenizeNumeral() (token.Token, error) {
	fmt.Println("tokenizeNumeral")
	var tok token.Token
	var err error

	if l.cur == '.' && !unicode.IsDigit(l.next) {
		lexError := l.lexError("`.` needs to be either double-quoted to be a quoted identifier or prefixed or followed by a digit to be a numeral identifier")

		// TODO read til a valid separator or eof? so I can potentially lex more valid tokens?
		err = l.readRune()
		if err != nil {
			return tok, err
		}

		return tok, lexError
	} else if l.cur == '-' && (l.next != '.' && !unicode.IsDigit(l.next)) {
		lexError := l.lexError("`-` needs to be either double-quoted to be a quoted identifier or followed by an optional `.` and at least one digit to be a numeral identifier")

		// TODO read til a valid separator or eof? so I can potentially lex more valid tokens?
		err = l.readRune()
		if err != nil {
			return tok, err
		}

		return tok, lexError
	}

	// TODO validate every l.cur is a digit
	id := []rune{l.cur}
	for err = l.readRune(); l.hasNext() && err == nil && !isSeparator(l.cur); err = l.readRune() {
		id = append(id, l.cur)
	}

	if err != nil {
		return tok, err
	}

	return token.Token{Type: token.Identifier, Literal: string(id)}, nil
}

// tokenizeUnquotedString considers the current rune(s) as an identifier that might be a dot
// keyword.
func (l *Lexer) tokenizeUnquotedString() (token.Token, error) {
	var tok token.Token
	var err error

	id := []rune{l.cur}
	for err = l.readRune(); l.hasNext() && err == nil && !isSeparator(l.cur); err = l.readRune() {
		id = append(id, l.cur)
	}

	if err != nil {
		return tok, err
	}

	literal := string(id)
	tok = token.Token{Type: token.LookupIdentifier(literal), Literal: literal}

	return tok, err
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
