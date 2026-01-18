package registry

import (
	"time"

	"github.com/misfitdev/proto-mcp/go/mcp"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// GenerateLeadEvent creates a deterministic mock event for testing.
func GenerateLeadEvent() *mcp.Event {
	// Fixed data for reproducibility
	payload, _ := structpb.NewStruct(map[string]interface{}{
		"email":      "test-lead@example.com",
		"ip_address": "192.168.1.1",
		"user_id":    "user_999",
		"source":     "google_ads",
		"meta": map[string]interface{}{
			"campaign": "proto_mcp_launch",
			"priority": 5,
		},
	})

	return &mcp.Event{
		Id:        "evt_12345",
		Domain:    "marketing_leads",
		Timestamp: timestamppb.New(time.Date(2026, 1, 17, 12, 0, 0, 0, time.UTC)),
		Payload:   payload,
	}
}
