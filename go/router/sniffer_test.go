package router

import (
	"bytes"
	"io"
	"testing"
)

func TestSniffer(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected Protocol
	}{
		{
			name:     "JSON-RPC (Open Brace)",
			input:    []byte("{\"jsonrpc\": \"2.0\"}"),
			expected: ProtocolJSON,
		},
		{
			name:     "JSON-RPC (Leading Whitespace)",
			input:    []byte(" \n\t{\"jsonrpc\": \"2.0\"}"),
			expected: ProtocolJSON,
		},
		{
			name:     "Binary (Small Length Prefix)",
			input:    []byte{0x00, 0x00, 0x00, 0x01, 0x0A},
			expected: ProtocolBinary,
		},
		{
			name:     "JSON-RPC (Content-Length Header)",
			input:    []byte("Content-Length: 18\r\n\r\n{\"jsonrpc\":\"2.0\"}"),
			expected: ProtocolJSON,
		},
		{
			name:     "Unknown",
			input:    []byte("GET / HTTP/1.1"),
			expected: ProtocolUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bytes.NewReader(tt.input)
			sniffer := NewSniffer(r)
			proto, err := sniffer.Detect()
			if err != nil {
				t.Fatalf("Detect failed: %v", err)
			}
			if proto != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, proto)
			}

			// Ensure the stream is still readable from the beginning
			all, err := io.ReadAll(sniffer)
			if err != nil {
				t.Fatalf("ReadAll failed: %v", err)
			}
			if !bytes.Equal(all, tt.input) {
				t.Errorf("Stream consumed incorrectly. Expected %v, got %v", tt.input, all)
			}
		})
	}
}

func TestSniffer_Empty(t *testing.T) {
	r := bytes.NewReader([]byte{})
	sniffer := NewSniffer(r)
	proto, err := sniffer.Detect()
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}
	if proto != ProtocolUnknown {
		t.Errorf("Expected ProtocolUnknown for empty input, got %v", proto)
	}
}

func TestSniffer_WhitespaceOnly(t *testing.T) {
	r := bytes.NewReader([]byte("   \n\t\r\n  "))
	sniffer := NewSniffer(r)
	proto, err := sniffer.Detect()
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}
	if proto != ProtocolUnknown {
		t.Errorf("Expected ProtocolUnknown for whitespace-only input, got %v", proto)
	}
}
