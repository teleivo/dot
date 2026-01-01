// Package rpc implements JSON-RPC 2.0 message types for the Language Server Protocol.
//
// This package provides types for encoding and decoding JSON-RPC 2.0 messages as specified
// in https://www.jsonrpc.org/specification, with extensions for the Language Server Protocol
// defined in https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/.
//
// The central type is [Message], which represents all JSON-RPC message types (requests,
// responses, and notifications) in a single struct. Message discrimination is based on field
// presence:
//   - Request: has ID and Method
//   - Response: has ID and either Result or Error
//   - Notification: has Method but no ID
package rpc

import (
	"encoding/json"
	"fmt"

	"github.com/teleivo/dot/internal/version"
)

// ErrorCode represents a JSON-RPC error code.
type ErrorCode int32

// JSON-RPC 2.0 standard error codes.
const (
	ParseError     ErrorCode = -32700
	InvalidRequest ErrorCode = -32600
	MethodNotFound ErrorCode = -32601
	InvalidParams  ErrorCode = -32602
	InternalError  ErrorCode = -32603
)

// LSP-specific error codes.
const (
	ServerNotInitialized ErrorCode = -32002
	UnknownErrorCode     ErrorCode = -32001
)

// LSP method names.
const (
	MethodInitialize = "initialize"
	MethodShutdown   = "shutdown"
	MethodExit       = "exit"

	MethodDidOpen   = "textDocument/didOpen"
	MethodDidChange = "textDocument/didChange"
	MethodDidClose  = "textDocument/didClose"

	MethodPublishDiagnostics = "textDocument/publishDiagnostics"
	MethodFormatting         = "textDocument/formatting"
	MethodCompletion         = "textDocument/completion"
)

// Message has all the fields of request, response and notification. Presence/absence of fields is
// used to discriminate which one it is. Unmarshaling of those discriminatory fields is deferred
// until we know which it is.
// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#abstractMessage
type Message struct {
	Version Version          `json:"jsonrpc"`
	ID      *ID              `json:"id,omitempty"`
	Method  string           `json:"method,omitempty"`
	Params  *json.RawMessage `json:"params,omitempty"`
	Result  *json.RawMessage `json:"result,omitempty"`
	Error   *Error           `json:"error,omitempty"`
}

// Error represents a structured error in a response.
type Error struct {
	// Code indicating the type of error.
	Code ErrorCode `json:"code"`
	// Message is a short description of the error.
	Message string `json:"message"`
	// Data is optional structured data containing additional information about the error.
	Data *json.RawMessage `json:"data,omitempty"`
}

// Version is a zero-sized struct that encodes as the jsonrpc version tag.
// It will fail during decode if it is not the correct version tag in the stream.
type Version struct{}

// MarshalJSON encodes the version as the JSON string "2.0".
func (Version) MarshalJSON() ([]byte, error) {
	return json.Marshal("2.0")
}

// UnmarshalJSON decodes the version and returns an error if it is not "2.0".
func (v *Version) UnmarshalJSON(data []byte) error {
	var version string
	if err := json.Unmarshal(data, &version); err != nil {
		return err
	}
	if version != "2.0" {
		return fmt.Errorf("invalid RPC version %q", version)
	}
	return nil
}

// ID is a request identifier that can be either a string or integer.
type ID struct {
	name   string
	number int64
}

// MarshalJSON encodes the ID as either a JSON string or number.
func (id *ID) MarshalJSON() ([]byte, error) {
	if id.name != "" {
		return json.Marshal(id.name)
	}
	return json.Marshal(id.number)
}

// UnmarshalJSON decodes a JSON string or number into the ID.
func (id *ID) UnmarshalJSON(data []byte) error {
	*id = ID{} // reset to support reusing ID in unmarshal
	if err := json.Unmarshal(data, &id.number); err == nil {
		return nil
	}
	return json.Unmarshal(data, &id.name)
}

// initializeResult is pre-computed at init time since it's static.
var initializeResult = func() json.RawMessage {
	result := map[string]any{
		"capabilities": map[string]any{
			"completionProvider": map[string]any{
				"triggerCharacters": []string{"[", ",", ";", "{", "="},
			},
			"documentFormattingProvider": true,
			"positionEncoding":           EncodingUTF8,
			"textDocumentSync":           SyncIncremental,
		},
		"serverInfo": map[string]any{
			"name":    "dotls",
			"version": version.Version(),
		},
	}
	b, err := json.Marshal(result)
	if err != nil {
		panic(err)
	}
	return b
}()

// InitializeResult returns the Result field for a successful initialize response.
// It includes the server's capabilities and metadata.
func InitializeResult() *json.RawMessage {
	return &initializeResult
}

// TextDocumentSyncKind defines how the host (editor) should sync document changes to the
// language server.
// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocumentSyncKind
type TextDocumentSyncKind int

const (
	// SyncNone means documents should not be synced at all.
	SyncNone TextDocumentSyncKind = 0
	// SyncFull means documents are synced by always sending the full content of the document.
	SyncFull TextDocumentSyncKind = 1
	// SyncIncremental means documents are synced by sending the full content on open, then only
	// incremental updates describing the changed range and replacement text.
	SyncIncremental TextDocumentSyncKind = 2
)

