# Spec: Core Protobuf and Stdio Transport

## Overview
This track establishes the foundational binary communication layer for the proto-mcp spike. We will define the core messages using Protocol Buffers and implement the length-delimited stdio framing required for binary transport.

## Goals
- Define the `MCPMessage` and all its sub-messages in a `.proto` file.
- Generate type-safe code for both Go (server) and Python (client).
- Implement the 4-byte big-endian length-prefix framing for stdio communication.
- Ensure interoperability by testing serialization/deserialization across both languages.

## Technical Requirements
- **Protobuf Version:** 3
- **Framing:** `[4-byte length (big-endian)][protobuf message bytes]`
- **Go Package:** `mcp`
- **Python Package:** `buf.mcp`

## Deliverables
- `proto/mcp.proto` file containing the full protocol definition.
- Generated Go and Python source files.
- Framing library/utility in Go and Python.
- Unit tests for framing and message serialization.
