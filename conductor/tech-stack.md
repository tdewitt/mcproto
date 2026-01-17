# Tech Stack

## Core Technologies
- **Languages:** Go (Server), Python (Client)
- **Encoding:** Protocol Buffers (protobuf)
- **Schema Management:** Buf Schema Registry (BSR)

## Transport Layer
- **Primary:** Stdio (Length-delimited binary framing)
- **Optional:** HTTP/2 (gRPC), WebSocket

## Benchmarking & Analysis
- **Tokenization:** cl100k_base (standard for GPT-4/Claude)
- **Performance Measurement:** Go `testing` package, Python `pytest-benchmark` or `timeit`
- **Analysis:** Custom scripts to compare Protobuf vs JSON-RPC message sizes and latencies
