#!/usr/bin/env bash

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
CLAUDE_BIN="${CLAUDE_BIN:-/Users/tucker/.claude/local/claude}"
MODEL="${CLAUDE_MODEL:-}"
CONFIG="$(mktemp)"
ATTEMPTS="${CLAUDE_ATTEMPTS:-5}"
TIMEOUT="${CLAUDE_TIMEOUT:-60}"
DELAY="${CLAUDE_DELAY:-1.5}"

cleanup() {
  rm -f "$CONFIG"
}
trap cleanup EXIT

(
  cd "$ROOT/go"
  go build -o mcproto ./cmd/mcproto/main.go
)

PROMPT=${1:-"You have access to MCP tools search_registry, resolve_schema, and call_tool. Find a tool in the mcpb registry that can list data sources for analytics, resolve its schema, then call it. Use call_tool with the bsr_ref and tool_name if available. If arguments are required, supply minimal valid values. Return only the final tool output."}

cat > "$CONFIG" <<EOF
{
  "mcproto": {
    "command": "$ROOT/go/mcproto",
    "args": ["--transport", "stdio"],
    "env": {
      "BUF_TOKEN": "${BUF_TOKEN}",
      "GITHUB_PERSONAL_ACCESS_TOKEN": "${GITHUB_PERSONAL_ACCESS_TOKEN}"
    }
  }
}
EOF

python3 "$ROOT/scripts/claude_discovery_loop.py" \
  --claude-bin "$CLAUDE_BIN" \
  --config "$CONFIG" \
  --prompt "$PROMPT" \
  --model "$MODEL" \
  --debug "${CLAUDE_DEBUG:-}" \
  ${CLAUDE_VERBOSE:+--verbose} \
  --attempts "$ATTEMPTS" \
  --timeout "$TIMEOUT" \
  --delay "$DELAY"
