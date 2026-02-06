package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/teleivo/assertive/assert"
)

// TestVisualOutput tests that dotx fmt preserves visual output by comparing
// plain text renderings of original and formatted DOT files.
//
// By default, it tests files in testdata/. Set DOTX_TEST_DIR to test external files.
// Temp files are preserved on failure for debugging, or always if DOTX_KEEP_TEMP=1.
func TestVisualOutput(t *testing.T) {
	if _, err := exec.LookPath("dot"); err != nil {
		t.Skip("dot (Graphviz) not found in PATH, skipping visual test")
	}

	testDir := os.Getenv("DOTX_TEST_DIR")
	if testDir == "" {
		testDir = "testdata"
	}

	if _, err := os.Stat(testDir); errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("test directory %q does not exist, skipping visual test", testDir)
	}

	dotFiles, err := filepath.Glob(filepath.Join(testDir, "*.dot"))
	if err != nil {
		t.Fatalf("failed to find .dot files in %q: %v", testDir, err)
	}

	gvFiles, err := filepath.Glob(filepath.Join(testDir, "*.gv"))
	if err != nil {
		t.Fatalf("failed to find .gv files in %q: %v", testDir, err)
	}

	dotFiles = append(dotFiles, gvFiles...)

	if len(dotFiles) == 0 {
		t.Skipf("no .dot or .gv files found in %q, skipping visual test", testDir)
	}

	keepTemp := os.Getenv("DOTX_KEEP_TEMP") == "1"
	for _, dotFile := range dotFiles {
		t.Run(filepath.Base(dotFile), func(t *testing.T) {
			t.Parallel()

			tempDir, err := os.MkdirTemp("", "dotx-visual-*")
			if err != nil {
				t.Fatalf("failed to create temp directory: %v", err)
			}

			// Clean up temp directory unless we're keeping it
			shouldCleanup := !keepTemp
			defer func() {
				if shouldCleanup {
					_ = os.RemoveAll(tempDir)
				} else if !t.Failed() {
					t.Logf("Temp files preserved at: %s", tempDir)
				}
			}()

			originalDot, err := os.ReadFile(dotFile)
			if err != nil {
				t.Fatalf("failed to read %q: %v", dotFile, err)
			}

			originalPlain, err := generatePlain(t, originalDot)
			if err != nil {
				t.Skipf("Skipping: original file fails with dot: %v", err)
			}
			originalPlainPath := filepath.Join(tempDir, "original.plain")
			if err := os.WriteFile(originalPlainPath, originalPlain, 0o644); err != nil {
				t.Fatalf("failed to write original plain output: %v", err)
			}

			formattedDot, err := formatDot(t, originalDot)
			if err != nil {
				t.Fatalf("failed to format DOT file: %v", err)
			}
			formattedDotPath := filepath.Join(tempDir, "formatted.dot")
			if err := os.WriteFile(formattedDotPath, formattedDot, 0o644); err != nil {
				t.Fatalf("failed to write formatted DOT: %v", err)
			}

			// Check idempotency: formatting the formatted output should produce identical result
			formattedDotSecond, err := formatDot(t, formattedDot)
			if err != nil {
				t.Fatalf("failed to format DOT file (second pass): %v", err)
			}
			if string(formattedDotSecond) != string(formattedDot) {
				// Preserve temp files on failure
				shouldCleanup = false
				assert.NoDiff(t, string(formattedDotSecond), string(formattedDot))
			}

			formattedPlain, err := generatePlain(t, formattedDot)
			if err != nil {
				t.Fatalf("failed to generate plain output from formatted: %v", err)
			}
			formattedPlainPath := filepath.Join(tempDir, "formatted.plain")
			if err := os.WriteFile(formattedPlainPath, formattedPlain, 0o644); err != nil {
				t.Fatalf("failed to write formatted plain output: %v", err)
			}

			if string(originalPlain) != string(formattedPlain) {
				// Preserve temp files on failure
				shouldCleanup = false
				t.Logf("plain output differs after formatting\n"+
					"  Original plain:  %s\n"+
					"  Formatted plain: %s\n"+
					"  Formatted DOT:   %s\n",
					originalPlainPath, formattedPlainPath, formattedDotPath)
				assert.EqualValues(t, string(originalPlain), string(formattedPlain))
			}
		})
	}
}

// generatePlain runs Graphviz dot to generate plain text output from DOT source
func generatePlain(t *testing.T, dotSource []byte) ([]byte, error) {
	t.Helper()

	timeout := 5 * time.Second
	if timeoutStr := os.Getenv("DOTX_FILE_TIMEOUT"); timeoutStr != "" {
		if d, err := time.ParseDuration(timeoutStr); err == nil {
			timeout = d
		}
	}

	ctx, cancel := context.WithTimeout(t.Context(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "dot", "-Tplain")
	cmd.Stdin = bytes.NewReader(dotSource)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("dot command timed out after %v (set DOTX_FILE_TIMEOUT to override)\nstderr: %s", timeout, stderr.String())
		}
		return nil, fmt.Errorf("dot command failed: %v\nstderr: %s", err, stderr.String())
	}

	return stdout.Bytes(), nil
}

// formatDot runs dotx fmt on DOT source and returns the formatted output
func formatDot(t *testing.T, dotSource []byte) ([]byte, error) {
	t.Helper()

	timeout := 5 * time.Second
	if timeoutStr := os.Getenv("DOTX_FILE_TIMEOUT"); timeoutStr != "" {
		if d, err := time.ParseDuration(timeoutStr); err == nil {
			timeout = d
		}
	}

	ctx, cancel := context.WithTimeout(t.Context(), timeout)
	defer cancel()

	var stdout, stderr bytes.Buffer

	// run dotx fmt directly in-process
	done := make(chan error, 1)
	go func() {
		_, err := run([]string{"dotx", "fmt"}, bytes.NewReader(dotSource), &stdout, &stderr)
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			return nil, fmt.Errorf("dotx fmt failed: %v\nstderr: %s", err, stderr.String())
		}
		return stdout.Bytes(), nil
	case <-ctx.Done():
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("dotx fmt timed out after %v (set DOTX_FILE_TIMEOUT to override)\nstderr: %s", timeout, stderr.String())
		}
		return nil, fmt.Errorf("dotx fmt failed: %v\nstderr: %s", ctx.Err(), stderr.String())
	}
}
