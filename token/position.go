package token

import (
	"strconv"
)

// Position describes a position in dot source code.
type Position struct {
	Row    int // Row is the line number starting at 1. A row of zero is not valid.
	Column int // Column is the horizontal position of in terms of runes starting at 1. A column of zero is not valid.
}

// String returns the position in line:column format.
func (p Position) String() string {
	return strconv.Itoa(p.Row) + ":" + strconv.Itoa(p.Column)
}

// Before reports whether the position p is before o.
func (p Position) Before(o Position) bool {
	if p.Row < o.Row {
		return true
	} else if p.Row == o.Row && p.Column < o.Column {
		return true
	}
	return false
}

// Before reports whether the position p is after o.
func (p Position) After(o Position) bool {
	if p.Row > o.Row {
		return true
	} else if p.Row == o.Row && p.Column > o.Column {
		return true
	}
	return false
}
