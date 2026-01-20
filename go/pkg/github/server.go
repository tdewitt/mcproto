package github

import (
	"context"
	"fmt"
	"os"

	gh "github.com/google/go-github/v69/github"
	"golang.org/x/oauth2"
	"google.golang.org/protobuf/proto"
)

type Server struct {
	UnimplementedGitHubServiceServer
	client *gh.Client
}

func NewServer() (*Server, error) {
	token := os.Getenv("GITHUB_PERSONAL_ACCESS_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("GITHUB_PERSONAL_ACCESS_TOKEN is not set")
	}

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(context.Background(), ts)
	client := gh.NewClient(tc)

	return &Server{
		client: client,
	}, nil
}

func (s *Server) SearchRepositories(ctx context.Context, req *SearchRepositoriesRequest) (*SearchRepositoriesResponse, error) {
	opts := &gh.SearchOptions{
		ListOptions: gh.ListOptions{
			Page:    int(req.GetPage()),
			PerPage: int(req.GetPerPage()),
		},
	}

	result, _, err := s.client.Search.Repositories(ctx, req.Query, opts)
	if err != nil {
		return nil, err
	}

	repos := make([]*Repository, len(result.Repositories))
	for i, r := range result.Repositories {
		repos[i] = convertRepository(r)
	}

	return &SearchRepositoriesResponse{
		TotalCount:        int32(result.GetTotal()),
		IncompleteResults: result.GetIncompleteResults(),
		Repositories:      repos,
	}, nil
}

func convertRepository(r *gh.Repository) *Repository {
	if r == nil {
		return nil
	}
	repo := &Repository{
		Id:              r.GetID(),
		NodeId:          r.GetNodeID(),
		Name:            r.GetName(),
		FullName:        r.GetFullName(),
		Private:         r.GetPrivate(),
		HtmlUrl:         r.GetHTMLURL(),
		Description:     proto.String(r.GetDescription()),
		Fork:            r.GetFork(),
		Url:             r.GetURL(),
		CreatedAt:       r.GetCreatedAt().String(),
		UpdatedAt:       r.GetUpdatedAt().String(),
		PushedAt:        r.GetPushedAt().String(),
		GitUrl:          r.GetGitURL(),
		SshUrl:          r.GetSSHURL(),
		CloneUrl:        r.GetCloneURL(),
		Homepage:        proto.String(r.GetHomepage()),
		Size:            int32(r.GetSize()),
		StargazersCount: int32(r.GetStargazersCount()),
		WatchersCount:   int32(r.GetWatchersCount()),
		Language:        proto.String(r.GetLanguage()),
		HasIssues:       r.GetHasIssues(),
		HasProjects:     r.GetHasProjects(),
		HasDownloads:    r.GetHasDownloads(),
		HasWiki:         r.GetHasWiki(),
		HasPages:        r.GetHasPages(),
		ForksCount:      int32(r.GetForksCount()),
		OpenIssuesCount: int32(r.GetOpenIssuesCount()),
		DefaultBranch:   r.GetDefaultBranch(),
	}
	if r.Owner != nil {
		repo.Owner = convertUser(r.Owner)
	}
	return repo
}

func convertUser(u *gh.User) *User {
	if u == nil {
		return nil
	}
	return &User{
		Login:      u.GetLogin(),
		Id:         u.GetID(),
		NodeId:     u.GetNodeID(),
		AvatarUrl:  u.GetAvatarURL(),
		GravatarId: u.GetGravatarID(),
		Url:        u.GetURL(),
		HtmlUrl:    u.GetHTMLURL(),
		Type:       u.GetType(),
		SiteAdmin:  u.GetSiteAdmin(),
	}
}

func (s *Server) GetRepository(ctx context.Context, req *GetRepositoryRequest) (*GetRepositoryResponse, error) {
	repo, _, err := s.client.Repositories.Get(ctx, req.Owner, req.Repo)
	if err != nil {
		return nil, err
	}
	return &GetRepositoryResponse{
		Repository: convertRepository(repo),
	}, nil
}

