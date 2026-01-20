import grpc
import sys
import os
import subprocess
import time

# Add python directory to path
sys.path.append(os.path.join(os.getcwd(), 'python'))

from mcp import github_pb2, github_pb2_grpc

def validate_github():
    print("\n" + "="*65)
    print(" GITHUB NATIVE SERVER VALIDATION")
    print("="*65 + "\n")

    # 1. Build and Start Server
    print("Building Go GitHub server...")
    subprocess.run(["go", "build", "-o", "github-server", "./cmd/github-server/main.go"], cwd="go", check=True)
    server_path = os.path.join(os.getcwd(), "go", "github-server")
    
    server_process = subprocess.Popen([server_path], stdout=subprocess.DEVNULL, stderr=sys.stderr)
    time.sleep(2)

    target = "localhost:50051"
    channel = grpc.insecure_channel(target)
    stub = github_pb2_grpc.GitHubServiceStub(channel)

    try:
        # 1. Search Repositories
        print("Testing SearchRepositories (query: 'proto-mcp')...")
        req = github_pb2.SearchRepositoriesRequest(query="proto-mcp")
        resp = stub.SearchRepositories(req)
        print(f"SUCCESS: Found {resp.total_count} repositories.")
        for repo in resp.repositories[:3]:
            print(f" - {repo.full_name} ({repo.html_url})")

        # 2. Get Repository
        if resp.repositories:
            target_repo = resp.repositories[0]
            print(f"\nTesting GetRepository ({target_repo.full_name})...")
            get_req = github_pb2.GetRepositoryRequest(owner=target_repo.owner.login, repo=target_repo.name)
            get_resp = stub.GetRepository(get_req)
            print(f"SUCCESS: Retrieved {get_resp.repository.full_name}")
            print(f"Description: {get_resp.repository.description}")

        # 3. List Issues
        if resp.repositories:
            target_repo = resp.repositories[0]
            print(f"\nTesting ListIssues ({target_repo.full_name})...")
            list_req = github_pb2.ListIssuesRequest(owner=target_repo.owner.login, repo=target_repo.name)
            list_resp = stub.ListIssues(list_req)
            print(f"SUCCESS: Found {len(list_resp.issues)} issues.")
            for issue in list_resp.issues[:3]:
                print(f" - #{issue.number}: {issue.title}")

    except Exception as e:
        print(f"ERROR: {e}")
    finally:
        channel.close()
        server_process.terminate()
        if os.path.exists(server_path):
            os.remove(server_path)

if __name__ == "__main__":
    validate_github()
