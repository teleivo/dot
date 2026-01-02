package lsp

import (
	"errors"
	"fmt"
	"io"
	"testing"
	"testing/iotest"

	"github.com/teleivo/assertive/assert"
	"github.com/teleivo/assertive/require"
	"github.com/teleivo/dot/lsp/internal/rpc"
)

func TestServer(t *testing.T) {
	// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#initialize
	t.Run("ShutdownBeforeInitialize", func(t *testing.T) {
		s, in := setup(t)

		t.Log("shutdown before initialize returns ServerNotInitialized error (-32002)")
		msg := `{"jsonrpc":"2.0","method":"shutdown","id":3}`
		writeMessage(t, in, msg)

		want := `{"jsonrpc":"2.0","id":3,"error":{"code":-32002,"message":"server not initialized"}}`
		assert.Truef(t, s.Scan(), "expecting response from server")
		require.EqualValuesf(t, s.Text(), want, "unexpected response")
	})

	// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#initialize
	t.Run("InitializeTwice", func(t *testing.T) {
		s, in := setup(t)

		t.Log("first initialize succeeds")
		initMsg := `{"jsonrpc":"2.0","method":"initialize","id":1,"params":{}}`
		writeMessage(t, in, initMsg)

		assert.Truef(t, s.Scan(), "expecting first initialize response")

		t.Log("second initialize returns InvalidRequest error (-32600)")
		initMsg2 := `{"jsonrpc":"2.0","method":"initialize","id":2,"params":{}}`
		writeMessage(t, in, initMsg2)

		want := `{"jsonrpc":"2.0","id":2,"error":{"code":-32600,"message":"server already initialized"}}`
		assert.Truef(t, s.Scan(), "expecting error response for second initialize")
		require.EqualValuesf(t, s.Text(), want, "unexpected response")
	})

	// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#shutdown
	t.Run("ShutdownAfterInitialize", func(t *testing.T) {
		s, in := setup(t)

		t.Log("initialize and get capabilities")
		initMsg := `{"jsonrpc":"2.0","method":"initialize","id":1,"params":{}}`
		writeMessage(t, in, initMsg)

		wantInit := `{"jsonrpc":"2.0","id":1,"result":{"capabilities":{"completionProvider":{"triggerCharacters":["[",",",";","{","="]},"documentFormattingProvider":true,"hoverProvider":true,"positionEncoding":"utf-8","signatureHelpProvider":{"triggerCharacters":["="]},"textDocumentSync":2},"serverInfo":{"name":"dotls","version":"(devel)"}}}`
		assert.Truef(t, s.Scan(), "expecting initialize response")
		require.EqualValuesf(t, s.Text(), wantInit, "unexpected initialize response")

		t.Log("send initialized notification")
		initializedMsg := `{"jsonrpc":"2.0","method":"initialized","params":{}}`
		writeMessage(t, in, initializedMsg)

		t.Log("shutdown returns null result")
		shutdownMsg := `{"jsonrpc":"2.0","method":"shutdown","id":2}`
		writeMessage(t, in, shutdownMsg)

		want := `{"jsonrpc":"2.0","id":2,"result":null}`
		assert.Truef(t, s.Scan(), "expecting shutdown response")
		require.EqualValuesf(t, s.Text(), want, "unexpected response")

		t.Log("requests after shutdown return InvalidRequest error (-32600)")
		postShutdownMsg := `{"jsonrpc":"2.0","method":"textDocument/hover","id":3,"params":{"textDocument":{"uri":"file:///test.dot"},"position":{"line":0,"character":0}}}`
		writeMessage(t, in, postShutdownMsg)

		wantErr := `{"jsonrpc":"2.0","id":3,"error":{"code":-32600,"message":"server is shutting down"}}`
		assert.Truef(t, s.Scan(), "expecting error response after shutdown")
		require.EqualValuesf(t, s.Text(), wantErr, "unexpected response after shutdown")

		t.Log("exit notification terminates server")
		exitMsg := `{"jsonrpc":"2.0","method":"exit"}`
		writeMessage(t, in, exitMsg)
	})

	// https://www.jsonrpc.org/specification#error_object
	t.Run("ParseError", func(t *testing.T) {
		s, in := setup(t)

		t.Log("invalid JSON returns ParseError (-32700)")
		writeMessage(t, in, `{not valid json}`)

		want := `{"jsonrpc":"2.0","error":{"code":-32700,"message":"invalid JSON"}}`
		assert.Truef(t, s.Scan(), "expecting ParseError response")
		require.EqualValuesf(t, s.Text(), want, "unexpected response")

		t.Log("server continues processing after parse error")
		initMsg := `{"jsonrpc":"2.0","method":"initialize","id":1,"params":{}}`
		writeMessage(t, in, initMsg)

		assert.Truef(t, s.Scan(), "expecting initialize response after parse error")
	})

	// https://www.jsonrpc.org/specification#error_object
	t.Run("MethodNotFound", func(t *testing.T) {
		s, in := setup(t)

		t.Log("initialize handshake")
		initMsg := `{"jsonrpc":"2.0","method":"initialize","id":1,"params":{}}`
		writeMessage(t, in, initMsg)
		assert.Truef(t, s.Scan(), "expecting initialize response")

		initializedMsg := `{"jsonrpc":"2.0","method":"initialized","params":{}}`
		writeMessage(t, in, initializedMsg)

		t.Log("unknown notification is silently dropped")
		unknownNotification := `{"jsonrpc":"2.0","method":"$/unknownNotification","params":{}}`
		writeMessage(t, in, unknownNotification)

		t.Log("unknown request returns MethodNotFound error (-32601)")
		unknownRequest := `{"jsonrpc":"2.0","method":"textDocument/unknown","id":2,"params":{}}`
		writeMessage(t, in, unknownRequest)

		want := `{"jsonrpc":"2.0","id":2,"error":{"code":-32601,"message":"method not found"}}`
		assert.Truef(t, s.Scan(), "expecting MethodNotFound error response")
		require.EqualValuesf(t, s.Text(), want, "unexpected response")
	})

	// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_publishDiagnostics
	t.Run("PublishDiagnostics", func(t *testing.T) {
		s, in := setup(t)

		t.Log("initialize handshake")
		initMsg := `{"jsonrpc":"2.0","method":"initialize","id":1,"params":{}}`
		writeMessage(t, in, initMsg)

		wantInit := `{"jsonrpc":"2.0","id":1,"result":{"capabilities":{"completionProvider":{"triggerCharacters":["[",",",";","{","="]},"documentFormattingProvider":true,"hoverProvider":true,"positionEncoding":"utf-8","signatureHelpProvider":{"triggerCharacters":["="]},"textDocumentSync":2},"serverInfo":{"name":"dotls","version":"(devel)"}}}`
		assert.Truef(t, s.Scan(), "expecting initialize response")
		require.EqualValuesf(t, s.Text(), wantInit, "unexpected initialize response")

		initializedMsg := `{"jsonrpc":"2.0","method":"initialized","params":{}}`
		writeMessage(t, in, initializedMsg)

		t.Log("open document with 2 parse errors publishes diagnostics")
		firstDocContent := `digraph {\n  a [label=]\n  b ->\n}`
		didOpenFirst := `{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"file:///first.dot","languageId":"dot","version":1,"text":"` + firstDocContent + `"}}}`
		writeMessage(t, in, didOpenFirst)

		want := `{"jsonrpc":"2.0","method":"textDocument/publishDiagnostics","params":{"uri":"file:///first.dot","version":1,"diagnostics":[{"range":{"start":{"line":1,"character":11},"end":{"line":1,"character":11}},"severity":1,"message":"expected attribute value"},{"range":{"start":{"line":3,"character":0},"end":{"line":3,"character":0}},"severity":1,"message":"expected node or subgraph as edge operand"}]}}`
		assert.Truef(t, s.Scan(), "expecting publishDiagnostics notification for didOpen")
		require.EqualValuesf(t, s.Text(), want, "unexpected diagnostics for didOpen")

		t.Log("fix first error via incremental change, one error remains")
		didChangeMsg1 := `{"jsonrpc":"2.0","method":"textDocument/didChange","params":{"textDocument":{"uri":"file:///first.dot","version":2},"contentChanges":[{"range":{"start":{"line":1,"character":5},"end":{"line":1,"character":11}},"text":"label=\"hello\""}]}}`
		writeMessage(t, in, didChangeMsg1)

		wantOneError := `{"jsonrpc":"2.0","method":"textDocument/publishDiagnostics","params":{"uri":"file:///first.dot","version":2,"diagnostics":[{"range":{"start":{"line":3,"character":0},"end":{"line":3,"character":0}},"severity":1,"message":"expected node or subgraph as edge operand"}]}}`
		assert.Truef(t, s.Scan(), "expecting publishDiagnostics after first incremental change")
		require.EqualValuesf(t, s.Text(), wantOneError, "unexpected diagnostics after first fix")

		t.Log("open second valid document, empty diagnostics")
		secondDocContent := `graph { x -- y }`
		didOpenSecond := `{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"file:///second.dot","languageId":"dot","version":1,"text":"` + secondDocContent + `"}}}`
		writeMessage(t, in, didOpenSecond)

		wantSecondEmpty := `{"jsonrpc":"2.0","method":"textDocument/publishDiagnostics","params":{"uri":"file:///second.dot","version":1,"diagnostics":[]}}`
		assert.Truef(t, s.Scan(), "expecting publishDiagnostics for second document")
		require.EqualValuesf(t, s.Text(), wantSecondEmpty, "unexpected diagnostics for second document")

		t.Log("fix second error in first document, no errors remain")
		didChangeMsg2 := `{"jsonrpc":"2.0","method":"textDocument/didChange","params":{"textDocument":{"uri":"file:///first.dot","version":3},"contentChanges":[{"range":{"start":{"line":2,"character":2},"end":{"line":2,"character":6}},"text":"b -> c"}]}}`
		writeMessage(t, in, didChangeMsg2)

		wantFirstEmpty := `{"jsonrpc":"2.0","method":"textDocument/publishDiagnostics","params":{"uri":"file:///first.dot","version":3,"diagnostics":[]}}`
		assert.Truef(t, s.Scan(), "expecting publishDiagnostics after second incremental change")
		require.EqualValuesf(t, s.Text(), wantFirstEmpty, "unexpected diagnostics after all fixes")

		t.Log("change second document, still valid")
		didChangeSecond := `{"jsonrpc":"2.0","method":"textDocument/didChange","params":{"textDocument":{"uri":"file:///second.dot","version":2},"contentChanges":[{"range":{"start":{"line":0,"character":14},"end":{"line":0,"character":14}},"text":" -- z"}]}}`
		writeMessage(t, in, didChangeSecond)

		wantSecondEmptyV2 := `{"jsonrpc":"2.0","method":"textDocument/publishDiagnostics","params":{"uri":"file:///second.dot","version":2,"diagnostics":[]}}`
		assert.Truef(t, s.Scan(), "expecting publishDiagnostics after changing second document")
		require.EqualValuesf(t, s.Text(), wantSecondEmptyV2, "unexpected diagnostics after changing second document")

		t.Log("close documents and verify server still responsive")
		didCloseSecond := `{"jsonrpc":"2.0","method":"textDocument/didClose","params":{"textDocument":{"uri":"file:///second.dot"}}}`
		writeMessage(t, in, didCloseSecond)

		didSaveMsg := `{"jsonrpc":"2.0","method":"textDocument/didSave","params":{"textDocument":{"uri":"file:///first.dot"}}}`
		writeMessage(t, in, didSaveMsg)

		didCloseFirst := `{"jsonrpc":"2.0","method":"textDocument/didClose","params":{"textDocument":{"uri":"file:///first.dot"}}}`
		writeMessage(t, in, didCloseFirst)

		shutdownMsg := `{"jsonrpc":"2.0","method":"shutdown","id":2}`
		writeMessage(t, in, shutdownMsg)

		wantShutdown := `{"jsonrpc":"2.0","id":2,"result":null}`
		assert.Truef(t, s.Scan(), "expecting shutdown response")
		require.EqualValuesf(t, s.Text(), wantShutdown, "unexpected shutdown response")
	})

	// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_completion
	t.Run("Completion", func(t *testing.T) {
		s, in := setup(t)

		t.Log("initialize handshake")
		initMsg := `{"jsonrpc":"2.0","method":"initialize","id":1,"params":{}}`
		writeMessage(t, in, initMsg)
		assert.Truef(t, s.Scan(), "expecting initialize response")

		initializedMsg := `{"jsonrpc":"2.0","method":"initialized","params":{}}`
		writeMessage(t, in, initializedMsg)

		t.Log("open document with node attribute list")
		docContent := `digraph { a [lab] }`
		didOpen := `{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"file:///test.dot","languageId":"dot","version":1,"text":"` + docContent + `"}}}`
		writeMessage(t, in, didOpen)

		assert.Truef(t, s.Scan(), "expecting publishDiagnostics")

		t.Log("complete 'lab' in node context returns node-applicable attributes")
		completionReq := `{"jsonrpc":"2.0","method":"textDocument/completion","id":2,"params":{"textDocument":{"uri":"file:///test.dot"},"position":{"line":0,"character":16}}}`
		writeMessage(t, in, completionReq)

		wantCompletion := `{"jsonrpc":"2.0","id":2,"result":{"isIncomplete":false,"items":[{"label":"label","kind":10,"detail":"lblString","documentation":{"kind":"markdown","value":"Text label attached to objects\n\n**Type:** [lblString](https://graphviz.org/docs/attr-types/lblString/)\n\nLabel: escString or HTML-like \u003ctable\u003e...\u003c/table\u003e\n\n[Docs](https://graphviz.org/docs/attrs/label/)"},"insertText":"label="},{"label":"labelloc","kind":10,"detail":"string","documentation":{"kind":"markdown","value":"Vertical placement of labels\n\n**Type:** [string](https://graphviz.org/docs/attr-types/string/)\n\nText string\n\n[Docs](https://graphviz.org/docs/attrs/labelloc/)"},"insertText":"labelloc="}]}}`
		assert.Truef(t, s.Scan(), "expecting completion response")
		require.EqualValuesf(t, s.Text(), wantCompletion, "unexpected completion response")

		t.Log("continue typing narrows completions")
		didChange := `{"jsonrpc":"2.0","method":"textDocument/didChange","params":{"textDocument":{"uri":"file:///test.dot","version":2},"contentChanges":[{"range":{"start":{"line":0,"character":16},"end":{"line":0,"character":16}},"text":"el"}]}}`
		writeMessage(t, in, didChange)

		assert.Truef(t, s.Scan(), "expecting publishDiagnostics after change")

		completionReq2 := `{"jsonrpc":"2.0","method":"textDocument/completion","id":3,"params":{"textDocument":{"uri":"file:///test.dot"},"position":{"line":0,"character":18}}}`
		writeMessage(t, in, completionReq2)

		wantCompletion2 := `{"jsonrpc":"2.0","id":3,"result":{"isIncomplete":false,"items":[{"label":"label","kind":10,"detail":"lblString","documentation":{"kind":"markdown","value":"Text label attached to objects\n\n**Type:** [lblString](https://graphviz.org/docs/attr-types/lblString/)\n\nLabel: escString or HTML-like \u003ctable\u003e...\u003c/table\u003e\n\n[Docs](https://graphviz.org/docs/attrs/label/)"},"insertText":"label="},{"label":"labelloc","kind":10,"detail":"string","documentation":{"kind":"markdown","value":"Vertical placement of labels\n\n**Type:** [string](https://graphviz.org/docs/attr-types/string/)\n\nText string\n\n[Docs](https://graphviz.org/docs/attrs/labelloc/)"},"insertText":"labelloc="}]}}`
		assert.Truef(t, s.Scan(), "expecting narrowed completion response")
		require.EqualValuesf(t, s.Text(), wantCompletion2, "unexpected narrowed completion response")

		t.Log("complete 'arr' in edge context returns edge-specific attributes")
		didChange2 := `{"jsonrpc":"2.0","method":"textDocument/didChange","params":{"textDocument":{"uri":"file:///test.dot","version":3},"contentChanges":[{"range":{"start":{"line":0,"character":10},"end":{"line":0,"character":20}},"text":"a -> b [arr"}]}}`
		writeMessage(t, in, didChange2)

		assert.Truef(t, s.Scan(), "expecting publishDiagnostics after edge change")

		completionReq3 := `{"jsonrpc":"2.0","method":"textDocument/completion","id":4,"params":{"textDocument":{"uri":"file:///test.dot"},"position":{"line":0,"character":21}}}`
		writeMessage(t, in, completionReq3)

		wantCompletion3 := `{"jsonrpc":"2.0","id":4,"result":{"isIncomplete":false,"items":[{"label":"arrowhead","kind":10,"detail":"arrowType","documentation":{"kind":"markdown","value":"Style of arrowhead on edge head node\n\n**Type:** [arrowType](https://graphviz.org/docs/attr-types/arrowType/): ` + "`box` | `crow` | `curve` | `diamond` | `dot` | `icurve` | `inv` | `none` | `normal` | `tee` | `vee`" + `\n\n[Docs](https://graphviz.org/docs/attrs/arrowhead/)"},"insertText":"arrowhead="},{"label":"arrowsize","kind":10,"detail":"double","documentation":{"kind":"markdown","value":"Multiplicative scale factor for arrowheads\n\n**Type:** [double](https://graphviz.org/docs/attr-types/double/)\n\nDouble-precision floating point number\n\n[Docs](https://graphviz.org/docs/attrs/arrowsize/)"},"insertText":"arrowsize="},{"label":"arrowtail","kind":10,"detail":"arrowType","documentation":{"kind":"markdown","value":"Style of arrowhead on edge tail node\n\n**Type:** [arrowType](https://graphviz.org/docs/attr-types/arrowType/): ` + "`box` | `crow` | `curve` | `diamond` | `dot` | `icurve` | `inv` | `none` | `normal` | `tee` | `vee`" + `\n\n[Docs](https://graphviz.org/docs/attrs/arrowtail/)"},"insertText":"arrowtail="}]}}`
		assert.Truef(t, s.Scan(), "expecting edge completion response")
		require.EqualValuesf(t, s.Text(), wantCompletion3, "unexpected edge completion response")
	})

	// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_completion
	t.Run("CompletionAttributeValues", func(t *testing.T) {
		s, in := setup(t)

		t.Log("initialize handshake")
		initMsg := `{"jsonrpc":"2.0","method":"initialize","id":1,"params":{}}`
		writeMessage(t, in, initMsg)
		assert.Truef(t, s.Scan(), "expecting initialize response")

		initializedMsg := `{"jsonrpc":"2.0","method":"initialized","params":{}}`
		writeMessage(t, in, initializedMsg)

		t.Log("open document with cursor after 'dir='")
		docContent := `digraph { a -> b [dir=] }`
		didOpen := `{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"file:///test.dot","languageId":"dot","version":1,"text":"` + docContent + `"}}}`
		writeMessage(t, in, didOpen)

		assert.Truef(t, s.Scan(), "expecting publishDiagnostics notification")

		t.Log("complete after '=' returns dirType values")
		completionReq := `{"jsonrpc":"2.0","method":"textDocument/completion","id":2,"params":{"textDocument":{"uri":"file:///test.dot"},"position":{"line":0,"character":21}}}`
		writeMessage(t, in, completionReq)

		want := `{"jsonrpc":"2.0","id":2,"result":{"isIncomplete":false,"items":[{"label":"back","kind":12,"detail":"dirType","documentation":{"kind":"markdown","value":"[dirType](https://graphviz.org/docs/attr-types/dirType/)"}},{"label":"both","kind":12,"detail":"dirType","documentation":{"kind":"markdown","value":"[dirType](https://graphviz.org/docs/attr-types/dirType/)"}},{"label":"forward","kind":12,"detail":"dirType","documentation":{"kind":"markdown","value":"[dirType](https://graphviz.org/docs/attr-types/dirType/)"}},{"label":"none","kind":12,"detail":"dirType","documentation":{"kind":"markdown","value":"[dirType](https://graphviz.org/docs/attr-types/dirType/)"}}]}}`
		assert.Truef(t, s.Scan(), "expecting completion response")
		require.EqualValuesf(t, s.Text(), want, "unexpected completion response")
	})

	// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_formatting
	t.Run("Formatting", func(t *testing.T) {
		s, in := setup(t)

		t.Log("initialize handshake")
		initMsg := `{"jsonrpc":"2.0","method":"initialize","id":1,"params":{}}`
		writeMessage(t, in, initMsg)
		assert.Truef(t, s.Scan(), "expecting initialize response")

		initializedMsg := `{"jsonrpc":"2.0","method":"initialized","params":{}}`
		writeMessage(t, in, initializedMsg)

		t.Log("open document with parse error")
		invalidContent := `digraph { a -> }`
		didOpen := `{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"file:///test.dot","languageId":"dot","version":1,"text":"` + invalidContent + `"}}}`
		writeMessage(t, in, didOpen)

		assert.Truef(t, s.Scan(), "expecting publishDiagnostics for invalid document")

		t.Log("formatting invalid document returns error")
		formatInvalid := `{"jsonrpc":"2.0","method":"textDocument/formatting","id":2,"params":{"textDocument":{"uri":"file:///test.dot"},"options":{"tabSize":2,"insertSpaces":false}}}`
		writeMessage(t, in, formatInvalid)

		wantError := `{"jsonrpc":"2.0","id":2,"error":{"code":-32603,"message":"formatting failed: 1:16: expected node or subgraph as edge operand"}}`
		assert.Truef(t, s.Scan(), "expecting error response for invalid document")
		require.EqualValuesf(t, s.Text(), wantError, "unexpected error response")

		t.Log("fix document and format successfully")
		didChange := `{"jsonrpc":"2.0","method":"textDocument/didChange","params":{"textDocument":{"uri":"file:///test.dot","version":2},"contentChanges":[{"range":{"start":{"line":0,"character":14},"end":{"line":0,"character":14}},"text":"b "}]}}`
		writeMessage(t, in, didChange)

		assert.Truef(t, s.Scan(), "expecting publishDiagnostics for fixed document")

		formatValid := `{"jsonrpc":"2.0","method":"textDocument/formatting","id":3,"params":{"textDocument":{"uri":"file:///test.dot"},"options":{"tabSize":2,"insertSpaces":false}}}`
		writeMessage(t, in, formatValid)

		wantFormatting := `{"jsonrpc":"2.0","id":3,"result":[{"range":{"start":{"line":0,"character":0},"end":{"line":0,"character":18}},"newText":"digraph {\n\ta -\u003e b\n}"}]}`
		assert.Truef(t, s.Scan(), "expecting formatting response")
		require.EqualValuesf(t, s.Text(), wantFormatting, "unexpected formatting response")
	})

	// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_hover
	t.Run("Hover", func(t *testing.T) {
		s, in := setup(t)

		t.Log("initialize handshake")
		initMsg := `{"jsonrpc":"2.0","method":"initialize","id":1,"params":{}}`
		writeMessage(t, in, initMsg)
		assert.Truef(t, s.Scan(), "expecting initialize response")

		initializedMsg := `{"jsonrpc":"2.0","method":"initialized","params":{}}`
		writeMessage(t, in, initializedMsg)

		t.Log("open document with label attribute")
		docContent := `digraph { a [label=\"hello\"] }`
		didOpen := `{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"file:///test.dot","languageId":"dot","version":1,"text":"` + docContent + `"}}}`
		writeMessage(t, in, didOpen)

		assert.Truef(t, s.Scan(), "expecting publishDiagnostics notification")

		t.Log("hover over 'label' attribute name returns documentation")
		// position is on 'label' (character 14 is the 'a' in 'label')
		hoverReq := `{"jsonrpc":"2.0","method":"textDocument/hover","id":2,"params":{"textDocument":{"uri":"file:///test.dot"},"position":{"line":0,"character":14}}}`
		writeMessage(t, in, hoverReq)

		// Same documentation as completion item for 'label'
		want := `{"jsonrpc":"2.0","id":2,"result":{"contents":{"kind":"markdown","value":"Text label attached to objects\n\n**Type:** [lblString](https://graphviz.org/docs/attr-types/lblString/)\n\nLabel: escString or HTML-like \u003ctable\u003e...\u003c/table\u003e\n\n[Docs](https://graphviz.org/docs/attrs/label/)"}}}`
		assert.Truef(t, s.Scan(), "expecting hover response")
		require.EqualValuesf(t, s.Text(), want, "unexpected hover response")
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
