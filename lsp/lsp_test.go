package lsp

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/teleivo/assertive/assert"
	"github.com/teleivo/assertive/require"
)

// TODO this initialize request is kept here for my server tests
// 	in := `Content-Length: 4182
//
// {"jsonrpc":"2.0","method":"initialize","id":1,"params":{"workDoneToken":"1","capabilities":{"textDocument":{"typeDefinition":{"linkSupport":true},"definition":{"linkSupport":true,"dynamicRegistration":true},"synchronization":{"willSaveWaitUntil":true,"dynamicRegistration":false,"willSave":true,"didSave":true},"rename":{"prepareSupport":true,"dynamicRegistration":true},"semanticTokens":{"dynamicRegistration":false,"requests":{"full":{"delta":true},"range":false},"augmentsSyntaxTokens":true,"serverCancelSupport":false,"multilineTokenSupport":false,"overlappingTokenSupport":true,"tokenModifiers":["declaration","definition","readonly","static","deprecated","abstract","async","modification","documentation","defaultLibrary"],"tokenTypes":["namespace","type","class","enum","interface","struct","typeParameter","parameter","variable","property","enumMember","event","function","method","macro","keyword","modifier","comment","string","number","regexp","operator","decorator"],"formats":["relative"]},"inlayHint":{"resolveSupport":{"properties":["textEdits","tooltip","location","command"]},"dynamicRegistration":true},"references":{"dynamicRegistration":false},"implementation":{"linkSupport":true},"callHierarchy":{"dynamicRegistration":false},"publishDiagnostics":{"dataSupport":true,"relatedInformation":true,"tagSupport":{"valueSet":[1,2]}},"hover":{"contentFormat":["markdown","plaintext"],"dynamicRegistration":true},"documentSymbol":{"dynamicRegistration":false,"symbolKind":{"valueSet":[1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22,23,24,25,26]},"hierarchicalDocumentSymbolSupport":true},"rangeFormatting":{"rangesSupport":true,"dynamicRegistration":true},"diagnostic":{"tagSupport":{"valueSet":[1,2]},"dynamicRegistration":false},"formatting":{"dynamicRegistration":true},"foldingRange":{"foldingRange":{"collapsedText":true},"lineFoldingOnly":true,"dynamicRegistration":false,"foldingRangeKind":{"valueSet":["comment","imports","region"]}},"documentHighlight":{"dynamicRegistration":false},"completion":{"completionItemKind":{"valueSet":[1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22,23,24,25]},"dynamicRegistration":false,"completionItem":{"snippetSupport":true,"resolveSupport":{"properties":["documentation","detail","additionalTextEdits","command","data"]},"deprecatedSupport":true,"tagSupport":{"valueSet":[1]},"commitCharactersSupport":false,"insertTextModeSupport":{"valueSet":[1]},"insertReplaceSupport":true,"documentationFormat":["markdown","plaintext"],"labelDetailsSupport":true,"preselectSupport":false},"completionList":{"itemDefaults":["commitCharacters","editRange","insertTextFormat","insertTextMode","data"]},"contextSupport":true,"insertTextMode":1},"declaration":{"linkSupport":true},"signatureHelp":{"signatureInformation":{"documentationFormat":["markdown","plaintext"],"activeParameterSupport":true,"parameterInformation":{"labelOffsetSupport":true}},"dynamicRegistration":false},"codeLens":{"resolveSupport":{"properties":["command"]},"dynamicRegistration":false},"codeAction":{"codeActionLiteralSupport":{"codeActionKind":{"valueSet":["","quickfix","refactor","refactor.extract","refactor.inline","refactor.rewrite","source","source.organizeImports"]}},"dynamicRegistration":true,"isPreferredSupport":true,"dataSupport":true,"resolveSupport":{"properties":["edit","command"]}}},"workspace":{"didChangeWatchedFiles":{"relativePatternSupport":true,"dynamicRegistration":false},"symbol":{"symbolKind":{"valueSet":[1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22,23,24,25,26]},"dynamicRegistration":false},"workspaceEdit":{"resourceOperations":["rename","create","delete"]},"didChangeConfiguration":{"dynamicRegistration":false},"applyEdit":true,"workspaceFolders":true,"configuration":true,"semanticTokens":{"refreshSupport":true},"inlayHint":{"refreshSupport":true}},"general":{"positionEncodings":["utf-8","utf-16","utf-32"]},"window":{"showDocument":{"support":true},"workDoneProgress":true,"showMessage":{"messageActionItem":{"additionalPropertiesSupport":true}}}},"workspaceFolders":null,"trace":"off","rootUri":null,"rootPath":null,"clientInfo":{"name":"Neovim","version":"0.11.5+v0.11.5"},"processId":92548}}`

