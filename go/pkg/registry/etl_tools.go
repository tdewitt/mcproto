package registry

import (
	"context"
	"fmt"

	"github.com/misfitdev/proto-mcp/go/mcp"
)

// PopulateETLTools adds 12 realistic ETL tools to the registry.
func (r *UnifiedRegistry) PopulateETLTools() {
	// BSR references for the analytics specs
	const base = "buf.build/mcpb/analytics"
	extractRef := base + "/misfit.analytics.v1.ExtractRequest:main"
	transformRef := base + "/misfit.analytics.v1.TransformRequest:main"
	loadRef := base + "/misfit.analytics.v1.LoadRequest:main"

	// --- EXTRACT TOOLS ---
	r.Register(&mcp.Tool{
		Name:         "fetch_activity_stream",
		Description:  "Extracts a stream of raw user activity events from the source database.",
		SchemaSource: &mcp.Tool_BsrRef{BsrRef: extractRef},
	}, mockHandler("fetch_activity_stream"))

	r.Register(&mcp.Tool{
		Name:         "list_data_sources",
		Description:  "Lists all available data domains and their stream status.",
		SchemaSource: &mcp.Tool_BsrRef{BsrRef: extractRef},
	}, mockHandler("list_data_sources"))

	r.Register(&mcp.Tool{
		Name:         "get_stream_metadata",
		Description:  "Retrieves schema and throughput metadata for a specific activity stream.",
		SchemaSource: &mcp.Tool_BsrRef{BsrRef: extractRef},
	}, mockHandler("get_stream_metadata"))

	// --- TRANSFORM TOOLS ---
	r.Register(&mcp.Tool{
		Name:         "apply_data_mapping",
		Description:  "Applies a set of transformation rules to an event's payload.",
		SchemaSource: &mcp.Tool_BsrRef{BsrRef: transformRef},
	}, mockHandler("apply_data_mapping"))

	r.Register(&mcp.Tool{
		Name:         "enrich_geo_location",
		Description:  "Enriches event data with geographical information based on IP address.",
		SchemaSource: &mcp.Tool_BsrRef{BsrRef: transformRef},
	}, mockHandler("enrich_geo_location"))

	r.Register(&mcp.Tool{
		Name:         "validate_event_schema",
		Description:  "Checks if an event payload conforms to its domain-specific Protobuf schema.",
		SchemaSource: &mcp.Tool_BsrRef{BsrRef: transformRef},
	}, mockHandler("validate_event_schema"))

	r.Register(&mcp.Tool{
		Name:         "anonymize_pii_fields",
		Description:  "Masks or hashes personally identifiable information in the event payload.",
		SchemaSource: &mcp.Tool_BsrRef{BsrRef: transformRef},
	}, mockHandler("anonymize_pii_fields"))

	// --- LOAD TOOLS ---
	r.Register(&mcp.Tool{
		Name:         "write_to_warehouse",
		Description:  "Commits a batch of events to the long-term analytics warehouse.",
		SchemaSource: &mcp.Tool_BsrRef{BsrRef: loadRef},
	}, mockHandler("write_to_warehouse"))

	r.Register(&mcp.Tool{
		Name:         "emit_to_webhook",
		Description:  "Forwards events to an external real-time processing webhook.",
		SchemaSource: &mcp.Tool_BsrRef{BsrRef: loadRef},
	}, mockHandler("emit_to_webhook"))

	r.Register(&mcp.Tool{
		Name:         "archive_to_cold_storage",
		Description:  "Moves processed events to S3 cold storage for compliance.",
		SchemaSource: &mcp.Tool_BsrRef{BsrRef: loadRef},
	}, mockHandler("archive_to_cold_storage"))

	r.Register(&mcp.Tool{
		Name:         "push_to_valkey_cache",
		Description:  "Updates the real-time Valkey JSON cache with the latest user state.",
		SchemaSource: &mcp.Tool_BsrRef{BsrRef: loadRef},
	}, mockHandler("push_to_valkey_cache"))

	r.Register(&mcp.Tool{
		Name:         "log_pipeline_event",
		Description:  "Logs a summary of the ETL operation to the internal monitoring system.",
		SchemaSource: &mcp.Tool_BsrRef{BsrRef: loadRef},
	}, mockHandler("log_pipeline_event"))
}

func mockHandler(name string) ToolHandler {
	return func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		return &mcp.ToolResult{
			Content: []*mcp.ToolContent{
				{
					Content: &mcp.ToolContent_Text{
						Text: fmt.Sprintf("Tool [%s] executed successfully.", name),
					},
				},
			},
		}, nil
	}
}
