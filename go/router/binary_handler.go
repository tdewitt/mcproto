package router

import (
	"fmt"
	"io"

	"github.com/misfitdev/proto-mcp/go/stdio"
)

type BinaryHandler struct{}

func (h *BinaryHandler) Handle(rw io.ReadWriter) error {
	reader := stdio.NewReader(rw)
	writer := stdio.NewWriter(rw)

	for {
		msg, err := reader.ReadMessage()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("binary handler read error: %w", err)
		}

		// Echo the message back
		if err := writer.WriteMessage(msg); err != nil {
			return fmt.Errorf("binary handler write error: %w", err)
		}
	}
}
