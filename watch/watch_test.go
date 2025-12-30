package watch

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/teleivo/assertive/assert"
)

func TestHandleGenerateSuccess(t *testing.T) {
	dotFile := tempDOT(t, `digraph { a -> b }`)
	wa := newTestWatcher(t, dotFile)

	req := httptest.NewRequest(http.MethodGet, "/graph.svg", nil)
	rec := httptest.NewRecorder()

	wa.handleGenerate(rec, req)

	assert.EqualValuesf(t, rec.Code, http.StatusOK, "status code")
	assert.EqualValuesf(t, rec.Header().Get("Content-Type"), "image/svg+xml", "Content-Type")
	assert.Truef(t, strings.Contains(rec.Body.String(), "<svg"), "body should contain <svg")
}

func TestHandleGenerateInvalidDOT(t *testing.T) {
	dotFile := tempDOT(t, `digraph { A `)
	wa := newTestWatcher(t, dotFile)

	req := httptest.NewRequest(http.MethodGet, "/graph.svg", nil)
	rec := httptest.NewRecorder()

	wa.handleGenerate(rec, req)

	assert.EqualValuesf(t, rec.Code, http.StatusOK, "status code")
	assert.EqualValuesf(t, rec.Header().Get("Content-Type"), "image/svg+xml", "Content-Type")
	body := rec.Body.String()
	assert.Truef(t, strings.Contains(body, "<svg"), "body should contain <svg")
	assert.Truef(t, strings.Contains(body, "syntax error"), "body should contain syntax error")
}

func tempDOT(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.dot")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	return path
}

func newTestWatcher(t *testing.T, dotFile string) *Watcher {
	t.Helper()
	wa, err := New(Config{
		File:   dotFile,
		Port:   "0",
		Stdout: io.Discard,
		Stderr: io.Discard,
	})
	if err != nil {
		t.Fatalf("failed to create watcher: %v", err)
	}
	return wa
}
