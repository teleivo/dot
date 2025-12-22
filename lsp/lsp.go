// Package lsp ...
// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/
package lsp

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"
)

const maxContentLength = 10 << 20 // 10MB

// Config ...
type Config struct {
	Debug  bool      // enable debug logging
	Stdin  io.Reader // input for ...
	Stdout io.Writer // output for ...
	Stderr io.Writer // output for error logging
}

type Server struct {
	stdin   io.Reader
	stdout  io.Writer
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
	logger := slog.New(slog.NewTextHandler(cfg.Stderr, &slog.HandlerOptions{Level: level}))
	srv := &Server{
		stdin:   cfg.Stdin,
		stdout:  cfg.Stdout,
		logger:  logger,
		logFile: f,
	}
	return srv, nil
}

// Watch starts the ...
func (srv *Server) Start(ctx context.Context) error {
	// TODO log to file with -debug for now
	// TODO create server loop that reads from stdin
	// TODO create my own scanner for the json-rpc messages?
	go func() {
		r := io.TeeReader(srv.stdin, srv.logFile)
		sc := bufio.NewScanner(r)
		for sc.Scan() {
			srv.logger.Debug("received", "msg", sc.Text())
		}
	}()
	<-ctx.Done()
	srv.logger.Debug("shutting down")
	_ = srv.logFile.Close()
	return nil
}

type Scanner struct {
	r       *bufio.Reader
	content string
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
	s.content = "" // clear previous content

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
		if line == "" && !hasLength {
			s.err = errors.New("expected content-length header")
			s.done = true
			return false
		}
		if line == "" && hasLength {
			break
		}

		header, value, ok := s.readHeader(line)
		if !ok {
			return false
		}
		if !strings.EqualFold(header, "Content-Length") {
			continue // skip any other header as they provide no value
		}

		var err error
		length, err = strconv.Atoi(value)
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
	s.content = string(m)
	return true
}

func (s *Scanner) readLine() (string, bool) {
	line, err := s.r.ReadString('\n')
	if err != nil {
		if !errors.Is(err, io.EOF) {
			s.err = err
		}
		s.done = true
		return "", false
	}

	// be lenient: spec expects \r\n terminated headers but \n is accepted as well
	line = strings.TrimSuffix(line, "\n")
	line = strings.TrimSuffix(line, "\r")
	return line, true
}

func (s *Scanner) readHeader(line string) (string, string, bool) {
	header, value, found := strings.Cut(line, ":")
	if !found {
		s.err = fmt.Errorf("invalid header: expected 'name: value', got %q", line)
		s.done = true
		return "", "", false
	}
	return strings.TrimSpace(header), strings.TrimSpace(value), true
}

func (s *Scanner) Err() error {
	return s.err
}

func (s *Scanner) Next() string {
	if s.done {
		return ""
	}
	return s.content
}
