import subprocess
import os
import sys
import json
import base64

# Add python directory to path
sys.path.append(os.path.join(os.getcwd(), 'python'))

from mcp.stdio import StdioReader, StdioWriter
from mcp import mcp_pb2

def print_box(title, content, color="\033[94m"):
    reset = "\033[0m"
    print(f"{color}┌── {title} " + "─"*(60-len(title)) + reset)
    for line in content.split("\n"):
        print(f"{color}│{reset} {line}")
    print(f"{color}└" + "─"*62 + reset)

def run_visual_wiretap():
    print("\n" + "="*65)
    print(" DUAL-PROTOCOL WIRE-TAP DEMO")
    print("="*65 + "\n")

    # 1. Compile
    subprocess.run(["go", "build", "-o", "echo-server", "./cmd/echo-server/main.go"], cwd="go", check=True)
    server_path = os.path.join(os.getcwd(), "go", "echo-server")

    def test_protocol(name, payload_gen_func, is_binary=False):
        print(f"Testing Protocol: {name}")
        process = subprocess.Popen(
            [server_path],
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=False,
            bufsize=0
        )
        
        payload = payload_gen_func()
        
        # Show what we are sending
        if is_binary:
            hex_view = " ".join(f"{b:02x}" for b in payload[:16]) + "..."
            print_box("WIRE: Client -> Server (BINARY)", f"Raw Bytes (Hex): {hex_view}\nLength Prefix: {payload[0]:02x}")
        else:
            print_box("WIRE: Client -> Server (JSON)", payload.decode('utf-8'))

        process.stdin.write(payload)
        process.stdin.flush()

        # Capture server's "thought" from stderr
        server_thought = process.stderr.readline().decode('utf-8').strip()
        print(f"\033[93m{server_thought}\033[0m")

        # Capture response
        response = process.stdout.readline()
        if is_binary:
            print_box("WIRE: Server -> Client (BINARY)", f"Received {len(response)} bytes of Protobuf")
        else:
            print_box("WIRE: Server -> Client (JSON)", response.decode('utf-8'))

        process.terminate()
        print("\n")

    # JSON-RPC Test
    def gen_json():
        return json.dumps({"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {}}).encode('utf-8') + b"\n"
    
    test_protocol("Legacy JSON-RPC", gen_json, is_binary=False)

    # Protobuf Test
    def gen_proto():
        msg = mcp_pb2.MCPMessage(id=99)
        msg.initialize_request.protocol_version = "1.0.0"
        data = msg.SerializeToString()
        import struct
        return struct.pack(">I", len(data)) + data

    test_protocol("High-Efficiency Protobuf", gen_proto, is_binary=True)

    if os.path.exists(server_path):
        os.remove(server_path)

if __name__ == "__main__":
    run_visual_wiretap()
