# Jira MCP Server for mcproto

## Overview

A Jira integration for mcproto that exposes Jira operations as MCP tools with protobuf schemas. This enables token-efficient access to Jira from AI agents while maintaining full type safety.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     mcproto Server                          │
│  ┌─────────────────────────────────────────────────────┐   │
│  │              UnifiedRegistry                         │   │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐            │   │
│  │  │  GitHub  │ │   ETL    │ │   Jira   │  ← NEW     │   │
│  │  │  Tools   │ │  Tools   │ │  Tools   │            │   │
│  │  └──────────┘ └──────────┘ └──────────┘            │   │
│  └─────────────────────────────────────────────────────┘   │
│                           │                                 │
│  ┌────────────────────────┴────────────────────────────┐   │
│  │                    Transports                        │   │
│  │         gRPC (:50051)    │    stdio (binary)        │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                  Jira Cloud REST API                        │
│         https://{instance}.atlassian.net/rest/api/3         │
└─────────────────────────────────────────────────────────────┘
```

## Jira Tools to Implement

### Phase 1: Core Operations
| Tool | Description | Jira API |
|------|-------------|----------|
| `SearchIssues` | Search issues with JQL | `GET /search` |
| `GetIssue` | Get issue details | `GET /issue/{issueKey}` |
| `CreateIssue` | Create new issue | `POST /issue` |
| `UpdateIssue` | Update issue fields | `PUT /issue/{issueKey}` |
| `TransitionIssue` | Change issue status | `POST /issue/{issueKey}/transitions` |
| `AddComment` | Add comment to issue | `POST /issue/{issueKey}/comment` |
| `AssignIssue` | Assign issue to user | `PUT /issue/{issueKey}/assignee` |

### Phase 2: Extended Operations
| Tool | Description | Jira API |
|------|-------------|----------|
| `GetTransitions` | Get available transitions | `GET /issue/{issueKey}/transitions` |
| `GetComments` | Get issue comments | `GET /issue/{issueKey}/comment` |
| `LinkIssues` | Link two issues | `POST /issueLink` |
| `GetProject` | Get project details | `GET /project/{projectKey}` |
| `ListProjects` | List accessible projects | `GET /project` |
| `GetUser` | Get user by account ID | `GET /user` |
| `SearchUsers` | Search for users | `GET /user/search` |

### Phase 3: Workflow Automation
| Tool | Description | Use Case |
|------|-------------|----------|
| `BulkTransition` | Transition multiple issues | Sprint completion |
| `CloneIssue` | Clone issue with customization | Template-based creation |
| `GetSprintIssues` | Get issues in sprint | Sprint reporting |

## Proto Definitions

### File: `proto/jira/jira.proto`

```protobuf
syntax = "proto3";

package tucker.mcproto.jira.v1;

option go_package = "github.com/tdewitt/mcproto/go/gen/jira";

import "google/protobuf/struct.proto";

// === Search ===

message SearchIssuesRequest {
  string jql = 1;                    // JQL query string
  repeated string fields = 2;         // Fields to return (empty = all)
  int32 start_at = 3;                // Pagination offset
  int32 max_results = 4;             // Max results (default 50)
  repeated string expand = 5;         // Expand options (changelog, renderedFields, etc.)
}

message SearchIssuesResponse {
  repeated Issue issues = 1;
  int32 start_at = 2;
  int32 max_results = 3;
  int32 total = 4;
}

// === Get Issue ===

message GetIssueRequest {
  string issue_key = 1;              // e.g., "PROJ-123"
  repeated string fields = 2;
  repeated string expand = 3;
}

message GetIssueResponse {
  Issue issue = 1;
}

// === Create Issue ===

message CreateIssueRequest {
  string project_key = 1;            // e.g., "PROJ"
  string issue_type = 2;             // e.g., "Bug", "Story", "Task"
  string summary = 3;
  string description = 4;            // Markdown or ADF JSON
  string assignee_account_id = 5;    // Optional
  string priority = 6;               // Optional: "Highest", "High", "Medium", "Low", "Lowest"
  repeated string labels = 7;
  map<string, string> custom_fields = 8;  // customfield_XXXXX -> value
}

message CreateIssueResponse {
  string issue_key = 1;
  string issue_id = 2;
  string self = 3;                   // API URL
}

// === Update Issue ===

