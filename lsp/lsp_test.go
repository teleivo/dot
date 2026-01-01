package lsp

import (
	"errors"
	"fmt"
	"io"
	"testing"
	"testing/iotest"

	"github.com/teleivo/assertive/assert"
	"github.com/teleivo/assertive/require"
	"github.com/teleivo/dot"
	"github.com/teleivo/dot/lsp/internal/rpc"
	"github.com/teleivo/dot/token"
)

func TestServer(t *testing.T) {
	// Per LSP 3.17 spec: "If the server receives a request or notification before the
	// `initialize` request it should act as follows:
	// - For a request the response should be an error with `code: -32002`
	//   (ServerNotInitialized). The message can be picked by the server.
	// - Notifications should be dropped, except for the exit notification."
	// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#initialize
	t.Run("ShutdownBeforeInitialize", func(t *testing.T) {
		s, in := setup(t)

		// Send shutdown request before initialize
		msg := `{"jsonrpc":"2.0","method":"shutdown","id":3}`
		writeMessage(t, in, msg)

		// Server must respond with ServerNotInitialized error (shutdown is a request)
		want := `{"jsonrpc":"2.0","id":3,"error":{"code":-32002,"message":"server not initialized"}}`
		assert.Truef(t, s.Scan(), "expecting response from server")
		require.EqualValuesf(t, s.Text(), want, "unexpected response")
	})

	// Per LSP 3.17 spec: "The initialize request may only be sent once."
	// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#initialize
	t.Run("InitializeTwice", func(t *testing.T) {
		s, in := setup(t)

		initMsg := `{"jsonrpc":"2.0","method":"initialize","id":1,"params":{}}`
		writeMessage(t, in, initMsg)

		// First initialize should succeed
		assert.Truef(t, s.Scan(), "expecting first initialize response")

		// Send initialize again
		initMsg2 := `{"jsonrpc":"2.0","method":"initialize","id":2,"params":{}}`
		writeMessage(t, in, initMsg2)

		// Second initialize should fail with InvalidRequest error (-32600)
		want := `{"jsonrpc":"2.0","id":2,"error":{"code":-32600,"message":"server already initialized"}}`
		assert.Truef(t, s.Scan(), "expecting error response for second initialize")
		require.EqualValuesf(t, s.Text(), want, "unexpected response")
	})

	t.Run("ShutdownAfterInitialize", func(t *testing.T) {
		s, in := setup(t)

		// Step 1: initialize request
		initMsg := `{"jsonrpc":"2.0","method":"initialize","id":1,"params":{}}`
		writeMessage(t, in, initMsg)

		// Server responds with capabilities:
		// - completionProvider: completion with trigger characters
		// - positionEncoding: "utf-8" (negotiated encoding for character offsets)
		// - textDocumentSync: 2 (TextDocumentSyncKind.Incremental)
		// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#initializeResult
		wantInit := `{"jsonrpc":"2.0","id":1,"result":{"capabilities":{"completionProvider":{"triggerCharacters":["[",",",";","{","="]},"documentFormattingProvider":true,"positionEncoding":"utf-8","textDocumentSync":2},"serverInfo":{"name":"dotls","version":"(devel)"}}}`
		assert.Truef(t, s.Scan(), "expecting initialize response")
		require.EqualValuesf(t, s.Text(), wantInit, "unexpected initialize response")

		// Step 2: initialized notification (no id, no response expected)
		initializedMsg := `{"jsonrpc":"2.0","method":"initialized","params":{}}`
		writeMessage(t, in, initializedMsg)

		// Step 3: shutdown request - server should respond but NOT exit yet
		// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#shutdown
		shutdownMsg := `{"jsonrpc":"2.0","method":"shutdown","id":2}`
		writeMessage(t, in, shutdownMsg)

		want := `{"jsonrpc":"2.0","id":2,"result":null}`
		assert.Truef(t, s.Scan(), "expecting shutdown response")
		require.EqualValuesf(t, s.Text(), want, "unexpected response")

		// Step 4: requests after shutdown should error with InvalidRequest (-32600)
		// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#shutdown
		postShutdownMsg := `{"jsonrpc":"2.0","method":"textDocument/hover","id":3,"params":{"textDocument":{"uri":"file:///test.dot"},"position":{"line":0,"character":0}}}`
		writeMessage(t, in, postShutdownMsg)

		wantErr := `{"jsonrpc":"2.0","id":3,"error":{"code":-32600,"message":"server is shutting down"}}`
		assert.Truef(t, s.Scan(), "expecting error response after shutdown")
		require.EqualValuesf(t, s.Text(), wantErr, "unexpected response after shutdown")

		// Step 5: exit notification - server should exit (no response expected)
		// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#exit
		exitMsg := `{"jsonrpc":"2.0","method":"exit"}`
		writeMessage(t, in, exitMsg)
	})

	// Per JSON-RPC 2.0 spec: invalid JSON should return ParseError (-32700) with id null.
	// The server should continue processing subsequent messages.
	t.Run("ParseError", func(t *testing.T) {
		s, in := setup(t)

		// Send invalid JSON
		writeMessage(t, in, `{not valid json}`)

		// Server should respond with ParseError (id omitted when unknown)
		want := `{"jsonrpc":"2.0","error":{"code":-32700,"message":"invalid JSON"}}`
		assert.Truef(t, s.Scan(), "expecting ParseError response")
		require.EqualValuesf(t, s.Text(), want, "unexpected response")

		// Server should still be alive - send valid initialize
		initMsg := `{"jsonrpc":"2.0","method":"initialize","id":1,"params":{}}`
		writeMessage(t, in, initMsg)

		assert.Truef(t, s.Scan(), "expecting initialize response after parse error")
	})

	// Per JSON-RPC 2.0 and LSP spec: unknown request methods should return MethodNotFound (-32601).
	// Unknown notifications are silently dropped (no response since notifications have no id).
	t.Run("MethodNotFound", func(t *testing.T) {
		s, in := setup(t)

		// Initialize handshake
		initMsg := `{"jsonrpc":"2.0","method":"initialize","id":1,"params":{}}`
		writeMessage(t, in, initMsg)
		assert.Truef(t, s.Scan(), "expecting initialize response")

		initializedMsg := `{"jsonrpc":"2.0","method":"initialized","params":{}}`
		writeMessage(t, in, initializedMsg)

		// Send an unknown notification - should be silently ignored (no response)
		unknownNotification := `{"jsonrpc":"2.0","method":"$/unknownNotification","params":{}}`
		writeMessage(t, in, unknownNotification)

		// Send an unknown request method - should get MethodNotFound error
		unknownRequest := `{"jsonrpc":"2.0","method":"textDocument/unknown","id":2,"params":{}}`
		writeMessage(t, in, unknownRequest)

		// Only the request should get a response; notification was dropped
		want := `{"jsonrpc":"2.0","id":2,"error":{"code":-32601,"message":"method not found"}}`
		assert.Truef(t, s.Scan(), "expecting MethodNotFound error response")
		require.EqualValuesf(t, s.Text(), want, "unexpected response")
	})

	// textDocument/didOpen and textDocument/didChange trigger diagnostics via
	// textDocument/publishDiagnostics notification.
	// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_publishDiagnostics
	t.Run("PublishDiagnostics", func(t *testing.T) {
		s, in := setup(t)

		// Initialize handshake - server advertises:
		// - completionProvider: completion with trigger characters
		// - positionEncoding: "utf-8" (character offsets count bytes)
		// - textDocumentSync: 2 (TextDocumentSyncKind.Incremental)
		initMsg := `{"jsonrpc":"2.0","method":"initialize","id":1,"params":{}}`
		writeMessage(t, in, initMsg)
		wantInit := `{"jsonrpc":"2.0","id":1,"result":{"capabilities":{"completionProvider":{"triggerCharacters":["[",",",";","{","="]},"documentFormattingProvider":true,"positionEncoding":"utf-8","textDocumentSync":2},"serverInfo":{"name":"dotls","version":"(devel)"}}}`
		assert.Truef(t, s.Scan(), "expecting initialize response")
		require.EqualValuesf(t, s.Text(), wantInit, "unexpected initialize response")

		initializedMsg := `{"jsonrpc":"2.0","method":"initialized","params":{}}`
		writeMessage(t, in, initializedMsg)

		// Open a document with 2 parse errors:
		// Line 2: "a [label=]" - missing attribute value
		// Line 3: "b ->" - missing edge target
		//
		// Document content (with actual newlines for clarity):
		// digraph {
		//   a [label=]
		//   b ->
		// }
		//
		// Parser reports (1-based): 2:12 and 4:1
		// LSP positions are 0-based: line 1 char 11, line 3 char 0
		firstDocContent := `digraph {\n  a [label=]\n  b ->\n}`
		didOpenFirst := `{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"file:///first.dot","languageId":"dot","version":1,"text":"` + firstDocContent + `"}}}`
		writeMessage(t, in, didOpenFirst)

		// Server sends publishDiagnostics notification (no id field)
		// Diagnostics use point ranges (start == end) for error positions
		// Severity 1 = Error
		// Version matches the document version from didOpen (version: 1)
		want := `{"jsonrpc":"2.0","method":"textDocument/publishDiagnostics","params":{"uri":"file:///first.dot","version":1,"diagnostics":[{"range":{"start":{"line":1,"character":11},"end":{"line":1,"character":11}},"severity":1,"message":"expected attribute value"},{"range":{"start":{"line":3,"character":0},"end":{"line":3,"character":0}},"severity":1,"message":"expected node or subgraph as edge operand"}]}}`
		assert.Truef(t, s.Scan(), "expecting publishDiagnostics notification for didOpen")
		require.EqualValuesf(t, s.Text(), want, "unexpected diagnostics for didOpen")

		// Fix the first error using incremental sync (TextDocumentSyncKind.Incremental = 2)
		// Replace "label=" with "label=\"hello\"" on line 2 (0-based: line 1)
		// Range: start {line: 1, character: 5} to end {line: 1, character: 11}
		// This changes "a [label=]" to "a [label=\"hello\"]"
		didChangeMsg1 := `{"jsonrpc":"2.0","method":"textDocument/didChange","params":{"textDocument":{"uri":"file:///first.dot","version":2},"contentChanges":[{"range":{"start":{"line":1,"character":5},"end":{"line":1,"character":11}},"text":"label=\"hello\""}]}}`
		writeMessage(t, in, didChangeMsg1)

		// Now only one error remains: the missing edge target on line 3
		// Version matches the document version from didChange (version: 2)
		wantOneError := `{"jsonrpc":"2.0","method":"textDocument/publishDiagnostics","params":{"uri":"file:///first.dot","version":2,"diagnostics":[{"range":{"start":{"line":3,"character":0},"end":{"line":3,"character":0}},"severity":1,"message":"expected node or subgraph as edge operand"}]}}`
		assert.Truef(t, s.Scan(), "expecting publishDiagnostics after first incremental change")
		require.EqualValuesf(t, s.Text(), wantOneError, "unexpected diagnostics after first fix")

		// Open a second document (valid DOT, no errors)
		secondDocContent := `graph { x -- y }`
		didOpenSecond := `{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"file:///second.dot","languageId":"dot","version":1,"text":"` + secondDocContent + `"}}}`
		writeMessage(t, in, didOpenSecond)

		// Version matches the document version from didOpen (version: 1)
		wantSecondEmpty := `{"jsonrpc":"2.0","method":"textDocument/publishDiagnostics","params":{"uri":"file:///second.dot","version":1,"diagnostics":[]}}`
		assert.Truef(t, s.Scan(), "expecting publishDiagnostics for second document")
		require.EqualValuesf(t, s.Text(), wantSecondEmpty, "unexpected diagnostics for second document")

		// Fix the second error in first document
		// Replace "b ->" with "b -> c" on line 3 (0-based: line 2)
		// Range: start {line: 2, character: 2} to end {line: 2, character: 6}
		didChangeMsg2 := `{"jsonrpc":"2.0","method":"textDocument/didChange","params":{"textDocument":{"uri":"file:///first.dot","version":3},"contentChanges":[{"range":{"start":{"line":2,"character":2},"end":{"line":2,"character":6}},"text":"b -> c"}]}}`
		writeMessage(t, in, didChangeMsg2)

		// Version matches the document version from didChange (version: 3)
		wantFirstEmpty := `{"jsonrpc":"2.0","method":"textDocument/publishDiagnostics","params":{"uri":"file:///first.dot","version":3,"diagnostics":[]}}`
		assert.Truef(t, s.Scan(), "expecting publishDiagnostics after second incremental change")
		require.EqualValuesf(t, s.Text(), wantFirstEmpty, "unexpected diagnostics after all fixes")

		// Change second document: insert " -- z" at end before "}"
		// "graph { x -- y }" -> "graph { x -- y -- z }"
		didChangeSecond := `{"jsonrpc":"2.0","method":"textDocument/didChange","params":{"textDocument":{"uri":"file:///second.dot","version":2},"contentChanges":[{"range":{"start":{"line":0,"character":14},"end":{"line":0,"character":14}},"text":" -- z"}]}}`
		writeMessage(t, in, didChangeSecond)

		// Version matches the document version from didChange (version: 2)
		wantSecondEmptyV2 := `{"jsonrpc":"2.0","method":"textDocument/publishDiagnostics","params":{"uri":"file:///second.dot","version":2,"diagnostics":[]}}`
		assert.Truef(t, s.Scan(), "expecting publishDiagnostics after changing second document")
		require.EqualValuesf(t, s.Text(), wantSecondEmptyV2, "unexpected diagnostics after changing second document")

		// Close second document
		didCloseSecond := `{"jsonrpc":"2.0","method":"textDocument/didClose","params":{"textDocument":{"uri":"file:///second.dot"}}}`
		writeMessage(t, in, didCloseSecond)

		// Send didSave and didClose for first document
		didSaveMsg := `{"jsonrpc":"2.0","method":"textDocument/didSave","params":{"textDocument":{"uri":"file:///first.dot"}}}`
		writeMessage(t, in, didSaveMsg)

		didCloseFirst := `{"jsonrpc":"2.0","method":"textDocument/didClose","params":{"textDocument":{"uri":"file:///first.dot"}}}`
		writeMessage(t, in, didCloseFirst)

		// Verify server is still responsive after closing both documents
		shutdownMsg := `{"jsonrpc":"2.0","method":"shutdown","id":2}`
		writeMessage(t, in, shutdownMsg)

		wantShutdown := `{"jsonrpc":"2.0","id":2,"result":null}`
		assert.Truef(t, s.Scan(), "expecting shutdown response")
		require.EqualValuesf(t, s.Text(), wantShutdown, "unexpected shutdown response")
	})

	// textDocument/completion returns attribute completions filtered by prefix.
	// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_completion
	t.Run("Completion", func(t *testing.T) {
		s, in := setup(t)

		// Initialize handshake
		initMsg := `{"jsonrpc":"2.0","method":"initialize","id":1,"params":{}}`
		writeMessage(t, in, initMsg)
		assert.Truef(t, s.Scan(), "expecting initialize response")

		initializedMsg := `{"jsonrpc":"2.0","method":"initialized","params":{}}`
		writeMessage(t, in, initializedMsg)

		// Open a document with a node that has an attribute list
		// Document: digraph { a [lab] }
		// The cursor will be positioned after "lab" to test prefix filtering
		docContent := `digraph { a [lab] }`
		didOpen := `{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"file:///test.dot","languageId":"dot","version":1,"text":"` + docContent + `"}}}`
		writeMessage(t, in, didOpen)

		// Receive diagnostics (document is valid)
		assert.Truef(t, s.Scan(), "expecting publishDiagnostics")

		// Request completion at position after "lab" (line 0, character 16)
		// This should return all attributes starting with "lab": label, labelangle, etc.
		completionReq := `{"jsonrpc":"2.0","method":"textDocument/completion","id":2,"params":{"textDocument":{"uri":"file:///test.dot"},"position":{"line":0,"character":16}}}`
		writeMessage(t, in, completionReq)

		// Expect completions filtered to attributes starting with "lab"
		// All "lab*" attributes are edge-only except: label (G,C,N,E), labelloc (G,C,N)
		// Since we're in a node context [lab], we expect node-applicable attributes
		wantCompletion := `{"jsonrpc":"2.0","id":2,"result":{"isIncomplete":false,"items":[{"label":"label","kind":10,"detail":"Graph, Cluster, Node, Edge","documentation":"Text label attached to objects","insertText":"label="},{"label":"labelloc","kind":10,"detail":"Graph, Cluster, Node","documentation":"Vertical placement of labels","insertText":"labelloc="}]}}`
		assert.Truef(t, s.Scan(), "expecting completion response")
		require.EqualValuesf(t, s.Text(), wantCompletion, "unexpected completion response")

		// Now test narrowing: user continues typing "label"
		// Change document: "digraph { a [lab] }" -> "digraph { a [label] }"
		didChange := `{"jsonrpc":"2.0","method":"textDocument/didChange","params":{"textDocument":{"uri":"file:///test.dot","version":2},"contentChanges":[{"range":{"start":{"line":0,"character":16},"end":{"line":0,"character":16}},"text":"el"}]}}`
		writeMessage(t, in, didChange)

		// Receive diagnostics
		assert.Truef(t, s.Scan(), "expecting publishDiagnostics after change")

		// Request completion at position after "label" (line 0, character 18)
		completionReq2 := `{"jsonrpc":"2.0","method":"textDocument/completion","id":3,"params":{"textDocument":{"uri":"file:///test.dot"},"position":{"line":0,"character":18}}}`
		writeMessage(t, in, completionReq2)

		// Now only "label" and "labelloc" match (both apply to nodes)
		wantCompletion2 := `{"jsonrpc":"2.0","id":3,"result":{"isIncomplete":false,"items":[{"label":"label","kind":10,"detail":"Graph, Cluster, Node, Edge","documentation":"Text label attached to objects","insertText":"label="},{"label":"labelloc","kind":10,"detail":"Graph, Cluster, Node","documentation":"Vertical placement of labels","insertText":"labelloc="}]}}`
		assert.Truef(t, s.Scan(), "expecting narrowed completion response")
		require.EqualValuesf(t, s.Text(), wantCompletion2, "unexpected narrowed completion response")

		// Test edge context: completions should include edge-specific attributes
		// Change document to have an edge with attributes
		// "digraph { a [label] }" -> "digraph { a -> b [arr] }"
		didChange2 := `{"jsonrpc":"2.0","method":"textDocument/didChange","params":{"textDocument":{"uri":"file:///test.dot","version":3},"contentChanges":[{"range":{"start":{"line":0,"character":10},"end":{"line":0,"character":20}},"text":"a -> b [arr"}]}}`
		writeMessage(t, in, didChange2)

		// Receive diagnostics
		assert.Truef(t, s.Scan(), "expecting publishDiagnostics after edge change")

		// Request completion at position after "arr" (line 0, character 21)
		completionReq3 := `{"jsonrpc":"2.0","method":"textDocument/completion","id":4,"params":{"textDocument":{"uri":"file:///test.dot"},"position":{"line":0,"character":21}}}`
		writeMessage(t, in, completionReq3)

		// Expect edge attributes starting with "arr": arrowhead, arrowsize, arrowtail
		wantCompletion3 := `{"jsonrpc":"2.0","id":4,"result":{"isIncomplete":false,"items":[{"label":"arrowhead","kind":10,"detail":"Edge","documentation":"Style of arrowhead on edge head node","insertText":"arrowhead="},{"label":"arrowsize","kind":10,"detail":"Edge","documentation":"Multiplicative scale factor for arrowheads","insertText":"arrowsize="},{"label":"arrowtail","kind":10,"detail":"Edge","documentation":"Style of arrowhead on edge tail node","insertText":"arrowtail="}]}}`
		assert.Truef(t, s.Scan(), "expecting edge completion response")
		require.EqualValuesf(t, s.Text(), wantCompletion3, "unexpected edge completion response")
	})

	t.Run("Formatting", func(t *testing.T) {
		s, in := setup(t)

		// Initialize
		initMsg := `{"jsonrpc":"2.0","method":"initialize","id":1,"params":{}}`
		writeMessage(t, in, initMsg)
		assert.Truef(t, s.Scan(), "expecting initialize response")

		initializedMsg := `{"jsonrpc":"2.0","method":"initialized","params":{}}`
		writeMessage(t, in, initializedMsg)

		// Open document with parse error
		invalidContent := `digraph { a -> }`
		didOpen := `{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"file:///test.dot","languageId":"dot","version":1,"text":"` + invalidContent + `"}}}`
		writeMessage(t, in, didOpen)

		// Receive diagnostics for invalid document
		assert.Truef(t, s.Scan(), "expecting publishDiagnostics for invalid document")

		// Format invalid document - should return error
		formatInvalid := `{"jsonrpc":"2.0","method":"textDocument/formatting","id":2,"params":{"textDocument":{"uri":"file:///test.dot"},"options":{"tabSize":2,"insertSpaces":false}}}`
		writeMessage(t, in, formatInvalid)

		wantError := `{"jsonrpc":"2.0","id":2,"error":{"code":-32603,"message":"formatting failed: 1:16: expected node or subgraph as edge operand"}}`
		assert.Truef(t, s.Scan(), "expecting error response for invalid document")
		require.EqualValuesf(t, s.Text(), wantError, "unexpected error response")

		// Fix document: "digraph { a -> }" -> "digraph { a -> b }"
		didChange := `{"jsonrpc":"2.0","method":"textDocument/didChange","params":{"textDocument":{"uri":"file:///test.dot","version":2},"contentChanges":[{"range":{"start":{"line":0,"character":14},"end":{"line":0,"character":14}},"text":"b "}]}}`
		writeMessage(t, in, didChange)

		// Receive diagnostics for fixed document
		assert.Truef(t, s.Scan(), "expecting publishDiagnostics for fixed document")

		// Format valid document - should succeed
		formatValid := `{"jsonrpc":"2.0","method":"textDocument/formatting","id":3,"params":{"textDocument":{"uri":"file:///test.dot"},"options":{"tabSize":2,"insertSpaces":false}}}`
		writeMessage(t, in, formatValid)

		wantFormatting := `{"jsonrpc":"2.0","id":3,"result":[{"range":{"start":{"line":0,"character":0},"end":{"line":0,"character":18}},"newText":"digraph {\n\ta -\u003e b\n}"}]}`
		assert.Truef(t, s.Scan(), "expecting formatting response")
		require.EqualValuesf(t, s.Text(), wantFormatting, "unexpected formatting response")
	})
}

