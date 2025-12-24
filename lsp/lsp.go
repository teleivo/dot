// Package lsp ...
// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/
package lsp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/teleivo/dot"
	"github.com/teleivo/dot/lsp/internal/rpc"
)

// Config ...
type Config struct {
	Debug bool      // enable debug logging
	In    io.Reader // input for ...
	Out   io.Writer // output for ...
	Err   io.Writer // output for error logging
}

type Server struct {
	in      io.Reader
	out     io.Writer
	logger  *slog.Logger
	logFile *os.File
	state   state
}

// New creates a ...
func New(cfg Config) (*Server, error) {
	level := slog.LevelInfo
	if cfg.Debug {
		level = slog.LevelDebug
	}
	f, err := os.Create("/tmp/dotls.log")
	if err != nil {
		return nil, err
	}
	// logger := slog.New(slog.NewTextHandler(f, &slog.HandlerOptions{Level: level}))
	logger := slog.New(slog.NewTextHandler(cfg.Err, &slog.HandlerOptions{Level: level}))
	srv := &Server{
		in:      cfg.In,
		out:     cfg.Out,
		logger:  logger,
		logFile: f,
	}
	return srv, nil
}

type state int

const (
	uninitialized state = iota
	initialized
	shuttingDown
)

// Start starts the ...
func (srv *Server) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	// TODO log to file with -debug for now
	// TODO setup state with type for statemachine states
	// unitialized/initialized/shutdown/terminated or so
	go func() {
		defer func() {
			if err := recover(); err != nil {
				fmt.Println(err)
			}
		}()
		r := io.TeeReader(srv.in, srv.logFile)

		s := rpc.NewScanner(r)
		for s.Scan() {
			// TODO should the scanner alread do the Unmarshal?
			var message rpc.Message
			err := json.Unmarshal(s.Bytes(), &message)
			// TODO what to do in case of err? what can I respond with according to the spec?
			// TODO error handling in general
			if err != nil {
				fmt.Printf("DEBUG: json.Unmarshal failed: %v\nbytes: %q\n", err, s.Bytes())
				break
			}

			var response rpc.Message
			// TODO create rpc.Writer and respond with the id from the request
			switch srv.state {
			case uninitialized:
				fmt.Println("uninitialized case")
				if message.Method == "initialize" {
					// TODO do I need to wait on the initialized notification to set the state? or
					// can I just ignore if it is sent or not
					// TODO what if sending response errors? still move to initialized state?
					srv.state = initialized
					response = rpc.Message{ID: message.ID, Result: rpc.InitializeResult()}
					content, _ := json.Marshal(response)
					srv.writeMessage(content)
				} else {
					response = rpc.Message{ID: message.ID, Error: &rpc.Error{Code: rpc.ServerNotInitialized, Message: "server not initialized"}}
					content, _ := json.Marshal(response)
					srv.writeMessage(content)
				}
			case initialized:
				fmt.Println("initialized case")
				switch message.Method {
				case "initialize":
					response = rpc.Message{ID: message.ID, Error: &rpc.Error{Code: rpc.InvalidRequest, Message: "server already initialized"}}
					content, _ := json.Marshal(response)
					srv.writeMessage(content)
				case "shutdown":
					// TODO what if sending response errors? still move to shuttingDown state?
					srv.state = shuttingDown
					response = rpc.Message{ID: message.ID} // TODO does the "result" need to be set to null explicitly?
					content, _ := json.Marshal(response)
					srv.writeMessage(content)
					srv.logger.Debug("shutdown message received")
				case "textDocument/didOpen":
					if message.Params == nil {
						panic("TODO handle")
					}
					// TODO implement
					var requestParams rpc.DidOpenTextDocumentParams
					err := json.Unmarshal(*message.Params, &requestParams)
					if err != nil {
						panic("TODO handle")
					} else {
						r := strings.NewReader(requestParams.TextDocument.Text)

						ps, err := dot.NewParser(r)
						if err != nil {
							panic("TODO handle")
						}

						_, err = ps.Parse()
						if err != nil {
							panic("TODO handle")
						}

						response.Method = "textDocument/publishDiagnostics"
						responseParams := rpc.PublishDiagnosticsParams{
							URI: requestParams.TextDocument.URI,
						}
						responseParams.Diagnostics = []rpc.Diagnostic{{Message: "test"}}
						if errs := ps.Errors(); len(errs) > 0 {
							// TODO map to lsp
							// return errs[0]
						}
						puf, err := json.Marshal(responseParams)
						if err != nil {
							panic("TODO handle")
						}
						rm := json.RawMessage(puf)
						response.Params = &rm
						content, _ := json.Marshal(response)
						srv.writeMessage(content)
					}
				}
			case shuttingDown:
				switch message.Method {
				case "exit":
					srv.logger.Debug("exit notification received")
					cancel()
				default:
					response = rpc.Message{ID: message.ID, Error: &rpc.Error{Code: rpc.InvalidRequest, Message: "server is shutting down"}}
					content, _ := json.Marshal(response)
					srv.writeMessage(content)
				}
			}
			srv.logger.Debug("received", "msg", s.Text())
		}
		fmt.Printf("DEBUG: loop exited, scanner err: %v\n", s.Err())
	}()

	<-ctx.Done()
	srv.logger.Debug("shutting down")
	_ = srv.logFile.Close()
	return nil
}

func (srv *Server) writeMessage(content []byte) error {
	// TODO user bufio and its API
	// TODO do I need to flush?
	_, err := fmt.Fprintf(srv.out, "Content-Length:  %d \r\n", len(content))
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(srv.out, "\r\n")
	if err != nil {
		return err
	}
	_, err = srv.out.Write(content)
	if err != nil {
		return err
	}
	return nil
}
