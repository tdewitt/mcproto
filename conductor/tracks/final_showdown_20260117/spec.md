# Spec: Fluid Analytics & Token Showdown

## Overview
This final track implements a realistic ETL pipeline demo to compare token efficiency across three modes: Explicit MCP (Legacy), Anthropic Tool Search, and proto-mcp. We use a fluid data model (variable inputs) to show how Protobuf and BSR handle complexity without bloating the LLM context.

## Goals
- Implement 12 ETL-themed tools in the Go server.
- Implement a "Protocol Simulator" that can calculate tokens for all three discovery modes.
- Run a reproducible ETL task (Extract -> Transform -> Load) using deterministic data.
- Produce a final "Economic and Technical Value Report".

## Technical Requirements
- **Protobuf Spec:** `misfit.analytics.v1` (defined in `proto/analytics.proto`).
- **Data Model:** Use `google.protobuf.Struct` for variable data handling.
- **Tokenizer:** `cl100k_base` for precise GPT-4/Claude token counting.

## Deliverables
- Multi-mode Go server supporting Standard, Search, and Proto-MCP.
- Python benchmark script (`scripts/final_showdown.py`).
- Final value report in `docs/spike_summary.md`.
