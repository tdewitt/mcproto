import json
import tiktoken
import sys
import os

def count_tokens(text):
    encoding = tiktoken.get_encoding("cl100k_base")
    return len(encoding.encode(text))

# 1. Explicit Schema Generator (Simulating Legacy MCP)
# Converts our ETL tools into heavy JSON-RPC definitions
def get_explicit_mcp_context():
    tools = [
        {
            "name": "fetch_activity_stream",
            "description": "Extracts a stream of raw user activity events from the source database.",
            "inputSchema": {
                "type": "object",
                "properties": {
                    "source_id": {"type": "string", "description": "The unique identifier for the data source"},
                    "batch_size": {"type": "integer", "description": "Number of events to fetch per batch", "minimum": 1, "maximum": 1000},
                    "filters": {
                        "type": "object",
                        "description": "Key-value pairs for filtering the stream",
                        "additionalProperties": {"type": "string"}
                    }
                },
                "required": ["source_id"]
            }
        },
        {
            "name": "apply_data_mapping",
            "description": "Applies a set of transformation rules to an event's payload.",
            "inputSchema": {
                "type": "object",
                "properties": {
                    "event": {
                        "type": "object",
                        "properties": {
                            "id": {"type": "string"},
                            "domain": {"type": "string"},
                            "timestamp": {"type": "string", "format": "date-time"},
                            "payload": {"type": "object", "additionalProperties": True}
                        }
                    },
                    "rules": {
                        "type": "array",
                        "items": {
                            "type": "object",
                            "properties": {
                                "target_field": {"type": "string"},
                                "transform_op": {"type": "string", "enum": ["uppercase", "hash", "mask"]},
                                "fallback_value": {"type": "string"}
                            }
                        }
                    }
                }
            }
        },
        # ... We will simulate 10 more of these to reach the 12 tools count
    ]
    
    # Pad to 12 tools with similar complexity
    for i in range(10):
        tools.append({
            "name": f"load_tool_{i}",
            "description": f"Mock load tool instance {i} for high-volume data ingestion.",
            "inputSchema": {
                "type": "object",
                "properties": {
                    "target": {"type": "string"},
                    "options": {"type": "object", "properties": {"overwrite": {"type": "boolean"}}}
                }
            }
        })
        
    return json.dumps(tools, indent=2)

# 2. Anthropic Search Simulator (Simulating multi-turn discovery)
def get_anthropic_search_context():
    # Phase 1: Only the search tool is visible
    search_tool = {
        "name": "search_tools",
        "description": "Search for available tools by keyword.",
        "inputSchema": {
            "type": "object",
            "properties": {"query": {"type": "string"}}
        }
    }
    return json.dumps([search_tool], indent=2)

# 3. proto-mcp Summary Generator
def get_proto_mcp_context():
    summaries = []
    # 12 tools
    tools_info = [
        ("fetch_activity_stream", "Extracts a stream of raw user activity events."),
        ("list_data_sources", "Lists all available data domains."),
        ("get_stream_metadata", "Retrieves schema and throughput metadata."),
        ("apply_data_mapping", "Applies transformation rules."),
        ("enrich_geo_location", "Enriches data with geographical info."),
        ("validate_event_schema", "Checks if payload conforms to schema."),
        ("anonymize_pii_fields", "Masks or hashes PII."),
        ("write_to_warehouse", "Commits batch to warehouse."),
        ("emit_to_webhook", "Forwards to external webhook."),
        ("archive_to_cold_storage", "Moves to S3 cold storage."),
        ("push_to_valkey_cache", "Updates Valkey JSON cache."),
        ("log_pipeline_event", "Logs summary of ETL operation.")
    ]
    
    context = ""
    for name, desc in tools_info:
        context += f"tool: {name}\ndesc: {desc}\nref: buf.build/misfit/analytics/v1.{name}:main\n\n"
    return context

def run_simulation():
    print("--- Phase 2: Token Simulation & Logic ---")
    
    explicit_ctx = get_explicit_mcp_context()
    explicit_tokens = count_tokens(explicit_ctx)
    
    search_ctx = get_anthropic_search_context()
    search_tokens = count_tokens(search_ctx)
    
    proto_ctx = get_proto_mcp_context()
    proto_tokens = count_tokens(proto_ctx)
    
    print(f"Explicit MCP Context: {explicit_tokens} tokens")
    print(f"Anthropic Search (Init): {search_tokens} tokens")
    print(f"proto-mcp Context: {proto_tokens} tokens")
    
    # Save results for Phase 3
    results = {
        "explicit": explicit_tokens,
        "search_init": search_tokens,
        "proto": proto_tokens
    }
    with open("docs/simulation_results.json", "w") as f:
        json.dump(results, f)

if __name__ == "__main__":
    run_simulation()
