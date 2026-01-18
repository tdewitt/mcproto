import io
import struct
import pytest
from mcp.stdio import StdioReader, StdioWriter
from mcp import mcp_pb2

def test_roundtrip():
    buf = io.BytesIO()
    writer = StdioWriter(buf)
    
    msg = mcp_pb2.MCPMessage(id=1)
    msg.initialize_request.protocol_version = "1.0.0"
    
    writer.write_message(msg)
    
    buf.seek(0)
    reader = StdioReader(buf)
    read_msg = reader.read_message()
    
    assert read_msg.id == msg.id
    assert read_msg.initialize_request.protocol_version == msg.initialize_request.protocol_version

def test_eof():
    buf = io.BytesIO()
    reader = StdioReader(buf)
    assert reader.read_message() is None

def test_incomplete_length():
    buf = io.BytesIO(b"\x00\x00\x00")
    reader = StdioReader(buf)
    with pytest.raises(EOFError):
        reader.read_message()

def test_incomplete_body():
    buf = io.BytesIO(b"\x00\x00\x00\x05\x01\x02") # Claim 5, give 2
    reader = StdioReader(buf)
    with pytest.raises(EOFError):
        reader.read_message()
