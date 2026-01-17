# Product Guidelines

## 1. Technical Rigor
- **Precise Benchmarking:** All performance claims must be backed by reproducible benchmarks. Use high-resolution timers and account for warm-up periods. Compare against a standard JSON-RPC baseline under identical conditions.
- **Detailed Logging:** Implement comprehensive logging for all protocol exchanges. Log binary sizes, serialization/deserialization times, and schema resolution latencies. Use structured logging to facilitate analysis.
- **Token Consumption Measurement:** Accurately measure and report token consumption for both proto-mcp and JSON-RPC. Use the same tokenizer (e.g., cl100k_base) for fair comparison.

## 2. Minimalist Utility
- **Focus on Core Value:** Prioritize features that directly validate the core hypotheses (token efficiency, latency). Avoid over-engineering edge cases that don't contribute to the spike's goals.
- **Just-Enough Documentation:** Documentation should be concise and focused on how to run the benchmarks and interpret the results. Avoid extensive theoretical explanations unless necessary for understanding the implementation.

## 3. Educational & Demonstrative Value
- **Clear Codebase:** Write clean, idiomatic code (Go for server, Python for client) that serves as a reference implementation. Use comments to explain protocol-specific logic.
- **Exploratory Documentation:** Include a "Lessons Learned" or "Architecture Decisions" section to document the reasoning behind key design choices. This will help future developers understand the trade-offs involved.
- **How-it-Works Guide:** Create a high-level guide explaining the proto-mcp protocol flow, including schema resolution and message framing. This will help onboard new contributors and users.
