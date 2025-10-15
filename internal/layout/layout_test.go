package layout_test

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/teleivo/assertive/assert"
	"github.com/teleivo/assertive/require"
	"github.com/teleivo/dot/internal/layout"
)

func TestLayout(t *testing.T) {
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
		"RootDocIsConsideredBroken": {
			in:          layout.NewDoc(10).TextIf("hello", layout.Broken),
			wantDefault: "hello",
			wantLayout: `<text cond="broken" content="hello"/>
`,
		},
		"GroupDoesNotBreakIfOnDocLimit": {
			in: layout.NewDoc(10).Group(func(d *layout.Doc) {
				d.Text("01234").BreakIf(3, layout.Broken).Text("56789")
			}),
			wantDefault: "0123456789",
			wantLayout: `<group width=10>
	<text width=5 content="01234"/>
	<break cond="broken" count=3/>
	<text width=5 content="56789"/>
</group>
`,
		},
		"GroupBreaksIfExceedsDocLimit": {
			in: layout.NewDoc(10).Group(func(d *layout.Doc) {
				d.Text("01234").BreakIf(3, layout.Broken).Text("56789a")
			}),
			wantDefault: "01234\n\n\n56789a",
			wantLayout: `<group width=broken>
	<text width=5 content="01234"/>
	<break cond="broken" count=3/>
	<text width=6 content="56789a"/>
</group>
`,
		},
		"IndentAndDeIndent": {
			in: layout.NewDoc(10).Indent(2, func(d *layout.Doc) {
				d.
					Break(1).
					Text("hello").
					Indent(-1, func(d *layout.Doc) {
						d.
							Break(1).
							Text("world")
					})
			}),
			wantDefault: "\n\t\thello\n\tworld",
			wantLayout: `<indent columns=2>
	<break count=1/>
	<text width=5 content="hello"/>
	<indent columns=-1>
		<break count=1/>
		<text width=5 content="world"/>
	</indent>
</indent>
`,
		},
		"IndentNotDoneAtStartOfLine": {
			in: layout.NewDoc(10).Indent(1, func(d *layout.Doc) {
				d.Text("hello")
			}),
			wantDefault: "hello",
			wantLayout: `<indent columns=1>
	<text width=5 content="hello"/>
</indent>
`,
		},
		"SkipTrailingSpaces": {
			in:          layout.NewDoc(10).Space().Text("012345678").Space().Break(1),
			wantDefault: " 012345678\n",
			wantLayout: `<space/>
<text width=9 content="012345678"/>
<space/>
<break count=1/>
`,
		},
		"TopLevelSkipTrailingSpacesButRenderConditionalBreak": {
			in:          layout.NewDoc(10).Text("01234").BreakIf(1, layout.Broken).Text("56789").Space(),
			wantDefault: "01234\n56789",
			wantLayout: `<text width=5 content="01234"/>
<break cond="broken" count=1/>
<text width=5 content="56789"/>
<space/>
`,
		},
		"TrailingSpaceBeforeConditionalBreakShouldNotCauseGroupToBreak": {
			in: layout.NewDoc(10).Group(func(d *layout.Doc) {
				d.Text("0123456789").Space().BreakIf(1, layout.Broken)
			}),
			wantDefault: `0123456789`,
			wantLayout: `<group width=10>
	<text width=10 content="0123456789"/>
	<space/>
	<break cond="broken" count=1/>
</group>
`,
		},
		"TrailingSpaceInInnerGroupShouldCauseGroupToBreak": {
			in: layout.NewDoc(10).
				Group(func(d *layout.Doc) {
					d.Group(func(d *layout.Doc) {
						d.Text("01234").Space()
					}).
						BreakIf(1, layout.Broken)
					d.Group(func(d *layout.Doc) {
						d.Text("56789").Space()
					})
				}),
			wantDefault: "01234\n56789",
			wantLayout: `<group width=broken>
	<group width=5>
		<text width=5 content="01234"/>
		<space/>
	</group>
	<break cond="broken" count=1/>
	<group width=5>
		<text width=5 content="56789"/>
		<space/>
	</group>
</group>
`,
		},
		"MergeConsecutiveUnconditionalSpaces": {
			in:          layout.NewDoc(80).Space().Space().Text("in between"),
			wantDefault: ` in between`,
			wantLayout: `<space/>
<text width=10 content="in between"/>
`,
		},
		"MergeConsecutiveConditionalSpaces": {
			in:          layout.NewDoc(80).SpaceIf(layout.Broken).SpaceIf(layout.Broken).Text("in between"),
			wantDefault: ` in between`,
			wantLayout: `<space cond="broken"/>
<text width=10 content="in between"/>
`,
		},
		"DontMergeConsecutiveSpacesWithDifferingCondition": {
			in:          layout.NewDoc(80).SpaceIf(layout.Broken).Space().Text("in between"),
			wantDefault: ` in between`,
			wantLayout: `<space cond="broken"/>
<space/>
<text width=10 content="in between"/>
`,
		},
		"SpaceFollowedByNonRenderingConditionalText": {
			// When a group breaks, TextIf(Flat) doesn't render, so the space before it is trailing
			in: layout.NewDoc(5).Group(func(d *layout.Doc) {
				d.Text("hello").Space().TextIf("world", layout.Flat)
			}),
			wantDefault: "hello",
			wantLayout: `<group width=broken>
	<text width=5 content="hello"/>
	<space/>
	<text cond="flat" width=5 content="world"/>
</group>
`,
		},
		"MultipleConsecutiveTrailingSpaces": {
			in: layout.NewDoc(10).Group(func(d *layout.Doc) {
				d.Text("hello").Space().Space()
			}),
			wantDefault: "hello",
			wantLayout: `<group width=5>
	<text width=5 content="hello"/>
	<space/>
</group>
`,
		},
		"ConditionalTrailingSpace": {
			in: layout.NewDoc(5).Group(func(d *layout.Doc) {
				d.Text("hello").SpaceIf(layout.Flat).BreakIf(1, layout.Broken)
			}),
			wantDefault: "hello",
			wantLayout: `<group width=5>
	<text width=5 content="hello"/>
	<space cond="flat"/>
	<break cond="broken" count=1/>
</group>
`,
		},
		"SpaceBetweenBreaks": {
			// Space between breaks is trailing and doesn't render; breaks merge to single newline
			in:          layout.NewDoc(80).Text("hello").Break(1).Space().Break(1).Text("world"),
			wantDefault: "hello\nworld",
			wantLayout: `<text width=5 content="hello"/>
<break count=1/>
<space/>
<break count=1/>
<text width=5 content="world"/>
`,
		},
		"GroupWithOnlySpaces": {
			// consecutive spaces are merged and since the space trailing its not rendered
			in: layout.NewDoc(80).Group(func(d *layout.Doc) {
				d.Space().Space()
			}),
			wantDefault: "",
			wantLayout: `<group width=0>
	<space/>
</group>
`,
		},
		"LeadingSpacesBeforeBreak": {
			// Spaces at the start before any content followed by break should not render
			in:          layout.NewDoc(80).Space().Space().Break(1).Text("hello"),
			wantDefault: "\nhello",
			wantLayout: `<space/>
<break count=1/>
<text width=5 content="hello"/>
`,
		},
		"TrailingSpacesWithMixedConditions": {
			// consecutive spaces with different conditions are not merged
			in: layout.NewDoc(5).Group(func(d *layout.Doc) {
				d.Text("hello").Space().SpaceIf(layout.Broken)
			}),
			wantDefault: "hello",
			wantLayout: `<group width=5>
	<text width=5 content="hello"/>
	<space/>
	<space cond="broken"/>
</group>
`,
		},
		"SpaceAfterEmptyGroup": {
			// space after an empty group but before content should be counted
			in:          layout.NewDoc(80).Group(func(d *layout.Doc) {}).Space().Text("hello"),
			wantDefault: " hello",
			wantLayout: `<group width=0>
</group>
<space/>
<text width=5 content="hello"/>
`,
		},
		"TrailingSpaceAfterIndent": {
			in: layout.NewDoc(80).Group(func(d *layout.Doc) {
				d.Indent(1, func(d *layout.Doc) {
					d.Text("hello")
				}).Space()
			}),
			wantDefault: "hello",
			wantLayout: `<group width=5>
	<indent columns=1>
		<text width=5 content="hello"/>
	</indent>
	<space/>
</group>
`,
		},
		"SpaceInGroupFollowedByConditionalBreakAndMoreText": {
			in: layout.NewDoc(20).Group(func(d *layout.Doc) {
				d.Text("hello").Space().BreakIf(1, layout.Broken).Text("world")
			}),
			wantDefault: "hello world",
			wantLayout: `<group width=11>
	<text width=5 content="hello"/>
	<space/>
	<break cond="broken" count=1/>
	<text width=5 content="world"/>
</group>
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
	rank=same
}`,
			wantLayout: `<text width=7 content="digraph"/>
<space/>
<text width=1 content="{"/>
<group width=broken>
	<indent columns=1>
		<break count=1/>
		<group width=50>
			<text width=1 content="3"/>
			<space/>
			<text width=2 content="->"/>
			<space cond="flat"/>
			<break cond="broken" count=1/>
			<text width=1 content="2"/>
			<space/>
			<group width=43>
				<group width=43>
					<text width=1 content="["/>
					<break cond="broken" count=1/>
					<indent columns=1>
						<text width=5 content="color"/>
						<text width=1 content="="/>
						<text width=6 content="\"blue\""/>
						<text cond="flat" width=1 content=","/>
						<break cond="broken" count=1/>
						<text width=10 content="background"/>
						<text width=1 content="="/>
						<text width=17 content="\"transparent red\""/>
					</indent>
					<break cond="broken" count=1/>
					<text width=1 content="]"/>
					<space/>
				</group>
				<break cond="broken" count=1/>
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
				err := tc.in.Clone().Render(&got, layout.Default)
				require.NoErrorf(t, err, "failed to render default format")

				assert.EqualValues(t, got.String(), tc.wantDefault)
			})
		}
	})
	t.Run("RenderLayout", func(t *testing.T) {
		for name, tc := range tests {
			t.Run(name, func(t *testing.T) {
				var got strings.Builder
				err := tc.in.Clone().Render(&got, layout.Layout)
				require.NoErrorf(t, err, "failed to render layout format")

				assert.EqualValues(t, got.String(), tc.wantLayout)
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
				err = tc.in.Clone().Render(f, layout.Go)
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

				assert.EqualValues(t, string(got), want)
			})
		}
	})
}
