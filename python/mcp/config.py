"""Configuration constants for proto-mcp client and server."""

# gRPC configuration
DEFAULT_GRPC_TARGET = "localhost:50051"
DEFAULT_GRPC_PORT = 50051

# Timeout configuration (in seconds)
DEFAULT_REQUEST_TIMEOUT = 30
DEFAULT_BSR_TIMEOUT = 30

# Message size limits
MAX_MESSAGE_SIZE = 32 * 1024 * 1024  # 32MB

# Protocol configuration
SUPPORTED_PROTOCOL_VERSION = "1.0.0"

# BSR configuration
DEFAULT_BSR_BASE_URL = "https://api.buf.build"

# Registry cache configuration
MAX_REGISTRY_CACHE_SIZE = 100  # Maximum schemas to keep in memory
