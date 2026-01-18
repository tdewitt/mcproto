# Plan: Core Protobuf and Stdio Transport

## Phase 1: Proto Definition & Code Generation [checkpoint: 898aa02]
- [x] Task: Create `proto/mcp.proto` based on the spec 9f65a38
- [x] Task: Generate Go source code from proto 165d3f3
- [x] Task: Generate Python source code from proto 165d3f3
- [ ] Task: Conductor - User Manual Verification 'Phase 1' (Protocol in workflow.md)

## Phase 2: Go Stdio Framing
- [x] Task: Implement message reading logic with 4-byte big-endian prefix 9f65a38
- [x] Task: Implement message writing logic with 4-byte big-endian prefix 012ad8b
- [x] Task: Verify Go framing with unit tests 012ad8b
- [ ] Task: Conductor - User Manual Verification 'Phase 2' (Protocol in workflow.md)

## Phase 3: Python Stdio Framing [checkpoint: bd94c08]
- [x] Task: Implement message reading logic with 4-byte big-endian prefix in Python bd94c08
- [x] Task: Implement message writing logic with 4-byte big-endian prefix in Python bd94c08
- [x] Task: Verify Python framing with unit tests bd94c08
- [ ] Task: Conductor - User Manual Verification 'Phase 3' (Protocol in workflow.md)

## Phase 4: Token Reduction Benchmark [checkpoint: 225cf13]
- [x] Task: Implement token comparison benchmark script cdff5ad
- [x] Task: Generate comparison report (Protobuf vs JSON-RPC) 225cf13
- [ ] Task: Conductor - User Manual Verification 'Phase 4' (Protocol in workflow.md)

## Phase 5: Interop Proof (Go Server <-> Python Client) [checkpoint: 1d2231a]
- [x] Task: Implement a minimal Go server that echoes MCP messages 1d2231a
- [x] Task: Implement a minimal Python client that sends a request to the Go server 1d2231a
- [x] Task: Demonstrate successful end-to-end binary exchange via stdio 1d2231a
- [ ] Task: Conductor - User Manual Verification 'Phase 5' (Protocol in workflow.md)
