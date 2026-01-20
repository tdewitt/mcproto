"""Run Claude MCP demo with retries until output is produced."""

import argparse
import subprocess
import sys
import time
from typing import List
from shlex import join as shlex_join


def build_command(args: argparse.Namespace) -> List[str]:
    """Build the Claude CLI command."""
    cmd = [
        args.claude_bin,
        "--print",
        "--verbose",
        "--mcp-config",
        args.config,
        "--permission-mode",
        "bypassPermissions",
        "--allow-dangerously-skip-permissions",
    ]
    if args.model:
        cmd.extend(["--model", args.model])
    if args.debug:
        cmd.extend(["--debug", args.debug])
    cmd.append(args.prompt)
    return cmd


def run_once(cmd: List[str], timeout: int) -> subprocess.CompletedProcess:
    """Run the Claude CLI once with a timeout."""
    return subprocess.run(
        cmd,
        capture_output=True,
        text=True,
        timeout=timeout,
    )


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--claude-bin", required=True)
    parser.add_argument("--config", required=True)
    parser.add_argument("--prompt", required=True)
    parser.add_argument("--model", default="")
    parser.add_argument("--debug", default="")
    parser.add_argument("--verbose", action="store_true")
    parser.add_argument("--attempts", type=int, default=5)
    parser.add_argument("--timeout", type=int, default=60)
    parser.add_argument("--delay", type=float, default=1.5)
    args = parser.parse_args()

    cmd = build_command(args)
    cmd_str = shlex_join(cmd)

    for attempt in range(1, args.attempts + 1):
        print(f"[demo] attempt {attempt}/{args.attempts}", file=sys.stderr)
        print(f"[demo] command: {cmd_str}", file=sys.stderr)
        start = time.time()
        try:
            result = run_once(cmd, args.timeout)
            duration = time.time() - start
        except subprocess.TimeoutExpired:
            duration = time.time() - start
            print(f"[demo] attempt timed out after {duration:.2f}s", file=sys.stderr)
            time.sleep(args.delay)
            continue

        stdout = (result.stdout or "").strip()
        stderr = (result.stderr or "").strip()

        if stdout:
            print(stdout)
            print(f"[demo] attempt {attempt} completed in {duration:.2f}s", file=sys.stderr)
            return 0

        if stderr:
            print("[demo] stderr:", file=sys.stderr)
            print(stderr, file=sys.stderr)

        if result.returncode not in (0, None):
            print(
                f"[demo] exit code {result.returncode}",
                file=sys.stderr,
            )
        print(f"[demo] attempt {attempt} finished in {duration:.2f}s with no stdout", file=sys.stderr)

        time.sleep(args.delay)

    print("[demo] no output after retries", file=sys.stderr)
    return 1


if __name__ == "__main__":
    raise SystemExit(main())
