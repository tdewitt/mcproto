package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/misfitdev/proto-mcp/go/pkg/inspector"
)

func main() {
	timeout := flag.Duration("timeout", 30*time.Second, "Timeout for inspection")
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		fmt.Println("Usage: inspector [flags] <command> [args...]")
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	tools, err := inspector.Inspect(ctx, args[0], args[1:]...)
	if err != nil {
		log.Fatalf("Inspection failed: %v", err)
	}

	data, err := json.MarshalIndent(tools, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal tools: %v", err)
	}

	fmt.Println(string(data))
}
