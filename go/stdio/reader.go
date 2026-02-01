package stdio

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/misfitdev/proto-mcp/go/mcp"
	"google.golang.org/protobuf/proto"
)

type Reader struct {
	r io.Reader
}

func NewReader(r io.Reader) *Reader {
	return &Reader{r: r}
}

const MaxMessageSize = 32 * 1024 * 1024 // 32MB

func (r *Reader) ReadMessage() (*mcp.MCPMessage, error) {
	// Read length prefix (4 bytes)
	lenBuf := make([]byte, 4)
	if _, err := io.ReadFull(r.r, lenBuf); err != nil {
		return nil, err
	}
	length := binary.BigEndian.Uint32(lenBuf)

	// Security: Limit message size to prevent OOM
	if length == 0 {
		return nil, fmt.Errorf("message size cannot be zero")
	}
	if length > MaxMessageSize {
		return nil, fmt.Errorf("message size %d exceeds limit of %d bytes", length, MaxMessageSize)
	}

	// Read message body with bounded allocation
	msgBuf := make([]byte, length)
	if _, err := io.ReadFull(r.r, msgBuf); err != nil {
		return nil, err
	}

	// Unmarshal protobuf message
	msg := &mcp.MCPMessage{}
	if err := proto.Unmarshal(msgBuf, msg); err != nil {
		return nil, err
	}

	return msg, nil
}
