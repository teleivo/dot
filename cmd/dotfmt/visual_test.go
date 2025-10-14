package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestVisualOutput tests that dotfmt preserves visual output by comparing
// SVG renderings of original and formatted DOT files.
//
// By default, it tests files in testdata/. Set DOTFMT_TEST_DIR to test external files.
// Temp files are preserved on failure for debugging, or always if DOTFMT_KEEP_TEMP=1.
func TestVisualOutput(t *testing.T) {
	if _, err := exec.LookPath("dot"); err != nil {
		t.Skip("dot (Graphviz) not found in PATH, skipping visual test")
	}

	testDir := os.Getenv("DOTFMT_TEST_DIR")
	if testDir == "" {
		testDir = "testdata"
	}

	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		t.Skipf("test directory %q does not exist, skipping visual test", testDir)
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

	keepTemp := os.Getenv("DOTFMT_KEEP_TEMP") == "1"
	for _, dotFile := range dotFiles {
		t.Run(filepath.Base(dotFile), func(t *testing.T) {
			t.Parallel()

			tempDir, err := os.MkdirTemp("", "dotfmt-visual-*")
			if err != nil {
				t.Fatalf("failed to create temp directory: %v", err)
			}

			// Clean up temp directory unless we're keeping it
			shouldCleanup := !keepTemp
			defer func() {
				if shouldCleanup {
					os.RemoveAll(tempDir)
				} else if !t.Failed() {
					t.Logf("Temp files preserved at: %s", tempDir)
				}
			}()

			originalDot, err := os.ReadFile(dotFile)
			if err != nil {
				t.Fatalf("failed to read %q: %v", dotFile, err)
			}

			originalSVG, err := generateSVG(originalDot)
			if err != nil {
				t.Fatalf("failed to generate SVG from original: %v", err)
			}
			originalSVGPath := filepath.Join(tempDir, "original.svg")
			if err := os.WriteFile(originalSVGPath, originalSVG, 0o644); err != nil {
				t.Fatalf("failed to write original SVG: %v", err)
			}

			formattedDot, err := formatDot(originalDot)
			if err != nil {
				t.Fatalf("failed to format DOT file: %v", err)
			}
			formattedDotPath := filepath.Join(tempDir, "formatted.dot")
			if err := os.WriteFile(formattedDotPath, formattedDot, 0o644); err != nil {
				t.Fatalf("failed to write formatted DOT: %v", err)
			}

			formattedSVG, err := generateSVG(formattedDot)
			if err != nil {
				t.Fatalf("failed to generate SVG from formatted: %v", err)
			}
			formattedSVGPath := filepath.Join(tempDir, "formatted.svg")
			if err := os.WriteFile(formattedSVGPath, formattedSVG, 0o644); err != nil {
				t.Fatalf("failed to write formatted SVG: %v", err)
			}

			originalHash := sha256.Sum256(originalSVG)
			formattedHash := sha256.Sum256(formattedSVG)
			if originalHash != formattedHash {
				// Preserve temp files on failure
				shouldCleanup = false
				t.Errorf("SVG output differs after formatting\n"+
					"  Original SVG:  %s\n"+
					"  Formatted SVG: %s\n"+
					"  Formatted DOT: %s\n"+
					"  Original hash:  %x\n"+
					"  Formatted hash: %x",
					originalSVGPath, formattedSVGPath, formattedDotPath,
					originalHash, formattedHash)
			}
		})
	}
}

// generateSVG runs Graphviz dot to generate SVG from DOT source
func generateSVG(dotSource []byte) ([]byte, error) {
	cmd := exec.Command("dot", "-Tsvg")
	cmd.Stdin = bytes.NewReader(dotSource)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("dot command failed: %v\nstderr: %s", err, stderr.String())
	}

	return stdout.Bytes(), nil
}

// formatDot runs dotfmt on DOT source and returns the formatted output
func formatDot(dotSource []byte) ([]byte, error) {
	var stdout, stderr bytes.Buffer

	if err := run([]string{"dotfmt"}, bytes.NewReader(dotSource), &stdout, &stderr); err != nil {
		return nil, fmt.Errorf("dotfmt failed: %v\nstderr: %s", err, stderr.String())
	}

	return stdout.Bytes(), nil
}
