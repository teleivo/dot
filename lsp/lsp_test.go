package lsp

import (
	"fmt"
	"io"
	"testing"

	"github.com/teleivo/assertive/assert"
	"github.com/teleivo/assertive/require"
	"github.com/teleivo/dot/lsp/internal/rpc"
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

		// Server responds with capabilities
		// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#initializeResult
		wantInit := `{"jsonrpc":"2.0","id":1,"result":{"capabilities":{"textDocumentSync":1},"serverInfo":{"name":"dotls","version":"(devel)"}}}`
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

		// Server should respond with ParseError and null id
		want := `{"jsonrpc":"2.0","id":null,"error":{"code":-32700,"message":"invalid JSON"}}`
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

		// Initialize handshake
		initMsg := `{"jsonrpc":"2.0","method":"initialize","id":1,"params":{}}`
		writeMessage(t, in, initMsg)
		assert.Truef(t, s.Scan(), "expecting initialize response")

		initializedMsg := `{"jsonrpc":"2.0","method":"initialized","params":{}}`
		writeMessage(t, in, initializedMsg)

		// Open a document with 2 parse errors:
		// Line 2: "a [label=]" - missing attribute value
		// Line 3: "b ->" - missing edge target
		//
		// Parser reports (1-based): 2:12 and 4:1
		// LSP positions are 0-based: line 1 char 11, line 3 char 0
		docContent := `digraph {\n  a [label=]\n  b ->\n}`
		didOpenMsg := `{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"file:///test.dot","languageId":"dot","version":1,"text":"` + docContent + `"}}}`
		writeMessage(t, in, didOpenMsg)

		// Server sends publishDiagnostics notification (no id field)
		// Diagnostics use point ranges (start == end) for error positions
		// Severity 1 = Error
		want := `{"jsonrpc":"2.0","method":"textDocument/publishDiagnostics","params":{"uri":"file:///test.dot","diagnostics":[{"range":{"start":{"line":1,"character":11},"end":{"line":1,"character":11}},"severity":1,"message":"expected attribute value"},{"range":{"start":{"line":3,"character":0},"end":{"line":3,"character":0}},"severity":1,"message":"expected node or subgraph as edge operand"}]}}`
		assert.Truef(t, s.Scan(), "expecting publishDiagnostics notification for didOpen")
		require.EqualValuesf(t, s.Text(), want, "unexpected diagnostics for didOpen")

		// Fix the errors by sending didChange with valid content
		// With TextDocumentSyncKind.Full (1), contentChanges contains a single element with full text
		fixedContent := `digraph {\n  a [label=\"hello\"]\n  b -> c\n}`
		didChangeMsg := `{"jsonrpc":"2.0","method":"textDocument/didChange","params":{"textDocument":{"uri":"file:///test.dot","version":2},"contentChanges":[{"text":"` + fixedContent + `"}]}}`
		writeMessage(t, in, didChangeMsg)

		// Server publishes empty diagnostics array to clear previous errors
		wantEmpty := `{"jsonrpc":"2.0","method":"textDocument/publishDiagnostics","params":{"uri":"file:///test.dot","diagnostics":[]}}`
		assert.Truef(t, s.Scan(), "expecting publishDiagnostics notification for didChange")
		require.EqualValuesf(t, s.Text(), wantEmpty, "unexpected diagnostics for didChange")

		// Send didSave and didClose notifications - server should ignore them (no response)
		// These are sent by editors on save and when closing a buffer
		didSaveMsg := `{"jsonrpc":"2.0","method":"textDocument/didSave","params":{"textDocument":{"uri":"file:///test.dot"}}}`
		writeMessage(t, in, didSaveMsg)

		didCloseMsg := `{"jsonrpc":"2.0","method":"textDocument/didClose","params":{"textDocument":{"uri":"file:///test.dot"}}}`
		writeMessage(t, in, didCloseMsg)

		// Send a request after the notifications to verify server is still responsive
		// and didn't block on the ignored notifications
		shutdownMsg := `{"jsonrpc":"2.0","method":"shutdown","id":2}`
		writeMessage(t, in, shutdownMsg)

		wantShutdown := `{"jsonrpc":"2.0","id":2,"result":null}`
		assert.Truef(t, s.Scan(), "expecting shutdown response after ignored notifications")
		require.EqualValuesf(t, s.Text(), wantShutdown, "unexpected shutdown response")
	})
}

func setup(t *testing.T) (*rpc.Scanner, io.Writer) {
	t.Helper()

	inR, inW := io.Pipe()
	outR, outW := io.Pipe()

	srv, err := New(Config{
		In:  inR,
		Out: outW,
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
