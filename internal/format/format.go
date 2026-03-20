// Package format provides file and directory formatting for DOT files.
package format

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"runtime/trace"
	"sync"

	"github.com/teleivo/dot/internal/layout"
	"github.com/teleivo/dot/printer"
)

// Reader formats DOT source from r and writes the result to w.
func Reader(r io.Reader, w io.Writer, ft layout.Format) error {
	src, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("error reading input: %v", err)
	}
	p := printer.New(src, w, ft)
	return p.Print()
}

type task struct {
	file string
	prev chan io.Writer
	next chan io.Writer
}

// Dir formats all DOT files (.dot, .gv) in a directory tree.
// Formatting errors are written to w in directory walk order.
func Dir(ctx context.Context, root string, ft layout.Format, w io.Writer) error {
	ctx, ta := trace.NewTask(ctx, "Dir")
	defer ta.End()

	var wg sync.WaitGroup
	defer wg.Wait()

	in := make(chan task)
	defer close(in)

	for range runtime.GOMAXPROCS(0) {
		wg.Go(func() {
			for t := range in {
				trace.Log(ctx, "file", t.file)
				var err error
				trace.WithRegion(ctx, "format", func() {
					err = File(t.file, ft)
				})
				trace.WithRegion(ctx, "report", func() {
					w := <-t.prev
					if err != nil {
						_, _ = fmt.Fprintf(w, "%s\n", err)
					}
					t.next <- w
				})
			}
		})
	}

	t := task{
		prev: make(chan io.Writer, 1),
		next: make(chan io.Writer, 1),
	}
	t.prev <- w
	if err := fs.WalkDir(os.DirFS(root), ".", func(path string, d fs.DirEntry, fsErr error) error {
		if fsErr != nil {
			return fsErr
		}
		if d.IsDir() {
			return nil
		}
		if ext := filepath.Ext(d.Name()); ext != ".dot" && ext != ".gv" {
			return nil
		}

		t.file = filepath.Join(root, path)
		select {
		case <-ctx.Done():
			return context.Cause(ctx)
		default:
		}
		select {
		case in <- t:
		case <-ctx.Done():
			return context.Cause(ctx)
		}
		t = task{
			prev: t.next,
			next: make(chan io.Writer, 1),
		}
		return nil
	}); err != nil {
		return err
	}

	return nil
}

// File formats a single DOT file in-place.
func File(path string, ft layout.Format) error {
	fi, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("%s: failed to open file: %s", path, err)
	}
	src, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("%s: error reading file: %s", path, err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+"*")
	if err != nil {
		return fmt.Errorf("%s: failed to create temp file for atomic rename: %s", path, err)
	}

	var success bool
	tmpPath := tmp.Name()
	defer func() {
		if !success {
			_ = os.Remove(tmpPath)
		}
	}()

	if perm := fi.Mode().Perm(); perm != 0o600 {
		if err := tmp.Chmod(perm); err != nil {
			_ = tmp.Close()
			return fmt.Errorf("%s: failed to set file mode: %s", path, err)
		}
	}

	p := printer.New(src, tmp, ft)
	if err := p.Print(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("%s:%s", path, err)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("%s: failed to close temp file: %s", path, err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("%s: failed to rename temp file: %s", path, err)
	}

	success = true
	return nil
}
