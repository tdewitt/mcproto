import subprocess
import os
import sys
import io

# Add python directory to path
sys.path.append(os.path.join(os.getcwd(), 'python'))

from mcp.stdio import StdioReader, StdioWriter
from mcp import mcp_pb2

def run_demo():
    # 1. Compile the Go server
    print("Building Go echo-server...")
    subprocess.run(["go", "build", "-o", "echo-server", "./cmd/echo-server/main.go"], cwd="go", check=True)
    
    # 2. Launch the server
    server_path = os.path.join(os.getcwd(), "go", "echo-server")
    process = subprocess.Popen(
        [server_path],
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=sys.stderr,
        bufsize=0
    )
    
    writer = StdioWriter(process.stdin)
    reader = StdioReader(process.stdout)
    
    try:
        # 3. Send a message
        print("Sending InitializeRequest to Go server...")
        msg = mcp_pb2.MCPMessage(id=42)
        msg.initialize_request.protocol_version = "1.0.0"
        writer.write_message(msg)
        
        # 4. Read the echo
        print("Waiting for echo response...")
        response = reader.read_message()
        
        if response and response.id == 42:
            print(f"SUCCESS: Received echo with ID {response.id}")
            print(f"Payload: {response.initialize_request.protocol_version}")
        else:
            print("FAILURE: Received incorrect or no response")
            
    finally:
        process.terminate()
        if os.path.exists(server_path):
            os.remove(server_path)

if __name__ == "__main__":
    run_demo()