message UpdateIssueRequest {
  string issue_key = 1;
  string summary = 2;                // Optional
  string description = 3;            // Optional
  string assignee_account_id = 4;    // Optional
  string priority = 5;               // Optional
  repeated string labels = 6;
  map<string, string> custom_fields = 7;
  bool notify_users = 8;             // Default true
}

message UpdateIssueResponse {
  bool success = 1;
}

// === Transition Issue ===

message TransitionIssueRequest {
  string issue_key = 1;
  string transition_id = 2;          // Numeric ID from GetTransitions
  string comment = 3;                // Optional comment with transition
  map<string, string> fields = 4;    // Fields required by transition
}

message TransitionIssueResponse {
  bool success = 1;
}

message GetTransitionsRequest {
  string issue_key = 1;
}

message GetTransitionsResponse {
  repeated Transition transitions = 1;
}

message Transition {
  string id = 1;
  string name = 2;
  Status to = 3;
}

// === Comments ===

message AddCommentRequest {
  string issue_key = 1;
  string body = 2;                   // Markdown or ADF JSON
  Visibility visibility = 3;         // Optional restriction
}

message AddCommentResponse {
  string comment_id = 1;
  string self = 2;
}

message GetCommentsRequest {
  string issue_key = 1;
  int32 start_at = 2;
  int32 max_results = 3;
  string order_by = 4;               // "created" or "-created"
}

message GetCommentsResponse {
  repeated Comment comments = 1;
  int32 total = 2;
}

// === Assignment ===

message AssignIssueRequest {
  string issue_key = 1;
  string account_id = 2;             // null/empty to unassign
}

message AssignIssueResponse {
  bool success = 1;
}

// === Users ===

message SearchUsersRequest {
  string query = 1;                  // Name or email
  int32 max_results = 2;
}

message SearchUsersResponse {
  repeated User users = 1;
}

// === Common Types ===

message Issue {
  string id = 1;
  string key = 2;
  string self = 3;
  IssueFields fields = 4;
}

message IssueFields {
  string summary = 1;
  string description = 2;
  IssueType issue_type = 3;
  Status status = 4;
  Priority priority = 5;
  User assignee = 6;
  User reporter = 7;
  Project project = 8;
  repeated string labels = 9;
  string created = 10;
  string updated = 11;
  repeated Comment comments = 12;
  google.protobuf.Struct custom_fields = 13;
}

message IssueType {
  string id = 1;
  string name = 2;
  string description = 3;
  bool subtask = 4;
}

message Status {
  string id = 1;
  string name = 2;
  string description = 3;
  StatusCategory category = 4;
}

message StatusCategory {
  string id = 1;
  string key = 2;              // "new", "indeterminate", "done"
  string name = 3;
}

message Priority {
  string id = 1;
  string name = 2;
}

message User {
  string account_id = 1;
  string display_name = 2;
  string email_address = 3;
  bool active = 4;
}

message Project {
  string id = 1;
  string key = 2;
  string name = 3;
}

message Comment {
  string id = 1;
  string body = 2;
  User author = 3;
  string created = 4;
  string updated = 5;
}

message Visibility {
  string type = 1;             // "group" or "role"
  string value = 2;            // Group name or role name
}
```

## Go Implementation

### File Structure

```
mcproto/
├── proto/
│   └── jira/
│       └── jira.proto           # NEW
├── go/
│   ├── gen/
│   │   └── jira/
│   │       └── jira.pb.go       # Generated
│   ├── pkg/
│   │   └── jira/
│   │       ├── client.go        # NEW: Jira REST client
│   │       └── server.go        # NEW: Server wrapper
│   └── pkg/
│       └── registry/
│           └── jira_tools.go    # NEW: Tool registration
└── buf.yaml                     # Update for jira module
```

### Jira Client (`go/pkg/jira/client.go`)

```go
package jira

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "os"
)

type Client struct {
    baseURL    string
    httpClient *http.Client
    email      string
    apiToken   string
}

func NewClient() (*Client, error) {
    url := os.Getenv("JIRA_URL")
    email := os.Getenv("JIRA_EMAIL")
    token := os.Getenv("JIRA_API_TOKEN")

    if url == "" || email == "" || token == "" {
        return nil, fmt.Errorf("JIRA_URL, JIRA_EMAIL, and JIRA_API_TOKEN required")
    }

    return &Client{
        baseURL:    url,
        httpClient: &http.Client{},
        email:      email,
        apiToken:   token,
    }, nil
}