func TestScanner(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		t.Parallel()

		var w bytes.Buffer
		s := NewScanner(&w)

		msg1 := `{"jsonrpc":"2.0","method":"initialize","id":1,"params":null}`
		write(t, &w, "Content-Length:  %d \r\n", len(msg1))
		write(t, &w, "\r\n")
		write(t, &w, "%s", msg1)

		assert.Truef(t, s.Scan(), "want true as msg1 is unread")
		require.EqualValuesf(t, s.Next(), msg1, "failed to read msg1")
		require.NoErrorf(t, s.Err(), "want no errors reading msg1")

		msg2 := `{"jsonrpc":"2.0","method":"initialized","id":2}`
		write(t, &w, "content-Length: %d\n", len(msg2))
		write(t, &w, "content-type: application/vscode-jsonrpc; charset=utf-8\r\n")
		write(t, &w, "\n")
		write(t, &w, "%s", msg2)

		assert.Truef(t, s.Scan(), "want true as msg2 is unread")
		require.EqualValuesf(t, s.Next(), msg2, "failed to read msg2")
		require.NoErrorf(t, s.Err(), "want no errors reading msg2")

		// Content-Type before Content-Length is valid per spec; unknown headers are skipped
		msg3 := `{"jsonrpc":"2.0","method":"shutdown","id":3}`
		write(t, &w, "Content-Type: application/vscode-jsonrpc; charset=utf-8\r\n")
		write(t, &w, "X-Custom-Header: some-value\r\n")
		write(t, &w, "Content-Length: %d\r\n", len(msg3))
		write(t, &w, "X-Another-Header: ignored\r\n")
		write(t, &w, "\r\n")
		write(t, &w, "%s", msg3)

		assert.Truef(t, s.Scan(), "want true as msg3 is unread")
		require.EqualValuesf(t, s.Next(), msg3, "failed to read msg3")
		require.NoErrorf(t, s.Err(), "want no errors reading msg3")

		// Content-Length: 0 is valid at protocol level (empty content)
		write(t, &w, "Content-Length: 0\r\n")
		write(t, &w, "\r\n")

		assert.Truef(t, s.Scan(), "want true as msg4 is unread")
		require.EqualValuesf(t, s.Next(), "", "msg4 should be empty content")
		require.NoErrorf(t, s.Err(), "want no errors reading msg4")

		assert.Falsef(t, s.Scan(), "want false as all msgs are read")
		assert.EqualValuesf(t, s.Next(), "", "should be empty")
		assert.NoErrorf(t, s.Err(), "want no errors reading all msgs")

		assert.Falsef(t, s.Scan(), "want false as all msgs are read")
		assert.EqualValuesf(t, s.Next(), "", "should be empty")
		assert.NoErrorf(t, s.Err(), "want no errors reading all msgs")
	})
	t.Run("Errors", func(t *testing.T) {
		t.Parallel()

		t.Run("HeaderLineWithoutNewline", func(t *testing.T) {
			t.Parallel()
			var w bytes.Buffer
			s := NewScanner(&w)

			write(t, &w, "Content-Length: 10")

			assert.Falsef(t, s.Scan(), "want false as header line incomplete")
			assert.Nilf(t, s.Err(), "EOF during header read is not an error")
			assert.EqualValuesf(t, s.Next(), "", "no content on EOF")
			// subsequent calls should remain stable
			assert.Falsef(t, s.Scan(), "still false")
			assert.Nilf(t, s.Err(), "still no error")
		})
		t.Run("InvalidHeaderFormat", func(t *testing.T) {
			t.Parallel()
			var w bytes.Buffer
			s := NewScanner(&w)

			msg1 := `{"jsonrpc":"2.0","method":"initialize","id":1,"params":null}`
			write(t, &w, "Content-Length %d\r\n", len(msg1)) // missing ':'
			write(t, &w, "\r\n")
			write(t, &w, "%s", msg1)

			assert.Falsef(t, s.Scan(), "want false as header format invalid")
			require.NotNilf(t, s.Err(), "expect error")
			assert.Truef(t, strings.Contains(s.Err().Error(), "invalid header"), "error should mention 'invalid header'")
			assert.EqualValuesf(t, s.Next(), "", "no content on error")
		})
		t.Run("InvalidContentLengthValue", func(t *testing.T) {
			t.Parallel()
			var w bytes.Buffer
			s := NewScanner(&w)

			msg1 := `{"jsonrpc":"2.0","method":"initialize","id":1,"params":null}`
			write(t, &w, "Content-Length: invalid\r\n")
			write(t, &w, "\r\n")
			write(t, &w, "%s", msg1)

			assert.Falsef(t, s.Scan(), "want false as content-length not a number")
			require.NotNilf(t, s.Err(), "expect error")
			assert.Truef(t, strings.Contains(s.Err().Error(), "invalid content-length"), "error should mention 'invalid content-length'")
			assert.EqualValuesf(t, s.Next(), "", "no content on error")
		})
		t.Run("NoContent", func(t *testing.T) {
			t.Parallel()
			var w bytes.Buffer
			s := NewScanner(&w)

			write(t, &w, "Content-Length: 100\r\n")
			write(t, &w, "\r\n")
			// no content written

			assert.Falsef(t, s.Scan(), "want false as content missing")
			require.NotNilf(t, s.Err(), "expect error")
			assert.Truef(t, strings.Contains(s.Err().Error(), "unexpected EOF"), "error should mention 'unexpected EOF'")
			assert.EqualValuesf(t, s.Next(), "", "no content on error")
		})
		t.Run("PartialContent", func(t *testing.T) {
			t.Parallel()
			var w bytes.Buffer
			s := NewScanner(&w)

			msg1 := `{"jsonrpc":"2.0","method":"initialize","id":1,"params":null}`
			write(t, &w, "Content-Length: %d\r\n", len(msg1))
			write(t, &w, "\r\n")
			write(t, &w, "%s", msg1[:len(msg1)-8]) // write less than promised

			assert.Falsef(t, s.Scan(), "want false as content is incomplete")
			require.NotNilf(t, s.Err(), "expect error")
			assert.Truef(t, strings.Contains(s.Err().Error(), "unexpected EOF"), "error should mention 'unexpected EOF'")
			assert.EqualValuesf(t, s.Next(), "", "no content on error")
		})
		t.Run("NegativeContentLength", func(t *testing.T) {
			t.Parallel()
			var w bytes.Buffer
			s := NewScanner(&w)

			write(t, &w, "Content-Length: -1\r\n")
			write(t, &w, "\r\n")

			assert.Falsef(t, s.Scan(), "want false as content-length is negative")
			require.NotNilf(t, s.Err(), "expect error")
			assert.Truef(t, strings.Contains(s.Err().Error(), "positive"), "error should mention 'content-length'")
			assert.EqualValuesf(t, s.Next(), "", "no content on error")
		})
		t.Run("ContentLengthTooLarge", func(t *testing.T) {
			t.Parallel()
			var w bytes.Buffer
			s := NewScanner(&w)

			// 1 byte over maxContentLength (10MB)
			write(t, &w, "Content-Length: %d\r\n", 10<<20+1)
			write(t, &w, "\r\n")

			assert.Falsef(t, s.Scan(), "want false as content-length exceeds max")
			require.NotNilf(t, s.Err(), "expect error")
			assert.Truef(t, strings.Contains(s.Err().Error(), "max"), "error should mention 'max'")
			assert.EqualValuesf(t, s.Next(), "", "no content on error")
		})
		t.Run("EmptyLineBeforeContentLength", func(t *testing.T) {
			t.Parallel()
			var w bytes.Buffer
			s := NewScanner(&w)

			write(t, &w, "\r\n") // empty line with no headers

			assert.Falsef(t, s.Scan(), "want false as no content-length header")
			require.NotNilf(t, s.Err(), "expect error")
			assert.Truef(t, strings.Contains(s.Err().Error(), "content-length"), "error should mention 'content-length'")
			assert.EqualValuesf(t, s.Next(), "", "no content on error")
		})
		t.Run("ContentInHeaderPosition", func(t *testing.T) {
			t.Parallel()
			var w bytes.Buffer
			s := NewScanner(&w)

			msg := `{"jsonrpc":"2.0","id":1}`
			write(t, &w, "Content-Length: %d\r\n", len(msg))
			write(t, &w, "%s\r\n", msg) // content directly, no empty line separator

			assert.Falsef(t, s.Scan(), "want false as empty line separator missing")
			require.NotNilf(t, s.Err(), "expect error")
			assert.Truef(t, strings.Contains(s.Err().Error(), "empty"), "error should mention 'empty'")
			assert.EqualValuesf(t, s.Next(), "", "no content on error")
		})
		t.Run("ReaderError", func(t *testing.T) {
			t.Parallel()
			r := iotest.ErrReader(errors.New("connection reset"))
			s := NewScanner(r)

			assert.Falsef(t, s.Scan(), "want false on reader error")
			require.NotNilf(t, s.Err(), "expect error")
			assert.Truef(t, strings.Contains(s.Err().Error(), "connection reset"), "error should contain underlying cause")
		})
	})
}

func write(t *testing.T, w io.Writer, format string, args ...any) {
	t.Helper()
	_, err := fmt.Fprintf(w, format, args...)
	require.NoErrorf(t, err, "failed to write message")
}
