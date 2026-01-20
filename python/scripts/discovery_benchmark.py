"""Benchmark on-demand tool discovery with BSR-backed schemas."""

import argparse
import json
import os
import subprocess
import sys
import time
import typing

DEFAULT_ADDR = "localhost:50051"
DEFAULT_QUERY = "git_read"
SERVER_BINARY = "mcproto"


class _FallbackEncoder:
    """Fallback tokenizer when tiktoken is unavailable."""

    def encode(self, text: str) -> list[str]:
        """Return a naive token list based on whitespace."""
        return text.split()


def get_encoder() -> typing.Any:
    """Return the cl100k_base encoder or a fallback.

    Returns:
        Token encoder with an encode method.
    """
    try:
        import tiktoken
    except ImportError:
        return _FallbackEncoder()
    return tiktoken.get_encoding("cl100k_base")


def token_count(text: str, encoder: typing.Any = None) -> int:
    """Count tokens for a text payload.

    Args:
        text: Input string to tokenize.
        encoder: Optional encoder with an encode method.

    Returns:
        Token count.
    """
    if encoder is None:
        encoder = get_encoder()
    return len(encoder.encode(text))


def build_proto_listing_payload(tools: typing.Iterable[typing.Any]) -> str:
    """Build a proto-mcp style listing payload.

    Args:
        tools: Iterable of tool-like objects.

    Returns:
        JSON payload string containing name/description/bsr_ref.
    """
    entries = []
    for tool in tools:
        entry = {
            "name": tool.name,
            "description": tool.description,
        }
        if getattr(tool, "bsr_ref", ""):
            entry["bsr_ref"] = tool.bsr_ref
        entries.append(entry)
    return json.dumps(
        {"tools": entries},
        separators=(",", ":"),
        sort_keys=True,
    )


def build_legacy_listing_payload(
    tools: typing.Iterable[typing.Any],
    schema_by_ref: dict[str, typing.Any],
) -> str:
    """Build a legacy MCP listing payload with inline schemas.

    Args:
        tools: Iterable of tool-like objects.
        schema_by_ref: Mapping from BSR ref to schema dict.

    Returns:
        JSON payload string containing inline schema per tool.
    """
    entries = []
    for tool in tools:
        entry = {
            "name": tool.name,
            "description": tool.description,
        }
        if getattr(tool, "bsr_ref", ""):
            entry["inputSchema"] = schema_by_ref[tool.bsr_ref]
        entries.append(entry)
    return json.dumps(
        {"tools": entries},
        separators=(",", ":"),
        sort_keys=True,
    )


def collect_bsr_refs(tools: typing.Iterable[typing.Any]) -> list[str]:
    """Collect unique BSR refs from tools.

    Args:
        tools: Iterable of tool-like objects.

    Returns:
        Sorted list of unique BSR refs.
    """
    refs = {tool.bsr_ref for tool in tools if getattr(tool, "bsr_ref", "")}
    return sorted(refs)


def fetch_schema_map(
    bsr_client: typing.Any,
    refs: typing.Iterable[str],
    parse_ref: typing.Callable[[str], typing.Any],
) -> dict[str, typing.Any]:
    """Fetch descriptor sets from BSR and convert to dicts.

    Args:
        bsr_client: BSR client instance.
        refs: Iterable of BSR ref strings.
        parse_ref: Function to parse a BSR ref string.

    Returns:
        Mapping from BSR ref to schema dict.
    """
    from google.protobuf import json_format

    schema_map = {}
    for ref_str in refs:
        ref = parse_ref(ref_str)
        fds = bsr_client.fetch_descriptor_set(ref)
        schema_map[ref_str] = json_format.MessageToDict(
            fds,
            preserving_proto_field_name=True,
        )
    return schema_map


def start_server(addr: str, build: bool) -> tuple[subprocess.Popen, str]:
    """Build and start the mcproto server.

    Args:
        addr: Address to bind the gRPC server.
        build: Whether to build the binary before starting.

    Returns:
        Tuple of (process, binary_path).
    """
    repo_root = os.getcwd()
    server_path = os.path.join(repo_root, "go", SERVER_BINARY)
    if build:
        subprocess.run(
            ["go", "build", "-o", SERVER_BINARY, "./cmd/mcproto/main.go"],
            cwd=os.path.join(repo_root, "go"),
            check=True,
        )
    process = subprocess.Popen(
        [server_path, "--addr", addr],
        stdout=subprocess.DEVNULL,
        stderr=sys.stderr,
    )
    time.sleep(1)
    return process, server_path


def format_savings(value: int, baseline: int) -> str:
    """Format a savings percentage string.

    Args:
        value: Current token count.
        baseline: Baseline token count.

    Returns:
        Savings percentage string.
    """
    if baseline <= 0:
        return "0.0%"
    savings = (1 - (value / baseline)) * 100
    return f"{savings:.1f}%"


