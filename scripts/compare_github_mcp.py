#!/usr/bin/env python3
"""
Compare the official GitHub MCP (hosted HTTP) vs. the proto MCP (stdio + BSR discovery)
for the task "create an issue in tdewitt/mcproto".

This is intentionally verbose and heavily commented because it is demo support code.
It explains what is happening and why at each step.
"""

from __future__ import annotations

import json
import os
import re
import subprocess
import sys
import tempfile
import time
from dataclasses import dataclass
from pathlib import Path
from typing import Dict, Optional, Tuple, List, Any

try:
    import httpx  # type: ignore
except Exception:
    httpx = None  # Will raise later if missing


# ----- Utility dataclasses ----------------------------------------------------

@dataclass
class RunResult:
    label: str
    prompt: str
    stdout: str
    duration_seconds: float
    token_count_prompt: int
    token_count_output: int
    token_count_tools_list: int
    issue_urls: Tuple[str, ...]

    @property
    def token_count_total(self) -> int:
        return self.token_count_prompt + self.token_count_output + self.token_count_tools_list


# ----- Token counting --------------------------------------------------------

def count_tokens(text: str) -> int:
    """
    Count tokens using tiktoken's cl100k_base if available; otherwise fall back
    to a naive whitespace split. For the demo we only need relative comparisons.
    """
    try:
        import tiktoken  # type: ignore

        enc = tiktoken.get_encoding("cl100k_base")
        return len(enc.encode(text))
    except Exception:
        return len(text.split())


# ----- Env loading -----------------------------------------------------------

def load_env_file(path: Path) -> Dict[str, str]:
    """
    Minimal .env loader: key=value per line, no interpolation.
    """
    env: Dict[str, str] = {}
    if not path.exists():
        return env
    for line in path.read_text().splitlines():
        line = line.strip()
        if not line or line.startswith("#") or "=" not in line:
            continue
        key, val = line.split("=", 1)
        env[key.strip()] = val.strip()
    return env


# ----- Config builders -------------------------------------------------------

def build_official_config(tmpdir: Path, pat: str) -> Path:
    """
    Build a Claude MCP config that points to the hosted GitHub MCP over HTTP.
    Auth uses PAT in the Authorization header.
    """
    cfg = {
        "mcpServers": {
            "github-official": {
                "type": "http",
                "url": "https://api.githubcopilot.com/mcp/",
                "headers": {"Authorization": f"Bearer {pat}"},
            }
        }
    }
    path = tmpdir / "official.json"
    path.write_text(json.dumps(cfg, indent=2))
    return path


def build_proto_config(tmpdir: Path, repo_root: Path, buf_token: str, gh_pat: str) -> Path:
    """
    Build and point to the local proto MCP (stdio transport).
    The server uses BSR for schema resolution and go-github with PAT for execution.
    """
    # Build mcproto into the temp dir to avoid polluting the repo.
    mcproto_bin = tmpdir / "mcproto"
    subprocess.run(
        ["go", "build", "-o", str(mcproto_bin), "./cmd/mcproto/main.go"],
        cwd=repo_root / "go",
        check=True,
    )

    cfg = {
        "mcpServers": {
            "mcproto": {
                "command": str(mcproto_bin),
                "args": ["--transport", "stdio"],
                "env": {
                    "BUF_TOKEN": buf_token,
                    "GITHUB_PERSONAL_ACCESS_TOKEN": gh_pat,
                },
            }
        }
    }
    path = tmpdir / "proto.json"
    path.write_text(json.dumps(cfg, indent=2))
    return path


# ----- Claude runner ---------------------------------------------------------

