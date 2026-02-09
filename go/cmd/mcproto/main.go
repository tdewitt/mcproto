package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/misfitdev/proto-mcp/go/mcp"
	"github.com/misfitdev/proto-mcp/go/pkg/bsr"
	"github.com/misfitdev/proto-mcp/go/pkg/github"
	grpc_pkg "github.com/misfitdev/proto-mcp/go/pkg/grpc"
	"github.com/misfitdev/proto-mcp/go/pkg/jira"
	"github.com/misfitdev/proto-mcp/go/pkg/linear"
	"github.com/misfitdev/proto-mcp/go/pkg/notion"
	"github.com/misfitdev/proto-mcp/go/pkg/registry"
	"github.com/misfitdev/proto-mcp/go/router"
	"google.golang.org/grpc"
)

type stdioReadWriter struct {
	reader *os.File
	writer *os.File
}

func (s *stdioReadWriter) Read(p []byte) (n int, err error)  { return s.reader.Read(p) }
func (s *stdioReadWriter) Write(p []byte) (n int, err error) { return s.writer.Write(p) }

func populateDefaultTools(reg *registry.UnifiedRegistry) {
	reg.PopulateETLTools()
	reg.PopulateDiscoveryTools()
	reg.GenerateMockCatalog() // Adds the 1,000 tools for the "Boss Demo"

	if ghServer, err := github.NewServer(); err != nil {
		fmt.Fprintf(os.Stderr, "Skipping GitHub tools: %v\n", err)
	} else {
		reg.PopulateGitHubTools(ghServer)
	}
	if jiraClient, err := jira.NewClient(); err != nil {
		fmt.Fprintf(os.Stderr, "Skipping Jira tools: %v\n", err)
	} else if err := reg.PopulateJiraTools(jiraClient); err != nil {
		fmt.Fprintf(os.Stderr, "Skipping Jira tools: %v\n", err)
	}

	if linearClient, err := linear.NewClient(); err != nil {
		fmt.Fprintf(os.Stderr, "Skipping Linear tools: %v\n", err)
	} else if err := reg.PopulateLinearTools(linearClient); err != nil {
		fmt.Fprintf(os.Stderr, "Skipping Linear tools: %v\n", err)
	}

	if notionClient, err := notion.NewClient(); err != nil {
		fmt.Fprintf(os.Stderr, "Skipping Notion tools: %v\n", err)
	} else if err := reg.PopulateNotionTools(notionClient); err != nil {
		fmt.Fprintf(os.Stderr, "Skipping Notion tools: %v\n", err)
	}
}

func main() {
	transport := flag.String("transport", "grpc", "Transport to use (grpc or stdio)")
	addr := flag.String("addr", ":50051", "gRPC listen address")
	populate := flag.Bool("populate", true, "Populate the server with the ETL and Discovery mock catalogs")
	flag.Parse()

	bsrClient := bsr.NewClient()
	reg := registry.NewUnifiedRegistry(bsrClient)

	if *populate {
		populateDefaultTools(reg)
	}

	fmt.Fprintf(os.Stderr, "MC Proto Server starting... [Transport: %s]\n", *transport)

	if *transport == "grpc" {
		lis, err := net.Listen("tcp", *addr)
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}
		s := grpc.NewServer()
		mcp.RegisterMCPServiceServer(s, grpc_pkg.NewServer(reg))
		fmt.Fprintf(os.Stderr, "gRPC listening on %s\n", *addr)
		if err := s.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	} else {
		// Stdio Transport with Dual-Protocol Router
		rw := &stdioReadWriter{reader: os.Stdin, writer: os.Stdout}
		pr := router.NewProtocolRouter(rw)

		// Support both Legacy JSON and high-efficiency Binary on the same pipe
		bsrRegistry := bsr.NewRegistry(bsrClient)
		pr.Register(router.ProtocolJSON, router.NewJSONHandler(reg, bsrRegistry))
		pr.Register(router.ProtocolBinary, router.NewBinaryHandler(reg))

		if err := pr.Route(); err != nil {
			log.Fatalf("Router session failed: %v", err)
		}
	}
}
