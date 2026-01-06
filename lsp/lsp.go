// Package lsp implements a Language Server Protocol server for the DOT graph language.
//
// The server provides diagnostics for DOT files, reporting parse errors to the client.
// It implements the LSP 3.17 specification:
// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/
package lsp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/teleivo/dot"
	"github.com/teleivo/dot/internal/layout"
	"github.com/teleivo/dot/lsp/internal/completion"
	"github.com/teleivo/dot/lsp/internal/diagnostic"
	"github.com/teleivo/dot/lsp/internal/hover"
	"github.com/teleivo/dot/lsp/internal/navigate"
	"github.com/teleivo/dot/lsp/internal/rpc"
	"github.com/teleivo/dot/printer"
	"github.com/teleivo/dot/token"
)

// Config holds the configuration for creating an LSP server.
type Config struct {
	In    io.Reader // input for LSP messages
	Out   io.Writer // output for LSP messages
	Debug bool      // enable debug logging
	Log   io.Writer // output for logging
	Trace io.Writer // output for JSON-RPC message traffic (nil to disable)
}

type Server struct {
	in     *rpc.Scanner
	out    *rpc.Writer
	logger *slog.Logger
	state  state
	docs   map[rpc.DocumentURI]*document
}

// New creates an LSP server with the given configuration.
func New(cfg Config) (*Server, error) {
	level := slog.LevelInfo
	if cfg.Debug {
		level = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(cfg.Log, &slog.HandlerOptions{Level: level}))

	in := cfg.In
	out := cfg.Out
	if cfg.Trace != nil {
		in = io.TeeReader(cfg.In, cfg.Trace)
		out = io.MultiWriter(cfg.Out, cfg.Trace)
	}

	srv := &Server{
		in:     rpc.NewScanner(in),
		out:    rpc.NewWriter(out),
		logger: logger,
		docs:   make(map[rpc.DocumentURI]*document),
	}
	return srv, nil
}

type state int

const (
	uninitialized state = iota
	initialized
	shuttingDown
)

type document struct {
	uri     rpc.DocumentURI
	version int32
	src     []byte
	lines   []int
	tree    *dot.Tree   // cached parse result, nil if stale
	errors  []dot.Error // cached parse errors from tree
}

func newDocument(item rpc.TextDocumentItem) *document {
	src := []byte(item.Text)
	lines := buildLines(src)
	return &document{uri: item.URI, version: item.Version, src: src, lines: lines}
}

func buildLines(src []byte) []int {
	lines := []int{0: 0}
	for i, r := range src {
		if r == '\n' {
			lines = append(lines, i+1) // start of the next line
		}
	}
	return lines
}

func (d *document) offset(pos rpc.Position) (int, error) {
	if int(pos.Line) >= len(d.lines) {
		return 0, fmt.Errorf("line %d not in document (%d lines)", pos.Line, len(d.lines))
	}
	lineStart := d.lines[pos.Line]
	offset := lineStart + int(pos.Character)
	if offset > len(d.src) {
		return 0, fmt.Errorf("offset %d beyond document length %d", offset, len(d.src))
	}
	return offset, nil
}

func (d *document) change(change rpc.TextDocumentContentChangeEvent) error {
	start, err := d.offset(change.Range.Start)
	if err != nil {
		return fmt.Errorf("invalid start position: %v", err)
	}
	end, err := d.offset(change.Range.End)
	if err != nil {
		return fmt.Errorf("invalid end position: %v", err)
	}
	if start > end {
		return fmt.Errorf("start offset %d > end offset %d", start, end)
	}

	text := change.Text
	newSrc := make([]byte, start+len(text)+len(d.src)-end)
	copy(newSrc, d.src[:start])
	copy(newSrc[start:], text)
	copy(newSrc[start+len(text):], d.src[end:])
	d.src = newSrc
	d.lines = buildLines(d.src)
	d.tree = nil   // invalidate cached tree
	d.errors = nil // invalidate cached errors

	return nil
}

