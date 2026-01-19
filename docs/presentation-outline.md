# Presentation Outline: proto-mcp
**The Global Utility Belt for AI**

---

### Slide 1: The Context Wall
*   **Headline:** Current AI Orchestration is Hits a Wall at Scale.
*   **Visual:** A graph showing Context Usage vs. Number of Tools.
*   **Key Point:** Standard MCP uses JSON-RPC. JSON schemas are heavy, repetitive, and expensive. 100 tools = 20k tokens.

### Slide 2: Introducing proto-mcp
*   **Headline:** Binary, Type-Safe, and Late-Binding.
*   **Visual:** Diagram of Client <-> Server <-> BSR.
*   **Key Point:** Decoupling tool *identity* (Name/Desc) from tool *implementation* (Binary Blueprint).

### Slide 3: The Magic of Late Binding
*   **Headline:** From "Compiled-In" to "Just-In-Time" Schemas.
*   **Visual:** Comparison of "Explicit Setup" vs. "Dynamic Resolution."
*   **Key Point:** The Agent fetches the blueprint from the BSR only when it needs to call the tool. No local code required.

### Slide 4: Recursive Discovery (The Killer Feature)
*   **Headline:** A Search Tool for the World's APIs.
*   **Visual:** Demo screenshot of `search_registry` finding tools.
*   **Key Point:** Don't tell the AI every tool you have. Give it a way to find what it needs in the global registry.

### Slide 5: The Showdown (The Proof)
*   **Headline:** 99.7% Token Reduction.
*   **Visual:** Bar chart of Legacy MCP (200k tokens) vs. proto-mcp (453 tokens).
*   **Key Point:** We've proven this with a live Go server and Python client talking to the real Buf Registry.

### Slide 6: The Future: Collaborative Spaces
*   **Headline:** The "Global Utility Belt."
*   **Visual:** A mesh of AIs and Tools across Network/gRPC.
*   **Key Point:** Any machine can serve tools. Any agent can discover and call them. Zero-config, High-efficiency.

---

### Call to Action:
**Adopt proto-mcp. Build the Global Utility Belt.**
