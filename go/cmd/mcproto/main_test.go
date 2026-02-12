package main

import (
	"testing"

	"github.com/misfitdev/proto-mcp/go/pkg/bsr"
	"github.com/misfitdev/proto-mcp/go/pkg/registry"
)

func TestPopulateDefaultToolsRegistersBaselineCatalog(t *testing.T) {
	t.Setenv("GITHUB_PERSONAL_ACCESS_TOKEN", "")
	t.Setenv("JIRA_URL", "")
	t.Setenv("JIRA_EMAIL", "")
	t.Setenv("JIRA_API_TOKEN", "")

	reg := registry.NewUnifiedRegistry(bsr.NewClient())
	populateDefaultTools(reg, true) // include mock catalog for baseline count

	tools := reg.List("")
	if len(tools) < 1000 {
		t.Fatalf("expected baseline populated tools, got %d", len(tools))
	}
}
