#!/usr/bin/env bash

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
CLAUDE_BIN="${CLAUDE_BIN:-/Users/tucker/.claude/local/claude}"
MODEL="${CLAUDE_MODEL:-}"
CONFIG="$(mktemp)"
ATTEMPTS="${CLAUDE_ATTEMPTS:-5}"
TIMEOUT="${CLAUDE_TIMEOUT:-60}"
DELAY="${CLAUDE_DELAY:-1.5}"

if [[ -f "$ROOT/.env" ]]; then
  set -a
  # shellcheck disable=SC1090
  source "$ROOT/.env"
  set +a
fi

cleanup() {
  rm -f "$CONFIG"
}
trap cleanup EXIT

(
  cd "$ROOT/go"
  go build -o mcproto ./cmd/mcproto/main.go
)

PROMPT=${1:-"You have meta tools: search_registry, resolve_schema, call_tool. I want to create an issue in GitHub using mcproto. First run search_registry with query \"mcproto github\" (mcpb-only) and pick the top GitHub MCP candidate. Resolve the schema for the tool that creates an issue, then call it to open a new issue in the GitHub repo tdewitt/mcproto. Include a short unique token in the title/body. Use call_tool with the bsr_ref and tool_name if available. Return the raw tool output (URL/issue number). Also print a step log with timings in seconds for each action you take (search, resolve, call)."}

cat > "$CONFIG" <<EOF
{
  "mcpServers": {
    "mcproto": {
      "command": "$ROOT/go/mcproto",
      "args": ["--transport", "stdio"],
      "env": {
        "BUF_TOKEN": "${BUF_TOKEN:-}",
        "GITHUB_PERSONAL_ACCESS_TOKEN": "${GITHUB_PERSONAL_ACCESS_TOKEN:-}"
      }
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