func TestStartReturnsReaderError(t *testing.T) {
	readErr := errors.New("input/output error")
	srv, err := New(Config{
		In:  iotest.ErrReader(readErr),
		Out: io.Discard,
	})
	require.NoError(t, err)

	err = srv.Start(t.Context())

	require.Truef(t, errors.Is(err, readErr), "want %v, got %v", readErr, err)
}

func setup(t *testing.T) (*rpc.Scanner, io.Writer) {
	t.Helper()

	inR, inW := io.Pipe()
	outR, outW := io.Pipe()

	srv, err := New(Config{
		In:  inR,
		Out: outW,
		Log: io.Discard,
	})
	require.NoErrorf(t, err, "want no errors creating server")
	go func() {
		require.NoError(t, srv.Start(t.Context()))
	}()

	t.Cleanup(func() {
		require.NoErrorf(t, inW.Close(), "failed to close inW")
		require.NoErrorf(t, outW.Close(), "failed to close outW")
	})

	return rpc.NewScanner(outR), inW
}

func writeMessage(t *testing.T, w io.Writer, content string) {
	t.Helper()
	write(t, w, "Content-Length:  %d \r\n", len(content))
	write(t, w, "\r\n")
	write(t, w, "%s", content)
}

func write(t *testing.T, w io.Writer, format string, args ...any) {
	t.Helper()
	_, err := fmt.Fprintf(w, format, args...)
	require.NoErrorf(t, err, "failed to write message")
}

