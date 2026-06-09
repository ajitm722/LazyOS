package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/ajitm722/LazyOS/internal/daemons"
)

// tableEntry is the JSON shape returned by list_tables and describe_table.
type tableEntry struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Columns     string `json:"columns"`
	Backend     string `json:"backend"`
}

// handleListTables returns all available osquery tables across every backend.
// An optional "backend" argument filters results to a single backend.
func (s *Server) handleListTables(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	filterBackend := strings.TrimSpace(req.GetString("backend", ""))

	var tables []tableEntry
	for name, backend := range s.backends {
		if filterBackend != "" && name != filterBackend {
			continue
		}
		for _, t := range backend.GetSchema() {
			tables = append(tables, tableEntry{
				Name:        t.Name,
				Description: t.Description,
				Columns:     t.Columns,
				Backend:     name,
			})
		}
	}

	result := struct {
		Tables []tableEntry `json:"tables"`
		Count  int          `json:"count"`
	}{
		Tables: tables,
		Count:  len(tables),
	}

	payload, _ := json.Marshal(result)
	return mcp.NewToolResultText(string(payload)), nil
}

// handleDescribeTable returns full schema details for a single table,
// including a generated sample query. Searches across all backends.
func (s *Server) handleDescribeTable(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tableName, err := req.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError("missing required argument: name"), nil
	}
	tableName = strings.TrimSpace(tableName)

	for backendName, backend := range s.backends {
		for _, t := range backend.GetSchema() {
			if strings.EqualFold(t.Name, tableName) {
				cols := daemons.ExtractColumnNames(t.Columns)
				sampleQuery := fmt.Sprintf("SELECT %s FROM %s LIMIT 10",
					strings.Join(cols, ", "), t.Name)

				result := struct {
					Name        string   `json:"name"`
					Description string   `json:"description"`
					Columns     []string `json:"columns"`
					Backend     string   `json:"backend"`
					SampleQuery string   `json:"sample_query"`
				}{
					Name:        t.Name,
					Description: t.Description,
					Columns:     cols,
					Backend:     backendName,
					SampleQuery: sampleQuery,
				}

				payload, _ := json.Marshal(result)
				return mcp.NewToolResultText(string(payload)), nil
			}
		}
	}

	return mcp.NewToolResultError(fmt.Sprintf("table not found: %s", tableName)), nil
}

// handleOsqueryQuery executes a SQL query against the appropriate backend and
// returns rows, columns, and metadata.
func (s *Server) handleOsqueryQuery(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sql, err := req.RequireString("sql")
	if err != nil {
		return mcp.NewToolResultError("missing required argument: sql"), nil
	}
	sql = strings.TrimSpace(sql)

	backend := s.resolveBackend(sql)
	if backend == nil {
		return mcp.NewToolResultError("could not determine backend for query; no backends available"), nil
	}

	start := time.Now()
	rows, columns, err := backend.Query(ctx, sql)
	elapsed := time.Since(start)

	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("query failed: %v", err)), nil
	}

	result := struct {
		Columns     []string            `json:"columns"`
		Rows        []map[string]string `json:"rows"`
		RowCount    int                 `json:"row_count"`
		Backend     string              `json:"backend"`
		QueryTimeMs int64               `json:"query_time_ms"`
		SQL         string              `json:"sql"`
	}{
		Columns:     columns,
		Rows:        rows,
		RowCount:    len(rows),
		Backend:     s.backendName(backend),
		QueryTimeMs: elapsed.Milliseconds(),
		SQL:         sql,
	}

	payload, _ := json.Marshal(result)
	return mcp.NewToolResultText(string(payload)), nil
}

// resolveBackend determines which backend should handle the given SQL by
// parsing the table name and matching it against each backend's schema.
// Falls back to the first available backend if the table cannot be resolved.
func (s *Server) resolveBackend(sql string) daemons.Queryer {
	for name, backend := range s.backends {
		cols := daemons.DeriveColumnsFromSchema(sql, backend.GetSchema())
		if cols != nil {
			return s.backends[name]
		}
	}

	// Fallback: return the first available backend
	for _, backend := range s.backends {
		return backend
	}
	return nil
}

// backendName returns the registered name for the given Queryer instance.
func (s *Server) backendName(target daemons.Queryer) string {
	for name, backend := range s.backends {
		if backend == target {
			return name
		}
	}
	return "unknown"
}
