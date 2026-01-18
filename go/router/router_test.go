package router

import (
	"bytes"
	"io"
	"testing"
)

type mockHandler struct {
	called bool
}

func (h *mockHandler) Handle(rw io.ReadWriter) error {
	h.called = true
	// Consume some data to simulate handling
	_, _ = io.ReadAll(rw)
	return nil
}

func TestProtocolRouter(t *testing.T) {
	jsonHandler := &mockHandler{}
	binaryHandler := &mockHandler{}

	input := []byte("{\"jsonrpc\": \"2.0\"}")
	rw := struct {
		io.Reader
		io.Writer
	}{
		Reader: bytes.NewReader(input),
		Writer: &bytes.Buffer{},
	}

	router := NewProtocolRouter(rw)
	router.Register(ProtocolJSON, jsonHandler)
	router.Register(ProtocolBinary, binaryHandler)

	if err := router.Route(); err != nil {
		t.Fatalf("Route failed: %v", err)
	}

	if !jsonHandler.called {
		t.Error("JSON handler was not called")
	}
	if binaryHandler.called {
		t.Error("Binary handler was called for JSON input")
	}
}

