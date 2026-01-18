package stdio

import (
	"encoding/binary"
	"io"

	"github.com/misfitdev/proto-mcp/go/mcp"
	"google.golang.org/protobuf/proto"
)

type Writer struct {
	w io.Writer
}

func NewWriter(w io.Writer) *Writer {
	return &Writer{w: w}
}

func (w *Writer) WriteMessage(msg *mcp.MCPMessage) error {
	// Marshal protobuf message
	data, err := proto.Marshal(msg)
	if err != nil {
		return err
	}

	// Write length prefix (4 bytes, big-endian)
	lenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBuf, uint32(len(data)))
	if _, err := w.w.Write(lenBuf); err != nil {
		return err
	}

	// Write message body
	if _, err := w.w.Write(data); err != nil {
		return err
	}

	return nil
}