// parse parses the document if needed, caching both tree and errors.
func (d *document) parse() {
	if d.tree == nil {
		ps := dot.NewParser(d.src)
		d.tree = ps.Parse()
		d.errors = ps.Errors()
	}
}

// Tree returns the cached parse tree, parsing the document if needed.
func (d *document) Tree() *dot.Tree {
	d.parse()
	return d.tree
}

// Errors returns the cached parse errors, parsing the document if needed.
func (d *document) Errors() []dot.Error {
	d.parse()
	return d.errors
}

func (d *document) startPos() rpc.Position {
	return rpc.Position{
		Line:      0,
		Character: 0,
	}
}

func (d *document) endPos() rpc.Position {
	return rpc.Position{
		Line:      uint32(len(d.lines)) - 1,
		Character: uint32(len(d.src) - d.lines[len(d.lines)-1]),
	}
}

func tokenPosition(pos rpc.Position) token.Position {
	return token.Position{
		Line:   int(pos.Line) + 1,
		Column: int(pos.Character) + 1,
	}
}

func (srv *Server) write(cancel context.CancelCauseFunc, msg rpc.Message) {
	content, err := json.Marshal(msg)
	if err != nil {
		srv.logger.Error("failed to marshal response", "err", err)
		return
	}
	if err := srv.out.Write(content); err != nil {
		srv.logger.Error("failed to write response", "err", err)
		cancel(err)
	}
}

// writeResult marshals the result and sends it as a response.
func (srv *Server) writeResult(cancel context.CancelCauseFunc, id *rpc.ID, result any) {
	rp, err := json.Marshal(result)
	if err != nil {
		srv.logger.Error("failed to marshal result", "err", err)
		return
	}
	rm := json.RawMessage(rp)
	srv.write(cancel, rpc.Message{ID: id, Result: &rm})
}

// writeErr sends an error response for the given request ID.
func (srv *Server) writeErr(cancel context.CancelCauseFunc, id *rpc.ID, err *rpc.Error) {
	srv.write(cancel, rpc.Message{ID: id, Error: err})
}

func (srv *Server) publishDiagnostics(cancel context.CancelCauseFunc, doc *document) {
	params := diagnostic.Compute(doc.Errors(), doc.uri, doc.version)
	rp, err := json.Marshal(params)
	if err != nil {
		srv.logger.Error("failed to marshal diagnostics", "uri", doc.uri, "err", err)
		return
	}
	rm := json.RawMessage(rp)
	srv.write(cancel, rpc.Message{
		Method: rpc.MethodPublishDiagnostics,
		Params: &rm,
	})
}

// unmarshalParams unmarshals JSON-RPC params into the given type.
// Returns the parsed params and nil on success, or zero value and an rpc.Error on failure.
func unmarshalParams[T any](params *json.RawMessage) (T, *rpc.Error) {
	var result T
	if params == nil {
		return result, &rpc.Error{Code: rpc.InvalidParams, Message: "missing params"}
	}
	if err := json.Unmarshal(*params, &result); err != nil {
		return result, &rpc.Error{Code: rpc.InvalidParams, Message: err.Error()}
	}
	return result, nil
}

// getDoc retrieves a document by URI, returning nil and an rpc.Error if not found.
func (srv *Server) getDoc(uri rpc.DocumentURI) (*document, *rpc.Error) {
	doc, ok := srv.docs[uri]
	if !ok {
		return nil, &rpc.Error{Code: rpc.InvalidParams, Message: fmt.Sprintf("unknown document: %s", uri)}
	}
	return doc, nil
}

