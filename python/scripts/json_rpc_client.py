import subprocess
import os
import sys
import json

def run_json_rpc_test():
    # 1. Compile the Go server
    print("Building Go dual-protocol echo-server...")
    subprocess.run(["go", "build", "-o", "echo-server", "./cmd/echo-server/main.go"], cwd="go", check=True)
    
    # 2. Launch the server
    server_path = os.path.join(os.getcwd(), "go", "echo-server")
    process = subprocess.Popen(
        [server_path],
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=sys.stderr,
        text=True,
        bufsize=1
    )
    
    try:
        # 3. Send a JSON-RPC InitializeRequest
        print("Sending JSON-RPC InitializeRequest to Go server...")
        req = {
            "jsonrpc": "2.0",
            "id": 1,
            "method": "initialize",
            "params": {
                "protocolVersion": "2024-11-05"
            }
        }
        process.stdin.write(json.dumps(req) + "\n")
        process.stdin.flush()
        
        # 4. Read the JSON-RPC response
        print("Waiting for JSON-RPC response...")
        line = process.stdout.readline()
        if not line:
            print("FAILURE: No response from server")
            return

        resp = json.loads(line)
        if resp.get("id") == 1 and "result" in resp:
            print(f"SUCCESS: Received JSON-RPC response")
            print(f"Server Name: {resp['result']['serverInfo']['name']}")
            print(f"Protocol Version: {resp['result']['protocolVersion']}")
        else:
            print(f"FAILURE: Received incorrect response: {resp}")
            
    finally:
        process.terminate()
        if os.path.exists(server_path):
            os.remove(server_path)

if __name__ == "__main__":
    run_json_rpc_test()
