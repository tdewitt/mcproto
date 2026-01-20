# MC Proto
> **High-efficiency AI tool orchestration.** 99% Token reduction. Zero-config interop. Late-binding schema resolution.

![MC Proto Hero](./docs/images/mc-proto-hero.png)

## Motivation
The standard Model Context Protocol (MCP) relies on JSON-RPC, which introduces significant overhead as agent environments scale. In enterprise settings with hundreds of tools, technical JSON schemas consume the majority of the AI's context window, increasing inference costs and reducing the available space for reasoning and task data.

**MC Proto** (aka `proto-mcp`) is a Protocol Buffer-based implementation of MCP. It replaces JSON-RPC with binary serialization and leverages the Buf Schema Registry (BSR) to decouple tool definitions from the transport layer.

## Key Features
*   **99.9% Token Reduction:** Implements "Thin Discovery" where the LLM only receives tool names and descriptions. Technical schemas are resolved out-of-band by the client library using BSR references.
*   **Late-Binding Schema Resolution:** Tool blueprints are fetched from the BSR at runtime. This allows agents to execute tools without requiring the corresponding code to be compiled into the client or server at build-time.
*   **Multi-Transport Engine:** A unified implementation that supports both local Stdio pipes (length-delimited binary framing) and remote gRPC sockets.
*   **Recursive Discovery:** Includes a discovery meta-tool that performs live global searches of the Buf Schema Registry to find and load new tool blueprints on-demand.
*   **Dual-Protocol Sniffer:** A non-destructive sniffer that detects the protocol format (JSON vs. Protobuf) based on the initial bytes of the stream, allowing for backward compatibility with standard MCP clients.

## Performance Metrics
In a benchmark of 1,000 tools:
*   **Standard MCP:** ~200,000 tokens (Exceeds most context windows).
*   **MC Proto (Search-based):** **453 tokens** (99.77% reduction).
*   **Serialization Latency:** Protobuf serialization performed **~5x faster** than JSON encoding in Go and Python environments.

## Setup and Execution

### 1. Environment Configuration
```bash
# Install dependencies via the Proto toolchain
proto use
source .venv/bin/activate
pip install -r requirements.txt
```

### 2. Run the Reference Demo
The following script demonstrates recursive discovery and a late-binding gRPC call against the consolidated Go server:
```bash
export BUF_TOKEN=<your_token>
export PYTHONPATH=$PYTHONPATH:$(pwd)/python
python3 python/examples/boss_demo.py
```

## Architecture
MC Proto utilizes the **Buf Schema Registry** as a runtime blueprint provider:
1. **Server** (`go/cmd/mcproto`) advertises a `bsr_ref` pointer.
2. **Client** library (`python/mcp`) fetches the `FileDescriptorSet` from the BSR.
3. **LLM** operates on a semantic summary of the tool.
4. **Binary Execution** is performed using dynamic message reflection (`dynamicpb` in Go, `DescriptorPool` in Python).

---
*Developed as a technical spike to verify the viability of binary-first AI orchestration.*
