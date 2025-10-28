// Stream DOT tokens from stdin to stdout.
//
// This is a development and debugging tool for the [dot.Scanner]. It is not intended for
// distribution or production use.
package main

import (
	"fmt"
	"io"
	"os"
	"text/tabwriter"

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

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	defer tw.Flush()

	fmt.Fprintf(tw, "POSITION\tTYPE\tVALUE\n")

	for tok, err := sc.Next(); tok.Type != token.EOF; tok, err = sc.Next() {
		fmt.Fprintf(tw, "%s\t%s\t%s\n", position(tok), tok.Type.String(), value(tok))
		if err != nil { // adapt once I collect errors
			return err
		}
	}

	return nil
}

func position(t token.Token) string {
	if t.Start == t.End {
		return t.Start.String()
	}
	return t.Start.String() + "-" + t.End.String()
}

func value(t token.Token) string {
	if t.Type == token.Identifier {
		return t.Literal
	}
	return t.Type.String()
}
