import subprocess
import os
import sys
import struct
from typing import Any

# Add python directory to path
sys.path.append(os.path.join(os.getcwd(), 'python'))

from mcp.stdio import StdioReader, StdioWriter
from mcp.bsr import BSRClient
from mcp.registry import Registry
from mcp import mcp_pb2
from google.protobuf.any_pb2 import Any as ProtoAny

def run_blind_demo():
    print("\n" + "="*65)
    print(" BLIND CLIENT DEMO (Late Binding)")
    print("="*65 + "\n")

    # 1. Compile Go Dynamic Server
    print("Building Go mcproto server...")
    subprocess.run(["go", "build", "-o", "mcproto", "./cmd/mcproto/main.go"], cwd="go", check=True)
    server_path = os.path.join(os.getcwd(), "go", "mcproto")

    # 2. Setup Client
    bsr = BSRClient()
    registry = Registry(bsr)

    process = subprocess.Popen(
        [server_path, "--transport", "stdio"],
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=sys.stderr,
        bufsize=0
    )
    
    writer = StdioWriter(process.stdin)
    reader = StdioReader(process.stdout)

    try:
        # 3. Discovery: Find tools
        print("Discovering tools...")
        msg = mcp_pb2.MCPMessage(id=1)
        msg.list_tools_request.SetInParent() # Empty request
        writer.write_message(msg)
        
        resp = reader.read_message()
        tool = resp.list_tools_response.tools[0]
        print(f"Discovered Tool: {tool.name}")
        print(f"BSR Reference:   {tool.bsr_ref}")

        # 4. Late Binding: Fetch schema from BSR
        print("\nFetching schema from BSR (Late Binding)...")
        ToolRequestClass = registry.resolve(tool.bsr_ref)
        print(f"Dynamically generated class: {ToolRequestClass.__name__}")

        # 5. Call Tool: Construct dynamic request
        print("\nCalling tool with dynamic payload...")
        # Neither this script nor the library has 'id' or 'name' in source
        req_instance = ToolRequestClass(id="demo-id", name="demo-module")
        
        args_any = ProtoAny()
        args_any.Pack(req_instance)
        
        call_msg = mcp_pb2.MCPMessage(id=2)
        call_msg.call_tool_request.name = tool.name
        call_msg.call_tool_request.arguments.CopyFrom(args_any)
        writer.write_message(call_msg)

        # 6. Unpack Response
        response_msg = reader.read_message()
        if response_msg.call_tool_response.HasField("success"):
            text = response_msg.call_tool_response.success.content[0].text
            print(f"SUCCESS: Server responded -> '{text}'")
            
    finally:
        process.terminate()
        if os.path.exists(server_path):
            os.remove(server_path)

if __name__ == "__main__":
    run_blind_demo()