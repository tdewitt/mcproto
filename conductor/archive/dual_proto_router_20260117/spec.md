# Spec: Dual-Protocol Router

## Overview
This track implements the "Protocol Negotiation" layer specified in the proto-mcp spec. We will build a router that sniffs the initial bytes of a stream to determine if the client is using standard JSON-RPC or the new binary Protobuf protocol.

## Goals
- Implement a "Sniffer" in Go that differentiates between `{` (JSON-RPC) and binary length-prefixes (0x00-0x1F).
- Create a routing layer that dispatches requests to the appropriate protocol handler.
- Ensure the Go server can handle a standard JSON-RPC `initialize` request.
- Maintain backward compatibility so that standard MCP tools can still use the server.

## Technical Requirements
- **Detection Logic:** Check first byte. If `0x7B` (`{`), route to JSON. If in range `0x00-0x1F`, route to Protobuf.
- **Go Server Integration:** Update `echo-server` to use the new router.

## Deliverables
- `go/pkg/router` package with sniffing and dispatching logic.
- Updated `echo-server` with dual-protocol support.
- Integration tests showing successful communication from both JSON and Protobuf clients.
