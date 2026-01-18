package main

import (
	"log"
	"os"

	"github.com/misfitdev/proto-mcp/go/stdio"
)

func main() {
	reader := stdio.NewReader(os.Stdin)
	writer := stdio.NewWriter(os.Stdout)

	for {
		msg, err := reader.ReadMessage()
		if err != nil {
			log.Fatalf("Error reading message: %v", err)
		}

		// Echo the message back
		if err := writer.WriteMessage(msg); err != nil {
			log.Fatalf("Error writing message: %v", err)
		}
	}
}
