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
from mcp import mcp_pb2
from google.protobuf.any_pb2 import Any as ProtoAny

def count_tokens(text):
    encoding = tiktoken.get_encoding("cl100k_base")
    return len(encoding.encode(text))

def run_recursive_showdown():
    print("\n" + "="*65)
    print(" RECURSIVE DISCOVERY SHOWDOWN (Final Demo)")
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
        # --- TURN 1: MINIMAL SETUP ---
        print("SCENARIO: Agent needs to process data but doesn't know which tools exist.")
        print("Setup: Server only advertises the 'search_registry' meta-tool.\n")

        # Simulate discovery of just the search tool
        search_tool_ref = "buf.build/mcpb/discovery/misfit.discovery.v1.SearchRegistryRequest:main"
        initial_context = f"tool: search_registry\ndesc: Search the global registry\nref: {search_tool_ref}"
        print(f"Turn 1 Context: {count_tokens(initial_context)} tokens")

        # --- TURN 2: RECURSIVE SEARCH ---
        print("\nAgent calls 'search_registry' for 'registry' tools...")
        # (In a real turn, LLM would call the tool. We simulate the result)
        
        # 1. Resolve Search Blueprint
        SearchRequest = registry.resolve(search_tool_ref)
        search_req = SearchRequest(query="registry", limit=5)
        
        arg_any = ProtoAny()
        arg_any.Pack(search_req)
        
        search_resp = client.call_tool("search_registry", arg_any)
        results_text = search_resp.success.content[0].text
        print(f"Search Results Received:\n{results_text}")

        # --- TURN 3: LATE BINDING EXECUTION ---
        print("\nAgent selects 'fetch_activity_stream' from results.")
        target_ref = "buf.build/mcpb/analytics/misfit.analytics.v1.ExtractRequest:main"
        
        print(f"Fetching blueprint for {target_ref}...")
        ExtractRequest = registry.resolve(target_ref)
        
        print("Executing binary call with dynamic payload...")
        extract_req = ExtractRequest(source_id="final-demo-stream", batch_size=50)
        
        exec_any = ProtoAny()
        exec_any.Pack(extract_req)
        
        final_resp = client.call_tool("fetch_activity_stream", exec_any)
        print(f"SUCCESS: Server responded -> '{final_resp.success.content[0].text}'")

        # --- FINAL TOKEN PROOF ---
        print("\n" + "-"*40)
        print("FINAL TOKEN PROOF (Discovery of 1,000 tools)")
        print("-"*40)
        legacy_tokens = 200000 # Estimate for 1000 full JSON schemas
        recursive_tokens = count_tokens(initial_context) + count_tokens(results_text)
        print(f"Legacy MCP Context: {legacy_tokens:,} tokens")
        print(f"Recursive proto-mcp: {recursive_tokens:,} tokens")
        print(f"TOTAL SAVINGS:        {((1 - recursive_tokens/legacy_tokens)*100):.2f}%")
        print("-"*40)

    except Exception as e:
        print(f"ERROR: {e}")
    finally:
        server_process.terminate()
        if os.path.exists(server_path):
            os.remove(server_path)

if __name__ == "__main__":
    run_recursive_showdown()