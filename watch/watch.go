// Package watch provides a file watcher that serves DOT graphs as SVG via HTTP.
package watch

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"os"
	"os/exec"
	"sync"
	"time"
)

// Config configures a Watcher.
type Config struct {
	File   string    // DOT file to serve
	Port   string    // HTTP server port (use "0" for a random available port)
	Debug  bool      // enable debug logging
	Stdout io.Writer // output for status messages
	Stderr io.Writer // output for error logging
}

// Watcher watches a DOT file for changes and serves the rendered SVG via HTTP.
// It provides an SSE endpoint that notifies connected browsers when the file changes.
type Watcher struct {
	file     string
	stdout   io.Writer
	logger   *slog.Logger
	server   *http.Server
	shutdown chan struct{}
	clients  sync.WaitGroup
}

const dotBinary = "dot"

//go:embed index.html
var indexHTML []byte

// New creates a Watcher that serves the given DOT file as SVG on the specified port.
func New(cfg Config) (*Watcher, error) {
	_, err := os.Stat(cfg.File)
	if err != nil {
		return nil, fmt.Errorf("file error: %v", err)
	}
	addr, err := netip.ParseAddrPort("127.0.0.1:" + cfg.Port)
	if err != nil {
		return nil, fmt.Errorf("invalid port %q, must be in range 1-65535", cfg.Port)
	}
	_, err = exec.LookPath(dotBinary)
	if err != nil {
		return nil, fmt.Errorf("dot executable not found, install Graphviz from https://graphviz.org/download/")
	}

	handler := http.NewServeMux()
	server := http.Server{
		Addr:        addr.String(),
		Handler:     handler,
		ReadTimeout: 3 * time.Second,
		IdleTimeout: 120 * time.Second,
	}
	level := slog.LevelInfo
	if cfg.Debug {
		level = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(cfg.Stderr, &slog.HandlerOptions{Level: level}))
	wa := &Watcher{
		file:     cfg.File,
		stdout:   cfg.Stdout,
		logger:   logger,
		server:   &server,
		shutdown: make(chan struct{}),
	}
	handler.HandleFunc("GET /", wa.handleIndex)
	handler.HandleFunc("GET /events", wa.handleEvents)
	svgHandler := http.TimeoutHandler(http.HandlerFunc(wa.handleGenerate), 5*time.Second, "failed to generate svg in time")
	handler.Handle("GET /graph", svgHandler)
	handler.Handle("GET /graph.svg", svgHandler)
	return wa, nil
}

// Watch starts the HTTP server and blocks until the context is cancelled.
func (wa *Watcher) Watch(ctx context.Context) error {
	ln, err := net.Listen("tcp", wa.server.Addr)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(wa.stdout, "watching on http://%s\n", ln.Addr())

	go func() {
		<-ctx.Done()
		close(wa.shutdown)
		wa.logger.Debug("shutting down, notifying clients")
		wa.clients.Wait() // no timeout: localhost flushes complete nearly instantly
		ctxTimeout, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		if err := wa.server.Shutdown(ctxTimeout); err != nil && !errors.Is(err, context.Canceled) {
			wa.logger.Error("failed to shutdown", "error", err)
		}
	}()

	if err := wa.server.Serve(ln); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func (wa *Watcher) handleIndex(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	_, err := w.Write(indexHTML)
	if err != nil {
		wa.logger.Error("failed to write index.html", "error", err)
	}
}

func (wa *Watcher) handleEvents(w http.ResponseWriter, r *http.Request) {
	wa.clients.Add(1)
	defer wa.clients.Done()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	wa.logger.Debug("client connected")

	keepAliveTicker := time.NewTicker(15 * time.Second)
	defer keepAliveTicker.Stop()
	pollTicker := time.NewTicker(500 * time.Millisecond)
	defer pollTicker.Stop()

	var lastMod time.Time
	var lastSize int64

	for {
		select {
		case <-r.Context().Done():
			wa.logger.Debug("client disconnected")
			return
		case <-wa.shutdown:
			_, _ = fmt.Fprint(w, "event: close\ndata: shutdown\n\n")
			flusher.Flush()
			wa.logger.Debug("closing connection to client")
			return
		case <-keepAliveTicker.C:
			_, _ = w.Write([]byte(": keep-alive\n"))
			wa.logger.Debug("sent keep-alive")
			flusher.Flush()
		case <-pollTicker.C:
			stat, err := os.Stat(wa.file)
			if err != nil {
				wa.logger.Error("stat failed", "error", err)
				return
			}
			if !stat.ModTime().Equal(lastMod) || stat.Size() != lastSize {
				wa.logger.Debug("change detected", "modtime", stat.ModTime(), "size", stat.Size())
				_, _ = fmt.Fprintf(w, "data: %s\nretry: 5000\n\n", stat.ModTime())
				flusher.Flush()
			}
			lastMod = stat.ModTime()
			lastSize = stat.Size()
		}
	}
}

func (wa *Watcher) handleGenerate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/svg+xml")
	err := wa.generate(r.Context(), w)
	if err != nil {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprint(w, err.Error())
		return
	}
}

func (wa *Watcher) generate(ctx context.Context, w io.Writer) error {
	dotSource, err := os.ReadFile(wa.file)
	if err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, dotBinary, "-Tsvg", "-Gbgcolor=transparent")
	cmd.Stdin = bytes.NewReader(dotSource)

	var stderr bytes.Buffer
	cmd.Stdout = w
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("dot command failed: %v\nstderr: %s", err, stderr.String())
	}
	return nil
}