func TestDocumentChange(t *testing.T) {
	type test struct {
		initial string
		changes []rpc.TextDocumentContentChangeEvent
		want    string
	}

	tests := map[string]test{
		"InsertAtStart": {
			initial: "hello",
			changes: []rpc.TextDocumentContentChangeEvent{
				{Range: &rpc.Range{Start: rpc.Position{Line: 0, Character: 0}, End: rpc.Position{Line: 0, Character: 0}}, Text: "say "},
			},
			want: "say hello",
		},
		"InsertInMiddle": {
			initial: "helo",
			changes: []rpc.TextDocumentContentChangeEvent{
				{Range: &rpc.Range{Start: rpc.Position{Line: 0, Character: 2}, End: rpc.Position{Line: 0, Character: 2}}, Text: "l"},
			},
			want: "hello",
		},
		"InsertAtEnd": {
			initial: "hello",
			changes: []rpc.TextDocumentContentChangeEvent{
				{Range: &rpc.Range{Start: rpc.Position{Line: 0, Character: 5}, End: rpc.Position{Line: 0, Character: 5}}, Text: " world"},
			},
			want: "hello world",
		},
		"DeleteAtStart": {
			initial: "say hello",
			changes: []rpc.TextDocumentContentChangeEvent{
				{Range: &rpc.Range{Start: rpc.Position{Line: 0, Character: 0}, End: rpc.Position{Line: 0, Character: 4}}, Text: ""},
			},
			want: "hello",
		},
		"DeleteInMiddle": {
			initial: "helllo",
			changes: []rpc.TextDocumentContentChangeEvent{
				{Range: &rpc.Range{Start: rpc.Position{Line: 0, Character: 3}, End: rpc.Position{Line: 0, Character: 4}}, Text: ""},
			},
			want: "hello",
		},
		"DeleteAtEnd": {
			initial: "hello world",
			changes: []rpc.TextDocumentContentChangeEvent{
				{Range: &rpc.Range{Start: rpc.Position{Line: 0, Character: 5}, End: rpc.Position{Line: 0, Character: 11}}, Text: ""},
			},
			want: "hello",
		},
		"ReplaceShorter": {
			initial: "hello world",
			changes: []rpc.TextDocumentContentChangeEvent{
				{Range: &rpc.Range{Start: rpc.Position{Line: 0, Character: 6}, End: rpc.Position{Line: 0, Character: 11}}, Text: "go"},
			},
			want: "hello go",
		},
		"ReplaceLonger": {
			initial: "hello go",
			changes: []rpc.TextDocumentContentChangeEvent{
				{Range: &rpc.Range{Start: rpc.Position{Line: 0, Character: 6}, End: rpc.Position{Line: 0, Character: 8}}, Text: "world"},
			},
			want: "hello world",
		},
		"ReplaceSameLength": {
			initial: "hello world",
			changes: []rpc.TextDocumentContentChangeEvent{
				{Range: &rpc.Range{Start: rpc.Position{Line: 0, Character: 6}, End: rpc.Position{Line: 0, Character: 11}}, Text: "there"},
			},
			want: "hello there",
		},
		"InsertNewline": {
			initial: "ab",
			changes: []rpc.TextDocumentContentChangeEvent{
				{Range: &rpc.Range{Start: rpc.Position{Line: 0, Character: 1}, End: rpc.Position{Line: 0, Character: 1}}, Text: "\n"},
			},
			want: "a\nb",
		},
		"DeleteAcrossLines": {
			initial: "hello\nworld",
			changes: []rpc.TextDocumentContentChangeEvent{
				{Range: &rpc.Range{Start: rpc.Position{Line: 0, Character: 3}, End: rpc.Position{Line: 1, Character: 2}}, Text: ""},
			},
			want: "helrld",
		},
		"ReplaceAcrossLines": {
			initial: "aaa\nbbb\nccc",
			changes: []rpc.TextDocumentContentChangeEvent{
				{Range: &rpc.Range{Start: rpc.Position{Line: 0, Character: 1}, End: rpc.Position{Line: 2, Character: 2}}, Text: "X"},
			},
			want: "aXc",
		},
		"InsertMultipleLines": {
			initial: "ac",
			changes: []rpc.TextDocumentContentChangeEvent{
				{Range: &rpc.Range{Start: rpc.Position{Line: 0, Character: 1}, End: rpc.Position{Line: 0, Character: 1}}, Text: "\nb\n"},
			},
			want: "a\nb\nc",
		},
		"EditOnSecondLine": {
			initial: "first\nsecond",
			changes: []rpc.TextDocumentContentChangeEvent{
				{Range: &rpc.Range{Start: rpc.Position{Line: 1, Character: 0}, End: rpc.Position{Line: 1, Character: 6}}, Text: "2nd"},
			},
			want: "first\n2nd",
		},
		"EditOnThirdLine": {
			initial: "a\nb\nc",
			changes: []rpc.TextDocumentContentChangeEvent{
				{Range: &rpc.Range{Start: rpc.Position{Line: 2, Character: 0}, End: rpc.Position{Line: 2, Character: 1}}, Text: "C"},
			},
			want: "a\nb\nC",
		},
		"ChainedInserts": {
			initial: "ac",
			changes: []rpc.TextDocumentContentChangeEvent{
				{Range: &rpc.Range{Start: rpc.Position{Line: 0, Character: 1}, End: rpc.Position{Line: 0, Character: 1}}, Text: "b"},
				{Range: &rpc.Range{Start: rpc.Position{Line: 0, Character: 3}, End: rpc.Position{Line: 0, Character: 3}}, Text: "d"},
			},
			want: "abcd",
		},
		"ChainedDeleteThenInsert": {
			initial: "hello world",
			changes: []rpc.TextDocumentContentChangeEvent{
				{Range: &rpc.Range{Start: rpc.Position{Line: 0, Character: 5}, End: rpc.Position{Line: 0, Character: 11}}, Text: ""},
				{Range: &rpc.Range{Start: rpc.Position{Line: 0, Character: 5}, End: rpc.Position{Line: 0, Character: 5}}, Text: "!"},
			},
			want: "hello!",
		},
		"EmptyDocumentInsert": {
			initial: "",
			changes: []rpc.TextDocumentContentChangeEvent{
				{Range: &rpc.Range{Start: rpc.Position{Line: 0, Character: 0}, End: rpc.Position{Line: 0, Character: 0}}, Text: "hello"},
			},
			want: "hello",
		},
		"DeleteEntireDocument": {
			initial: "hello",
			changes: []rpc.TextDocumentContentChangeEvent{
				{Range: &rpc.Range{Start: rpc.Position{Line: 0, Character: 0}, End: rpc.Position{Line: 0, Character: 5}}, Text: ""},
			},
			want: "",
		},
		"DeleteEntireMultiLineDocument": {
			initial: "a\nb\nc",
			changes: []rpc.TextDocumentContentChangeEvent{
				{Range: &rpc.Range{Start: rpc.Position{Line: 0, Character: 0}, End: rpc.Position{Line: 2, Character: 1}}, Text: ""},
			},
			want: "",
		},
		"InsertAtEndOfLineBeforeNewline": {
			initial: "a\nb",
			changes: []rpc.TextDocumentContentChangeEvent{
				{Range: &rpc.Range{Start: rpc.Position{Line: 0, Character: 1}, End: rpc.Position{Line: 0, Character: 1}}, Text: "X"},
			},
			want: "aX\nb",
		},
		"DeleteNewlineJoiningLines": {
			initial: "hello\nworld",
			changes: []rpc.TextDocumentContentChangeEvent{
				{Range: &rpc.Range{Start: rpc.Position{Line: 0, Character: 5}, End: rpc.Position{Line: 1, Character: 0}}, Text: ""},
			},
			want: "helloworld",
		},
		"ReplaceNewlineWithSpace": {
			initial: "hello\nworld",
			changes: []rpc.TextDocumentContentChangeEvent{
				{Range: &rpc.Range{Start: rpc.Position{Line: 0, Character: 5}, End: rpc.Position{Line: 1, Character: 0}}, Text: " "},
			},
			want: "hello world",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			doc := newDocument(rpc.TextDocumentItem{Text: tt.initial})

			for _, change := range tt.changes {
				err := doc.change(change)
				require.NoErrorf(t, err, "unexpected error applying change")
			}

			assert.EqualValuesf(t, string(doc.src), tt.want, "unexpected document content")
		})
	}
}

