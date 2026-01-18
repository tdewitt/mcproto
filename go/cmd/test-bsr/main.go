package main

import (
	"context"
	"fmt"
	"os"

	"github.com/misfitdev/proto-mcp/go/pkg/bsr"
)

func main() {
	// Ensure BUF_TOKEN is in env for the client to find it
	client := bsr.NewClient()
	// Using a public module from bufbuild
	ref, err := bsr.ParseRef("buf.build/bufbuild/registry/buf.registry.module.v1.DownloadService:main")
	if err != nil {
		fmt.Printf("Parse Error: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Printf("Fetching descriptors for %s/%s...\n", ref.Owner, ref.Repository)
	fds, err := client.FetchDescriptorSet(context.Background(), ref)
	if err != nil {
		fmt.Printf("Fetch Error: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Printf("Successfully fetched FileDescriptorSet with %d files\n", len(fds.File))
}
