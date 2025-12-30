package rpc

import (
	"bufio"
	"fmt"
	"io"
)

// Writer writes JSON-RPC messages to an [io.Writer] using the base protocol framing.
// The base protocol consists of a header and content part, where the header is separated
// from the content by an empty line (\r\n).
//
// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#baseProtocol
type Writer struct {
	w *bufio.Writer
}

// NewWriter returns a new Writer that writes to w.
func NewWriter(w io.Writer) *Writer {
	return &Writer{w: bufio.NewWriter(w)}
}

// Write writes a JSON-RPC message with the appropriate Content-Length header.
// The content should be a valid JSON-RPC message (request, response, or notification).
func (w *Writer) Write(content []byte) error {
	_, err := fmt.Fprintf(w.w, "Content-Length: %d\r\n\r\n", len(content))
	if err != nil {
		return err
	}
	_, err = w.w.Write(content)
	if err != nil {
		return err
	}
	return w.w.Flush()
}
