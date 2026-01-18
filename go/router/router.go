package router

import (
	"fmt"
	"io"
	"os"
)

type Handler interface {
	Handle(rw io.ReadWriter) error
}

type ProtocolRouter struct {
	rw       io.ReadWriter
	handlers map[Protocol]Handler
}

func NewProtocolRouter(rw io.ReadWriter) *ProtocolRouter {
	return &ProtocolRouter{
		rw:       rw,
		handlers: make(map[Protocol]Handler),
	}
}

func (pr *ProtocolRouter) Register(p Protocol, h Handler) {
	pr.handlers[p] = h
}

func (pr *ProtocolRouter) Route() error {
	sniffer := NewSniffer(pr.rw)
	p, err := sniffer.Detect()
	if err != nil {
		return fmt.Errorf("failed to detect protocol: %w", err)
	}

	switch p {
	case ProtocolJSON:
		fmt.Fprintln(os.Stderr, "DEBUG: [Server] Sniffed '{' -> Routing to JSON-RPC Handler")
	case ProtocolBinary:
		fmt.Fprintln(os.Stderr, "DEBUG: [Server] Sniffed binary prefix -> Routing to Protobuf Handler")
	default:
		fmt.Fprintln(os.Stderr, "DEBUG: [Server] Sniffed unknown format -> Routing to Unknown Handler")
	}

	handler, ok := pr.handlers[p]
	if !ok {
		return fmt.Errorf("no handler registered for protocol: %v", p)
	}

	// Use combinedReadWriter to wrap the sniffer (which has the peeked bytes)
	// and the original writer.
	crw := &combinedReadWriter{
		Reader: sniffer,
		Writer: pr.rw,
	}

	return handler.Handle(crw)
}

type combinedReadWriter struct {
	io.Reader
	io.Writer
}
