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

	return nil
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
					srv.write(cancel, rpc.Message{ID: message.ID, Error: &rpc.Error{Code: rpc.ServerNotInitialized, Message: "server not initialized"}})
				}
			case initialized:
				switch message.Method {
				case rpc.MethodInitialize:
					if message.ID == nil {
						srv.logger.Error("missing request id", "method", message.Method)
						continue
					}
					srv.write(cancel, rpc.Message{ID: message.ID, Error: &rpc.Error{Code: rpc.InvalidRequest, Message: "server already initialized"}})
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
					if message.Params == nil {
						srv.logger.Error("missing params", "method", message.Method)
						continue
					}
					var params rpc.DidOpenTextDocumentParams
					if err := json.Unmarshal(*message.Params, &params); err != nil {
						srv.logger.Error("invalid params", "method", message.Method, "err", err)
						continue
					}
					doc := newDocument(params.TextDocument)
					srv.docs[params.TextDocument.URI] = doc
					response, err := diagnostics(doc)
					if err != nil {
						srv.logger.Error("diagnostics failed", "method", message.Method, "uri", doc.uri, "err", err)
						continue
					}
					srv.write(cancel, response)
				case rpc.MethodDidChange:
					if message.Params == nil {
						srv.logger.Error("missing params", "method", message.Method)
						continue
					}
					var params rpc.DidChangeTextDocumentParams
					if err := json.Unmarshal(*message.Params, &params); err != nil {
						srv.logger.Error("invalid params", "method", message.Method, "err", err)
						continue
					}
					if len(params.ContentChanges) == 0 {
						srv.logger.Error("no content changes", "method", message.Method, "uri", params.TextDocument.URI)
						continue
					}

					doc, ok := srv.docs[params.TextDocument.URI]
					if !ok {
						srv.logger.Error("unknown document", "uri", params.TextDocument.URI)
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
					response, err := diagnostics(doc)
					if err != nil {
						srv.logger.Error("diagnostics failed", "method", message.Method, "uri", doc.uri, "err", err)
						continue
					}
					srv.write(cancel, response)
				case rpc.MethodFormatting:
					if message.Params == nil {
						srv.logger.Error("missing params", "method", message.Method)
						continue
					}
					var params rpc.DocumentFormattingParams
					if err := json.Unmarshal(*message.Params, &params); err != nil {
						srv.logger.Error("invalid params", "method", message.Method, "err", err)
						continue
					}

					doc, ok := srv.docs[params.TextDocument.URI]
					if !ok {
						srv.logger.Error("unknown document", "uri", params.TextDocument.URI)
						continue
					}
					var text strings.Builder
					p := printer.New(doc.src, &text, layout.Default)
					if err := p.Print(); err != nil {
						srv.write(cancel, rpc.Message{ID: message.ID, Error: &rpc.Error{Code: rpc.InternalError, Message: fmt.Sprintf("formatting failed: %v", err)}})
						continue
					}
					start := doc.startPos()
					end := doc.endPos()

					response := rpc.Message{
						ID: message.ID,
					}
					edits := []rpc.TextEdit{
						{
							Range: rpc.Range{
								Start: start,
								End:   end,
							},
							NewText: text.String(),
						},
					}
					rp, err := json.Marshal(edits)
					if err != nil {
						srv.logger.Error("formatting failed due to marshaling edits", "method", message.Method, "err", err)
						continue
					}
					rm := json.RawMessage(rp)
					response.Result = &rm

					srv.write(cancel, response)
				case rpc.MethodCompletion:
					if message.Params == nil {
						srv.logger.Error("missing params", "method", message.Method)
						continue
					}
					var params rpc.CompletionParams
					if err := json.Unmarshal(*message.Params, &params); err != nil {
						srv.logger.Error("invalid params", "method", message.Method, "err", err)
						continue
					}
					doc, ok := srv.docs[params.TextDocument.URI]
					if !ok {
						srv.logger.Error("unknown document", "uri", params.TextDocument.URI)
						continue
					}

					// TODO cache the tree in the document
					ps := dot.NewParser(doc.src)
					tree := ps.Parse()
					items := completion.Items(tree, tokenPosition(params.Position))

					response := rpc.Message{
						ID: message.ID,
					}
					completions := rpc.CompletionList{
						Items: items,
					}
					rp, err := json.Marshal(completions)
					if err != nil {
						srv.logger.Error("formatting failed due to marshaling edits", "method", message.Method, "err", err)
						continue
					}
					rm := json.RawMessage(rp)
					response.Result = &rm

					srv.write(cancel, response)
				case rpc.MethodDidClose:
					if message.Params == nil {
						srv.logger.Error("missing params", "method", message.Method)
						continue
					}
					var params rpc.DidCloseTextDocumentParams
					if err := json.Unmarshal(*message.Params, &params); err != nil {
						srv.logger.Error("invalid params", "method", message.Method, "err", err)
						continue
					}
					_, ok := srv.docs[params.TextDocument.URI]
					if !ok {
						srv.logger.Error("unknown document", "uri", params.TextDocument.URI)
						continue
					}
					delete(srv.docs, params.TextDocument.URI)
				default:
					if message.ID == nil { // notifications are ignored
						continue
					}
					srv.write(cancel, rpc.Message{ID: message.ID, Error: &rpc.Error{Code: rpc.MethodNotFound, Message: "method not found"}})
				}
			case shuttingDown:
				switch message.Method {
				case rpc.MethodExit:
					srv.logger.Debug("exit notification received")
					cancel(nil)
				default:
					srv.write(cancel, rpc.Message{ID: message.ID, Error: &rpc.Error{Code: rpc.InvalidRequest, Message: "server is shutting down"}})
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

func diagnostics(doc *document) (rpc.Message, error) {
	ps := dot.NewParser(doc.src)
	ps.Parse()

	response := rpc.Message{
		Method: rpc.MethodPublishDiagnostics,
	}
	responseParams := rpc.PublishDiagnosticsParams{
		URI:     doc.uri,
		Version: &doc.version,
	}
	sev := rpc.SeverityError
	errs := ps.Errors()
	responseParams.Diagnostics = make([]rpc.Diagnostic, len(errs))
	for i, err := range errs {
		responseParams.Diagnostics[i] = rpc.Diagnostic{
			Range: rpc.Range{
				Start: rpc.Position{
					Line:      uint32(err.Pos.Line) - 1,
					Character: uint32(err.Pos.Column) - 1,
				},
				End: rpc.Position{
					Line:      uint32(err.Pos.Line) - 1,
					Character: uint32(err.Pos.Column) - 1,
				},
			},
			Severity: &sev,
			Message:  err.Msg,
		}
	}
	rp, err := json.Marshal(responseParams)
	if err != nil {
		return response, err
	}
	rm := json.RawMessage(rp)
	response.Params = &rm

	return response, nil
}
