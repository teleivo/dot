package main

import (
	"fmt"
	"io"
	"os"

	"github.com/teleivo/dot/printer"
)

func main() {
	if err := run(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func run(r io.Reader, w io.Writer) error {
	p := printer.NewPrinter(r, w)
	return p.Print()
}
