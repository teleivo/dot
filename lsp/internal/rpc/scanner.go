package rpc

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
)

const (
	maxHeaderLineLength = 4096
	maxContentLength    = 20 << 20 // 20MB
)

// Scanner reads JSON-RPC messages from an [io.Reader] using the base protocol framing.
// The base protocol consists of a header and content part, where the header is separated
// from the content by an empty line (\r\n).
//
// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#baseProtocol
type Scanner struct {
	r       *bufio.Reader
	buf     []byte
	content []byte
	done    bool
	err     error
}

// NewScanner returns a new Scanner that reads from r.
func NewScanner(r io.Reader) *Scanner {
	return &Scanner{
		r: bufio.NewReaderSize(r, maxHeaderLineLength),
	}
}

// Scan reads the next message from the input.
// It returns true if a message was successfully read, or false if an error occurred
// or EOF was reached. After Scan returns false, the [Scanner.Err] method will return
// any error that occurred during scanning, except for [io.EOF], which is not reported.
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
			s.err = fmt.Errorf("invalid content-length: exceeds maximum of 20MB, got %d", length)
			s.done = true
			return false
		}
		hasLength = true
	}

	if length > cap(s.buf) {
		s.buf = make([]byte, length)
	}
	n, err := io.ReadFull(s.r, s.buf[:length])
	if err != nil {
		s.err = fmt.Errorf("unexpected EOF: read %d of %d content bytes", n, length)
		s.done = true
		return false
	}
	s.content = s.buf[:length]
	return true
}

func (s *Scanner) readLine() ([]byte, bool) {
	line, err := s.r.ReadSlice('\n')
	if err != nil {
		if errors.Is(err, bufio.ErrBufferFull) {
			s.err = errors.New("header line too long: exceeds maximum of 4KB")
		} else if !errors.Is(err, io.EOF) {
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

// Err returns the first non-EOF error encountered by the Scanner.
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
