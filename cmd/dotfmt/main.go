package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/teleivo/dot/layout"
	"github.com/teleivo/dot/printer"
)

func main() {
	if err := run(os.Args, os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func run(args []string, r io.Reader, w io.Writer, wErr io.Writer) error {
	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flags.SetOutput(wErr)
	format := flags.String("format", "default", "Print the formatted DOT code using 'default', the intermediate representation (IR) used to layout the DOT code using 'layout' or a runnable main.go of the IR using 'go'")
	err := flags.Parse(args[1:])
	if err != nil {
		return err
	}
	ft, err := layout.NewFormat(*format)
	if err != nil {
		return fmt.Errorf("failed to convert -format=%q: %v", *format, err)
	}

	p := printer.NewPrinter(r, w, ft)
	return p.Print()
}
