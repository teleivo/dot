package lsp

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/teleivo/assertive/assert"
	"github.com/teleivo/assertive/require"
)

func TestScanner(t *testing.T) {
	// for later initialize request
	// 	in := `Content-Length: 4182
	//
	// {"jsonrpc":"2.0","method":"initialize","id":1,"params":{"workDoneToken":"1","capabilities":{"textDocument":{"typeDefinition":{"linkSupport":true},"definition":{"linkSupport":true,"dynamicRegistration":true},"synchronization":{"willSaveWaitUntil":true,"dynamicRegistration":false,"willSave":true,"didSave":true},"rename":{"prepareSupport":true,"dynamicRegistration":true},"semanticTokens":{"dynamicRegistration":false,"requests":{"full":{"delta":true},"range":false},"augmentsSyntaxTokens":true,"serverCancelSupport":false,"multilineTokenSupport":false,"overlappingTokenSupport":true,"tokenModifiers":["declaration","definition","readonly","static","deprecated","abstract","async","modification","documentation","defaultLibrary"],"tokenTypes":["namespace","type","class","enum","interface","struct","typeParameter","parameter","variable","property","enumMember","event","function","method","macro","keyword","modifier","comment","string","number","regexp","operator","decorator"],"formats":["relative"]},"inlayHint":{"resolveSupport":{"properties":["textEdits","tooltip","location","command"]},"dynamicRegistration":true},"references":{"dynamicRegistration":false},"implementation":{"linkSupport":true},"callHierarchy":{"dynamicRegistration":false},"publishDiagnostics":{"dataSupport":true,"relatedInformation":true,"tagSupport":{"valueSet":[1,2]}},"hover":{"contentFormat":["markdown","plaintext"],"dynamicRegistration":true},"documentSymbol":{"dynamicRegistration":false,"symbolKind":{"valueSet":[1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22,23,24,25,26]},"hierarchicalDocumentSymbolSupport":true},"rangeFormatting":{"rangesSupport":true,"dynamicRegistration":true},"diagnostic":{"tagSupport":{"valueSet":[1,2]},"dynamicRegistration":false},"formatting":{"dynamicRegistration":true},"foldingRange":{"foldingRange":{"collapsedText":true},"lineFoldingOnly":true,"dynamicRegistration":false,"foldingRangeKind":{"valueSet":["comment","imports","region"]}},"documentHighlight":{"dynamicRegistration":false},"completion":{"completionItemKind":{"valueSet":[1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22,23,24,25]},"dynamicRegistration":false,"completionItem":{"snippetSupport":true,"resolveSupport":{"properties":["documentation","detail","additionalTextEdits","command","data"]},"deprecatedSupport":true,"tagSupport":{"valueSet":[1]},"commitCharactersSupport":false,"insertTextModeSupport":{"valueSet":[1]},"insertReplaceSupport":true,"documentationFormat":["markdown","plaintext"],"labelDetailsSupport":true,"preselectSupport":false},"completionList":{"itemDefaults":["commitCharacters","editRange","insertTextFormat","insertTextMode","data"]},"contextSupport":true,"insertTextMode":1},"declaration":{"linkSupport":true},"signatureHelp":{"signatureInformation":{"documentationFormat":["markdown","plaintext"],"activeParameterSupport":true,"parameterInformation":{"labelOffsetSupport":true}},"dynamicRegistration":false},"codeLens":{"resolveSupport":{"properties":["command"]},"dynamicRegistration":false},"codeAction":{"codeActionLiteralSupport":{"codeActionKind":{"valueSet":["","quickfix","refactor","refactor.extract","refactor.inline","refactor.rewrite","source","source.organizeImports"]}},"dynamicRegistration":true,"isPreferredSupport":true,"dataSupport":true,"resolveSupport":{"properties":["edit","command"]}}},"workspace":{"didChangeWatchedFiles":{"relativePatternSupport":true,"dynamicRegistration":false},"symbol":{"symbolKind":{"valueSet":[1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22,23,24,25,26]},"dynamicRegistration":false},"workspaceEdit":{"resourceOperations":["rename","create","delete"]},"didChangeConfiguration":{"dynamicRegistration":false},"applyEdit":true,"workspaceFolders":true,"configuration":true,"semanticTokens":{"refreshSupport":true},"inlayHint":{"refreshSupport":true}},"general":{"positionEncodings":["utf-8","utf-16","utf-32"]},"window":{"showDocument":{"support":true},"workDoneProgress":true,"showMessage":{"messageActionItem":{"additionalPropertiesSupport":true}}}},"workspaceFolders":null,"trace":"off","rootUri":null,"rootPath":null,"clientInfo":{"name":"Neovim","version":"0.11.5+v0.11.5"},"processId":92548}}`

	// TODO what are some errors I need to handle?
	// no headers?
	// no content-length
	// no Content-Type as its optional so no error
	// invalid Content-Type
	// not the amount of bytes specified in the content-length
	// no number in content-length
	t.Run("Success", func(t *testing.T) {
		t.Parallel()

		var w bytes.Buffer
		s := NewScanner(&w)

		msg1 := `{"jsonrpc":"2.0","method":"initialize","id":1,"params":null}`
		write(t, &w, "Content-Length: %d\r\n", len(msg1))
		write(t, &w, "\r\n")
		write(t, &w, "%s", msg1)

		assert.Truef(t, s.Scan(), "want true as msg1 is unread")
		require.EqualValuesf(t, s.Next(), msg1, "failed to read msg1")
		require.NoErrorf(t, s.Err(), "want no errors reading msg1")

		msg2 := `{"jsonrpc":"2.0","method":"initialized","id":2}`
		write(t, &w, "content-Length: %d\n", len(msg2))
		write(t, &w, "\n")
		write(t, &w, "%s", msg2)

		// TODO add content-type as well

		assert.Truef(t, s.Scan(), "want true as msg2 is unread")
		require.EqualValuesf(t, s.Next(), msg2, "failed to read msg2")
		require.NoErrorf(t, s.Err(), "want no errors reading msg2")

		assert.Falsef(t, s.Scan(), "want false as all msgs are read")
		assert.EqualValuesf(t, s.Next(), "", "should be empty")
		assert.NoErrorf(t, s.Err(), "want no errors reading all msgs")

		assert.Falsef(t, s.Scan(), "want false as all msgs are read")
		assert.EqualValuesf(t, s.Next(), "", "should be empty")
		assert.NoErrorf(t, s.Err(), "want no errors reading all msgs")
	})
}

func write(t *testing.T, w io.Writer, format string, args ...any) {
	t.Helper()
	_, err := fmt.Fprintf(w, format, args...)
	require.NoErrorf(t, err, "failed to write message")
}
