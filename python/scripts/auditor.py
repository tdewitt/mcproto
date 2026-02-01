import os
import json
import requests
from notion_client import Client

def audit_notion():
    token = os.environ.get("NOTION_API_TOKEN")
    if not token:
        print("Skipping Notion audit (NOTION_API_TOKEN not set)")
        return {}
    
    notion = Client(auth=token)
    results = notion.search().get("results")
    
    types = {
        "property_types": set(),
        "block_types": set(),
        "parent_types": set()
    }
    
    for obj in results:
        types["parent_types"].add(obj.get("parent", {}).get("type"))
        if obj["object"] == "page":
            for prop in obj.get("properties", {}).values():
                types["property_types"].add(prop.get("type"))
        
        # Audit blocks for a few pages
        if obj["object"] == "page":
            try:
                blocks = notion.blocks.children.list(block_id=obj["id"]).get("results")
                for block in blocks:
                    types["block_types"].add(block.get("type"))
            except Exception:
                pass
                
    return {k: list(v) for k, v in types.items()}

def audit_linear():
    api_key = os.environ.get("LINEAR_API_KEY")
    if not api_key:
        print("Skipping Linear audit (LINEAR_API_KEY not set)")
        return {}
    
    url = "https://api.linear.app/graphql"
    headers = {"Authorization": api_key, "Content-Type": "application/json"}
    query = "{ issues { nodes { id title description state { name type } priority estimate assignee { id name } } } }"
    
    resp = requests.post(url, headers=headers, json={"query": query})
    if resp.status_code != 200:
        return {"error": resp.text}
    
    data = resp.json().get("data", {})
    return {"issue_fields": list(data.get("issues", {}).get("nodes", [{}])[0].keys()) if data.get("issues", {}).get("nodes") else []}

def audit_github():
    token = os.environ.get("GITHUB_PERSONAL_ACCESS_TOKEN")
    if not token:
        print("Skipping GitHub audit (GITHUB_PERSONAL_ACCESS_TOKEN not set)")
        return {}
    
    headers = {"Authorization": f"token {token}", "Accept": "application/vnd.github.v3+json"}
    
    # Audit repos
    repos_resp = requests.get("https://api.github.com/user/repos?per_page=1", headers=headers)
    repo_fields = list(repos_resp.json()[0].keys()) if repos_resp.status_code == 200 and repos_resp.json() else []
    
    # Audit issues
    issues_resp = requests.get("https://api.github.com/issues?per_page=1", headers=headers)
    issue_fields = list(issues_resp.json()[0].keys()) if issues_resp.status_code == 200 and issues_resp.json() else []
    
    return {"repo_fields": repo_fields, "issue_fields": issue_fields}

def main():
    report = {
        "notion": audit_notion(),
        "linear": audit_linear(),
        "github": audit_github()
    }
    
    os.makedirs("docs/audit", exist_ok=True)
    with open("docs/audit/report.json", "w") as f:
        json.dump(report, f, indent=2)
    
    print("Audit report generated at docs/audit/report.json")

if __name__ == "__main__":
    main()
