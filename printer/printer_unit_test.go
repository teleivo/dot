package printer

import (
	"bytes"
	"testing"

	"github.com/teleivo/assertive/assert"
	"github.com/teleivo/assertive/require"
	"github.com/teleivo/dot/ast"
)

func TestPrintID(t *testing.T) {
	tests := map[string]struct {
		in   string
		want string
	}{
		"UnquotedIDPastMaxColumnIsNotBrokenUp": {
			in: `aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb
1.11111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111
1.111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111112`,
			want: `aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb
1.11111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111
1.111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111112`,
		},
		// World in Chinese each rune is 3 bytes long 世界
		"QuotedIDOfMaxColumnIsNotBrokenUp": {
			in:   `"aaaaaaaaaaaaa aaaaaaaaa\"aaaaaaaaaaaaaaaaaaaaaaaa世界aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\""`,
			want: `"aaaaaaaaaaaaa aaaaaaaaa\"aaaaaaaaaaaaaaaaaaaaaaaa世界aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\""`,
		},
		// TODO URLs\ could actually be URLs \ and land right on the 100. How would I achieve that?
		"QuotedIDPastMaxColumnIsBrokenUp": {
			in: `"This is a test of a long attribute value that is past the max column which should be split on word boundaries several times of course as long as this is necessary it should also respect giant URLs https://github.com/teleivo/dot/blob/fake/27b6dbfe4b99f67df74bfb7323e19d6c547f68fd/parser_test.go#L13"`,
			want: `"This is a test of a long attribute value that is past the max column which should be split on word\
 boundaries several times of course as long as this is necessary it should also respect giant URLs \
https://github.com/teleivo/dot/blob/fake/27b6dbfe4b99f67df74bfb7323e19d6c547f68fd/parser_test.go#L13"`,
		},
		// 	// takes the output from QuotedIDPastMaxColumnIsBrokenUp as input and output
		// 	"BreakingUpQuotedIDIsIdempotent": {
		// 		in: `"This is a test of a long attribute value that is past the max column which should be split on word\
		// boundaries several times of course as long as this is necessary it should also respect giant URLs\
		// https://github.com/teleivo/dot/blob/fake/27b6dbfe4b99f67df74bfb7323e19d6c547f68fd/parser_test.go#L13"`,
		// 		want: `"This is a test of a long attribute value that is past the max column which should be split on word\
		// boundaries several times of course as long as this is necessary it should also respect giant URLs\
		// https://github.com/teleivo/dot/blob/fake/27b6dbfe4b99f67df74bfb7323e19d6c547f68fd/parser_test.go#L13"`,
		// 	},
		// TODO I think there is some off by one error in my placement of \ see ./example3.dot with
		// maxColumn=20. I see the \ appear on 21. How to elicit this with a test in here?
		// TODO add test with quoted ID containing newlines. Newlines in the ID should restart the counter towards maxcolumn
		// TODO add test with split quoted ID that is split in a different place than I would, these
		// should be stripped and \\n be placed as if the ID never had any.
		// TODO how does my current approach deal with special characters? as whitespace is used as a
		// word boundary
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			var got bytes.Buffer
			p := Printer{w: &got}
			in := ast.ID{Literal: test.in}

			err := p.printID(in)
			require.NoErrorf(t, err, "printID()")

			assert.EqualValuesf(t, got.String(), test.want, "printID")
		})
	}
}
