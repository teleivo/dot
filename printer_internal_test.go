package dot

import (
	"testing"

	"github.com/teleivo/assertive/assert"
)

func TestIsMultiLineComment(t *testing.T) {
	tests := []struct {
		column    int
		text      string
		wantCount int
		wantOk    bool
	}{
		{
			column:    4,
			text:      `	this is a  	comment! that has exactly 100 runes, which is the max column of dotfmt like it or not!`,
			wantCount: 20,
			wantOk:    false,
		},
		{
			column:    5,
			text:      `	this is a  	comment! that has exactly 100 runes, which is the max column of dotfmt like it or not!`,
			wantCount: 20,
			wantOk:    true,
		},
	}

	for _, test := range tests {
		gotCount, gotOk := isMultiLineComment(test.column, test.text)

		assert.Equalsf(t, gotCount, test.wantCount, "isMultiLineComment(%d, %q)", test.column, test.text)
		assert.Equalsf(t, gotOk, test.wantOk, "isMultiLineComment(%d, %q)", test.column, test.text)
	}
}