// PositionEncodingKind defines how character offsets are interpreted in positions.
// The encoding is negotiated during initialization: the client offers supported encodings,
// and the server picks one to use for the session.
// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#positionEncodingKind
type PositionEncodingKind string

const (
	// EncodingUTF8 means character offsets count UTF-8 code units (bytes).
	EncodingUTF8 PositionEncodingKind = "utf-8"
	// EncodingUTF16 means character offsets count UTF-16 code units.
	// This is the default and must always be supported by servers.
	EncodingUTF16 PositionEncodingKind = "utf-16"
	// EncodingUTF32 means character offsets count UTF-32 code units (Unicode code points).
	EncodingUTF32 PositionEncodingKind = "utf-32"
)

// DocumentURI represents a URI identifying a text document.
// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#documentUri
type DocumentURI string

// DidOpenTextDocumentParams contains the parameters for the textDocument/didOpen notification.
// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#didOpenTextDocumentParams
type DidOpenTextDocumentParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}

// TextDocumentItem represents an open text document with its content.
// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocumentItem
type TextDocumentItem struct {
	// URI is the text document's URI.
	URI DocumentURI `json:"uri"`
	// LanguageID is the text document's language identifier (e.g., "dot").
	LanguageID string `json:"languageId"`
	// Version is the version number of this document, incremented after each change.
	Version int32 `json:"version"`
	// Text is the content of the opened text document.
	Text string `json:"text"`
}

// DidChangeTextDocumentParams contains the parameters for the textDocument/didChange notification.
// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#didChangeTextDocumentParams
type DidChangeTextDocumentParams struct {
	// TextDocument identifies the document that changed. The version number points to the version
	// after all provided content changes have been applied.
	TextDocument VersionedTextDocumentIdentifier `json:"textDocument"`
	// ContentChanges contains the actual content changes. With TextDocumentSyncKind.Full,
	// this array contains a single element with the entire document content.
	ContentChanges []TextDocumentContentChangeEvent `json:"contentChanges"`
}

// VersionedTextDocumentIdentifier identifies a specific version of a text document.
// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#versionedTextDocumentIdentifier
type VersionedTextDocumentIdentifier struct {
	TextDocumentIdentifier
	// Version is the version number of this document. The version number increases after each
	// change, including undo/redo.
	Version int32 `json:"version"`
}

// TextDocumentIdentifier identifies a text document using a URI.
// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocumentIdentifier
type TextDocumentIdentifier struct {
	// URI is the text document's URI.
	URI DocumentURI `json:"uri"`
}

