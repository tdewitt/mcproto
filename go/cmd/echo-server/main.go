package main

import (
	"log"
	"os"

	"github.com/misfitdev/proto-mcp/go/pkg/mcp"
	"github.com/misfitdev/proto-mcp/go/router"
)

// ... stdioReadWriter implementation ...

func main() {
	rw := &stdioReadWriter{
		reader: os.Stdin,
		writer: os.Stdout,
	}

	registry := mcp.NewUnifiedRegistry()
	// No tools registered for the simple echo-server for now

	pr := router.NewProtocolRouter(rw)
	pr.Register(router.ProtocolJSON, &router.JSONHandler{})
	pr.Register(router.ProtocolBinary, router.NewBinaryHandler(registry))

	if err := pr.Route(); err != nil {
		log.Fatalf("Router failed: %v", err)
	}
}
