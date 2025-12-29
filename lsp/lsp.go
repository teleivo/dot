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
	"io"
	"log/slog"

	"github.com/teleivo/dot"
	"github.com/teleivo/dot/lsp/internal/rpc"
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
	}
	return srv, nil
}

type state int

const (
	uninitialized state = iota
	initialized
	shuttingDown
)

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
				if message.Method == "initialize" {
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
				case "initialize":
					if message.ID == nil {
						srv.logger.Error("missing request id", "method", message.Method)
						continue
					}
					srv.write(cancel, rpc.Message{ID: message.ID, Error: &rpc.Error{Code: rpc.InvalidRequest, Message: "server already initialized"}})
				case "shutdown":
					if message.ID == nil {
						srv.logger.Error("missing request id", "method", message.Method)
						continue
					}
					srv.state = shuttingDown
					nullResult := json.RawMessage("null")
					srv.write(cancel, rpc.Message{ID: message.ID, Result: &nullResult})
					srv.logger.Debug("shutdown message received")
				case "textDocument/didOpen":
					if message.Params == nil {
						srv.logger.Error("missing params", "method", message.Method)
						continue
					}
					var params rpc.DidOpenTextDocumentParams
					if err := json.Unmarshal(*message.Params, &params); err != nil {
						srv.logger.Error("invalid params", "method", message.Method, "err", err)
						continue
					}
					response, err := diagnostics(params.TextDocument.URI, params.TextDocument.Text)
					if err != nil {
						srv.logger.Error("diagnostics failed", "method", message.Method, "err", err)
						continue
					}
					srv.write(cancel, response)
				case "textDocument/didChange":
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
						srv.logger.Error("no content changes", "method", message.Method)
						continue
					}
					response, err := diagnostics(params.TextDocument.URI, params.ContentChanges[0].Text)
					if err != nil {
						srv.logger.Error("diagnostics failed", "method", message.Method, "err", err)
						continue
					}
					srv.write(cancel, response)
				default:
					if message.ID == nil { // notifications are ignored
						continue
					}
					srv.write(cancel, rpc.Message{ID: message.ID, Error: &rpc.Error{Code: rpc.MethodNotFound, Message: "method not found"}})
				}
			case shuttingDown:
				switch message.Method {
				case "exit":
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
		cancel(err)
	}
}

func diagnostics(uri rpc.DocumentURI, text string) (rpc.Message, error) {
	var response rpc.Message

	ps := dot.NewParser([]byte(text))
	ps.Parse()

	response.Method = "textDocument/publishDiagnostics"
	responseParams := rpc.PublishDiagnosticsParams{
		URI: uri,
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
	puf, err := json.Marshal(responseParams)
	if err != nil {
		return response, err
	}
	rm := json.RawMessage(puf)
	response.Params = &rm

	return response, nil
}
