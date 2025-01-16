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
		// TODO how hard / complex would it make my code to support newlines without line continuation?
		// TODO add test with more than one whitespace. for some reason two whitespaces after word lead
		// to an off by one error
		"QuotedIDPastMaxColumnWithMultipleWhitespaces": {
			in: `"This is a test of a long attribute value that is past the max column which should be
			split on word  boundaries several times of course as long as this is necessary it should also
			respect giant URLs
			https://github.com/teleivo/dot/blob/fake/27b6dbfe4b99f67df74bfb7323e19d6c547f68fd/parser_test.go#L13"`,
			want: `"This is a test of a long attribute value that is past the max column which should be split on word\
  boundaries several times of course as long as this is necessary it should also respect giant URLs \
https://github.com/teleivo/dot/blob/fake/27b6dbfe4b99f67df74bfb7323e19d6c547f68fd/parser_test.go#L13"`,
		},
		// TODO add test with pure newlines in the correct spot	where a line continuation should go
		// these should be normalized
		// input uses the same text as in QuotedIDPastMaxColumnIsBrokenUp with line continuations in
		// places they should not be i.e. too early and too late
		"QuotedIDWithOutOfPlaceLineContinuations": {
			in: `"This is a test of a long attribute \
value that is past the max column which\
 should be split on word boundaries several times of course as long as this is necessary it should also respect giant URLs\
 https://github.com/teleivo/dot/blob/fake/27b6dbfe4b99f67df74bfb7323e19d6c547f68fd/parser_test.go#L13"`,
			want: `"This is a test of a long attribute value that is past the max column which should be split on word\
 boundaries several times of course as long as this is necessary it should also respect giant URLs \
https://github.com/teleivo/dot/blob/fake/27b6dbfe4b99f67df74bfb7323e19d6c547f68fd/parser_test.go#L13"`,
		},
		"QuotedIDWithUnnecessaryLineContinuationBeforeClosingQuote": {
			in: `"This is an ID that does not need a split\
"`,
			want: `"This is an ID that does not need a split"`,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			var gotFirst bytes.Buffer
			p := Printer{w: &gotFirst}

			err := p.printID(ast.ID{Literal: test.in})
			require.NoErrorf(t, err, "printID()")

			require.EqualValuesf(t, gotFirst.String(), test.want, "printID")

			t.Logf("print again with the previous output as the input to ensure printing is idempotent")

			var gotSecond bytes.Buffer
			p = Printer{w: &gotSecond}

			err = p.printID(ast.ID{Literal: gotFirst.String()})
			require.NoErrorf(t, err, "printID()")

			assert.EqualValuesf(t, gotSecond.String(), gotFirst.String(), "printID")
		})
	}
}
