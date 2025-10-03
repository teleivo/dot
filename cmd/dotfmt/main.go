package main

import (
	"flag"
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
	debug := flag.Bool("debug", false, "Print the intermediate representation used to layout the DOT code instead of the code itself")
	flag.Parse()

	p := printer.NewPrinter(r, w, *debug)
	return p.Print()
}
