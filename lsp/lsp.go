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
	"strings"

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
func (srv *Server) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancelCause(ctx)
	go func() {
		for srv.in.Scan() {
			var message rpc.Message
			if err := json.Unmarshal(srv.in.Bytes(), &message); err != nil {
				// nullResult := json.RawMessage("null")
				srv.write(cancel, rpc.Message{
					// ID:    &nullResult,
					Error: &rpc.Error{Code: rpc.ParseError, Message: "invalid JSON"},
				})
				continue
			}

			switch srv.state {
			case uninitialized:
				if message.Method == "initialize" {
					// TODO do I need to wait on the initialized notification to set the state? or
					// can I just ignore if it is sent or not
					srv.state = initialized
					srv.write(cancel, rpc.Message{ID: message.ID, Result: rpc.InitializeResult()})
				} else {
					srv.write(cancel, rpc.Message{ID: message.ID, Error: &rpc.Error{Code: rpc.ServerNotInitialized, Message: "server not initialized"}})
				}
			case initialized:
				switch message.Method {
				case "initialize":
					// TODO expect this to be a method so it must have an id
					srv.write(cancel, rpc.Message{ID: message.ID, Error: &rpc.Error{Code: rpc.InvalidRequest, Message: "server already initialized"}})
				case "shutdown":
					// TODO expect this to be a method so it must have an id
					srv.state = shuttingDown
					nullResult := json.RawMessage("null")
					srv.write(cancel, rpc.Message{ID: message.ID, Result: &nullResult})
					srv.logger.Debug("shutdown message received")
				case "textDocument/didOpen":
					if message.Params == nil {
						panic("TODO handle")
					}
					var requestParams rpc.DidOpenTextDocumentParams
					err := json.Unmarshal(*message.Params, &requestParams)
					if err != nil {
						panic("TODO handle")
					} else {
						response, err := diagnostics(requestParams.TextDocument.URI, requestParams.TextDocument.Text)
						if err != nil {
							panic("TODO handle")
						}
						srv.write(cancel, response)
					}
				case "textDocument/didChange":
					if message.Params == nil {
						panic("TODO handle")
					}
					var requestParams rpc.DidChangeTextDocumentParams
					err := json.Unmarshal(*message.Params, &requestParams)
					if err != nil {
						panic("TODO handle")
					} else {
						response, err := diagnostics(requestParams.TextDocument.URI, requestParams.ContentChanges[0].Text)
						if err != nil {
							panic("TODO handle")
						}
						srv.write(cancel, response)
					}
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
	}()

	<-ctx.Done()
	srv.logger.Debug("shutting down")
	return nil
}

func (srv *Server) write(cancel context.CancelCauseFunc, msg rpc.Message) {
	content, err := json.Marshal(msg)
	if err != nil {
		srv.logger.Error("marshal failed", "err", err)
		return
	}
	if err := srv.out.Write(content); err != nil {
		cancel(err)
	}
}

func diagnostics(uri rpc.DocumentURI, text string) (rpc.Message, error) {
	var response rpc.Message
	r := strings.NewReader(text)

	ps, err := dot.NewParser(r)
	if err != nil {
		return response, err
	}

	_, err = ps.Parse()
	if err != nil {
		return response, err
	}

	response.Method = "textDocument/publishDiagnostics"
	responseParams := rpc.PublishDiagnosticsParams{
		URI: uri,
	}
	// TODO make clean, is every error in ps.Errors() one with a position? so a dot.Error
	// responseParams.Diagnostics = make([]rpc.Diagnostic, len(ps.Errors()))
	sev := rpc.SeverityError
	for _, err := range ps.Errors() {
		var perr dot.Error
		errors.As(err, &perr)
		responseParams.Diagnostics = append(responseParams.Diagnostics, rpc.Diagnostic{
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
			Message:  perr.Msg,
		})
	}
	if len(ps.Errors()) == 0 {
		responseParams.Diagnostics = []rpc.Diagnostic{}
	}
	puf, err := json.Marshal(responseParams)
	if err != nil {
		return response, err
	}
	rm := json.RawMessage(puf)
	response.Params = &rm

	return response, nil
}
