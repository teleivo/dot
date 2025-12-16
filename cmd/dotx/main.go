package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"runtime/pprof"
	"text/tabwriter"

	"github.com/teleivo/dot"
	"github.com/teleivo/dot/internal/layout"
	"github.com/teleivo/dot/printer"
	"github.com/teleivo/dot/token"
)

func main() {
	if len(os.Args) < 2 {
		usage(os.Stderr)
		os.Exit(1)
	}

	if err := run(os.Args, os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func run(args []string, r io.Reader, w io.Writer, wErr io.Writer) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: dotx <command> [args]\ncommands: fmt, inspect")
	}

	if args[1] == "-h" || args[1] == "--help" || args[1] == "help" {
		usage(wErr)
		return nil
	}

	switch args[1] {
	case "fmt":
		return runFmt(args[2:], r, w, wErr)
	case "inspect":
		return runInspect(args[2:], r, w, wErr)
	case "":
		return errors.New("no command specified")
	default:
		return fmt.Errorf("unknown command: %s", args[1])
	}
}

func usage(w io.Writer) {
	_, _ = fmt.Fprintln(w, "dotx is a tool for working with DOT (Graphviz) graph files")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "usage: dotx <command> [args]")
	_, _ = fmt.Fprintln(w, "commands: fmt, inspect")
}

func runFmt(args []string, r io.Reader, w io.Writer, wErr io.Writer) error {
	flags := flag.NewFlagSet("fmt", flag.ExitOnError)
	flags.SetOutput(wErr)
	flags.Usage = func() {
		_, _ = fmt.Fprintln(wErr, "usage: dotx fmt [flags]")
		_, _ = fmt.Fprintln(wErr, "flags:")
		flags.PrintDefaults()
	}
	format := flags.String("format", "default", "Print the formatted DOT code using 'default', the intermediate representation (IR) used to layout the DOT code using 'layout' or a runnable main.go of the IR using 'go'")
	cpuProfile := flags.String("cpuprofile", "", "write cpu profile to `file`")
	memProfile := flags.String("memprofile", "", "write memory profile to `file`")

	err := flags.Parse(args)
	if err != nil {
		return err
	}
	ft, err := layout.NewFormat(*format)
	if err != nil {
		return fmt.Errorf("failed to convert -format=%q: %v", *format, err)
	}

	return profile(func() error {
		p := printer.New(r, w, ft)
		if err := p.Print(); err != nil {
			return err
		}
		return nil
	}, *cpuProfile, *memProfile)
}

func profile(fn func() error, cpuProfile, memProfile string) error {
	if cpuProfile != "" {
		f, err := os.Create(cpuProfile)
		if err != nil {
			return fmt.Errorf("could not create CPU profile: %v", err)
		}
		defer func() { _ = f.Close() }()
		if err := pprof.StartCPUProfile(f); err != nil {
			return fmt.Errorf("could not start CPU profile: %v", err)
		}
		defer pprof.StopCPUProfile()
	}

	err := fn()
	if err != nil {
		return err
	}

	if memProfile != "" {
		f, err := os.Create(memProfile)
		if err != nil {
			return fmt.Errorf("could not create memory profile: %v", err)
		}
		defer func() { _ = f.Close() }()
		runtime.GC() // materialize all statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			return fmt.Errorf("could not write memory profile: %v", err)
		}
	}

	return nil
}

func runInspect(args []string, r io.Reader, w io.Writer, wErr io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: dotx inspect <subcommand>\nsubcommands: tree, tokens")
	}

	switch args[0] {
	case "tree":
		return runInspectTree(args[1:], r, w, wErr)
	case "tokens":
		return runInspectTokens(args[1:], r, w, wErr)
	case "":
		return errors.New("no inspect subcommand specified")
	default:
		return fmt.Errorf("unknown inspect subcommand: %s", args[0])
	}
}

func runInspectTree(args []string, r io.Reader, w io.Writer, wErr io.Writer) error {
	flags := flag.NewFlagSet("tree", flag.ExitOnError)
	flags.SetOutput(wErr)
	flags.Usage = func() {
		_, _ = fmt.Fprintln(wErr, "usage: dotx inspect tree [flags]")
		_, _ = fmt.Fprintln(wErr, "flags:")
		flags.PrintDefaults()
	}
	format := flags.String("format", "default", "Print the DOT code using its 'default' indented tree representation, or using 'scheme' for a scheme like tree with positions")
	cpuProfile := flags.String("cpuprofile", "", "write cpu profile to `file`")
	memProfile := flags.String("memprofile", "", "write memory profile to `file`")

	err := flags.Parse(args)
	if err != nil {
		return err
	}
	ft, err := dot.NewFormat(*format)
	if err != nil {
		return fmt.Errorf("failed to convert -format=%q: %v", *format, err)
	}

	return profile(func() error {
		p, err := dot.NewParser(r)
		if err != nil {
			return fmt.Errorf("error creating parser: %v", err)
		}

		t, err := p.Parse()
		if err != nil {
			return fmt.Errorf("error parsing: %v", err)
		}

		for _, parseErr := range p.Errors() {
			_, _ = fmt.Fprintln(wErr, parseErr)
		}

		if err := t.Render(w, ft); err != nil {
			return fmt.Errorf("error rendering tree: %v", err)
		}

		return nil
	}, *cpuProfile, *memProfile)
}

func runInspectTokens(args []string, r io.Reader, w io.Writer, wErr io.Writer) (err error) {
	flags := flag.NewFlagSet("tokens", flag.ExitOnError)
	flags.SetOutput(wErr)
	flags.Usage = func() {
		_, _ = fmt.Fprintln(wErr, "usage: dotx inspect tokens [flags]")
		_, _ = fmt.Fprintln(wErr, "flags:")
		flags.PrintDefaults()
	}
	cpuProfile := flags.String("cpuprofile", "", "write cpu profile to `file`")
	memProfile := flags.String("memprofile", "", "write memory profile to `file`")

	err = flags.Parse(args)
	if err != nil {
		return err
	}

	return profile(func() error {
		sc, err := dot.NewScanner(r)
		if err != nil {
			return fmt.Errorf("error scanning: %v", err)
		}

		tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		defer func() {
			if ferr := tw.Flush(); ferr != nil && err == nil {
				err = fmt.Errorf("error flushing output: %v", ferr)
			}
		}()

		_, _ = fmt.Fprintf(tw, "POSITION\tTYPE\tLITERAL\tERROR\n")

		for tok, err := sc.Next(); tok.Type != token.EOF; tok, err = sc.Next() {
			if err != nil {
				return fmt.Errorf("error scanning: %v", err)
			}
			_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", tokenPosition(tok), tok.Type.String(), tokenLiteral(tok), tok.Error)
		}

		return nil
	}, *cpuProfile, *memProfile)
}

func tokenPosition(tok token.Token) string {
	if tok.Start == tok.End {
		return tok.Start.String()
	}
	return tok.Start.String() + "-" + tok.End.String()
}

func tokenLiteral(tok token.Token) string {
	if tok.Type == token.ID || tok.Type == token.ERROR {
		return tok.Literal
	}
	return tok.Type.String()
}
