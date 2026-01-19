package registry

import (
	"context"
	"testing"
)

func TestPopulateETLTools(t *testing.T) {
	reg := NewUnifiedRegistry()
	reg.PopulateETLTools()

	tools := reg.List("")
	if len(tools) != 12 {
		t.Errorf("Expected 12 ETL tools, got %d", len(tools))
	}

	// Test a specific tool execution
	resp, err := reg.Call(context.Background(), "fetch_activity_stream", nil)
	if err != nil {
		t.Fatalf("Tool call failed: %v", err)
	}

	if len(resp.Content) == 0 {
		t.Fatal("Expected response content, got none")
	}

	expectedText := "Tool [fetch_activity_stream] executed successfully."
	if resp.Content[0].GetText() != expectedText {
		t.Errorf("Expected '%s', got '%s'", expectedText, resp.Content[0].GetText())
	}
}

func TestGenerateLeadEvent(t *testing.T) {
	evt := GenerateLeadEvent()
	if evt.Id != "evt_12345" {
		t.Errorf("Expected event ID evt_12345, got %s", evt.Id)
	}
	if evt.Domain != "marketing_leads" {
		t.Errorf("Expected domain marketing_leads, got %s", evt.Domain)
	}
	
	email := evt.Payload.Fields["email"].GetStringValue()
	if email != "test-lead@example.com" {
		t.Errorf("Expected email test-lead@example.com, got %s", email)
	}
}

func TestGenerateMockCatalog(t *testing.T) {
	reg := NewUnifiedRegistry()
	reg.GenerateMockCatalog()
	tools := reg.List("")
	if len(tools) != 1000 {
		t.Errorf("Expected 1000 tools, got %d", len(tools))
	}
}