def run_benchmark(args: argparse.Namespace) -> int:
    """Run the discovery benchmark.

    Args:
        args: Parsed CLI arguments.

    Returns:
        Exit code.
    """
    sys.path.append(os.path.join(os.getcwd(), "python"))

    from mcp import bsr
    from mcp import grpc_client
    from mcp import registry
    from google.protobuf import any_pb2

    process = None
    server_path = None
    if not args.no_server:
        process, server_path = start_server(args.addr, not args.no_build)

    client = grpc_client.GRPCClient(target=args.addr)
    bsr_client = bsr.BSRClient()
    dynamic_registry = registry.Registry(bsr_client)

    try:
        tools = client.list_tools().tools
        if args.limit and args.limit > 0:
            tools = tools[: args.limit]
        search_tools = client.list_tools(query=args.query).tools

        print(f"Catalog tools: {len(tools)}")
        print(f"Search results: {len(search_tools)}")

        refs = collect_bsr_refs(tools)
        schema_by_ref = fetch_schema_map(
            bsr_client,
            refs,
            bsr.BSRRef.parse,
        )

        proto_all_payload = build_proto_listing_payload(tools)
        legacy_payload = build_legacy_listing_payload(tools, schema_by_ref)
        proto_search_payload = build_proto_listing_payload(search_tools)

        encoder = get_encoder()
        legacy_tokens = token_count(legacy_payload, encoder)
        proto_all_tokens = token_count(proto_all_payload, encoder)
        proto_search_tokens = token_count(proto_search_payload, encoder)

        print("\n--- Token Comparison (Context Usage) ---")
        print(
            "Legacy MCP (inline schemas): "
            f"{legacy_tokens:,} tokens"
        )
        print(
            "Proto-mcp (full list):       "
            f"{proto_all_tokens:,} tokens "
            f"({format_savings(proto_all_tokens, legacy_tokens)} saved)"
        )
        print(
            "Proto-mcp (search only):     "
            f"{proto_search_tokens:,} tokens "
            f"({format_savings(proto_search_tokens, legacy_tokens)} saved)"
        )

        if not search_tools:
            print("\nNo search results to demonstrate BSR fetch.")
            return 1

        target_tool = search_tools[0]
        schema_tokens = 0
        if getattr(target_tool, "bsr_ref", ""):
            schema_payload = json.dumps(
                schema_by_ref[target_tool.bsr_ref],
                separators=(",", ":"),
                sort_keys=True,
            )
            schema_tokens = token_count(schema_payload, encoder)
            search_plus_tokens = proto_search_tokens + schema_tokens
            search_plus_savings = format_savings(
                search_plus_tokens,
                legacy_tokens,
            )
            print(
                "Proto-mcp (search + 1 schema): "
                f"{search_plus_tokens:,} tokens "
                f"({search_plus_savings} saved)"
            )

        print("\n--- On-Demand Tool Execution ---")
        print(f"Selected tool: {target_tool.name}")
        print(f"BSR ref: {target_tool.bsr_ref}")

        tool_class = dynamic_registry.resolve(target_tool.bsr_ref)
        request_msg = tool_class()
        args_any = any_pb2.Any()
        args_any.Pack(request_msg)

        response = client.call_tool(target_tool.name, args_any)
        if response.HasField("success"):
            text = response.success.content[0].text
            print(f"SUCCESS: {text}")
        else:
            print("ERROR: Tool call failed")

        return 0
    finally:
        client.close()
        if process is not None:
            process.terminate()
        if server_path and os.path.exists(server_path) and not args.no_build:
            os.remove(server_path)


def parse_args(argv: list[str]) -> argparse.Namespace:
    """Parse CLI arguments.

    Args:
        argv: Argument list.

    Returns:
        Parsed arguments.
    """
    parser = argparse.ArgumentParser(
        description="Benchmark on-demand discovery with BSR schemas."
    )
    parser.add_argument("--addr", default=DEFAULT_ADDR, help="gRPC addr")
    parser.add_argument("--query", default=DEFAULT_QUERY, help="Search query")
    parser.add_argument(
        "--limit",
        type=int,
        default=0,
        help="Optional max tools for benchmarking",
    )
    parser.add_argument(
        "--no-build",
        action="store_true",
        help="Skip building the server binary",
    )
    parser.add_argument(
        "--no-server",
        action="store_true",
        help="Skip launching the local server",
    )
    return parser.parse_args(argv)


def main() -> int:
    """Entry point for the benchmark script."""
    args = parse_args(sys.argv[1:])
    return run_benchmark(args)


if __name__ == "__main__":
    raise SystemExit(main())
