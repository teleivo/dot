package main

import (
	"fmt"
	"io"
	"os"
)

func main() {
	if err := run(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
	}
}

func run(in io.Reader, out io.Writer) error {
	return nil
}
