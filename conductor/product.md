# Initial Concept
C but this is a spike.  Not a full product.  Want to see if it makes any sense first

# Product Guide: proto-mcp Spike

## Purpose
This project is an experimental spike designed to validate the core architectural assumptions of the proto-mcp protocol. Before committing to a full product suite, we aim to verify the performance, efficiency, and developer experience (DX) claims made in the specification.

## Core Objectives
- **Validate Token Efficiency:** Empirically verify the claimed 99% reduction in AI context consumption compared to standard JSON-RPC MCP.
- **Benchmark Performance:** Measure the latency overhead of binary serialization and schema resolution to ensure it meets real-time AI orchestration requirements.
- **BSR Integration:** Test the end-to-end flow of resolving schemas via the actual Buf Schema Registry (BSR).
- **Dual-Protocol Viability:** Demonstrate a router capable of handling both JSON-RPC and Protobuf payloads seamlessly.

## Components
- **Go Server Implementation:** A reference server capable of hosting tools and serving requests over both stdio and network transports.
- **Python Client Implementation:** A reference client capable of tool discovery, schema resolution via BSR, and type-safe tool execution.

## Key Scenarios & Tools
- **Baseline Latency:** An "echo" tool for minimal-overhead benchmarking.
- **Realistic Payloads:** A "web search" mock tool to test typical text-heavy orchestration.
- **Binary Handling:** A tool for high-volume binary data (e.g., image processing) to validate protobuf's handling of non-text assets.
- **Protocol Routing:** A demonstration of the dual-protocol handler identifying and routing requests based on the initial wire bytes.

## Success Criteria
- Successful execution of a tool call using Protobuf encoding with schemas resolved from BSR.
- Comparative benchmarks showing significant token savings over JSON-RPC.
- A functional dual-protocol router that correctly identifies payload types.
