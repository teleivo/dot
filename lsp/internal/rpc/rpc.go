package rpc

import (
	"encoding/json"
	"fmt"
)

// TODO should I create a type ErrorCode int64 type? with these as there consts
// TODO do I need all of them? if so when do they make sense
const (
	ParseError     int64 = -32700
	InvalidRequest int64 = -32600
	MethodNotFound int64 = -32601
	InvalidParams  int64 = -32602
	InternalError  int64 = -32603
)

// Error code indicating that a server received a notification or request before the server received
// the `initialize` request.
const (
	ServerNotInitialized int64 = -32002
	UnknownErrorCode     int64 = -32001
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
	Code int64 `json:"code"`
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
