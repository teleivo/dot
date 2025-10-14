package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"runtime/pprof"

	"github.com/teleivo/dot/internal/layout"
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
	cpuProfile := flags.String("cpuprofile", "", "write cpu profile to `file`")
	memProfile := flags.String("memprofile", "", "write memory profile to `file`")

	err := flags.Parse(args[1:])
	if err != nil {
		return err
	}

	if *cpuProfile != "" {
		f, err := os.Create(*cpuProfile)
		if err != nil {
			return fmt.Errorf("could not create CPU profile: %v", err)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			return fmt.Errorf("could not start CPU profile: %v", err)
		}
		defer pprof.StopCPUProfile()
	}

	ft, err := layout.NewFormat(*format)
	if err != nil {
		return fmt.Errorf("failed to convert -format=%q: %v", *format, err)
	}
	p := printer.NewPrinter(r, w, ft)
	if err := p.Print(); err != nil {
		return err
	}

	if *memProfile != "" {
		f, err := os.Create(*memProfile)
		if err != nil {
			return fmt.Errorf("could not create memory profile: %v", err)
		}
		defer f.Close()
		runtime.GC() // materialize all statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			return fmt.Errorf("could not write memory profile: %v", err)
		}
	}

	return nil
}
