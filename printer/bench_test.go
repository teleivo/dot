package printer_test

import (
	"io"
	"os"
	"testing"

	"github.com/teleivo/dot/internal/layout"
	"github.com/teleivo/dot/printer"
)

func BenchmarkPrint(b *testing.B) {
	benchmarks := []struct {
		name string
		path string
	}{
		{"Small", "testdata/simple.dot"},
		{"Medium", "../cmd/dotx/testdata/graphviz_graphs_directed_cpuprofile.dot"},
		{"Large", "../samples-graphviz/share/examples/world.gv"},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			src, err := os.ReadFile(bm.path)
			if err != nil {
				b.Skipf("skipping: %v", err)
			}
			b.SetBytes(int64(len(src)))
			for b.Loop() {
				p := printer.New(src, io.Discard, layout.Default)
				if err := p.Print(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
