import subprocess
import os
import sys
import time
import json
import tiktoken
from typing import List

# Add python directory to path
sys.path.append(os.path.join(os.getcwd(), 'python'))

from mcp.grpc_client import GRPCClient
from mcp.bsr import BSRClient
from mcp.registry import Registry
from google.protobuf.any_pb2 import Any as ProtoAny

def count_tokens(text):
    encoding = tiktoken.get_encoding("cl100k_base")
    return len(encoding.encode(text))

def run_boss_demo():
    print("\n" + "="*65)
    print(" THE 1,000 TOOL CHALLENGE (Boss Demo)")
    print("="*65 + "\n")

    # 1. Start Server
    print("Building and starting Go gRPC server...")
    subprocess.run(["go", "build", "-o", "grpc-server", "./cmd/grpc-server/main.go"], cwd="go", check=True)
    server_path = os.path.join(os.getcwd(), "go", "grpc-server")
    server_process = subprocess.Popen([server_path], stdout=subprocess.DEVNULL, stderr=sys.stderr)
    time.sleep(2)

    client = GRPCClient()
    bsr = BSRClient()
    registry = Registry(bsr)

    try:
        # 2. The Benchmark
        print("SCENARIO: An Agent needs to find a tool to read git files among 1,000 tools.\n")

        # Standard MCP Simulation (Estimating)
        # Average tool schema is ~200 tokens. 1000 tools = 200,000 tokens.
        legacy_tokens = 200000 
        
        # Proto-MCP Full Listing (Name + Desc)
        all_tools = client.list_tools()
        proto_full_text = ""
        for tool in all_tools.tools:
            proto_full_text += f"tool: {tool.name}\ndesc: {tool.description}\n\n"
        full_listing_tokens = count_tokens(proto_full_text)

        # Proto-MCP Search (The Winner)
        search_results = client.list_tools(query="git_read")
        search_text = ""
        for tool in search_results.tools:
            search_text += f"tool: {tool.name}\ndesc: {tool.description}\n\n"
        search_tokens = count_tokens(search_text)

        print(f"--- Token Comparison (Context Usage) ---")
        print(f"Standard MCP (Estimate): {legacy_tokens:,} tokens")
        print(f"Proto-MCP (Full List):   {full_listing_tokens:,} tokens ({(1 - full_listing_tokens/legacy_tokens)*100:.1f}% saving)")
        print(f"Proto-MCP (Search):      {search_tokens:,} tokens ({(1 - search_tokens/legacy_tokens)*100:.1f}% saving)")
        print("-" * 40)

        # 3. Dynamic Execution
        target_tool = search_results.tools[0]
        print(f"\nDiscovered Tool: {target_tool.name}")
        print(f"Calling tool dynamically via gRPC + BSR...")

        # Late Binding
        ToolClass = registry.resolve(target_tool.bsr_ref)
        req = ToolClass(id="boss-demo", name="misfit-repo")
        
        arg_any = ProtoAny()
        arg_any.Pack(req)
        
        resp = client.call_tool(target_tool.name, arg_any)
        if resp.HasField("success"):
            print(f"SUCCESS: Server responded -> '{resp.success.content[0].text}'")

    except Exception as e:
        print(f"ERROR: {e}")
    finally:
        server_process.terminate()
        if os.path.exists(server_path):
            os.remove(server_path)

if __name__ == "__main__":
    run_boss_demo()