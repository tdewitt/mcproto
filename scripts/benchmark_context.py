import json
import tiktoken

def count_tokens(text):
    encoding = tiktoken.get_encoding("cl100k_base")
    return len(encoding.encode(text))

def benchmark_context():
    # 10 Tools
    tools_data = []
    for i in range(10):
        tools_data.append({
            "name": f"tool_{i}",
            "description": f"This is a detailed description for tool number {i}. It performs some specific action that requires context.",
            "inputSchema": {
                "type": "object",
                "properties": {
                    "query": {"type": "string"},
                    "max_results": {"type": "integer"},
                    "filter": {"type": "string"},
                    "options": {"type": "object", "properties": {"debug": {"type": "boolean"}}}
                },
                "required": ["query"]
            }
        })

    # Standard MCP: Sends the full JSON for all tools to the LLM context
    mcp_json_str = json.dumps(tools_data, indent=2)
    mcp_tokens = count_tokens(mcp_json_str)

    # proto-mcp: The LLM ONLY sees the name and description. 
    # The schema is handled by the transport layer and the BSR.
    proto_mcp_context = ""
    for t in tools_data:
        proto_mcp_context += f"tool: {t["name"]}\ndescription: {t["description"]}\n\n"
    
    proto_tokens = count_tokens(proto_mcp_context)

    print(f"--- Context Token Comparison (10 Tools) ---")
    print(f"Standard MCP (Full JSON Schema): {mcp_tokens} tokens")
    print(f"proto-mcp (Name + Description only): {proto_tokens} tokens")
    print(f"Reduction: {(1 - proto_tokens/mcp_tokens)*100:.2f}%")

    # 100 Tools
    tools_100 = []
    for i in range(100):
        tools_100.append({
            "name": f"tool_{i}",
            "description": f"Short desc {i}",
            "inputSchema": {"type": "object", "properties": {"q": {"type": "string"}}}
        })
    
    mcp_100_tokens = count_tokens(json.dumps(tools_100))
    proto_100_context = ""
    for t in tools_100:
        proto_100_context += f"{t["name"]}: {t["description"]}\n"
    proto_100_tokens = count_tokens(proto_100_context)

    print(f"\n--- Context Token Comparison (100 Tools) ---")
    print(f"Standard MCP (Full JSON Schema): {mcp_100_tokens} tokens")
    print(f"proto-mcp (Name + Description only): {proto_100_tokens} tokens")
    print(f"Reduction: {(1 - proto_100_tokens/mcp_100_tokens)*100:.2f}%")

if __name__ == "__main__":
    benchmark_context()
