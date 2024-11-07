package main

import (
	"fmt"
	"io"
	"os"

	"github.com/teleivo/dot"
)

func main() {
	if err := run(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
	}
}

func run(r io.Reader, w io.Writer) error {
	return dot.Print(r, w)
}
