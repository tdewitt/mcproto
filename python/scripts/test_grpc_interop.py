import sys
import os
import time

# Add python directory to path
sys.path.append(os.path.join(os.getcwd(), 'python'))

from mcp.grpc_client import GRPCClient

def test_interop():
    print("Testing gRPC Interop...")
    client = GRPCClient()
    try:
        resp = client.initialize()
        print(f"SUCCESS: Server responded with protocol version: {resp.protocol_version}")
        print(f"Server Metadata: {resp.metadata}")
    except Exception as e:
        print(f"FAILURE: {e}")
    finally:
        client.close()

if __name__ == "__main__":
    test_interop()
