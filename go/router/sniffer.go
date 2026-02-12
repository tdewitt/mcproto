package router

import (
	"bufio"
	"io"
	"strings"
)

type Protocol int

const (
	ProtocolUnknown Protocol = iota
	ProtocolJSON
	ProtocolBinary
)

type Sniffer struct {
	br *bufio.Reader
}

func NewSniffer(r io.Reader) *Sniffer {
	return &Sniffer{
		br: bufio.NewReader(r),
	}
}

func (s *Sniffer) Detect() (Protocol, error) {
	peek, err := s.br.Peek(32)
	if err != nil {
		if err == io.EOF && len(peek) == 0 {
			return ProtocolUnknown, nil
		}
		if len(peek) == 0 {
			return ProtocolUnknown, err
		}
	}

	firstByte, found := firstNonWhitespace(peek)
	if !found {
		// All peeked bytes are whitespace with no actual content byte.
		// Returning ProtocolJSON here would cause the router to hand the
		// connection to the JSON handler which would then hang waiting for
		// real data. Return ProtocolUnknown so the router surfaces a clear
		// "no handler for protocol" error instead.
		return ProtocolUnknown, nil
	}
	if hasContentLengthHeaderBytes(peek) {
		return ProtocolJSON, nil
	}
	if firstByte == '{' {
		return ProtocolJSON, nil
	}
	if firstByte <= 0x1F {
		return ProtocolBinary, nil
	}

	return ProtocolUnknown, nil
}

func (s *Sniffer) Read(p []byte) (n int, err error) {
	return s.br.Read(p)
}

func firstNonWhitespace(data []byte) (byte, bool) {
	for _, b := range data {
		if !isWhitespace(b) {
			return b, true
		}
	}
	return 0, false
}

func isWhitespace(b byte) bool {
	return b == ' ' || b == '\n' || b == '\r' || b == '\t'
}

func hasContentLengthHeaderBytes(data []byte) bool {
	trimmed := strings.TrimLeftFunc(string(data), func(r rune) bool {
		return r == ' ' || r == '\n' || r == '\r' || r == '\t'
	})
	return strings.HasPrefix(strings.ToLower(trimmed), "content-length:")
}
