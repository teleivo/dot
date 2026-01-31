// Package format provides file and directory formatting for DOT files.
package format

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

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

// File formats a single DOT file in-place.
func File(path string, ft layout.Format) error {
	src, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("error reading file: %v", err)
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("failed to open file: %v", err)
	}
	defer func() {
		_ = f.Close()
	}()

	p := printer.New(src, f, ft)
	if err := p.Print(); err != nil {
		return fmt.Errorf("%s:%s", path, err)
	}
	return nil
}

// Dir formats all DOT files (.dot, .gv) in a directory tree.
func Dir(root string, ft layout.Format) error {
	return fs.WalkDir(os.DirFS(root), ".", func(path string, d fs.DirEntry, fsErr error) error {
		if fsErr != nil {
			return fsErr
		}
		if d.IsDir() {
			return nil
		}
		if ext := filepath.Ext(d.Name()); ext != ".dot" && ext != ".gv" {
			return nil
		}

		file := filepath.Join(root, path)
		return File(file, ft)
	})
}
