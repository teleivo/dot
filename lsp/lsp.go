// Package lsp ...
// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/
package lsp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"

	"github.com/teleivo/dot/lsp/internal/rpc"
)

const maxContentLength = 10 << 20 // 10MB

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

// Watch starts the ...
func (srv *Server) Start(ctx context.Context) error {
	// TODO log to file with -debug for now
	// TODO setup state with type for statemachine states
	// unitialized/initialized/shutdown/terminated or so
	go func() {
		r := io.TeeReader(srv.in, srv.logFile)

		s := NewScanner(r)
		for s.Scan() {
			var message rpc.Message
			err := json.Unmarshal(s.Bytes(), &message)
			// TODO what to do in case of err? what can I respond with according to the spec?
			if err != nil {
				break
			}

			var response string
			if message.Method == "initialize" {
				response = `{"jsonrpc":"2.0","id":1,"result":{"capabilities":{"textDocumentSync":1},"serverInfo":{"name":"dotls","version":"0.1.0"}}}`
			} else {
				response = `{"jsonrpc":"2.0","id":1,"error":{"code":-32002,"message":"server not initialized"}}`
			}
			srv.writeMessage(response)
			srv.logger.Debug("received", "msg", s.Text())
		}
	}()
	<-ctx.Done()
	srv.logger.Debug("shutting down")
	_ = srv.logFile.Close()
	return nil
}

func (srv *Server) writeMessage(content string) error {
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
	_, err = fmt.Fprintf(srv.out, "%s", content)
	if err != nil {
		return err
	}
	return nil
}

type Scanner struct {
	r       *bufio.Reader
	content []byte
	done    bool
	err     error
}

func NewScanner(r io.Reader) *Scanner {
	return &Scanner{
		r: bufio.NewReader(r),
	}
}

func (s *Scanner) Scan() bool {
	if s.done {
		return false
	}
	s.content = nil // clear previous content

	var hasLength bool
	var length int
	for {
		line, ok := s.readLine()
		if !ok {
			if hasLength {
				s.err = errors.New("expected empty line before content")
			}
			return false
		}
		if len(line) == 0 && !hasLength {
			s.err = errors.New("expected content-length header")
			s.done = true
			return false
		}
		if len(line) == 0 && hasLength {
			break
		}

		header, value, ok := s.readHeader(line)
		if !ok {
			return false
		}
		if !bytes.EqualFold(header, []byte("Content-Length")) {
			continue // skip any other header as they provide no value
		}

		var err error
		length, err = strconv.Atoi(string(value))
		if err != nil {
			s.err = fmt.Errorf("invalid content-length: expected number, got %q", value)
			s.done = true
			return false
		}
		if length < 0 {
			s.err = fmt.Errorf("invalid content-length: expected positive number, got %q", value)
			s.done = true
			return false
		}
		if length > maxContentLength {
			s.err = fmt.Errorf("invalid content-length: exceeds maximum of 10MB, got %d", length)
			s.done = true
			return false
		}
		hasLength = true
	}

	m := make([]byte, length)
	n, err := io.ReadFull(s.r, m)
	if err != nil {
		s.err = fmt.Errorf("unexpected EOF: read %d of %d content bytes", n, length)
		s.done = true
		return false
	}
	s.content = m
	return true
}

func (s *Scanner) readLine() ([]byte, bool) {
	line, err := s.r.ReadBytes('\n')
	if err != nil {
		if !errors.Is(err, io.EOF) {
			s.err = err
		}
		s.done = true
		return nil, false
	}

	// be lenient: spec expects \r\n terminated headers but \n is accepted as well
	line = bytes.TrimSuffix(line, []byte("\n"))
	line = bytes.TrimSuffix(line, []byte("\r"))
	return line, true
}

func (s *Scanner) readHeader(line []byte) ([]byte, []byte, bool) {
	header, value, found := bytes.Cut(line, []byte(":"))
	if !found {
		s.err = fmt.Errorf("invalid header: expected 'name: value', got %q", line)
		s.done = true
		return nil, nil, false
	}
	return bytes.TrimSpace(header), bytes.TrimSpace(value), true
}

func (s *Scanner) Err() error {
	return s.err
}

// Bytes returns the most recent content read by a call to Scan.
// The underlying array may point to data that will be overwritten by a subsequent call to Scan.
func (s *Scanner) Bytes() []byte {
	return s.content
}

// Text returns the most recent content read by a call to Scan as a string.
func (s *Scanner) Text() string {
	return string(s.content)
}