def run_claude(
    label: str,
    claude_bin: Path,
    config_path: Path,
    prompt: str,
    model: Optional[str],
    log_dir: Path,
) -> RunResult:
    """
    Run Claude CLI once with the given MCP config and prompt.
    Capture stdout and wall-clock duration. Tokenize prompt/output for reporting.
    """
    cmd = [
        str(claude_bin),
        "--print",
        "--verbose",
        "--mcp-config",
        str(config_path),
        "--permission-mode",
        "bypassPermissions",
        "--allow-dangerously-skip-permissions",
        prompt,
    ]
    if model:
        cmd[1:1] = ["--model", model]  # insert after binary for readability

    start = time.time()
    proc = subprocess.run(cmd, capture_output=True, text=True)
    end = time.time()
    duration = round(end - start, 2)

    # Save stdout/stderr for inspection.
    (log_dir / f"{label}.stdout.txt").write_text(proc.stdout)
    (log_dir / f"{label}.stderr.txt").write_text(proc.stderr)

    if proc.returncode != 0:
        raise RuntimeError(
            f"{label} run failed (exit {proc.returncode}):\nSTDOUT:\n{proc.stdout}\nSTDERR:\n{proc.stderr}"
        )

    stdout = proc.stdout
    token_prompt = count_tokens(prompt)
    token_output = count_tokens(stdout)
    issues = tuple(re.findall(r"https?://github.com/[^\s)]+/issues/\d+", stdout))

    # Attempt to capture tools/list output if present in verbose logs.
    # Claude CLI does not expose a clean hook, so we rely on a best-effort JSON scrape.
    token_tools_list = 0
    tools_json_matches: List[str] = re.findall(r'"tools":\\s*\\[(.*?)\\]\\s*}', proc.stdout, re.DOTALL)
    if tools_json_matches:
        try:
            snippet = "{" + '"tools":[' + tools_json_matches[0] + "]}"
            token_tools_list = count_tokens(snippet)
        except Exception:
            token_tools_list = 0

    return RunResult(
        label=label,
        prompt=prompt,
        stdout=stdout,
        duration_seconds=duration,
        token_count_prompt=token_prompt,
        token_count_output=token_output,
        token_count_tools_list=token_tools_list,
        issue_urls=issues,
    )


# ----- MCP tools/list helpers ------------------------------------------------

def tools_list_http(url: str, auth_header: str) -> Dict[str, Any]:
    """
    Call tools/list over HTTP MCP (hosted GitHub MCP). This is a minimal client
    that sends initialize + tools/list.
    """
    if httpx is None:
        raise RuntimeError("httpx is required for HTTP tools/list; install in venv.")
    headers = {"Content-Type": "application/json"}
    if auth_header:
        headers["Authorization"] = auth_header
    init_req = {
        "jsonrpc": "2.0",
        "id": 1,
        "method": "initialize",
        "params": {},
    }
    list_req = {
        "jsonrpc": "2.0",
        "id": 2,
        "method": "tools/list",
        "params": {},
    }
    with httpx.Client(timeout=30) as client:
        r1 = client.post(url, json=init_req, headers=headers)
        r1.raise_for_status()
        r2 = client.post(url, json=list_req, headers=headers)
        if r2.status_code != 200:
            raise RuntimeError(f"tools/list HTTP failed: {r2.status_code} {r2.text[:200]}")
        text = r2.text
        # Handle SSE-style response (event: message\\ndata: {...}\\n\\n)
        data_lines = [line.strip()[5:].strip() for line in text.splitlines() if line.startswith("data:")]
        payload = data_lines[-1] if data_lines else text
        try:
            return json.loads(payload)
        except Exception as exc:
            raise RuntimeError(f"tools/list HTTP invalid JSON payload: {payload[:200]}") from exc


