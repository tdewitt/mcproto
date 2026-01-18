package main

import (
	"log"
	"os"

	"github.com/misfitdev/proto-mcp/go/pkg/registry"
	"github.com/misfitdev/proto-mcp/go/router"
)

// ... stdioReadWriter implementation ...

func main() {
	rw := &stdioReadWriter{
		reader: os.Stdin,
		writer: os.Stdout,
	}

	reg := registry.NewUnifiedRegistry()
	// No tools registered for the simple echo-server for now

	pr := router.NewProtocolRouter(rw)
	pr.Register(router.ProtocolJSON, &router.JSONHandler{})
	pr.Register(router.ProtocolBinary, router.NewBinaryHandler(reg))

	if err := pr.Route(); err != nil {
		log.Fatalf("Router failed: %v", err)
	}
}
