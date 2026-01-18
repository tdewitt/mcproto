import json
import base64
import tiktoken
import sys
import os

# Add gen/python to path
sys.path.append(os.path.join(os.getcwd(), "gen", "python"))

import mcp_pb2

def count_tokens(text):
    encoding = tiktoken.get_encoding("cl100k_base")
    return len(encoding.encode(text))

def benchmark():
    # 1. Create a sample ListToolsResponse with 10 tools
    tools = []
    for i in range(10):
        tool = mcp_pb2.Tool(
            name=f"tool_{i}",
            description=f"This is a detailed description for tool number {i}. It performs some specific action that requires context.",
            bsr_ref=f"buf.build/acme/tools/tool_{i}:v1"
        )
        tools.append(tool)
    
    proto_msg = mcp_pb2.ListToolsResponse(tools=tools)
    
    # 2. Protobuf Serialization (Base64 encoded for token counting)
    proto_bytes = proto_msg.SerializeToString()
    proto_b64 = base64.b64encode(proto_bytes).decode("utf-8")
    proto_tokens = count_tokens(proto_b64)
    
    # 3. JSON-RPC equivalent (with full schema as required by standard MCP)
    json_tools = []
    for i in range(10):
        json_tools.append({
            "name": f"tool_{i}",
            "description": f"This is a detailed description for tool number {i}. It performs some specific action that requires context.",
            "inputSchema": {
                "type": "object",
                "properties": {
                    "query": {"type": "string"},
                    "max_results": {"type": "integer"}
                },
                "required": ["query"]
            }
        })
    
    json_msg = {
        "jsonrpc": "2.0",
        "id": 1,
        "result": {
            "tools": json_tools
        }
    }
    
    json_str = json.dumps(json_msg, indent=2)
    json_tokens = count_tokens(json_str)
    
    # 4. Results
    print(f"--- Token Reduction Benchmark (10 Tools) ---")
    print(f"JSON-RPC Tokens: {json_tokens}")
    print(f"proto-mcp (BSR ref) Tokens: {proto_tokens}")
    reduction = (1 - (proto_tokens / json_tokens)) * 100
    print(f"Reduction: {reduction:.2f}%")
    
    # Benchmark with 100 tools
    tools_100 = []
    for i in range(100):
        tool = mcp_pb2.Tool(
            name=f"tool_{i}",
            description=f"Description for tool {i}",
            bsr_ref=f"buf.build/acme/tools/tool_{i}:v1"
        )
        tools_100.append(tool)
    
    proto_100 = mcp_pb2.ListToolsResponse(tools=tools_100)
    proto_100_tokens = count_tokens(base64.b64encode(proto_100.SerializeToString()).decode("utf-8"))
    
    json_100_tools = []
    for i in range(100):
        json_100_tools.append({
            "name": f"tool_{i}",
            "description": f"Description for tool {i}",
            "inputSchema": {
                "type": "object",
                "properties": {
                    "q": {"type": "string"}
                }
            }
        })
    json_100_tokens = count_tokens(json.dumps(json_100_tools))
    
    print(f"\n--- Token Reduction Benchmark (100 Tools) ---")
    print(f"JSON-RPC Tokens: {json_100_tokens}")
    print(f"proto-mcp (BSR ref) Tokens: {proto_100_tokens}")
    reduction_100 = (1 - (proto_100_tokens / json_100_tokens)) * 100
    print(f"Reduction: {reduction_100:.2f}%")

if __name__ == "__main__":
    benchmark()
