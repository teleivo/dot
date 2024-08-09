package dot_test

import (
	"strings"
	"testing"

	"github.com/teleivo/assertive/assert"
	"github.com/teleivo/assertive/require"
	"github.com/teleivo/dot"
)

func TestParser(t *testing.T) {
	tests := map[string]struct {
		in   string
		want dot.Graph
		err  error
	}{
		"Empty": {
			in:   "",
			want: dot.Graph{},
		},
		"EmptyDirectedGraph": {
			in: "digraph {}",
			want: dot.Graph{
				Directed: true,
			},
		},
		"EmptyUndirectedGraph": {
			in:   "graph {}",
			want: dot.Graph{},
		},
		"StrictDirectedUnnamedGraph": {
			in: `strict digraph {}`,
			want: dot.Graph{
				Strict:   true,
				Directed: true,
			},
		},
		"StrictDirectedNamedGraph": {
			in: `strict digraph dependencies {}`,
			want: dot.Graph{
				Strict:   true,
				Directed: true,
				ID:       "dependencies",
			},
		},
	}

	// TODO start parsing the simplest statement

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			p, err := dot.New(strings.NewReader(test.in))

			require.NoErrorf(t, err, "New(%q)", test.in)

			g, err := p.Parse()

			assert.NoErrorf(t, err, "Parse(%q)", test.in)
			assert.EqualValuesf(t, g, &test.want, "Parse(%q)", test.in)
		})
	}

	t.Run("Invalid", func(t *testing.T) {
		tests := map[string]struct {
			in     string
			errMsg string
		}{
			"StrictMustBeFirstKeyword": {
				in:     "digraph strict {}",
				errMsg: `got "strict" instead`,
			},
			"GraphIDMustComeAfterGraphKeywords": {
				in:     "dependencies {}",
				errMsg: `got "dependencies" instead`,
			},
			"LeftBraceMustFollow": {
				in:     "graph dependencies [",
				errMsg: `got "[" instead`,
			},
		}

		for name, test := range tests {
			t.Run(name, func(t *testing.T) {
				p, err := dot.New(strings.NewReader(test.in))

				require.NoErrorf(t, err, "New(%q)", test.in)

				_, err = p.Parse()

				assert.NotNilf(t, err, "Parse(%q)", test.in)
				assertContains(t, err.Error(), test.errMsg)
			})
		}
	})
}

func assertContains(t *testing.T, got, want string) {
	if !strings.Contains(got, want) {
		t.Errorf("got %q which does not contain %q", got, want)
	}
}
