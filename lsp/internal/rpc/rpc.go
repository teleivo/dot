// Package rpc implements JSON-RPC 2.0 message types for the Language Server Protocol.
package rpc

import (
	"encoding/json"
	"fmt"

	"github.com/teleivo/dot/internal/version"
)

// ErrorCode represents a JSON-RPC error code.
type ErrorCode int64

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
	Method  string           `json:"method"`
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

func (Version) MarshalJSON() ([]byte, error) {
	return json.Marshal("2.0")
}

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

func (id *ID) MarshalJSON() ([]byte, error) {
	if id.name != "" {
		return json.Marshal(id.name)
	}
	return json.Marshal(id.number)
}

func (id *ID) UnmarshalJSON(data []byte) error {
	*id = ID{} // reset to support reusing ID in unmarshal
	if err := json.Unmarshal(data, &id.number); err == nil {
		return nil
	}
	return json.Unmarshal(data, &id.name)
}

func InitializeResult() *json.RawMessage {
	// result := json.RawMessage(`{"capabilities":{"textDocumentSync":1},"serverInfo":{"name":"dotls","version":"0.1.0"}}`)
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
