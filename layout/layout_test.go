package layout_test

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/teleivo/assertive/assert"
	"github.com/teleivo/assertive/require"
	"github.com/teleivo/dot/layout"
)

func TestLayout(t *testing.T) {
	t.Run("GoString", func(t *testing.T) {
		// TODO reuse also test String()? and add wantString into table
		tests := map[string]struct {
			in *layout.Doc
		}{
			"EmptyDoc": {
				in: layout.NewDoc(80),
			},
			"EmptyGroup": {
				in: layout.NewDoc(80).Group(func(d *layout.Doc) {}),
			},
			"EmptyIndent": {
				in: layout.NewDoc(80).Indent(1, func(d *layout.Doc) {}),
			},
			// TODO add more nesting
			// TODO also test Indent and IndentIf
			"GoMain": {
				in: layout.NewDoc(80).
					Text("package main").
					Break(1).
					Text("func").
					Space().
					Text("main").
					Text("(").Text(")").
					Group(func(d *layout.Doc) {
						d.
							SpaceIf(layout.Flat).
							BreakIf(1, layout.Broken).
							Text("{").
							SpaceIf(layout.Flat).
							BreakIf(1, layout.Broken).
							Text(`print("yes")`).
							SpaceIf(layout.Flat).
							BreakIf(1, layout.Broken).
							Text("}")
					}),
			},
		}

		for name, tc := range tests {
			t.Run(name, func(t *testing.T) {
				var sb strings.Builder
				tc.in.Render(&sb, false)
				want := sb.String()

				// GoStringer should produce valid Go code
				goTemplate := `package main

		import (
			"os"

			"github.com/teleivo/dot/layout"
		)

		func main() {
			d := %s
			d.Render(os.Stdout, false)
		}`
				var goStringer strings.Builder
				fmt.Fprintf(&goStringer, "%#v", tc.in)

				const dir = "testdata/gostringer"
				err := os.Mkdir(dir, 0o700)
				if !errors.Is(err, os.ErrExist) {
					require.NoError(t, err)
				}
				t.Cleanup(func() {
					if t.Failed() {
						t.Logf("faulty Go code using layout.Doc generated using GoStringer is in: %s", dir)
					} else {
						if err := os.RemoveAll(dir); err != nil {
							t.Logf("failed to cleanup temp dir %s due to: %v", dir, err)
						}
					}
				})

				f, err := os.Create(dir + "/main.go")
				require.NoError(t, err)
				fmt.Fprintf(f, goTemplate, goStringer.String())
				cmd := exec.CommandContext(t.Context(), "go", "run", f.Name())
				got, err := cmd.Output()
				require.NoErrorf(t, err, "failed to execute Go code using layout.Doc generated using GoStringer")

				// GoStringer should render to the same layout as its source document
				assert.Equals(t, string(got), want)
			})
		}
	})
}
