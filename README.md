# MC Proto
> **High-efficiency AI tool orchestration.** 99% Token reduction. Zero-config interop. Late-binding schema resolution.

![MC Proto Hero](./docs/images/mc-proto-hero.png)

## Motivation
Standard Model Context Protocol (MCP) implementations rely on JSON-RPC, which can introduce overhead as agent environments scale. In enterprise settings with hundreds of tools, technical JSON schemas consume a significant portion of the AI's context window, increasing inference costs and reducing the available space for reasoning and task data.

**MC Proto** (aka `proto-mcp`) is a Protocol Buffer-based implementation of MCP. It replaces JSON-RPC with binary serialization and leverages the Buf Schema Registry (BSR) to decouple tool definitions from the transport layer.

## Key Features
*   **Token Efficiency:** The LLM receives only tool names and descriptions during the discovery phase. Technical schemas are resolved out-of-band by the client library using BSR references only when a tool is selected.
*   **Late-Binding Schema Resolution:** Tool blueprints are fetched from the BSR at runtime. This allows agents to execute tools without requiring the corresponding code to be compiled into the client or server at build-time.
*   **Multi-Transport Engine:** A unified implementation supports both local Stdio pipes (length-delimited binary framing) and remote gRPC sockets.
*   **Recursive Discovery:** Includes a discovery meta-tool that performs live global searches of the Buf Schema Registry to find and load new tool blueprints on-demand.
*   **Dual-Protocol Compatibility:** A non-destructive sniffer detects the protocol format (JSON vs. Protobuf) based on the initial bytes of the stream, allowing backward compatibility with standard MCP clients.

## Performance Simulation
A simulation using a catalog of 1,000 generated tools demonstrates the potential efficiency gains:

*   **Standard MCP (Projected):** ~200,000 tokens (Based on avg. 200 tokens/schema).
*   **MC Proto (Measured):** **453 tokens** (Tool names and descriptions only).

*Note: Actual savings depend on schema complexity and description length.*

## Setup and Execution

### 1. Environment Configuration
Create a `.env` file in the project root with the following credentials:
```bash
BUF_TOKEN=<your_buf_registry_token>
GITHUB_PERSONAL_ACCESS_TOKEN=<your_github_pat>
```

Install dependencies:
```bash
proto use
source .venv/bin/activate
pip install -r requirements.txt
```

### 2. Run the Reference Demo
The following script initiates a Claude agent that recursively discovers a GitHub tool from the registry, resolves its schema at runtime, and executes a task:

```bash
./scripts/run_claude_discovery_demo.sh
```

## Architecture
MC Proto utilizes the **Buf Schema Registry** as a runtime blueprint provider:
1. **Server** (`go/cmd/mcproto`) advertises a `bsr_ref` pointer.
2. **Client** library (`python/mcp`) fetches the `FileDescriptorSet` from the BSR.
3. **LLM** operates on a semantic summary of the tool.
4. **Binary Execution** is performed using dynamic message reflection (`dynamicpb` in Go, `DescriptorPool` in Python).

---
*Developed as a technical spike to verify the viability of binary-first AI orchestration.*