func (c *Client) do(ctx context.Context, method, path string, body, result interface{}) error {
    // Implementation: Basic auth, JSON marshaling, error handling
}

func (c *Client) SearchIssues(ctx context.Context, jql string, opts SearchOptions) (*SearchResult, error) {
    // GET /rest/api/3/search?jql=...
}

func (c *Client) GetIssue(ctx context.Context, key string) (*Issue, error) {
    // GET /rest/api/3/issue/{key}
}

func (c *Client) CreateIssue(ctx context.Context, req *CreateIssueRequest) (*CreateIssueResponse, error) {
    // POST /rest/api/3/issue
}

// ... additional methods
```

### Tool Registration (`go/pkg/registry/jira_tools.go`)

```go
package registry

import (
    "context"
    "fmt"

    "github.com/tdewitt/mcproto/go/gen/jira"
    jiraclient "github.com/tdewitt/mcproto/go/pkg/jira"
    mcp "github.com/tdewitt/mcproto/go/gen/mcp"
    "google.golang.org/protobuf/proto"
)

func (r *UnifiedRegistry) PopulateJiraTools(client *jiraclient.Client) error {
    tools := []struct {
        name        string
        description string
        bsrRef      string
        handler     ToolHandler
    }{
        {
            name:        "SearchIssues",
            description: "Search Jira issues using JQL. Returns matching issues with selected fields.",
            bsrRef:      "buf.build/mcpb/jira/tucker.mcproto.jira.v1.SearchIssuesRequest:main",
            handler:     makeSearchIssuesHandler(client),
        },
        {
            name:        "GetIssue",
            description: "Get details of a specific Jira issue by key (e.g., PROJ-123).",
            bsrRef:      "buf.build/mcpb/jira/tucker.mcproto.jira.v1.GetIssueRequest:main",
            handler:     makeGetIssueHandler(client),
        },
        {
            name:        "CreateIssue",
            description: "Create a new Jira issue in a project with specified type, summary, and fields.",
            bsrRef:      "buf.build/mcpb/jira/tucker.mcproto.jira.v1.CreateIssueRequest:main",
            handler:     makeCreateIssueHandler(client),
        },
        {
            name:        "TransitionIssue",
            description: "Transition a Jira issue to a new status. Use GetTransitions first to get available transition IDs.",
            bsrRef:      "buf.build/mcpb/jira/tucker.mcproto.jira.v1.TransitionIssueRequest:main",
            handler:     makeTransitionIssueHandler(client),
        },
        {
            name:        "AddComment",
            description: "Add a comment to a Jira issue.",
            bsrRef:      "buf.build/mcpb/jira/tucker.mcproto.jira.v1.AddCommentRequest:main",
            handler:     makeAddCommentHandler(client),
        },
        {
            name:        "AssignIssue",
            description: "Assign a Jira issue to a user by account ID. Use SearchUsers to find account IDs.",
            bsrRef:      "buf.build/mcpb/jira/tucker.mcproto.jira.v1.AssignIssueRequest:main",
            handler:     makeAssignIssueHandler(client),
        },
        {
            name:        "GetTransitions",
            description: "Get available status transitions for an issue. Returns transition IDs needed for TransitionIssue.",
            bsrRef:      "buf.build/mcpb/jira/tucker.mcproto.jira.v1.GetTransitionsRequest:main",
            handler:     makeGetTransitionsHandler(client),
        },
        {
            name:        "SearchUsers",
            description: "Search for Jira users by name or email. Returns account IDs for assignment.",
            bsrRef:      "buf.build/mcpb/jira/tucker.mcproto.jira.v1.SearchUsersRequest:main",
            handler:     makeSearchUsersHandler(client),
        },
    }

    for _, t := range tools {
        r.Register(&mcp.Tool{
            Name:        t.name,
            Description: t.description,
            SchemaSource: &mcp.Tool_BsrRef{
                BsrRef: t.bsrRef,
            },
        }, t.handler)
    }

    return nil
}

