// Package format provides file and directory formatting for DOT files.
package format

import (
	"errors"
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

// Dir formats all DOT files (.dot, .gv) in a directory tree.
func Dir(root string, ft layout.Format) error {
	var errs []error
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

		file := filepath.Join(root, path)
		if err := File(file, ft); err != nil {
			errs = append(errs, err)
		}
		return nil
	}); err != nil {
		return err
	}
	return errors.Join(errs...)
}

// File formats a single DOT file in-place.
func File(path string, ft layout.Format) error {
	fi, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to open file: %v", err)
	}
	src, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("error reading file: %v", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+"*")
	if err != nil {
		return fmt.Errorf("failed to create temp file for atomic rename: %v", err)
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
			return fmt.Errorf("failed to set file mode: %v", err)
		}
	}

	p := printer.New(src, tmp, ft)
	if err := p.Print(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("%s:%s", path, err)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %v", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("failed to rename temp file: %v", err)
	}

	success = true
	return nil
}
