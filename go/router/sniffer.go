package router

import (
	"bufio"
	"io"
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
	peek, err := s.br.Peek(1)
	if err != nil {
		if err == io.EOF {
			return ProtocolUnknown, nil
		}
		return ProtocolUnknown, err
	}

	firstByte := peek[0]
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
