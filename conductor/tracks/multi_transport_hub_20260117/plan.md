# Plan: Multi-Transport Tool Hub

## Phase 1: gRPC Transport Implementation
- [x] Task: Implement the `MCPService` gRPC server in Go a7ad620
- [x] Task: Implement the `MCPService` gRPC client in Python 08840e9
- [x] Task: Verify gRPC interop with a simple echo call f527ece
- [ ] Task: Conductor - User Manual Verification 'Phase 1' (Protocol in workflow.md)

## Phase 2: Unified Tool Engine & High-Scale Catalog
- [x] Task: Refactor Go server to use a shared Tool Registry for both transports b2bbaf6
- [x] Task: Implement a "Mock Factory" that generates 1,000 tools with unique descriptions 436c7d7
- [ ] Task: Implement the `ListTools` search/filter logic on the server
- [ ] Task: Conductor - User Manual Verification 'Phase 2' (Protocol in workflow.md)

## Phase 3: The 1,000 Tool Challenge (Boss Demo)
- [ ] Task: Create a script that benchmarks searching 1,000 tools vs loading them via standard MCP
- [ ] Task: Perform a "Blind" call over gRPC to a tool found via search
- [ ] Task: Final Spike Report: Summary of findings, interop, and cost analysis
- [ ] Task: Conductor - User Manual Verification 'Phase 3' (Protocol in workflow.md)
