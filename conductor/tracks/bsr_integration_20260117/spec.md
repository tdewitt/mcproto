# Spec: BSR Integration & Schema Resolution

## Overview
This track implements the core capability of proto-mcp: dynamic schema resolution. Instead of hardcoding tool definitions, the client and server will resolve Protobuf descriptors from the Buf Schema Registry (BSR) using `bsr_ref` strings.

## Goals
- Implement a BSR client in both Go and Python to fetch `FileDescriptorSet` from the registry.
- Implement a schema cache (in-memory) to minimize registry lookups.
- Implement dynamic message unpacking using `google.protobuf.Any` and the resolved descriptors.
- Enable the Python client to call a tool by only knowing its BSR reference.

## Technical Requirements
- **Authentication:** Use the `BUF_TOKEN` environment variable for BSR API calls.
- **Resolution Flow:** `bsr_ref` -> BSR API -> `FileDescriptorSet` -> Dynamic Message Registry.
- **Caching:** TTL-based in-memory cache for descriptors.

## Deliverables
- `go/pkg/bsr` and `python/mcp/bsr.py` client libraries.
- Updated `CallTool` implementation that handles dynamic unpacking.
- A demo showing a tool call where the schema was never present in the client's source code.
