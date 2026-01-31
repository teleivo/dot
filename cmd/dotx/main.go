package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"syscall"
	"text/tabwriter"

	"github.com/teleivo/dot"
	dotfmt "github.com/teleivo/dot/internal/format"
	"github.com/teleivo/dot/internal/layout"
	"github.com/teleivo/dot/internal/version"
	"github.com/teleivo/dot/lsp"
	"github.com/teleivo/dot/token"
	"github.com/teleivo/dot/watch"
)

// errFlagParse is a sentinel error indicating flag parsing failed.
// The flag package already printed the error, so main should not print again.
var errFlagParse = errors.New("flag parse error")

func main() {
	code, err := run(os.Args, os.Stdin, os.Stdout, os.Stderr)
	if err != nil && err != errFlagParse {
		fmt.Fprintf(os.Stderr, "%v\n", err)
	}
	os.Exit(code)
}

func run(args []string, r io.Reader, w io.Writer, wErr io.Writer) (int, error) {
	if len(args) < 2 {
		usage(wErr)
		return 2, nil
	}

	if args[1] == "-h" || args[1] == "--help" || args[1] == "help" {
		usage(wErr)
		return 0, nil
	}

	switch args[1] {
	case "fmt":
		return runFmt(args[2:], r, w, wErr)
	case "inspect":
		return runInspect(args[2:], r, w, wErr)
	case "lsp":
		return runLsp(args[2:], r, w, wErr)
	case "version":
		_, _ = fmt.Fprintln(w, version.Version())
		return 0, nil
	case "watch":
		return runWatch(args[2:], wErr)
	case "":
		return 2, errors.New("no command specified")
	default:
		return 2, fmt.Errorf("unknown command: %s", args[1])
	}
}

func usage(w io.Writer) {
	_, _ = fmt.Fprintln(w, "dotx is a tool for working with DOT (Graphviz) graph files")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "usage: dotx <command> [args]")
	_, _ = fmt.Fprintln(w, "commands: fmt, inspect, lsp, version, watch")
}

func runFmt(args []string, f io.Reader, w io.Writer, wErr io.Writer) (int, error) {
	flags := flag.NewFlagSet("fmt", flag.ContinueOnError)
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
		if err == flag.ErrHelp {
			return 0, nil
		}
		return 2, errFlagParse
	}
	ft, err := layout.NewFormat(*format)
	if err != nil {
		return 2, fmt.Errorf("failed to convert -format=%q: %v", *format, err)
	}

	err = profile(func() error {
		if flags.NArg() == 1 {
			arg := flags.Arg(0)
			fi, err := os.Stat(arg)
			if err != nil {
				return fmt.Errorf("failed to open file: %v", err)
			}
			root, err := filepath.Abs(arg)
			if err != nil {
				return fmt.Errorf("failed to get absolute path: %v", err)
			}

			if fi.IsDir() {
				return dotfmt.Dir(root, ft)
			}
			return dotfmt.File(root, ft)
		}
		// fmt stdin to stdout
		return dotfmt.Reader(f, w, ft)
	}, *cpuProfile, *memProfile)
	if err != nil {
		return 1, err
	}
	return 0, nil
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

func runInspect(args []string, r io.Reader, w io.Writer, wErr io.Writer) (int, error) {
	if len(args) == 0 {
		return 2, fmt.Errorf("usage: dotx inspect <subcommand>\nsubcommands: tree, tokens")
	}

	switch args[0] {
	case "tree":
		return runInspectTree(args[1:], r, w, wErr)
	case "tokens":
		return runInspectTokens(args[1:], r, w, wErr)
	case "":
		return 2, errors.New("no inspect subcommand specified")
	default:
		return 2, fmt.Errorf("unknown inspect subcommand: %s", args[0])
	}
}

func runInspectTree(args []string, r io.Reader, w io.Writer, wErr io.Writer) (int, error) {
	flags := flag.NewFlagSet("tree", flag.ContinueOnError)
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
		if err == flag.ErrHelp {
			return 0, nil
		}
		return 2, errFlagParse
	}
	ft, err := dot.NewFormat(*format)
	if err != nil {
		return 2, fmt.Errorf("failed to convert -format=%q: %v", *format, err)
	}

	err = profile(func() error {
		src, err := io.ReadAll(r)
		if err != nil {
			return fmt.Errorf("error reading input: %v", err)
		}

		p := dot.NewParser(src)
		t := p.Parse()

		for _, parseErr := range p.Errors() {
			_, _ = fmt.Fprintln(wErr, parseErr)
		}

		if err := t.Render(w, ft); err != nil {
			return fmt.Errorf("error rendering tree: %v", err)
		}

		return nil
	}, *cpuProfile, *memProfile)
	if err != nil {
		return 1, err
	}
	return 0, nil
}

