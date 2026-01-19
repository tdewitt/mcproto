# Plan: Native Integrations

## Phase 1: Discovery & Audit
- [x] Task: Create a "Inspector" script to query existing MCP servers for tool schemas 1727cc0
- [ ] Task: Create a "Data Auditor" script to catalog actual types in your Notion, Linear, and GitHub workspaces
- [ ] Task: Generate Audit Reports for all three services
- [ ] Task: Conductor - User Manual Verification 'Phase 1' (Protocol in workflow.md)

## Phase 2: Schema Definition
- [ ] Task: Define `proto/notion.proto` based on audit results
- [ ] Task: Define `proto/linear.proto` based on audit results
- [ ] Task: Define `proto/github.proto` based on audit results
- [ ] Task: Conductor - User Manual Verification 'Phase 2' (Protocol in workflow.md)

## Phase 3: Server Implementation
- [ ] Task: Implement Native Notion Server (Go)
- [ ] Task: Implement Native Linear Server (Go)
- [ ] Task: Implement Native GitHub Server (Go)
- [ ] Task: Conductor - User Manual Verification 'Phase 3' (Protocol in workflow.md)

## Phase 4: Verification
- [ ] Task: Create validation scripts to run RPCs against real data
- [ ] Task: Generate final success report
- [ ] Task: Conductor - User Manual Verification 'Phase 4' (Protocol in workflow.md)
