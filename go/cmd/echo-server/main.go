package main

import (
	"log"
	"os"

	"github.com/misfitdev/proto-mcp/go/router"
)

type stdioReadWriter struct {
	reader *os.File
	writer *os.File
}

func (s *stdioReadWriter) Read(p []byte) (n int, err error) {
	return s.reader.Read(p)
}

func (s *stdioReadWriter) Write(p []byte) (n int, err error) {
	return s.writer.Write(p)
}

func main() {
	rw := &stdioReadWriter{
		reader: os.Stdin,
		writer: os.Stdout,
	}

	pr := router.NewProtocolRouter(rw)
	pr.Register(router.ProtocolJSON, &router.JSONHandler{})
	pr.Register(router.ProtocolBinary, &router.BinaryHandler{})

	if err := pr.Route(); err != nil {
		log.Fatalf("Router failed: %v", err)
	}
}
