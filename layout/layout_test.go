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
	// TODO add test for trailing newline logic. is there any tag i need to reset the buffered
	// space? how about consecutive spaces? they are merged right now
	tests := map[string]struct {
		in          *layout.Doc
		wantDefault string
		wantLayout  string
	}{
		"EmptyDoc": {
			in:          layout.NewDoc(80),
			wantDefault: "",
			wantLayout:  "",
		},
		"EmptyGroup": {
			in:          layout.NewDoc(80).Group(func(d *layout.Doc) {}),
			wantDefault: "",
			wantLayout: `<group width=0>
</group>
`,
		},
		"EmptyIndent": {
			in:          layout.NewDoc(80).Indent(1, func(d *layout.Doc) {}),
			wantDefault: "",
			wantLayout: `<indent columns=1>
</indent>
`,
		},
		"MergeConsecutiveBreaks": {
			in: layout.NewDoc(80).Break(3).Break(2).Text("in between").Break(1),
			wantDefault: `


in between
`,
			wantLayout: `<break count=3/>
<break count=2/>
<text width=10 content="in between"/>
<break count=1/>
`,
		},
		"NestedDoc": {
			in: layout.NewDoc(80).
				Text("digraph").
				Space().
				Text("{").
				Group(func(d *layout.Doc) {
					d.
						Indent(1, func(d *layout.Doc) {
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
			wantDefault: `digraph {
	3 -> 2 [color="blue",background="transparent red"]
	rank =same
}`,
			wantLayout: `<text width=7 content="digraph"/>
<space/>
<text width=1 content="{"/>
<group width=broken>
	<indent columns=1>
		<break count=1/>
		<group width=51>
			<text width=1 content="3"/>
			<space/>
			<text width=2 content="->"/>
			<space/>
			<break count=1/>
			<text width=1 content="2"/>
			<space/>
			<group width=44>
				<group width=44>
					<text width=1 content="["/>
					<break count=1/>
					<indent columns=1>
						<text width=5 content="color"/>
						<text width=1 content="="/>
						<text width=6 content="\"blue\""/>
						<text width=1 content=","/>
						<break count=1/>
						<text width=10 content="background"/>
						<text width=1 content="="/>
						<text width=17 content="\"transparent red\""/>
					</indent>
					<break count=1/>
					<text width=1 content="]"/>
					<space/>
				</group>
				<break count=1/>
			</group>
		</group>
		<break count=1/>
		<text width=4 content="rank"/>
		<text width=1 content="="/>
		<text width=4 content="same"/>
	</indent>
	<break count=1/>
	<text width=1 content="}"/>
</group>
`,
		},
	}

	t.Run("RenderDefault", func(t *testing.T) {
		for name, tc := range tests {
			t.Run(name, func(t *testing.T) {
				var got strings.Builder
				err := tc.in.Render(&got, layout.Default)
				require.NoErrorf(t, err, "failed to render default format")

				assert.Equals(t, got.String(), tc.wantDefault)
			})
		}
	})
	t.Run("RenderLayout", func(t *testing.T) {
		for name, tc := range tests {
			t.Run(name, func(t *testing.T) {
				var got strings.Builder
				err := tc.in.Render(&got, layout.Layout)
				require.NoErrorf(t, err, "failed to render layout format")

				assert.Equals(t, got.String(), tc.wantLayout)
			})
		}
	})
	t.Run("RenderGo", func(t *testing.T) {
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
							t.Logf("failed to cleanup test dir %s due to: %v", dir, err)
						}
					}
				})

				f, err := os.Create(dir + "/main.go")
				require.NoError(t, err)
				err = tc.in.Render(f, layout.Go)
				require.NoErrorf(t, err, "failed to render Go format")
				cmd := exec.CommandContext(t.Context(), "go", "run", f.Name())
				got, err := cmd.Output()
				var exitErr *exec.ExitError
				if errors.As(err, &exitErr) {
					require.NoErrorf(t, err, "failed to execute Go code generated using GoStringer: %s", exitErr.Stderr)
				} else {
					require.NoErrorf(t, err, "failed to execute Go code generated using GoStringer")
				}

				// GoStringer should render to the same layout as its source document
				var sb strings.Builder
				tc.in.Clone().Render(&sb, layout.Default)
				want := sb.String()

				assert.Equals(t, string(got), want)
			})
		}
	})
}
