import json
import tiktoken
import sys
import os

def count_tokens(text):
    encoding = tiktoken.get_encoding("cl100k_base")
    return len(encoding.encode(text))

# 1. Explicit Schema Generator (Simulating Legacy MCP)
def get_explicit_mcp_context():
    tools = [
        {
            "name": "fetch_activity_stream",
            "description": "Extracts a stream of raw user activity events from the source database.",
            "inputSchema": {"type": "object", "properties": {"source_id": {"type": "string"}}}
        },
        {
            "name": "apply_data_mapping",
            "description": "Applies a set of transformation rules to an event's payload.",
            "inputSchema": {"type": "object", "properties": {"event": {"type": "object"}}}
        },
        {
            "name": "write_to_warehouse",
            "description": "Commits a batch of events to the long-term analytics warehouse.",
            "inputSchema": {"type": "object", "properties": {"target": {"type": "string"}}}
        }
    ]
    # Pad to 12 tools
    for i in range(9):
        tools.append({"name": f"extra_tool_{i}", "description": "Misc tool", "inputSchema": {"type": "object"}})
    return json.dumps(tools, indent=2)

# 2. Anthropic Search Simulator
def get_anthropic_search_context():
    search_tool = {"name": "search_tools", "description": "Search for tools.", "inputSchema": {"type": "object"}}
    return json.dumps([search_tool], indent=2)

# 3. proto-mcp Summary Generator
def get_proto_mcp_context():
    tools_info = [
        ("fetch_activity_stream", "Extracts a stream of raw user activity events."),
        ("apply_data_mapping", "Applies transformation rules."),
        ("write_to_warehouse", "Commits batch to warehouse.")
    ]
    for i in range(9):
        tools_info.append((f"extra_tool_{i}", "Misc tool"))
    
    context = ""
    for name, desc in tools_info:
        context += f"tool: {name}\ndesc: {desc}\nref: buf.build/misfit/analytics/v1.{name}:main\n\n"
    return context

def run_showdown():
    print("\n" + "="*65)
    print(" THE FLUID ANALYTICS PIPELINE SHOWDOWN")
    print("="*65)
    
    task_instructions = "TASK: Extract 'marketing_leads' from the source, apply mapping, and load to 'SalesWarehouse'."
    instr_tokens = count_tokens(task_instructions)

    # --- SCENARIO 1: EXPLICIT MCP ---
    prompt_1 = get_explicit_mcp_context()
    # Turn 1: Instructions + Full Schemas
    t1_tokens = count_tokens(prompt_1) + instr_tokens
    # Turns 2-4: LLM Decisions + Results (estimated 100 tokens per turn for simplicity)
    total_explicit = t1_tokens + (3 * 100)

    # --- SCENARIO 2: ANTHROPIC SEARCH ---
    prompt_2 = get_anthropic_search_context()
    # Turn 1: Instructions + Search Tool
    t2_turn1 = count_tokens(prompt_2) + instr_tokens
    # Turn 2: Search Result (3 Schemas) + Action
    search_result_schemas = json.dumps([t for t in json.loads(get_explicit_mcp_context())[:3]])
    t2_turn2 = count_tokens(search_result_schemas) + 100
    # Turns 3-5: Actions + Results
    total_search = t2_turn1 + t2_turn2 + (3 * 100)

    # --- SCENARIO 3: PROTO-MCP (The Winner) ---
    prompt_3 = get_proto_mcp_context()
    # Turn 1: Instructions + Summaries (All 12 visible)
    t3_turn1 = count_tokens(prompt_3) + instr_tokens
    # Turns 2-4: Actions + Results (No schemas in context!)
    total_proto = t3_turn1 + (3 * 100)

    print(f"\nRESULTS FOR THE 'DATA BRIDGE' TASK (Cumulative Tokens):")
    print(f"Mode 1: Explicit MCP:    {total_explicit:,} tokens")
    print(f"Mode 2: Anthropic Search: {total_search:,} tokens")
    print(f"Mode 3: proto-mcp:        {total_proto:,} tokens")
    
    saving_vs_explicit = (1 - total_proto/total_explicit) * 100
    saving_vs_search = (1 - total_proto/total_search) * 100
    
    print(f"\nSUMMARY:")
    print(f"proto-mcp is {saving_vs_explicit:.1f}% more context-efficient than Legacy MCP.")
    print(f"proto-mcp is {saving_vs_search:.1f}% more context-efficient than Tool Search (and saves 1 round-trip turn).")
    print("="*65 + "\n")

    # Generate the Summary Report
    with open("docs/spike_summary.md", "w") as f:
        f.write("# Final Spike Summary: proto-mcp Efficiency\n\n")
        f.write("## Token Showdown Results\n")
        f.write(f"| Mode | Total Task Tokens | Round-Trips | Efficiency |\n")
        f.write(f"| :--- | :--- | :--- | :--- |\n")
        f.write(f"| Explicit MCP | {total_explicit:,} | 4 | Baseline |\n")
        f.write(f"| Anthropic Search | {total_search:,} | 5 | +1 turn latency |\n")
        f.write(f"| **proto-mcp** | **{total_proto:,}** | **4** | **{saving_vs_explicit:.1f}% better** |\n\n")
        f.write("## Conclusion\n")
        f.write("proto-mcp provides the best of both worlds: the **instant availability** of all tools (like Explicit MCP) but with the **lightweight context** of Tool Search. By leveraging the BSR, we remove technical schema noise from the AI conversation entirely.\n")

if __name__ == "__main__":
    run_showdown()