func makeSearchIssuesHandler(client *jiraclient.Client) ToolHandler {
    return func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
        var req jira.SearchIssuesRequest
        if err := proto.Unmarshal(args, &req); err != nil {
            return nil, fmt.Errorf("unmarshal request: %w", err)
        }

        result, err := client.SearchIssues(ctx, req.Jql, jiraclient.SearchOptions{
            Fields:     req.Fields,
            StartAt:    int(req.StartAt),
            MaxResults: int(req.MaxResults),
            Expand:     req.Expand,
        })
        if err != nil {
            return nil, err
        }

        // Format as text for LLM consumption
        text := formatSearchResults(result)
        return mcpText(text), nil
    }
}

// ... additional handler implementations
```

## Environment Variables

```bash
# Jira Cloud API
JIRA_URL=https://yourcompany.atlassian.net
JIRA_EMAIL=your-email@company.com
JIRA_API_TOKEN=<api-token-from-atlassian>
```

## BSR Publishing

```bash
# From proto/jira directory
buf push --create-visibility public
```

This publishes the Jira schemas to BSR at `buf.build/mcpb/jira`, enabling runtime schema resolution.

## Implementation Phases

### Phase 1: Foundation
- [ ] Create `proto/jira/jira.proto` with core messages
- [ ] Update `buf.yaml` to include jira module
- [ ] Generate Go code with `buf generate`
- [ ] Implement basic `jira.Client` with auth

### Phase 2: Core Tools
- [ ] Implement `SearchIssues` handler
- [ ] Implement `GetIssue` handler
- [ ] Implement `CreateIssue` handler
- [ ] Implement `TransitionIssue` handler
- [ ] Add to registry in `main.go`

### Phase 3: Extended Tools
- [ ] Implement `AddComment` handler
- [ ] Implement `AssignIssue` handler
- [ ] Implement `GetTransitions` handler
- [ ] Implement `SearchUsers` handler

### Phase 4: Integration
- [ ] Publish protos to BSR
- [ ] Test with Python client
- [ ] Test with Claude via stdio transport
- [ ] Add to demo script

## Jira Service Management (JSM) Support

JSM has a separate API at `/rest/servicedeskapi/` for service desk-specific operations. The standard Jira API works for issues in JSM projects, but JSM adds customer-facing concepts.

### JSM-Specific Tools (Phase 4)

| Tool | Description | JSM API |
|------|-------------|---------|
| `GetServiceDesks` | List accessible service desks | `GET /servicedesk` |
| `GetServiceDesk` | Get service desk details | `GET /servicedesk/{serviceDeskId}` |
| `GetRequestTypes` | Get request types for a service desk | `GET /servicedesk/{id}/requesttype` |
| `CreateRequest` | Create customer request (public portal style) | `POST /servicedesk/{id}/request` |
| `GetRequest` | Get request with JSM-specific fields | `GET /request/{issueIdOrKey}` |
| `GetRequestComments` | Get comments with public/internal visibility | `GET /request/{id}/comment` |
| `AddRequestComment` | Add comment with visibility control | `POST /request/{id}/comment` |
| `GetRequestParticipants` | Get request participants | `GET /request/{id}/participant` |
| `AddRequestParticipants` | Add participants to request | `POST /request/{id}/participant` |
| `GetOrganizations` | List organizations | `GET /organization` |
| `GetOrganization` | Get organization details | `GET /organization/{organizationId}` |
| `GetCustomers` | Get customers in organization | `GET /organization/{id}/user` |
| `AddCustomer` | Add customer to service desk | `POST /servicedesk/{id}/customer` |
| `GetQueues` | Get queues for service desk | `GET /servicedesk/{id}/queue` |
| `GetSlaInfo` | Get SLA information for request | `GET /request/{id}/sla` |
| `SearchKnowledgeBase` | Search knowledge base articles | `GET /knowledgebase/article` |

### JSM Proto Messages

```protobuf
// === Service Desk ===

message GetServiceDesksRequest {
  int32 start = 1;
  int32 limit = 2;
}

message GetServiceDesksResponse {
  repeated ServiceDesk service_desks = 1;
  int32 size = 2;
  int32 start = 3;
  int32 limit = 4;
  bool is_last_page = 5;
}

message ServiceDesk {
  string id = 1;
  string project_id = 2;
  string project_name = 3;
  string project_key = 4;
}

// === Request Types ===

message GetRequestTypesRequest {
  string service_desk_id = 1;
  int32 start = 2;
  int32 limit = 3;
}

