import json
import tiktoken

def count_tokens(text):
    encoding = tiktoken.get_encoding("cl100k_base")
    return len(encoding.encode(text))

def prove_it():
    # Realistic "Heavy" Tool
    tool_name = "search_documents"
    description = "Searches a massive vector database for relevant documents based on semantic similarity."
    schema = {
        "type": "object",
        "properties": {
            "query": {"type": "string", "description": "The search query"},
            "top_k": {"type": "integer", "default": 10},
            "filters": {
                "type": "array",
                "items": {
                    "type": "object",
                    "properties": {
                        "field": {"type": "string"},
                        "operator": {"type": "string", "enum": ["eq", "neq", "gt", "lt"]},
                        "value": {"type": "string"}
                    }
                }
            },
            "include_metadata": {"type": "boolean"}
        },
        "required": ["query"]
    }

    # Standard MCP Representation
    mcp_view = json.dumps({"name": tool_name, "description": description, "inputSchema": schema}, indent=2)
    mcp_tokens = count_tokens(mcp_view)

    # proto-mcp Representation
    proto_view = f"tool: {tool_name}\ndescription: {description}\n(Schema resolved via BSR: buf.build/acme/tools/{tool_name})"
    proto_tokens = count_tokens(proto_view)

    print("="*60)
    print(" VISUAL PROOF: CONTEXT WINDOW COMPARISON (1 TOOL)")
    print("="*60)
    print(f"--- STANDARD MCP ({mcp_tokens} tokens) ---")
    print(mcp_view)
    print("\n" + "-"*30 + "\n")
    print(f"--- PROTO-MCP ({proto_tokens} tokens) ---")
    print(proto_view)
    print("="*60)

    # Economic Proof (for 100 tools)
    total_mcp = mcp_tokens * 100
    total_proto = proto_tokens * 100
    saved = total_mcp - total_proto
    
    # Assuming GPT-4o input prices ($2.50 per 1M tokens)
    price_per_m = 2.50
    monthly_calls = 10000
    dollars_saved = (saved * monthly_calls / 1000000) * price_per_m

    print(f"\nSCALED PROOF (100 Tools, 10k requests/mo):")
    print(f"Standard MCP Tokens: {total_mcp * monthly_calls:,}")
    print(f"proto-mcp Tokens:    {total_proto * monthly_calls:,}")
    print(f"Tokens Saved:        {saved * monthly_calls:,}")
    print(f"ESTIMATED SAVINGS:   ${dollars_saved:,.2f} / month")
    print("="*60)

if __name__ == "__main__":
    prove_it()