def tools_list_stdio(command: str, args: List[str], env: Dict[str, str]) -> Dict[str, Any]:
    """
    Call tools/list over stdio MCP for our proto server.
    """
    init_req = json.dumps({"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {}})
    list_req = json.dumps({"jsonrpc": "2.0", "id": 2, "method": "tools/list", "params": {}})
    payload = f"{init_req}\n{list_req}\n"
    proc = subprocess.run(
        [command] + args,
        input=payload,
        env={**os.environ, **env},
        capture_output=True,
        text=True,
    )
    if proc.returncode != 0:
        raise RuntimeError(f"tools/list stdio failed: {proc.stderr}")
    # Grab the second response (tools/list)
    lines = [l for l in proc.stdout.splitlines() if l.strip()]
    resp = json.loads(lines[-1])
    return resp


# ----- GitHub issue comment helper ------------------------------------------

def parse_issue_url(url: str) -> Optional[Tuple[str, str, int]]:
    """
    Parse a GitHub issue URL into (owner, repo, number).
    """
    m = re.match(r"https?://github\\.com/([^/]+)/([^/]+)/issues/(\\d+)", url)
    if not m:
        return None
    owner, repo, num = m.group(1), m.group(2), int(m.group(3))
    return owner, repo, num


def add_issue_comment(issue_url: str, pat: str, body: str) -> None:
    """
    Add a comment to a GitHub issue using the PAT.
    """
    if httpx is None:
        raise RuntimeError("httpx is required for commenting; install in venv.")
    parsed = parse_issue_url(issue_url)
    if not parsed:
        raise RuntimeError(f"Could not parse issue URL: {issue_url}")
    owner, repo, num = parsed
    api_url = f"https://api.github.com/repos/{owner}/{repo}/issues/{num}/comments"
    headers = {
        "Authorization": f"Bearer {pat}",
        "Accept": "application/vnd.github+json",
    }
    resp = httpx.post(api_url, headers=headers, json={"body": body}, timeout=30)
    if resp.status_code not in (200, 201):
        raise RuntimeError(f"Failed to comment on {issue_url}: {resp.status_code} {resp.text[:200]}")


# ----- Main orchestration ----------------------------------------------------

def main() -> int:
    repo_root = Path(__file__).resolve().parent.parent
    claude_bin = Path(os.environ.get("CLAUDE_BIN", "/Users/tucker/.claude/local/claude"))
    model = os.environ.get("CLAUDE_MODEL", "")
    prompt = (
        "You have MCP tools available. Create an issue in the GitHub repo tdewitt/mcproto "
        "with a short unique token in the title and body. Return the issue URL. "
        "If a GitHub issue tool exists (e.g., issue_write), call it with method=create, "
        "owner=tdewitt, repo=mcproto, and a unique title/body token. "
        "If only meta tools (search_registry/resolve_schema/call_tool) exist, use them to "
        "discover the GitHub CreateIssue schema via BSR, then call it."
    )

    # Load secrets from .env if present.
    env_path = repo_root / ".env"
    env = load_env_file(env_path)
    buf_token = env.get("BUF_TOKEN", "")
    gh_pat = env.get("GITHUB_PERSONAL_ACCESS_TOKEN", "")
    if not gh_pat:
        raise SystemExit("GITHUB_PERSONAL_ACCESS_TOKEN is required (set in .env).")

    tmpdir = Path(tempfile.mkdtemp(prefix="compare_mcp_"))

    # Build configs for both scenarios.
    official_cfg = build_official_config(tmpdir, gh_pat)
    proto_cfg = build_proto_config(tmpdir, repo_root, buf_token, gh_pat)

    # Preload tokens: tools/list
    official_tools = tools_list_http(
        url="https://api.githubcopilot.com/mcp/",
        auth_header=f"Bearer {gh_pat}",
    )
    proto_tools = tools_list_stdio(
        command=str((tmpdir / "mcproto")),
        args=["--transport", "stdio"],
        env={
            "BUF_TOKEN": buf_token,
            "GITHUB_PERSONAL_ACCESS_TOKEN": gh_pat,
        },
    )

    # Run official HTTP GitHub MCP.
    official = run_claude(
        label="official_github_mcp",
        claude_bin=claude_bin,
        config_path=official_cfg,
        prompt=prompt,
        model=model or None,
        log_dir=tmpdir,
    )
    official.token_count_tools_list = count_tokens(json.dumps(official_tools))

    # Run proto MCP (BSR discovery).
    proto = run_claude(
        label="proto_mcp",
        claude_bin=claude_bin,
        config_path=proto_cfg,
        prompt=prompt,
        model=model or None,
        log_dir=tmpdir,
    )
    proto.token_count_tools_list = count_tokens(json.dumps(proto_tools))

    # Build comparison summary.
    summary = {
        "prompt": prompt,
        "runs": [
            {
                "label": official.label,
                "duration_seconds": official.duration_seconds,
                "token_prompt": official.token_count_prompt,
                "token_output": official.token_count_output,
                "token_tools_list": official.token_count_tools_list,
                "token_total": official.token_count_total,
                "issue_urls": official.issue_urls,
            },
            {
                "label": proto.label,
                "duration_seconds": proto.duration_seconds,
                "token_prompt": proto.token_count_prompt,
                "token_output": proto.token_count_output,
                "token_tools_list": proto.token_count_tools_list,
                "token_total": proto.token_count_total,
                "issue_urls": proto.issue_urls,
            },
        ],
    }

    print(json.dumps(summary, indent=2))
    print(f"Logs: {tmpdir}")

    # Post summary comments into the created issues for transparency.
    for run in summary["runs"]:
        urls = run.get("issue_urls", [])
        if not urls:
            continue
        label = run["label"]
        body_lines = [
            f"Automated MCP comparison run: {label}",
            f"- Duration: {run['duration_seconds']}s",
            f"- Tokens: prompt={run['token_prompt']}, output={run['token_output']}, tools_list={run['token_tools_list']}, total={run['token_total']}",
            f"- Prompt: {summary['prompt']}",
        ]
        body = "\n".join(body_lines)
        for issue_url in urls:
            try:
                add_issue_comment(issue_url, gh_pat, body)
            except Exception as exc:
                print(f"Warning: failed to comment on {issue_url}: {exc}", file=sys.stderr)

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
