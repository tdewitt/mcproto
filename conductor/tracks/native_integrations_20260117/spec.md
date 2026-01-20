# Spec: Native Proto-MCP Integrations

## Overview
We are building a native, binary-first MCP server for GitHub. This server will bypass the legacy JSON-RPC implementation and speak directly to the GitHub API, exposing Protocol Buffer interfaces defined in the BSR.

## Goals
1.  **Reverse Engineer:** Extract tool definitions from the existing GitHub MCP server.
2.  **Audit:** Analyze your actual GitHub workspace to define minimal, correct Protobuf schemas.
3.  **Implement:** Build a high-performance Go server that maps Proto requests to GitHub API calls.
4.  **Verify:** Prove it works with real data.

## Scope
- **GitHub:** Search Repos, Get Repo, List Issues, Create Issue.

## Technical Constraints
- Pure `proto3` (no google wrappers).
- Auth via `GITHUB_PERSONAL_ACCESS_TOKEN`.
- Direct API integration (no wrapping legacy servers).
