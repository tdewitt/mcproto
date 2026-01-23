# Dual-Protocol Interoperability Proof

## Overview
The `proto-mcp` Go server supports the legacy JSON-RPC MCP standard and the high-efficiency binary Protobuf protocol on a single transport stream (stdio).

## Methodology
The implementation uses a `ProtocolRouter` that inspects the initial bytes of an incoming stream to determine the protocol:
- `0x7B` ('{') or leading whitespace → Dispatched to **JSONHandler**
- `0x00-0x1F` (Binary length prefix) → Dispatched to **BinaryHandler**

This detection is non-destructive; the router buffers the peeked bytes to ensure the downstream handlers receive the complete message.

## Functional Verification
Verification is performed using the `python/examples/visual_wiretap.py` demonstration script, which executes both JSON-RPC and Protobuf handshakes against the same server instance.

### 1. Legacy Compatibility (JSON-RPC)
The server identifies the JSON payload and responds with a standard JSON-RPC `InitializeResponse`.
- **Demo:** `python3 python/examples/visual_wiretap.py`
- **Result:** Successfully routes JSON-RPC `initialize` method to the JSON handler.

### 2. High-Efficiency Binary (Protobuf)
The server identifies the binary length prefix and responds with a Protobuf-encoded `InitializeResponse`.
- **Demo:** `python3 python/examples/visual_wiretap.py`
- **Result:** Successfully routes binary message to the Protobuf handler.

## Conclusion
The `proto-mcp` implementation is backward compatible, acting as a drop-in replacement for standard MCP servers. This allows incremental adoption of binary efficiency without disrupting existing JSON-RPC infrastructure.