func runInspectTokens(args []string, r io.Reader, w io.Writer, wErr io.Writer) (code int, err error) {
	flags := flag.NewFlagSet("tokens", flag.ContinueOnError)
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
		if err == flag.ErrHelp {
			return 0, nil
		}
		return 2, errFlagParse
	}

	err = profile(func() error {
		src, err := io.ReadAll(r)
		if err != nil {
			return fmt.Errorf("error reading input: %v", err)
		}

		sc := dot.NewScanner(src)
		tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		defer func() {
			if ferr := tw.Flush(); ferr != nil && err == nil {
				err = fmt.Errorf("error flushing output: %v", ferr)
			}
		}()

		_, _ = fmt.Fprintf(tw, "POSITION\tTYPE\tLITERAL\tERROR\n")

		for tok := sc.Next(); tok.Kind != token.EOF; tok = sc.Next() {
			_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", tokenPosition(tok), tok.Kind.String(), tokenLiteral(tok), tok.Error)
		}

		return nil
	}, *cpuProfile, *memProfile)
	if err != nil {
		return 1, err
	}
	return 0, nil
}

func tokenPosition(tok token.Token) string {
	if tok.Start == tok.End {
		return tok.Start.String()
	}
	return tok.Start.String() + "-" + tok.End.String()
}

func tokenLiteral(tok token.Token) string {
	if tok.Kind == token.ID || tok.Kind == token.ERROR {
		return tok.Literal
	}
	return tok.Kind.String()
}

func runLsp(args []string, r io.Reader, w io.Writer, wErr io.Writer) (int, error) {
	flags := flag.NewFlagSet("lsp", flag.ContinueOnError)
	flags.SetOutput(wErr)
	flags.Usage = func() {
		_, _ = fmt.Fprintln(wErr, "usage: dotx lsp [flags]")
		_, _ = fmt.Fprintln(wErr, "flags:")
		flags.PrintDefaults()
	}
	debug := flags.Bool("debug", false, "enable debug logging")
	tracePath := flags.String("tracefile", "", "write JSON-RPC messages to `file`")
	cpuProfile := flags.String("cpuprofile", "", "write cpu profile to `file`")
	memProfile := flags.String("memprofile", "", "write memory profile to `file`")

	err := flags.Parse(args)
	if err != nil {
		if err == flag.ErrHelp {
			return 0, nil
		}
		return 2, errFlagParse
	}

	var traceWriter io.Writer
	if *tracePath != "" {
		f, err := os.OpenFile(*tracePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return 1, fmt.Errorf("failed to open tracefile: %v", err)
		}
		defer func() { _ = f.Close() }()
		traceWriter = f
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	err = profile(func() error {
		l, err := lsp.New(lsp.Config{
			In:    r,
			Out:   w,
			Debug: *debug,
			Log:   os.Stderr,
			Trace: traceWriter,
		})
		if err != nil {
			return err
		}
		return l.Start(ctx)
	}, *cpuProfile, *memProfile)
	if err != nil {
		return 1, err
	}
	return 0, nil
}

func runWatch(args []string, wErr io.Writer) (int, error) {
	flags := flag.NewFlagSet("watch", flag.ContinueOnError)
	flags.SetOutput(wErr)
	flags.Usage = func() {
		_, _ = fmt.Fprintln(wErr, "usage: dotx watch [flags] <file>")
		_, _ = fmt.Fprintln(wErr, "flags:")
		flags.PrintDefaults()
	}
	port := flags.String("port", "0", "HTTP server port (0 for a random available port)")
	debug := flags.Bool("debug", false, "enable debug logging")
	cpuProfile := flags.String("cpuprofile", "", "write cpu profile to `file`")
	memProfile := flags.String("memprofile", "", "write memory profile to `file`")

	err := flags.Parse(args)
	if err != nil {
		if err == flag.ErrHelp {
			return 0, nil
		}
		return 2, errFlagParse
	}
	if flags.NArg() != 1 {
		flags.Usage()
		return 2, nil
	}
	file := flags.Arg(0)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	err = profile(func() error {
		w, err := watch.New(watch.Config{
			File:   file,
			Port:   *port,
			Debug:  *debug,
			Stdout: os.Stdout,
			Stderr: os.Stderr,
		})
		if err != nil {
			return err
		}
		return w.Watch(ctx)
	}, *cpuProfile, *memProfile)
	if err != nil {
		return 1, err
	}
	return 0, nil
}
