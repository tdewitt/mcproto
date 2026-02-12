package bsr

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestParseRef(t *testing.T) {
	refStr := "buf.build/acme/tools/acme.tools.v1.WebSearchRequest:v1"
	ref, err := ParseRef(refStr)
	if err != nil {
		t.Fatalf("ParseRef failed: %v", err)
	}

	if ref.Owner != "acme" {
		t.Errorf("Expected owner acme, got %s", ref.Owner)
	}
	if ref.Repository != "tools" {
		t.Errorf("Expected repository tools, got %s", ref.Repository)
	}
	if ref.Message != "acme.tools.v1.WebSearchRequest" {
		t.Errorf("Expected message acme.tools.v1.WebSearchRequest, got %s", ref.Message)
	}
	if ref.Version != "v1" {
		t.Errorf("Expected version v1, got %s", ref.Version)
	}
}

func TestFetchDescriptorSet(t *testing.T) {
	// Mock a FileDescriptorProto
	fd := &descriptorpb.FileDescriptorProto{
		Name:    proto.String("test.proto"),
		Package: proto.String("test.v1"),
	}

	fdJSON, _ := protojson.Marshal(fd)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/buf.alpha.registry.v1alpha1.ImageService/GetImage" {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}

		resp := map[string]interface{}{
			"image": map[string]interface{}{
				"file": []json.RawMessage{fdJSON},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client := &Client{
		httpClient: ts.Client(),
		baseURL:    ts.URL,
	}

	ref := &BSRRef{
		Owner:      "test",
		Repository: "test",
		Version:    "main",
	}

	fds, err := client.FetchDescriptorSet(context.Background(), ref)
	if err != nil {
		t.Fatalf("FetchDescriptorSet failed: %v", err)
	}

	if len(fds.File) != 1 {
		t.Fatalf("Expected 1 file, got %d", len(fds.File))
	}

	if fds.File[0].GetName() != "test.proto" {
		t.Errorf("Expected filename test.proto, got %s", fds.File[0].GetName())
	}
}

func TestParseRef_Errors(t *testing.T) {
	tests := []struct {
		ref string
	}{
		{"wrong.build/owner/repo/msg"},
		{"buf.build/owner/repo"},
	}

	for _, tt := range tests {
		_, err := ParseRef(tt.ref)
		if err == nil {
			t.Errorf("Expected error for ref %s, got nil", tt.ref)
		}
	}
}

func TestNewClientWithTimeout(t *testing.T) {
	c := NewClientWithTimeout(5 * time.Second)
	if c == nil {
		t.Fatal("NewClientWithTimeout returned nil")
	}
	if c.httpClient.Timeout != 5*time.Second {
		t.Errorf("Expected 5s timeout, got %v", c.httpClient.Timeout)
	}
}

func TestNewClientDefaultTimeout(t *testing.T) {
	c := NewClient()
	if c == nil {
		t.Fatal("NewClient returned nil")
	}
	if c.httpClient.Timeout != defaultHTTPTimeout {
		t.Errorf("Expected default timeout %v, got %v", defaultHTTPTimeout, c.httpClient.Timeout)
	}
}

func TestFetchDescriptorSet_Errors(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("not found"))
	}))
	defer ts.Close()

	client := &Client{
		httpClient: ts.Client(),
		baseURL:    ts.URL,
	}

	ref := &BSRRef{Owner: "test", Repository: "test"}
	_, err := client.FetchDescriptorSet(context.Background(), ref)
	if err == nil {
		t.Error("Expected error for 404 response, got nil")
	}
}
