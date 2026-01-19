# Spec: Native Proto-MCP Integrations

## Overview
We are building native, binary-first MCP servers for Notion, Linear, and GitHub. These servers will bypass the legacy JSON-RPC implementations and speak directly to the APIs, exposing Protocol Buffer interfaces defined in the BSR.

## Goals
1.  **Reverse Engineer:** Extract tool definitions from existing JSON-RPC MCP servers.
2.  **Audit:** Analyze your actual workspaces to define minimal, correct Protobuf schemas.
3.  **Implement:** Build high-performance Go servers that map Proto requests to API calls.
4.  **Verify:** Prove they work with real data.

## Scope
- **Notion:** Search, Get, Create, Update, Append.
- **Linear:** Search, Get, Create, Update.
- **GitHub:** Search Repos, Get Repo, List Issues, Create Issue.

## Technical Constraints
- Pure `proto3` (no google wrappers).
- Auth via `NOTION_API_TOKEN`, `LINEAR_API_KEY`, `GITHUB_PERSONAL_ACCESS_TOKEN`.
- Direct API integration (no wrapping legacy servers).