// DidCloseTextDocumentParams contains the parameters for the textDocument/didClose notification.
// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#didCloseTextDocumentParams
type DidCloseTextDocumentParams struct {
	// TextDocument is the document that was closed.
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

// DocumentFormattingParams contains the parameters for the textDocument/formatting request.
// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#documentFormattingParams
type DocumentFormattingParams struct {
	// TextDocument is the document to format.
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	// Options are the format options.
	Options FormattingOptions `json:"options"`
}

// FormattingOptions contains value-pairs describing format options.
// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#formattingOptions
type FormattingOptions struct {
	// TabSize is the size of a tab in spaces.
	TabSize uint32 `json:"tabSize"`
	// InsertSpaces indicates whether to prefer spaces over tabs.
	InsertSpaces bool `json:"insertSpaces"`
}

// TextEdit represents a textual edit applicable to a text document.
// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textEdit
type TextEdit struct {
	// Range is the range of the text document to be manipulated.
	Range Range `json:"range"`
	// NewText is the string to be inserted. For delete operations use an empty string.
	NewText string `json:"newText"`
}

// TextDocumentContentChangeEvent describes a change to a text document.
// When TextDocumentSyncKind.Full is used, only the Text field is set and it contains the full
// content of the document.
// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocumentContentChangeEvent
type TextDocumentContentChangeEvent struct {
	// Range is the range of the document that changed. Only present for incremental sync.
	Range *Range `json:"range,omitempty"`
	// RangeLength is the optional length of the range that got replaced. Deprecated: use Range.
	RangeLength *uint32 `json:"rangeLength,omitempty"`
	// Text is the new text for the provided range, or the full document content when Range is nil.
	Text string `json:"text"`
}

// PublishDiagnosticsParams is sent from the server to the client to signal results of validation.
// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#publishDiagnosticsParams
type PublishDiagnosticsParams struct {
	URI         DocumentURI  `json:"uri"`
	Version     *int32       `json:"version,omitempty"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}

// Diagnostic represents a diagnostic, such as a compiler error or warning.
// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#diagnostic
type Diagnostic struct {
	Range    Range               `json:"range"`
	Severity *DiagnosticSeverity `json:"severity,omitempty"`
	Message  string              `json:"message"`
}

// DiagnosticSeverity indicates the severity of a diagnostic.
// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#diagnosticSeverity
type DiagnosticSeverity int32

// Diagnostic severity levels.
const (
	SeverityError       DiagnosticSeverity = 1
	SeverityWarning     DiagnosticSeverity = 2
	SeverityInformation DiagnosticSeverity = 3
	SeverityHint        DiagnosticSeverity = 4
)

// Range represents a range in a text document.
// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#range
type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// Position represents a position in a text document (zero-based line and character).
// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#position
type Position struct {
	Line      uint32 `json:"line"`
	Character uint32 `json:"character"`
}

// CompletionParams contains the parameters for the textDocument/completion request.
// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#completionParams
type CompletionParams struct {
	// TextDocument is the text document.
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	// Position is the position inside the text document.
	Position Position `json:"position"`
	// Context is the completion context. Only available if the client specifies contextSupport.
	Context *CompletionContext `json:"context,omitempty"`
}

// CompletionContext contains additional information about the context in which a completion
// request is triggered.
// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#completionContext
type CompletionContext struct {
	// TriggerKind describes how the completion was triggered.
	TriggerKind CompletionTriggerKind `json:"triggerKind"`
	// TriggerCharacter is the trigger character that has trigger code complete.
	// Only set if TriggerKind is TriggerCharacter.
	TriggerCharacter *string `json:"triggerCharacter,omitempty"`
}

// CompletionTriggerKind describes how a completion was triggered.
// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#completionTriggerKind
type CompletionTriggerKind int

const (
	// TriggerInvoked means completion was triggered by typing an identifier, manual invocation
	// (e.g. Ctrl+Space) or via API.
	TriggerInvoked CompletionTriggerKind = 1
	// TriggerCharacter means completion was triggered by a trigger character specified by the
	// triggerCharacters property of the CompletionOptions.
	TriggerCharacter CompletionTriggerKind = 2
	// TriggerForIncompleteCompletions means completion was re-triggered as the current completion
	// list is incomplete.
	TriggerForIncompleteCompletions CompletionTriggerKind = 3
)

// CompletionList represents a collection of completion items to be presented in the editor.
// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#completionList
type CompletionList struct {
	// IsIncomplete indicates this list is not complete. Further typing should result in
	// recomputing this list.
	IsIncomplete bool `json:"isIncomplete"`
	// Items are the completion items.
	Items []CompletionItem `json:"items"`
}

// CompletionItem represents a completion item to be presented in the editor.
// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#completionItem
type CompletionItem struct {
	// Label is the label of this completion item. Also the text that is inserted when selecting
	// this completion unless insertText is provided.
	Label string `json:"label"`
	// Kind is the kind of this completion item. Based on the kind an icon is chosen by the editor.
	Kind *CompletionItemKind `json:"kind,omitempty"`
	// Detail is a human-readable string with additional information about this item,
	// like type or symbol information.
	Detail *string `json:"detail,omitempty"`
	// Documentation is a human-readable string or MarkupContent that represents a doc-comment.
	Documentation *MarkupContent `json:"documentation,omitempty"`
	// InsertText is a string that should be inserted into a document when selecting this
	// completion. When omitted the label is used.
	InsertText *string `json:"insertText,omitempty"`
}

// MarkupContent represents a string value with a specific format (plaintext or markdown).
// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#markupContent
type MarkupContent struct {
	// Kind is the format of the content ("plaintext" or "markdown").
	Kind string `json:"kind"`
	// Value is the content itself.
	Value string `json:"value"`
}

// CompletionItemKind is the kind of a completion entry.
// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#completionItemKind
type CompletionItemKind int

const (
	CompletionItemKindText          CompletionItemKind = 1
	CompletionItemKindMethod        CompletionItemKind = 2
	CompletionItemKindFunction      CompletionItemKind = 3
	CompletionItemKindConstructor   CompletionItemKind = 4
	CompletionItemKindField         CompletionItemKind = 5
	CompletionItemKindVariable      CompletionItemKind = 6
	CompletionItemKindClass         CompletionItemKind = 7
	CompletionItemKindInterface     CompletionItemKind = 8
	CompletionItemKindModule        CompletionItemKind = 9
	CompletionItemKindProperty      CompletionItemKind = 10
	CompletionItemKindUnit          CompletionItemKind = 11
	CompletionItemKindValue         CompletionItemKind = 12
	CompletionItemKindEnum          CompletionItemKind = 13
	CompletionItemKindKeyword       CompletionItemKind = 14
	CompletionItemKindSnippet       CompletionItemKind = 15
	CompletionItemKindColor         CompletionItemKind = 16
	CompletionItemKindFile          CompletionItemKind = 17
	CompletionItemKindReference     CompletionItemKind = 18
	CompletionItemKindFolder        CompletionItemKind = 19
	CompletionItemKindEnumMember    CompletionItemKind = 20
	CompletionItemKindConstant      CompletionItemKind = 21
	CompletionItemKindStruct        CompletionItemKind = 22
	CompletionItemKindEvent         CompletionItemKind = 23
	CompletionItemKindOperator      CompletionItemKind = 24
	CompletionItemKindTypeParameter CompletionItemKind = 25
)
