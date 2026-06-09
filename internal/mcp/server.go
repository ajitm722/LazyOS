// Package mcp provides an MCP-over-HTTP server that exposes osquery tables
// and query execution to AI agents via the Model Context Protocol.
//
// The server uses Streamable HTTP transport, making it accessible from
// browsers, remote AI clients, and any MCP-compatible consumer. When
// paired with Tailscale, it provides secure cross-network access without
// API keys or TLS configuration.
package mcp

import (
	"context"
	"log"
	"runtime/debug"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/ajitm722/LazyOS/internal/daemons"
)

// Server wraps the mark3labs MCP server with osquery backends.
type Server struct {
	backends map[string]daemons.Queryer
	srv      *server.MCPServer
}

// New creates an MCP server, registers all osquery tools, and returns the
// Server ready for HTTP serving.
//
// backends is a map of backend name → Queryer (e.g. "osquery-kernel" → kernel.Queryer).
// version is the server version string. If empty, it is derived from
// runtime/debug.ReadBuildInfo. If that fails, "dev" is used.
func New(backends map[string]daemons.Queryer, version string) *Server {
	if version == "" {
		version = readVersion()
	}

	s := &Server{
		backends: backends,
		srv: server.NewMCPServer(
			"lazyos-mcp",
			version,
			server.WithToolCapabilities(false),
		),
	}

	s.registerTools()
	return s
}

// ListenAndServe starts the Streamable HTTP server on the given address
// (e.g. "127.0.0.1:8080" or "0.0.0.0:8080"). It blocks until ctx is
// cancelled, then initiates a graceful shutdown. All backend connections
// are closed when the server stops.
func (s *Server) ListenAndServe(ctx context.Context, addr string) error {
	defer func() {
		for name, backend := range s.backends {
			if err := backend.Close(); err != nil {
				log.Printf("mcp: failed to close backend %s: %v", name, err)
			}
		}
	}()

	httpServer := server.NewStreamableHTTPServer(s.srv,
		server.WithEndpointPath("/mcp"),
		server.WithStateLess(true),
	)

	log.Printf("mcp: lazyos-mcp listening on http://%s/mcp", addr)

	errCh := make(chan error, 1)
	go func() {
		errCh <- httpServer.Start(addr)
	}()

	select {
	case <-ctx.Done():
		log.Println("mcp: shutting down...")
		return httpServer.Shutdown(ctx)
	case err := <-errCh:
		return err
	}
}

// registerTools defines and registers all MCP tools on the server.
func (s *Server) registerTools() {
	// list_tables — enumerate all available osquery tables
	listTool := mcp.NewTool("list_tables",
		mcp.WithDescription("List all available osquery tables across enabled backends. Returns table name, description, column list, and backend ownership for each table."),
		mcp.WithString("backend",
			mcp.Description("Filter by backend: 'osquery-kernel', 'osquery-aws', or omit for all"),
		),
	)
	s.srv.AddTool(listTool, s.handleListTables)

	// describe_table — full schema detail for a single table
	describeTool := mcp.NewTool("describe_table",
		mcp.WithDescription("Get detailed schema for a specific osquery table including all column names and a sample query."),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Table name (e.g. 'processes', 'aws_ec2_instance')"),
		),
	)
	s.srv.AddTool(describeTool, s.handleDescribeTable)

	// osquery_query — execute SQL against the live osquery daemon
	queryTool := mcp.NewTool("osquery_query",
		mcp.WithDescription("Execute a SQL query against the live osquery daemon. Returns structured results with columns, rows, and metadata. All queries are read-only (osquery has no DML)."),
		mcp.WithString("sql",
			mcp.Required(),
			mcp.Description("SQL query to execute (e.g. 'SELECT pid, name, resident_size FROM processes ORDER BY resident_size DESC LIMIT 10')"),
		),
	)
	s.srv.AddTool(queryTool, s.handleOsqueryQuery)
}

// readVersion extracts the module version from build info, or returns "dev".
func readVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		return info.Main.Version
	}
	return "dev"
}
