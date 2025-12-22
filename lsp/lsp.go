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

func (sc *Scanner) Scan() bool {
	if sc.done {
		return false
	}

	// TODO extract read header?
	// TODO headers are case-insensitive
	line, err := sc.r.ReadString('\n')
	line = strings.TrimSuffix(line, "\n")
	line = strings.TrimSuffix(line, "\r")
	if err != nil {
		if !errors.Is(err, io.EOF) {
			sc.err = err
		}
		sc.done = true
		return false
	}

	header, value, found := strings.Cut(line, ":")
	if !found {
		sc.err = errors.New("invalid header, expected Content-Length")
		sc.done = true
		return false
	}
	header = strings.TrimSpace(header)
	if !strings.EqualFold(header, "Content-Length") {
		sc.err = errors.New("expected Content-Length")
		sc.done = true
		return false
	}
	value = strings.TrimSpace(value)
	length, err := strconv.Atoi(value)
	if err != nil {
		sc.err = err
		sc.done = true
		return false
	}

	// TODO validate potential Content-Type header
	// TODO expect empty line if Content-Type provided
	line, err = sc.r.ReadString('\n')
	line = strings.TrimSuffix(line, "\n")
	line = strings.TrimSuffix(line, "\r")
	if err != nil {
		if !errors.Is(err, io.EOF) {
			sc.err = err
		}
		sc.done = true
		return false
	}

	m := make([]byte, length)
	n, err := io.ReadFull(sc.r, m)
	if n != length {
		sc.err = fmt.Errorf("failed to read full content, read %d instead of %d", n, length)
	}
	if err != nil {
		// TODO react to ErrUnexpectedEOF?
		if !errors.Is(err, io.EOF) {
			sc.err = err
		}
		sc.done = true
		return false
	}
	sc.content = string(m)
	return true
}

func (sc *Scanner) Err() error {
	return sc.err
}

func (sc *Scanner) Next() string {
	if sc.done {
		return ""
	}
	return sc.content
}
