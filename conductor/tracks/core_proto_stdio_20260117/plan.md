# Plan: Core Protobuf and Stdio Transport

## Phase 1: Proto Definition & Code Generation
- [x] Task: Create `proto/mcp.proto` based on the spec 9f65a38
- [x] Task: Generate Go source code from proto 165d3f3
- [x] Task: Generate Python source code from proto 165d3f3
- [ ] Task: Conductor - User Manual Verification 'Phase 1' (Protocol in workflow.md)

## Phase 2: Go Stdio Framing
- [ ] Task: Implement message reading logic with 4-byte big-endian prefix
- [ ] Task: Implement message writing logic with 4-byte big-endian prefix
- [ ] Task: Verify Go framing with unit tests
- [ ] Task: Conductor - User Manual Verification 'Phase 2' (Protocol in workflow.md)

## Phase 3: Python Stdio Framing
- [ ] Task: Implement message reading logic with 4-byte big-endian prefix in Python
- [ ] Task: Implement message writing logic with 4-byte big-endian prefix in Python
- [ ] Task: Verify Python framing with unit tests
- [ ] Task: Conductor - User Manual Verification 'Phase 3' (Protocol in workflow.md)
