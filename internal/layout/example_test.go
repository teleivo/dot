package layout_test

import (
	"fmt"
	"os"

	"github.com/teleivo/dot/internal/layout"
)

func Example() {
	d := layout.NewDoc(40)
	d.Text("person := Person{")
	d.Group(func(d *layout.Doc) {
		d.BreakIf(1, layout.Broken)
		d.Indent(1, func(d *layout.Doc) {
			d.Text("Name: \"Alice\",")
			d.SpaceIf(layout.Flat)
			d.BreakIf(1, layout.Broken)
			d.Text("Age: 30,")
			d.SpaceIf(layout.Flat)
			d.BreakIf(1, layout.Broken)
			d.Text("Email: \"alice@example.com\"")
			d.TextIf(",", layout.Broken)
		})
		d.SpaceIf(layout.Broken)
		d.BreakIf(1, layout.Broken)
	})
	d.Text("}")
	_ = d.Render(os.Stdout, layout.Default)
	fmt.Println()
	// Output:
	// person := Person{
	// 	Name: "Alice",
	// 	Age: 30,
	// 	Email: "alice@example.com",
	// }
}
