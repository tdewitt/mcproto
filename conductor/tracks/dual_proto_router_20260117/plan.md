# Plan: Dual-Protocol Router

## Phase 1: Sniffing Logic [checkpoint: b78e352]
- [x] Task: Implement the `Sniffer` in Go that reads the first byte without consuming the stream incorrectly 142ebc5
- [x] Task: Unit tests for protocol detection (JSON vs Binary) 142ebc5
- [ ] Task: Conductor - User Manual Verification 'Phase 1' (Protocol in workflow.md)

## Phase 2: Router Implementation
- [x] Task: Implement the `ProtocolRouter` that wraps an `io.ReadWriter` and dispatches to handlers 106fe3e
- [x] Task: Implement a minimal JSON-RPC handler for `initialize` to prove compatibility 70c6790
- [x] Task: Update the Go server to use the `ProtocolRouter` 68cb291
- [ ] Task: Conductor - User Manual Verification 'Phase 2' (Protocol in workflow.md)

## Phase 3: Integration & Compatibility Proof
- [ ] Task: Create a simple JSON-RPC test script to verify backward compatibility
- [ ] Task: Run the Python Protobuf client from the previous track to verify no regressions
- [ ] Task: Document the results in `docs/dual_protocol_proof.md`
- [ ] Task: Conductor - User Manual Verification 'Phase 3' (Protocol in workflow.md)
