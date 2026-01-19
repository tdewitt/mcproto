import struct
from typing import BinaryIO, Optional
from . import mcp_pb2

MAX_MESSAGE_SIZE = 32 * 1024 * 1024  # 32MB

class StdioReader:
    def __init__(self, reader: BinaryIO):
        self.reader = reader

    def read_message(self) -> Optional[mcp_pb2.MCPMessage]:
        # Read length prefix (4 bytes, big-endian)
        len_buf = self.reader.read(4)
        if not len_buf:
            return None
        if len(len_buf) < 4:
            raise EOFError("Incomplete length prefix")
        
        length = struct.unpack(">I", len_buf)[0]
        
        # Security: Limit message size to prevent OOM
        if length > MAX_MESSAGE_SIZE:
            raise ValueError(f"Message size {length} exceeds limit of {MAX_MESSAGE_SIZE} bytes")

        # Read message body
        msg_buf = self.reader.read(length)
        if len(msg_buf) < length:
            raise EOFError("Incomplete message body")
        
        msg = mcp_pb2.MCPMessage()
        msg.ParseFromString(msg_buf)
        return msg

class StdioWriter:
    def __init__(self, writer: BinaryIO):
        self.writer = writer

    def write_message(self, msg: mcp_pb2.MCPMessage):
        data = msg.SerializeToString()
        
        # Write length prefix (4 bytes, big-endian)
        len_buf = struct.pack(">I", len(data))
        self.writer.write(len_buf)
        
        # Write message body
        self.writer.write(data)
        self.writer.flush()
