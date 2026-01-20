#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
CLAUDE_BIN="${CLAUDE_BIN:-/Users/tucker/.claude/local/claude}"
CONFIG="$ROOT/scripts/claude_mcp.json"
MODEL="${CLAUDE_MODEL:-haiku}"

(
  cd "$ROOT/go"
  go build -o mcproto ./cmd/mcproto/main.go
)

PROMPT=${1:-"You have access to MCP tools search_registry, resolve_schema, and call_tool. Find a tool in the mcpb registry that can list data sources for analytics, resolve its schema, then call it. Use call_tool with the bsr_ref and tool_name if available. If arguments are required, supply minimal valid values. Return only the final tool output."}

"$CLAUDE_BIN" \
  --print \
  --model "$MODEL" \
  --strict-mcp-config \
  --mcp-config "$CONFIG" \
  --tools "" \
  --permission-mode dontAsk \
  "$PROMPT"
