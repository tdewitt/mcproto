import grpc
from typing import Optional, Dict
from . import mcp_pb2, mcp_pb2_grpc
from .config import DEFAULT_GRPC_TARGET

class GRPCClient:
    def __init__(self, target: str = DEFAULT_GRPC_TARGET, use_secure: bool = False):
        if use_secure:
            credentials = grpc.ssl_channel_credentials()
            self.channel = grpc.secure_channel(target, credentials)
        else:
            self.channel = grpc.insecure_channel(target)
        self.stub = mcp_pb2_grpc.MCPServiceStub(self.channel)

    def __enter__(self):
        """Context manager entry: return self for use in 'with' statements."""
        return self

    def __exit__(self, exc_type, exc_val, exc_tb):
        """Context manager exit: close the channel on context exit."""
        self.close()
        return False

    def __del__(self):
        """Destructor: ensure channel is closed when object is garbage collected."""
        try:
            self.close()
        except:
            pass

    def initialize(self, protocol_version: str = "1.0.0") -> mcp_pb2.InitializeResponse:
        req = mcp_pb2.InitializeRequest(
            protocol_version=protocol_version,
            capabilities=mcp_pb2.ClientCapabilities(
                supports_bsr_refs=True,
                encodings=["protobuf"]
            )
        )
        return self.stub.Initialize(req)

    def list_tools(self, query: Optional[str] = None, cursor: Optional[str] = None) -> mcp_pb2.ListToolsResponse:
        req = mcp_pb2.ListToolsRequest(query=query, cursor=cursor)
        return self.stub.ListTools(req)

    def call_tool(self, name: str, arguments: mcp_pb2.google_dot_protobuf_dot_any__pb2.Any) -> mcp_pb2.CallToolResponse:
        req = mcp_pb2.CallToolRequest(
            name=name,
            arguments=arguments
        )
        return self.stub.CallTool(req)

    def close(self):
        self.channel.close()
