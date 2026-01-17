package stdio

import (
	"encoding/binary"
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

func (r *Reader) ReadMessage() (*mcp.MCPMessage, error) {
	// Read length prefix (4 bytes)
	lenBuf := make([]byte, 4)
	if _, err := io.ReadFull(r.r, lenBuf); err != nil {
		return nil, err
	}
	length := binary.BigEndian.Uint32(lenBuf)

	// Read message body
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
