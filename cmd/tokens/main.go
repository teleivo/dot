// Stream dot tokens from stdin to stdout. This is mainly meant as a demonstration and debugging aid
// for the [dot.Scanner].
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/teleivo/dot"
	"github.com/teleivo/dot/token"
)

func main() {
	if err := run(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "stopped scanning due to err: %v\n", err)
		os.Exit(1)
	}
}

func run(r io.Reader, w io.Writer) error {
	sc, err := dot.NewScanner(r)
	if err != nil {
		return fmt.Errorf("error scanning: %v", err)
	}

	for tok, err := sc.Next(); tok.Type != token.EOF; tok, err = sc.Next() {
		fmt.Fprintf(w, "%s, err: %v\n", tok, err)
		if err != nil { // adapt once I collect errors
			return err
		}
	}

	return nil
}
