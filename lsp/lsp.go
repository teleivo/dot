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
)

// Watch starts the ...
func (srv *Server) Start(ctx context.Context) error {
	// TODO log to file with -debug for now
	// TODO setup state with type for statemachine states
	// unitialized/initialized/shutdown/terminated or so
	go func() {
		r := io.TeeReader(srv.in, srv.logFile)

		s := rpc.NewScanner(r)
		for s.Scan() {
			var message rpc.Message
			err := json.Unmarshal(s.Bytes(), &message)
			// TODO what to do in case of err? what can I respond with according to the spec?
			// TODO error handling in general
			if err != nil {
				break
			}

			var response rpc.Message
			// TODO create rpc.Writer and respond with the id from the request
			if srv.state == uninitialized {
				if message.Method == "initialize" {
					// TODO do I need to wait on the initialized notification to set the state? or
					// can I just ignore if it is sent or not
					srv.state = initialized
					response = rpc.Message{ID: message.ID, Result: rpc.InitializeResult()}
					content, _ := json.Marshal(response)
					srv.writeMessage(content)
				} else {
					response = rpc.Message{ID: message.ID, Error: &rpc.Error{Code: rpc.ServerNotInitialized, Message: "server not initialized"}} // TODO use responseMsg
					content, _ := json.Marshal(response)
					srv.writeMessage(content)
				}
			} else {
				switch message.Method {
				case "initialize":
					response = rpc.Message{ID: message.ID, Error: &rpc.Error{Code: rpc.InvalidRequest, Message: "server already initialized"}} // TODO use responseMsg
					content, _ := json.Marshal(response)
					srv.writeMessage(content)
				case "shutdown":
					response = rpc.Message{ID: message.ID} // TODO does the "result" need to be set to null explicitly?
					content, _ := json.Marshal(response)
					srv.writeMessage(content)
				case "textDocument/didOpen":
					// TODO implement
				}
			}
			srv.logger.Debug("received", "msg", s.Text())
		}
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
