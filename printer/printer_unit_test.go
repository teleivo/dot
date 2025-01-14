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
		"QuotedIDOfWithNonLineContinuationNewlines": {
			in: `"aaaaaaaaaaaaa aaaaaaaaa
			aaaaaaaaaaaaaaaaaaaaaaaa世界aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\" bbbbb cccccc ddddd"`,
			want: `"aaaaaaaaaaaaa aaaaaaaaa
			aaaaaaaaaaaaaaaaaaaaaaaa世界aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\" bbbbb cccccc ddddd"`,
		},
		"QuotedIDPastMaxColumnIsBrokenUp": {
			in: `"This is a test of a long attribute value that is past the max column which should be split on word boundaries several times of course as long as this is necessary it should also respect giant URLs https://github.com/teleivo/dot/blob/fake/27b6dbfe4b99f67df74bfb7323e19d6c547f68fd/parser_test.go#L13"`,
			want: `"This is a test of a long attribute value that is past the max column which should be split on word\
 boundaries several times of course as long as this is necessary it should also respect giant URLs \
https://github.com/teleivo/dot/blob/fake/27b6dbfe4b99f67df74bfb7323e19d6c547f68fd/parser_test.go#L13"`,
		},
		// TODO add test with split quoted ID that is split in a different place than I would, these
		// should be stripped and \\n be placed as if the ID never had any.
		// TODO how does my current approach deal with special characters? as whitespace is used as a
		// word boundary
		// TODO add idempotency test to main printer test as well
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			var gotFirst bytes.Buffer
			p := Printer{w: &gotFirst}

			err := p.printID(ast.ID{Literal: test.in})
			require.NoErrorf(t, err, "printID()")

			require.EqualValuesf(t, gotFirst.String(), test.want, "printID")

			t.Logf("printID should be idempotent")

			var gotSecond bytes.Buffer
			p = Printer{w: &gotSecond}

			err = p.printID(ast.ID{Literal: gotFirst.String()})
			require.NoErrorf(t, err, "printID()")

			assert.EqualValuesf(t, gotSecond.String(), gotFirst.String(), "printID")
		})
	}
}
