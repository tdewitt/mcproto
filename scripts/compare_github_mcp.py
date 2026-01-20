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
import shutil
import subprocess
import sys
import tempfile
import time
from dataclasses import dataclass
from pathlib import Path
from typing import Dict, Optional, Tuple


# ----- Utility dataclasses ----------------------------------------------------

@dataclass
class RunResult:
    label: str
    prompt: str
    stdout: str
    duration_seconds: float
    token_count_prompt: int
    token_count_output: int
    issue_urls: Tuple[str, ...]

    @property
    def token_count_total(self) -> int:
        return self.token_count_prompt + self.token_count_output


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

    if proc.returncode != 0:
        raise RuntimeError(
            f"{label} run failed (exit {proc.returncode}):\nSTDOUT:\n{proc.stdout}\nSTDERR:\n{proc.stderr}"
        )

    stdout = proc.stdout
    token_prompt = count_tokens(prompt)
    token_output = count_tokens(stdout)
    issues = tuple(re.findall(r"https?://github.com/[^\s)]+/issues/\\d+", stdout))

    return RunResult(
        label=label,
        prompt=prompt,
        stdout=stdout,
        duration_seconds=duration,
        token_count_prompt=token_prompt,
        token_count_output=token_output,
        issue_urls=issues,
    )


# ----- Main orchestration ----------------------------------------------------

def main() -> int:
    repo_root = Path(__file__).resolve().parent.parent
    claude_bin = Path(os.environ.get("CLAUDE_BIN", "/Users/tucker/.claude/local/claude"))
    model = os.environ.get("CLAUDE_MODEL", "")
    prompt = (
        "You have MCP tools available. Create an issue in the GitHub repo tdewitt/mcproto "
        "with a short unique token in the title and body. Return the issue URL."
    )

    # Load secrets from .env if present.
    env_path = repo_root / ".env"
    env = load_env_file(env_path)
    buf_token = env.get("BUF_TOKEN", "")
    gh_pat = env.get("GITHUB_PERSONAL_ACCESS_TOKEN", "")
    if not gh_pat:
        raise SystemExit("GITHUB_PERSONAL_ACCESS_TOKEN is required (set in .env).")

    with tempfile.TemporaryDirectory() as tmp:
        tmpdir = Path(tmp)

        # Build configs for both scenarios.
        official_cfg = build_official_config(tmpdir, gh_pat)
        proto_cfg = build_proto_config(tmpdir, repo_root, buf_token, gh_pat)

        # Run official HTTP GitHub MCP.
        official = run_claude(
            label="official_github_mcp",
            claude_bin=claude_bin,
            config_path=official_cfg,
            prompt=prompt,
            model=model or None,
        )

        # Run proto MCP (BSR discovery).
        proto = run_claude(
            label="proto_mcp",
            claude_bin=claude_bin,
            config_path=proto_cfg,
            prompt=prompt,
            model=model or None,
        )

        # Build comparison summary.
        summary = {
            "prompt": prompt,
            "runs": [
                {
                    "label": official.label,
                    "duration_seconds": official.duration_seconds,
                    "token_prompt": official.token_count_prompt,
                    "token_output": official.token_count_output,
                    "token_total": official.token_count_total,
                    "issue_urls": official.issue_urls,
                },
                {
                    "label": proto.label,
                    "duration_seconds": proto.duration_seconds,
                    "token_prompt": proto.token_count_prompt,
                    "token_output": proto.token_count_output,
                    "token_total": proto.token_count_total,
                    "issue_urls": proto.issue_urls,
                },
            ],
        }

        print(json.dumps(summary, indent=2))

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
