package bsr

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/descriptorpb"
)

// Client is a minimal BSR client to fetch descriptors.
type Client struct {
	httpClient *http.Client
	token      string
	baseURL    string
}

// defaultHTTPTimeout is the default timeout for HTTP requests to the BSR API.
const defaultHTTPTimeout = 30 * time.Second

// defaultFetchTimeout is the default context timeout for FetchDescriptorSet
// and Search when the caller does not supply a deadline.
const defaultFetchTimeout = 60 * time.Second

// NewClient creates a new BSR client.
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: defaultHTTPTimeout},
		token:      os.Getenv("BUF_TOKEN"),
		baseURL:    "https://api.buf.build",
	}
}

// BSRRef represents a parsed BSR reference.
type BSRRef struct {
	Owner      string
	Repository string
	Message    string
	Version    string
}

// ParseRef parses a BSR reference string.
// Format: buf.build/{owner}/{repository}/{full_message_name}:{version}
func ParseRef(ref string) (*BSRRef, error) {
	if !strings.HasPrefix(ref, "buf.build/") {
		return nil, fmt.Errorf("invalid BSR ref: must start with buf.build/")
	}
	parts := strings.Split(strings.TrimPrefix(ref, "buf.build/"), "/")
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid BSR ref: too few parts")
	}

	owner := parts[0]
	repo := parts[1]

	// The rest contains message name and version
	rest := strings.Join(parts[2:], "/")
	messageParts := strings.Split(rest, ":")
	messageName := messageParts[0]
	version := "main"
	if len(messageParts) > 1 {
		version = messageParts[1]
	}

	return &BSRRef{
		Owner:      owner,
		Repository: repo,
		Message:    messageName,
		Version:    version,
	}, nil
}

// FetchDescriptorSet fetches the FileDescriptorSet for a given BSR reference.
func (c *Client) FetchDescriptorSet(ctx context.Context, ref *BSRRef) (*descriptorpb.FileDescriptorSet, error) {
	// Apply a default timeout if the caller's context has no deadline.
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, defaultFetchTimeout)
		defer cancel()
	}

	url := fmt.Sprintf("%s/buf.alpha.registry.v1alpha1.ImageService/GetImage", c.baseURL)

	reqBody, err := json.Marshal(map[string]interface{}{
		"owner":      ref.Owner,
		"repository": ref.Repository,
		"reference":  ref.Version,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 8*1024))
		if readErr != nil {
			log.Printf("warning: failed to read BSR error response body: %v", readErr)
			return nil, fmt.Errorf("BSR API error (%d): <body unreadable>", resp.StatusCode)
		}
		return nil, fmt.Errorf("BSR API error (%d): %s", resp.StatusCode, string(body))
	}

	var imageResp struct {
		Image struct {
			File []json.RawMessage `json:"file"`
		} `json:"image"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&imageResp); err != nil {
		return nil, fmt.Errorf("failed to decode BSR response: %w", err)
	}

	fds := &descriptorpb.FileDescriptorSet{}
	unmarshaler := protojson.UnmarshalOptions{
		DiscardUnknown: true,
	}
	for _, fileJSON := range imageResp.Image.File {
		fd := &descriptorpb.FileDescriptorProto{}
		if err := unmarshaler.Unmarshal(fileJSON, fd); err != nil {
			return nil, fmt.Errorf("failed to unmarshal file descriptor: %w", err)
		}
		fds.File = append(fds.File, fd)
	}

	return fds, nil
}

// SearchResult represents a repository found in the registry.
type SearchResult struct {
	Owner      string `json:"owner"`
	Repository string `json:"name"`
}

// Search queries the BSR for repositories matching the query.
func (c *Client) Search(ctx context.Context, query string) ([]SearchResult, error) {
	// Apply a default timeout if the caller's context has no deadline.
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, defaultFetchTimeout)
		defer cancel()
	}

	url := fmt.Sprintf("%s/buf.alpha.registry.v1alpha1.SearchService/Search", c.baseURL)

	reqBody, err := json.Marshal(map[string]interface{}{
		"query":    query,
		"pageSize": 5,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 8*1024))
		if readErr != nil {
			log.Printf("warning: failed to read BSR Search error response body: %v", readErr)
			return nil, fmt.Errorf("BSR Search API error (%d): <body unreadable>", resp.StatusCode)
		}
		return nil, fmt.Errorf("BSR Search API error (%d): %s", resp.StatusCode, string(body))
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read BSR Search response body: %w", err)
	}

	var searchResp struct {
		Results []struct {
			Repository SearchResult `json:"repository"`
		} `json:"searchResults"`
	}

	if err := json.Unmarshal(bodyBytes, &searchResp); err != nil {
		return nil, fmt.Errorf("failed to decode search response: %w", err)
	}

	results := make([]SearchResult, 0, len(searchResp.Results))
	for _, res := range searchResp.Results {
		if res.Repository.Owner != "" {
			results = append(results, res.Repository)
		}
	}

	return results, nil
}
