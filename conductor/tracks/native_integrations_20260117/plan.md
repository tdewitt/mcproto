# Plan: Native Integrations

## Phase 1: Discovery & Audit
- [x] Task: Create a "Inspector" script to query existing MCP servers for tool schemas 1727cc0
- [x] Task: Create a "Data Auditor" script to catalog actual types in your Notion, Linear, and GitHub workspaces 115bf20
- [x] Task: Generate Audit Reports for GitHub service 1a58623
- [ ] Task: Conductor - User Manual Verification 'Phase 1' (Protocol in workflow.md)

## Phase 2: Schema Definition
- [x] Task: Inspect official GitHub MCP server to capture tool definitions d85091b
- [x] Task: Define `proto/github.proto` based on audit results and tool definitions 1a58623
- [ ] Task: Conductor - User Manual Verification 'Phase 2' (Protocol in workflow.md)

## Phase 3: Server Implementation
- [x] Task: Implement Native GitHub Server (Go) c00cab1
- [ ] Task: Conductor - User Manual Verification 'Phase 3' (Protocol in workflow.md)

## Phase 4: Verification
- [x] Task: Create validation scripts to run RPCs against real data 591b9dc
- [x] Task: Align GitHub tool naming with official MCP (drop GetRepository, add snake_case aliases) 4ea45f2
- [x] Task: Create on-demand discovery benchmark script (BSR + token savings) 5663e87
- [ ] Task: Generate final success report
- [ ] Task: Conductor - User Manual Verification 'Phase 4' (Protocol in workflow.md)
