package token

import (
	"strconv"
)

// Position describes a position in DOT source code.
// A Position is valid if the line number is > 0.
type Position struct {
	Line   int // line number, starting at 1
	Column int // column number, starting at 1 (byte offset)
}

// IsValid reports whether the position is valid.
func (p Position) IsValid() bool { return p.Line > 0 }

// String returns the position in line:column format.
func (p Position) String() string {
	return strconv.Itoa(p.Line) + ":" + strconv.Itoa(p.Column)
}

// Before reports whether the position p is before o.
func (p Position) Before(o Position) bool {
	if p.Line < o.Line {
		return true
	} else if p.Line == o.Line && p.Column < o.Column {
		return true
	}
	return false
}

// After reports whether the position p is after o.
func (p Position) After(o Position) bool {
	if p.Line > o.Line {
		return true
	} else if p.Line == o.Line && p.Column > o.Column {
		return true
	}
	return false
}
