package layout_test

import (
	"errors"
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
		// TODO make simple test for GoStringer/String on literal want structures
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
			"NestedDoc": {
				in: layout.NewDoc(80).
					Text("digraph").
					Space().
					Text("{").
					Group(func(d *layout.Doc) {
						d.
							IndentIf(1, layout.Broken, func(d *layout.Doc) {
								d.
									Break(1).
									Group(func(d *layout.Doc) {
										d.
											Text("3").
											Space().
											Text("->").
											SpaceIf(layout.Flat).
											BreakIf(1, layout.Broken).
											Text("2").
											Space().
											Group(func(d *layout.Doc) {
												d.
													Group(func(d *layout.Doc) {
														d.
															Text("[").
															BreakIf(1, layout.Broken).
															Indent(1, func(d *layout.Doc) {
																d.
																	Text("color").
																	Text("=").
																	Text("\"blue\"").
																	TextIf(",", layout.Flat).
																	BreakIf(1, layout.Broken).
																	Text("background").
																	Text("=").
																	Text("\"transparent red\"")
															}).
															BreakIf(1, layout.Broken).
															Text("]").
															Space()
													}).
													BreakIf(1, layout.Broken)
											})
									}).
									Break(1).
									Text("rank").
									Text("=").
									Text("same")
							}).
							Break(1).
							Text("}")
					}),
			},
		}

		for name, tc := range tests {
			t.Run(name, func(t *testing.T) {
				// GoStringer should produce valid Go code
				const dir = "testdata/gostringer"
				err := os.Mkdir(dir, 0o700)
				if !errors.Is(err, os.ErrExist) {
					require.NoError(t, err)
				}
				t.Cleanup(func() {
					if t.Failed() {
						t.Logf("faulty Go code generated using GoStringer is in: %s", dir)
					} else {
						if err := os.RemoveAll(dir); err != nil {
							t.Logf("failed to cleanup temp dir %s due to: %v", dir, err)
						}
					}
				})

				f, err := os.Create(dir + "/main.go")
				require.NoError(t, err)
				err = tc.in.Render(f, layout.Go)
				require.NoErrorf(t, err, "failed to render Go format")
				cmd := exec.CommandContext(t.Context(), "go", "run", f.Name())
				got, err := cmd.Output()
				require.NoErrorf(t, err, "failed to execute Go code generated using GoStringer")

				// GoStringer should render to the same layout as its source document
				var sb strings.Builder
				tc.in.Clone().Render(&sb, layout.Default)
				want := sb.String()

				assert.Equals(t, string(got), want)
			})
		}
	})
}
