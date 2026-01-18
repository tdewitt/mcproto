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

def run_live_showdown():
    print("\n" + "="*65)
    print(" LIVE ANALYTICS PIPELINE SHOWDOWN")
    print("="*65 + "\n")

    # 1. Build and Start Server
    print("Building Go showdown-server...")
    subprocess.run(["go", "build", "-o", "showdown-server", "./cmd/showdown-server/main.go"], cwd="go", check=True)
    server_path = os.path.join(os.getcwd(), "go", "showdown-server")
    
    server_process = subprocess.Popen([server_path], stdout=subprocess.DEVNULL, stderr=sys.stderr)
    time.sleep(2)

    client = GRPCClient()
    bsr = BSRClient()
    registry = Registry(bsr)

    try:
        # --- PHASE 1: DISCOVERY ---
        print("SCENARIO: Extract leads, transform them, and load to warehouse.\n")

        # Get tools list
        resp = client.list_tools()
        
        # Simulated standard MCP Context
        # (This is what standard MCP would send in the first turn)
        legacy_context = json.dumps([{"name": t.name, "description": t.description, "inputSchema": {"type": "object", "properties": {"id": {"type": "string"}}}} for t in resp.tools], indent=2)
        legacy_tokens = count_tokens(legacy_context)

        # Proto-MCP Context (What we actually send)
        proto_context = ""
        for t in resp.tools:
            proto_context += f"tool: {t.name}\n" # Corrected newline
            proto_context += f"desc: {t.description}\n" # Corrected newline
            proto_context += f"ref: {t.bsr_ref}\n\n"
        proto_tokens = count_tokens(proto_context)

        print(f"--- Token Savings (Initial Turn) ---")
        print(f"Legacy MCP (JSON Schemas): {legacy_tokens:,} tokens")
        print(f"proto-mcp (Summaries):     {proto_tokens:,} tokens")
        print(f"IMMEDIATE SAVING:          {((1 - proto_tokens/legacy_tokens)*100):.1f}%")
        print("-" * 40)

        # --- PHASE 2: DYNAMIC EXECUTION (LATE BINDING) ---
        target_tool = next(t for t in resp.tools if t.name == "fetch_activity_stream")
        print(f"\nTarget Tool Found: {target_tool.name}")
        print(f"Blueprint: {target_tool.bsr_ref}")

        print("\nFETCHING BLUEPRINT FROM BSR...")
        RequestClass = registry.resolve(target_tool.bsr_ref)
        print(f"Dynamically Bound Class: {RequestClass.__name__}")

        print("\nEXECUTING BINARY CALL VIA gRPC...")
        req = RequestClass(source_id="misfit-leads-2026", batch_size=100)
        args_any = ProtoAny()
        args_any.Pack(req)
        
        call_resp = client.call_tool(target_tool.name, args_any)
        if call_resp.HasField("success"):
            print(f"SUCCESS: Server responded -> '{call_resp.success.content[0].text}'")

    except Exception as e:
        print(f"ERROR: {e}")
    finally:
        server_process.terminate()
        if os.path.exists(server_path):
            os.remove(server_path)

if __name__ == "__main__":
    run_live_showdown()
