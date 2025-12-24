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
		return fmt.Errorf("invalid RPC version %v", version)
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

// InitializeResult returns the server's response to an initialize request.
// It includes the server's capabilities and metadata.
func InitializeResult() *json.RawMessage {
	init := map[string]any{
		"capabilities": map[string]any{
			"textDocumentSync": 1,
		},
		"serverInfo": map[string]any{
			"name":    "dotls",
			"version": version.Version(),
		},
	}
	b, err := json.Marshal(init)
	if err != nil {
		panic(err)
	}
	result := json.RawMessage(b)
	return &result
}

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
type DiagnosticSeverity int32

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