func (s *Server) ListIssues(ctx context.Context, req *ListIssuesRequest) (*ListIssuesResponse, error) {
	opts := &gh.IssueListByRepoOptions{
		State:     req.GetState(),
		Sort:      req.GetSort(),
		Direction: req.GetDirection(),
		ListOptions: gh.ListOptions{
			Page:    int(req.GetPage()),
			PerPage: int(req.GetPerPage()),
		},
	}

	issues, _, err := s.client.Issues.ListByRepo(ctx, req.Owner, req.Repo, opts)
	if err != nil {
		return nil, err
	}

	protoIssues := make([]*Issue, len(issues))
	for i, issue := range issues {
		protoIssues[i] = convertIssue(issue)
	}

	return &ListIssuesResponse{
		Issues: protoIssues,
	}, nil
}

func convertIssue(i *gh.Issue) *Issue {
	if i == nil {
		return nil
	}
	issue := &Issue{
		Id:            i.GetID(),
		NodeId:        i.GetNodeID(),
		Url:           i.GetURL(),
		RepositoryUrl: i.GetRepositoryURL(),
		HtmlUrl:       i.GetHTMLURL(),
		Number:        int32(i.GetNumber()),
		State:         i.GetState(),
		Title:         i.GetTitle(),
		Body:          proto.String(i.GetBody()),
		User:          convertUser(i.User),
		Comments:      int32(i.GetComments()),
		CreatedAt:     i.GetCreatedAt().String(),
		UpdatedAt:     i.GetUpdatedAt().String(),
	}
	// TODO: Add Labels, Assignee, etc.
	return issue
}

func (s *Server) CreateIssue(ctx context.Context, req *CreateIssueRequest) (*CreateIssueResponse, error) {
	issueRequest := &gh.IssueRequest{
		Title:     &req.Title,
		Body:      req.Body,
		Assignees: &req.Assignees,
		Labels:    &req.Labels,
	}
	if req.Milestone != nil {
		m := int(req.GetMilestone())
		issueRequest.Milestone = &m
	}

	issue, _, err := s.client.Issues.Create(ctx, req.Owner, req.Repo, issueRequest)
	if err != nil {
		return nil, err
	}

	return &CreateIssueResponse{
		Issue: convertIssue(issue),
	}, nil
}

func (s *Server) CreateOrUpdateFile(ctx context.Context, req *CreateOrUpdateFileRequest) (*FileCommitResponse, error) {
	opts := &gh.RepositoryContentFileOptions{
		Message: &req.Message,
		Content: []byte(req.Content),
		Branch:  &req.Branch,
		SHA:     req.Sha,
	}

	content, _, err := s.client.Repositories.CreateFile(ctx, req.Owner, req.Repo, req.Path, opts)
	if err != nil {
		return nil, err
	}

	return &FileCommitResponse{
		Content: convertContent(content.Content),
		Commit:  convertCommit(&content.Commit),
	}, nil
}

func convertContent(c *gh.RepositoryContent) *Content {
	if c == nil {
		return nil
	}
	return &Content{
		Name:        c.GetName(),
		Path:        c.GetPath(),
		Sha:         c.GetSHA(),
		Size:        int32(c.GetSize()),
		Url:         c.GetURL(),
		HtmlUrl:     c.GetHTMLURL(),
		GitUrl:      c.GetGitURL(),
		DownloadUrl: c.GetDownloadURL(),
		Type:        c.GetType(),
	}
}

func convertCommit(c *gh.Commit) *Commit {
	if c == nil {
		return nil
	}
	return &Commit{
		Sha:     c.GetSHA(),
		NodeId:  c.GetNodeID(),
		Url:     c.GetURL(),
		HtmlUrl: c.GetHTMLURL(),
		Message: c.GetMessage(),
	}
}