// Start runs the server's main loop, processing LSP messages until the context is cancelled.
// Note: If the context is cancelled externally (e.g., via SIGTERM), the goroutine reading from
// input will remain blocked on Scan() until the process exits. This is acceptable since the
// process terminates shortly after. For clean shutdown, use the LSP shutdown/exit sequence.
func (srv *Server) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancelCause(ctx)
	go func() {
		for srv.in.Scan() {
			select {
			case <-ctx.Done():
				return
			default:
			}

			var message rpc.Message
			if err := json.Unmarshal(srv.in.Bytes(), &message); err != nil {
				srv.write(cancel, rpc.Message{
					Error: &rpc.Error{Code: rpc.ParseError, Message: "invalid JSON"},
				})
				continue
			}

			switch srv.state {
			case uninitialized:
				if message.Method == rpc.MethodInitialize {
					if message.ID == nil {
						srv.logger.Error("missing request id", "method", message.Method)
						continue
					}
					srv.state = initialized
					srv.write(cancel, rpc.Message{ID: message.ID, Result: rpc.InitializeResult()})
				} else if message.ID != nil {
					srv.writeErr(cancel, message.ID, &rpc.Error{Code: rpc.ServerNotInitialized, Message: "server not initialized"})
				}
			case initialized:
				switch message.Method {
				case rpc.MethodInitialize:
					if message.ID == nil {
						srv.logger.Error("missing request id", "method", message.Method)
						continue
					}
					srv.writeErr(cancel, message.ID, &rpc.Error{Code: rpc.InvalidRequest, Message: "server already initialized"})
				case rpc.MethodShutdown:
					if message.ID == nil {
						srv.logger.Error("missing request id", "method", message.Method)
						continue
					}
					srv.state = shuttingDown
					nullResult := json.RawMessage("null")
					srv.write(cancel, rpc.Message{ID: message.ID, Result: &nullResult})
					srv.logger.Debug("shutdown", "id", *message.ID)
				case rpc.MethodDidOpen:
					params, rpcErr := unmarshalParams[rpc.DidOpenTextDocumentParams](message.Params)
					if rpcErr != nil {
						srv.logger.Error("invalid notification", "method", message.Method, "err", rpcErr.Message)
						continue
					}
					doc := newDocument(params.TextDocument)
					srv.docs[params.TextDocument.URI] = doc
					srv.publishDiagnostics(cancel, doc)
				case rpc.MethodDidChange:
					params, rpcErr := unmarshalParams[rpc.DidChangeTextDocumentParams](message.Params)
					if rpcErr != nil {
						srv.logger.Error("invalid notification", "method", message.Method, "err", rpcErr.Message)
						continue
					}
					if len(params.ContentChanges) == 0 {
						srv.logger.Error("no content changes", "method", message.Method, "uri", params.TextDocument.URI)
						continue
					}
					doc, rpcErr := srv.getDoc(params.TextDocument.URI)
					if rpcErr != nil {
						srv.logger.Error("invalid notification", "method", message.Method, "err", rpcErr.Message)
						continue
					}
					for _, change := range params.ContentChanges {
						// change positions depend on each other so we exit on first error to avoid
						// cascading errors
						if err := doc.change(change); err != nil {
							srv.logger.Error("invalid change",
								"uri", params.TextDocument.URI, "err", err)
							break
						}
					}
					doc.version = params.TextDocument.Version
					srv.publishDiagnostics(cancel, doc)
				case rpc.MethodFormatting:
					if message.ID == nil {
						srv.logger.Error("missing request id", "method", message.Method)
						continue
					}
					params, rpcErr := unmarshalParams[rpc.DocumentFormattingParams](message.Params)
					if rpcErr != nil {
						srv.writeErr(cancel, message.ID, rpcErr)
						continue
					}
					doc, rpcErr := srv.getDoc(params.TextDocument.URI)
					if rpcErr != nil {
						srv.writeErr(cancel, message.ID, rpcErr)
						continue
					}
					var text strings.Builder
					p := printer.New(doc.src, &text, layout.Default)
					if err := p.Print(); err != nil {
						srv.writeErr(cancel, message.ID, &rpc.Error{Code: rpc.InternalError, Message: fmt.Sprintf("formatting failed: %v", err)})
						continue
					}
					edits := []rpc.TextEdit{
						{
							Range:   rpc.Range{Start: doc.startPos(), End: doc.endPos()},
							NewText: text.String(),
						},
					}
					srv.writeResult(cancel, message.ID, edits)
				case rpc.MethodCompletion:
					if message.ID == nil {
						srv.logger.Error("missing request id", "method", message.Method)
						continue
					}
					params, rpcErr := unmarshalParams[rpc.CompletionParams](message.Params)
					if rpcErr != nil {
						srv.writeErr(cancel, message.ID, rpcErr)
						continue
					}
					doc, rpcErr := srv.getDoc(params.TextDocument.URI)
					if rpcErr != nil {
						srv.writeErr(cancel, message.ID, rpcErr)
						continue
					}
					items := completion.Items(doc.Tree(), tokenPosition(params.Position))
					srv.writeResult(cancel, message.ID, rpc.CompletionList{Items: items})
				case rpc.MethodHover:
					if message.ID == nil {
						srv.logger.Error("missing request id", "method", message.Method)
						continue
					}
					params, rpcErr := unmarshalParams[rpc.HoverParams](message.Params)
					if rpcErr != nil {
						srv.writeErr(cancel, message.ID, rpcErr)
						continue
					}
					doc, rpcErr := srv.getDoc(params.TextDocument.URI)
					if rpcErr != nil {
						srv.writeErr(cancel, message.ID, rpcErr)
						continue
					}
					srv.writeResult(cancel, message.ID, hover.Info(doc.Tree(), tokenPosition(params.Position)))
				case rpc.MethodDocumentSymbol:
					if message.ID == nil {
						srv.logger.Error("missing request id", "method", message.Method)
						continue
					}
					params, rpcErr := unmarshalParams[rpc.DocumentSymbolParams](message.Params)
					if rpcErr != nil {
						srv.writeErr(cancel, message.ID, rpcErr)
						continue
					}
					doc, rpcErr := srv.getDoc(params.TextDocument.URI)
					if rpcErr != nil {
						srv.writeErr(cancel, message.ID, rpcErr)
						continue
					}
					srv.writeResult(cancel, message.ID, navigate.DocumentSymbols(doc.Tree()))
				case rpc.MethodDefinition:
					if message.ID == nil {
						srv.logger.Error("missing request id", "method", message.Method)
						continue
					}
					params, rpcErr := unmarshalParams[rpc.DefinitionParams](message.Params)
					if rpcErr != nil {
						srv.writeErr(cancel, message.ID, rpcErr)
						continue
					}
					doc, rpcErr := srv.getDoc(params.TextDocument.URI)
					if rpcErr != nil {
						srv.writeErr(cancel, message.ID, rpcErr)
						continue
					}
					srv.writeResult(cancel, message.ID, navigate.Definition(doc.Tree(), params.TextDocument.URI, tokenPosition(params.Position)))
				case rpc.MethodDidClose:
					params, rpcErr := unmarshalParams[rpc.DidCloseTextDocumentParams](message.Params)
					if rpcErr != nil {
						srv.logger.Error("invalid notification", "method", message.Method, "err", rpcErr.Message)
						continue
					}
					if _, rpcErr := srv.getDoc(params.TextDocument.URI); rpcErr != nil {
						srv.logger.Error("invalid notification", "method", message.Method, "err", rpcErr.Message)
						continue
					}
					delete(srv.docs, params.TextDocument.URI)
				default:
					if message.ID == nil { // notifications are ignored
						continue
					}
					srv.writeErr(cancel, message.ID, &rpc.Error{Code: rpc.MethodNotFound, Message: "method not found"})
				}
			case shuttingDown:
				switch message.Method {
				case rpc.MethodExit:
					srv.logger.Debug("exit notification received")
					cancel(nil)
				default:
					if message.ID == nil { // notifications are ignored
						continue
					}
					srv.writeErr(cancel, message.ID, &rpc.Error{Code: rpc.InvalidRequest, Message: "server is shutting down"})
				}
			}
		}
		if err := srv.in.Err(); err != nil {
			cancel(err)
		} else {
			cancel(nil)
		}
	}()

	<-ctx.Done()
	srv.logger.Debug("shutting down")
	if err := context.Cause(ctx); !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}