message GetRequestTypesResponse {
  repeated RequestType request_types = 1;
  int32 size = 2;
  int32 start = 3;
  int32 limit = 4;
  bool is_last_page = 5;
}

message RequestType {
  string id = 1;
  string name = 2;
  string description = 3;
  string help_text = 4;
  string service_desk_id = 5;
  repeated string group_ids = 6;
}

// === Customer Requests ===

message CreateRequestRequest {
  string service_desk_id = 1;
  string request_type_id = 2;
  map<string, string> request_field_values = 3;  // Field ID -> value
  repeated string request_participants = 4;       // Account IDs
  bool raise_on_behalf_of = 5;
  string raise_on_behalf_of_account_id = 6;
}

message CreateRequestResponse {
  string issue_id = 1;
  string issue_key = 2;
  string request_type_id = 3;
  string service_desk_id = 4;
  CustomerRequestStatus current_status = 5;
}

message CustomerRequestStatus {
  string status = 1;
  string status_category = 2;
  string status_date = 3;
}

// === Comments with Visibility ===

message AddRequestCommentRequest {
  string issue_id_or_key = 1;
  string body = 2;
  bool public = 3;  // true = visible to customer, false = internal only
}

message AddRequestCommentResponse {
  string id = 1;
  string body = 2;
  bool public = 3;
  User author = 4;
  string created = 5;
}

// === Organizations ===

message GetOrganizationsRequest {
  int32 start = 1;
  int32 limit = 2;
}

message GetOrganizationsResponse {
  repeated Organization organizations = 1;
  int32 size = 2;
  int32 start = 3;
  int32 limit = 4;
  bool is_last_page = 5;
}

message Organization {
  string id = 1;
  string name = 2;
}

// === SLA ===

message GetSlaInfoRequest {
  string issue_id_or_key = 1;
}

message GetSlaInfoResponse {
  repeated SlaInfo values = 1;
}

message SlaInfo {
  string id = 1;
  string name = 2;
  CompletedCycle completed_cycles = 3;
  OngoingCycle ongoing_cycle = 4;
}

message CompletedCycle {
  string start_time = 1;
  string stop_time = 2;
  bool breached = 3;
  Duration goal_duration = 4;
  Duration elapsed_time = 5;
  Duration remaining_time = 6;
}

message OngoingCycle {
  string start_time = 1;
  bool breached = 2;
  bool paused = 3;
  bool within_calendar_hours = 4;
  Duration goal_duration = 5;
  Duration elapsed_time = 6;
  Duration remaining_time = 7;
}

message Duration {
  int64 millis = 1;
  string friendly = 2;  // e.g., "2h 30m"
}
```

## Decisions Made

1. **Location**: In mcproto repo (custom work)
2. **Markdown/ADF**: Accept markdown, convert to ADF internally
3. **Custom Fields**: Deferred for now
4. **Pagination**: Manual with `start` and `limit` parameters on all list operations

## Updated Implementation Phases

### Phase 1: Foundation
- [ ] Create `proto/jira/jira.proto` with core messages
- [ ] Update `buf.yaml` to include jira module
- [ ] Generate Go code with `buf generate`
- [ ] Implement basic `jira.Client` with auth
- [ ] Implement markdown-to-ADF converter

### Phase 2: Core Jira Tools
- [ ] Implement `SearchIssues` handler
- [ ] Implement `GetIssue` handler
- [ ] Implement `CreateIssue` handler
- [ ] Implement `TransitionIssue` handler
- [ ] Add to registry in `main.go`

### Phase 3: Extended Jira Tools
- [ ] Implement `AddComment` handler
- [ ] Implement `AssignIssue` handler
- [ ] Implement `GetTransitions` handler
- [ ] Implement `SearchUsers` handler

### Phase 4: JSM Tools
- [ ] Implement JSM client (`/rest/servicedeskapi/`)
- [ ] Implement `GetServiceDesks`, `GetRequestTypes`
- [ ] Implement `CreateRequest`, `GetRequest`
- [ ] Implement `AddRequestComment` (with public/internal)
- [ ] Implement `GetOrganizations`, `GetCustomers`
- [ ] Implement `GetSlaInfo`

### Phase 5: Integration & Testing
- [ ] Publish protos to BSR
- [ ] Test with Python client
- [ ] Test with Claude via stdio transport
- [ ] Add to demo script