func TestCompletionContext(t *testing.T) {
	tests := map[string]struct {
		src          string
		position     token.Position // 1-based line and column
		wantPrefix   string
		wantAttrCtx  attributeContext
		wantAttrName string // empty means completing name, non-empty means completing value
	}{
		// === Attribute name completion ===

		// Cursor inside node's attr_list after typing "lab"
		// Input: `graph { A [lab] }`
		//                       ^-- cursor at line 1, col 15 (after "lab")
		// Tree structure:
		//   NodeStmt > AttrList > AList > Attribute > ID > 'lab'
		"NodeAttrListPartialAttribute": {
			src:         `graph { A [lab] }`,
			position:    token.Position{Line: 1, Column: 15},
			wantPrefix:  "lab",
			wantAttrCtx: Node,
		},
		// Multi-line: cursor on line 2 should still be inside a node that starts on line 1
		// Input:
		//   graph {
		//     A [lab]
		//   }
		// Tree: 'lab' (@ 2 6 2 8), ']' (@ 2 9 2 9)
		// Cursor at line 2, col 9 (after "lab", on "]")
		"MultiLineNodeAttr": {
			src:         "graph {\n  A [lab]\n}",
			position:    token.Position{Line: 2, Column: 9},
			wantPrefix:  "lab",
			wantAttrCtx: Node,
		},
		// Bug case: naive column check fails when pos.Line > start.Line but pos.Column < start.Column
		// Input (note leading spaces on line 1):
		//   "  graph {\nA\n}"
		// Tree: Graph (@ 1 3 ...), 'A' (@ 2 1 2 1)
		// Cursor at line 2, col 2 (after "A")
		// Naive check: pos.Column (2) < Graph.Start.Column (3) → incorrectly returns "not inside"
		"MultiLineColumnBug": {
			src:         "  graph {\nA\n}",
			position:    token.Position{Line: 2, Column: 2},
			wantPrefix:  "A",
			wantAttrCtx: Node,
		},
		// Edge attributes: cursor after "arr" in edge attr list
		// Input: `digraph { a -> b [arr] }`
		// Tree: EdgeStmt > AttrList > AList > Attribute > ID > 'arr' (@ 1 19 1 21)
		// Cursor at line 1, col 22 (after "arr", on "]")
		"EdgeAttrList": {
			src:         `digraph { a -> b [arr] }`,
			position:    token.Position{Line: 1, Column: 22},
			wantPrefix:  "arr",
			wantAttrCtx: Edge,
		},
		// Empty prefix: cursor right after "[" with nothing typed yet
		// Input: `graph { a [ }` (malformed but parser recovers)
		// Tree: AttrList '[' (@ 1 11 1 11), no AList children
		// Cursor at line 1, col 12 (after "[")
		// Should return empty prefix, Node context
		"EmptyPrefixAfterBracket": {
			src:         `graph { a [ }`,
			position:    token.Position{Line: 1, Column: 12},
			wantPrefix:  "",
			wantAttrCtx: Node,
		},
		// Cursor after comma in attr list - ready for next attribute
		// Input: `graph { a [label=red,] }`
		// Tree: ',' (@ 1 21 1 21), ']' (@ 1 22 1 22)
		// Cursor at line 1, col 22 (after ",", on "]")
		"AfterCommaInAttrList": {
			src:         `graph { a [label=red,] }`,
			position:    token.Position{Line: 1, Column: 22},
			wantPrefix:  "",
			wantAttrCtx: Node,
		},
		// AttrStmt with "node" keyword - sets default node attributes
		// Input: `graph { node [lab] }`
		// Tree: AttrStmt > 'node' > AttrList > AList > Attribute > ID > 'lab' (@ 1 15 1 17)
		// Cursor at line 1, col 18 (after "lab")
		// Context should be Node (from AttrStmt with "node" keyword)
		"AttrStmtNode": {
			src:         `graph { node [lab] }`,
			position:    token.Position{Line: 1, Column: 18},
			wantPrefix:  "lab",
			wantAttrCtx: Node,
		},
		// AttrStmt with "edge" keyword - sets default edge attributes
		// Input: `graph { edge [lab] }`
		// Tree: AttrStmt > 'edge' > AttrList > AList > Attribute > ID > 'lab' (@ 1 15 1 17)
		// Cursor at line 1, col 18 (after "lab")
		// Context should be Edge (from AttrStmt with "edge" keyword)
		"AttrStmtEdge": {
			src:         `graph { edge [lab] }`,
			position:    token.Position{Line: 1, Column: 18},
			wantPrefix:  "lab",
			wantAttrCtx: Edge,
		},
		// AttrStmt with "graph" keyword - sets graph attributes
		// Input: `graph { graph [lab] }`
		// Tree: AttrStmt > 'graph' > AttrList > AList > Attribute > ID > 'lab' (@ 1 16 1 18)
		// Cursor at line 1, col 19 (after "lab")
		// Context should be Graph (from AttrStmt with "graph" keyword)
		"AttrStmtGraph": {
			src:         `graph { graph [lab] }`,
			position:    token.Position{Line: 1, Column: 19},
			wantPrefix:  "lab",
			wantAttrCtx: Graph,
		},
		// Subgraph: node inside subgraph still gets Node context
		// Input: `graph { subgraph { a [lab] } }`
		// Tree: Subgraph > StmtList > NodeStmt > AttrList > AList > Attribute > ID > 'lab' (@ 1 23 1 25)
		// Cursor at line 1, col 26 (after "lab")
		"NodeInSubgraph": {
			src:         `graph { subgraph { a [lab] } }`,
			position:    token.Position{Line: 1, Column: 26},
			wantPrefix:  "lab",
			wantAttrCtx: Node,
		},
		// Nil tree: should return empty prefix and Graph context
		"NilTree": {
			src:         ``,
			position:    token.Position{Line: 1, Column: 1},
			wantPrefix:  "",
			wantAttrCtx: Graph,
		},
		// Anonymous subgraph: node attributes inside anonymous subgraph get Node context
		// Input: `graph { subgraph { a [pen] } }`
		// Tree: Subgraph > StmtList > NodeStmt > AttrList > AList > Attribute > ID > 'pen' (@ 1 23 1 25)
		// Cursor at line 1, col 26 (after "pen")
		"AnonymousSubgraphNodeAttr": {
			src:         `graph { subgraph { a [pen] } }`,
			position:    token.Position{Line: 1, Column: 26},
			wantPrefix:  "pen",
			wantAttrCtx: Node,
		},
		// Named subgraph (non-cluster): node attributes inside named subgraph get Node context
		// Input: `graph { subgraph foo { a [pen] } }`
		// Tree: Subgraph > ID('foo') > StmtList > NodeStmt > AttrList > AList > Attribute > ID > 'pen' (@ 1 27 1 29)
		// Cursor at line 1, col 30 (after "pen")
		"NamedSubgraphNodeAttr": {
			src:         `graph { subgraph foo { a [pen] } }`,
			position:    token.Position{Line: 1, Column: 30},
			wantPrefix:  "pen",
			wantAttrCtx: Node,
		},
		// Cluster subgraph: node attributes inside cluster subgraph still get Node context
		// Input: `graph { subgraph cluster_foo { a [pen] } }`
		// Tree: Subgraph > ID('cluster_foo') > StmtList > NodeStmt > AttrList > AList > Attribute > ID > 'pen' (@ 1 35 1 37)
		// Cursor at line 1, col 38 (after "pen")
		// Context should be Node because we're on a NodeStmt, not the cluster itself
		"ClusterSubgraphNodeAttr": {
			src:         `graph { subgraph cluster_foo { a [pen] } }`,
			position:    token.Position{Line: 1, Column: 38},
			wantPrefix:  "pen",
			wantAttrCtx: Node,
		},
		// Cluster subgraph: graph attributes inside cluster get Cluster context
		// Input: `graph { subgraph cluster_foo { graph [pen] } }`
		// Tree: Subgraph > ID('cluster_foo') > StmtList > AttrStmt('graph') > AttrList > AList > Attribute > ID > 'pen' (@ 1 39 1 41)
		// Cursor at line 1, col 42 (after "pen")
		// Context should be Cluster because AttrStmt 'graph' inside a cluster_ subgraph
		"ClusterSubgraphGraphAttr": {
			src:         `graph { subgraph cluster_foo { graph [pen] } }`,
			position:    token.Position{Line: 1, Column: 42},
			wantPrefix:  "pen",
			wantAttrCtx: Cluster,
		},
		// Still completing name (no = yet)
		// Input: `graph { a [sha] }`
		// Cursor at line 1, col 15 (after "sha")
		"StillCompletingName": {
			src:         `graph { a [sha] }`,
			position:    token.Position{Line: 1, Column: 15},
			wantPrefix:  "sha",
			wantAttrCtx: Node,
		},

		// === Attribute value completion ===

		// Cursor right after "=" - ready to type value
		// Input: `graph { a [shape=] }`
		// Cursor at line 1, col 18 (after "=")
		"ValueAfterEquals": {
			src:          `graph { a [shape=] }`,
			position:     token.Position{Line: 1, Column: 18},
			wantPrefix:   "",
			wantAttrCtx:  Node,
			wantAttrName: "shape",
		},
		// Cursor after partial value
		// Input: `graph { a [shape=bo] }`
		// Cursor at line 1, col 20 (after "bo")
		"ValuePartial": {
			src:          `graph { a [shape=bo] }`,
			position:     token.Position{Line: 1, Column: 20},
			wantPrefix:   "bo",
			wantAttrCtx:  Node,
			wantAttrName: "shape",
		},
		// Edge with dir attribute
		// Input: `digraph { a -> b [dir=] }`
		// Cursor at line 1, col 22 (after "=")
		"ValueEdgeDir": {
			src:          `digraph { a -> b [dir=] }`,
			position:     token.Position{Line: 1, Column: 22},
			wantPrefix:   "",
			wantAttrCtx:  Edge,
			wantAttrName: "dir",
		},
		// Partial dir value
		// Input: `digraph { a -> b [dir=ba] }`
		// Cursor at line 1, col 24 (after "ba")
		"ValueEdgeDirPartial": {
			src:          `digraph { a -> b [dir=ba] }`,
			position:     token.Position{Line: 1, Column: 24},
			wantPrefix:   "ba",
			wantAttrCtx:  Edge,
			wantAttrName: "dir",
		},
		// Second attribute value after comma
		// Input: `graph { a [label=foo, shape=] }`
		// Cursor at line 1, col 28 (after second "=")
		"ValueSecondAttr": {
			src:          `graph { a [label=foo, shape=] }`,
			position:     token.Position{Line: 1, Column: 28},
			wantPrefix:   "",
			wantAttrCtx:  Node,
			wantAttrName: "shape",
		},
		// Graph-level rankdir
		// Input: `digraph { rankdir=L }`
		// Cursor at line 1, col 19 (after "L")
		"ValueGraphRankdir": {
			src:          `digraph { rankdir=L }`,
			position:     token.Position{Line: 1, Column: 19},
			wantPrefix:   "L",
			wantAttrCtx:  Graph,
			wantAttrName: "rankdir",
		},
		// Quoted value - unclosed quote creates error node outside Attribute,
		// can't determine we're in value position
		// Input: `graph { a [shape="bo] }`
		// Cursor at line 1, col 21 (inside ErrorTree, not Attribute)
		"ValueQuotedPartial": {
			src:         `graph { a [shape="bo] }`,
			position:    token.Position{Line: 1, Column: 21},
			wantPrefix:  "",
			wantAttrCtx: Node,
		},
		// Multi-line: value on next line
		// Input:
		//   graph {
		//     a [shape=
		//       box]
		//   }
		// Cursor at line 3, col 6 (after "bo")
		"ValueMultiLine": {
			src:          "graph {\n  a [shape=\n    bo]\n}",
			position:     token.Position{Line: 3, Column: 6},
			wantPrefix:   "bo",
			wantAttrCtx:  Node,
			wantAttrName: "shape",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ps := dot.NewParser([]byte(tt.src))
			tree := ps.Parse()

			got := completionContext(tree, tt.position)
			want := completionResult{Prefix: tt.wantPrefix, AttrCtx: tt.wantAttrCtx, AttrName: tt.wantAttrName}

			assert.EqualValuesf(t, got, want, "for %q at %s", tt.src, tt.position)
		})
	}
}
