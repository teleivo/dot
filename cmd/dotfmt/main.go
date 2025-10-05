package main

import (
	"fmt"
	"io"
	"os"

	"github.com/teleivo/dot/layout"
	"github.com/teleivo/dot/printer"
)

func main() {
	if err := run(os.Args, os.Stdin, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func run(args []string, r io.Reader, w io.Writer) error {
	// flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	// debug := flags.String("debug", "ff", "Print the intermediate representation used to layout the DOT code using 'layout' or print it as a a main.go 'go'")
	// err := flags.Parse(args[1:])
	// if err != nil {
	// 	return err
	// }

	// TODO fix
	// TODO create a main.go I could pipe to a file and run. extract logic from test?
	// TODO can I improve the indentation of GoStringer?
	// _ = debug
	p := printer.NewPrinter(r, w, layout.DebugGo)
	return p.Print()
}
