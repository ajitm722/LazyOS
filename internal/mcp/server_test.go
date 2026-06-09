package mcp

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/ajitm722/LazyOS/internal/daemons"
	"github.com/ajitm722/LazyOS/internal/daemons/mock"
)

// testBackends creates a Server with two mock backends for testing.
func testBackends() map[string]daemons.Queryer {
	kernelMock := &mock.MockQueryer{
		Schema: mock.MockTables,
		Results: map[string][]map[string]string{
			"SELECT pid, name, path, cmdline, state, cwd, root, uid, gid, on_disk, resident_size, total_size FROM processes": {
				{"pid": "1", "name": "systemd", "state": "S"},
				{"pid": "42", "name": "nginx", "state": "S"},
			},
		},
	}

	awsMock := &mock.MockQueryer{
		Schema: []daemons.TableSchema{
			{Name: "aws_ec2_instance", Description: "EC2 instances", Columns: "account_id, region_code, instances_instance_id, instances_state"},
		},
		Results: map[string][]map[string]string{
			"SELECT account_id, region_code, instances_instance_id, instances_state FROM aws_ec2_instance": {
				{"instances_instance_id": "i-abc123", "instances_state": "running"},
			},
		},
	}

	return map[string]daemons.Queryer{
		"osquery-kernel": kernelMock,
		"osquery-aws":    awsMock,
	}
}

func newCallToolRequest(args map[string]any) mcp.CallToolRequest {
	return mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: args,
		},
	}
}

func TestListTablesAll(t *testing.T) {
	s := New(testBackends(), "test")
	req := newCallToolRequest(map[string]any{})

	result, err := s.handleListTables(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(result.Content))
	}

	text := result.Content[0].(mcp.TextContent).Text
	var parsed struct {
		Tables []tableEntry `json:"tables"`
		Count  int          `json:"count"`
	}
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if parsed.Count < 3 {
		t.Errorf("expected at least 3 tables (3 mock + 1 aws), got %d", parsed.Count)
	}
}

func TestListTablesFiltered(t *testing.T) {
	s := New(testBackends(), "test")
	req := newCallToolRequest(map[string]any{"backend": "osquery-kernel"})

	result, err := s.handleListTables(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	text := result.Content[0].(mcp.TextContent).Text
	var parsed struct {
		Tables []tableEntry `json:"tables"`
		Count  int          `json:"count"`
	}
	json.Unmarshal([]byte(text), &parsed)

	for _, entry := range parsed.Tables {
		if entry.Backend != "osquery-kernel" {
			t.Errorf("expected only kernel tables, got backend=%s for table %s", entry.Backend, entry.Name)
		}
	}
}

func TestDescribeTableFound(t *testing.T) {
	s := New(testBackends(), "test")
	req := newCallToolRequest(map[string]any{"name": "processes"})

	result, err := s.handleDescribeTable(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	text := result.Content[0].(mcp.TextContent).Text
	var parsed struct {
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Columns     []string `json:"columns"`
		Backend     string   `json:"backend"`
		SampleQuery string   `json:"sample_query"`
	}
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if parsed.Name != "processes" {
		t.Errorf("expected name=processes, got %s", parsed.Name)
	}
	if parsed.Backend != "osquery-kernel" {
		t.Errorf("expected backend=osquery-kernel, got %s", parsed.Backend)
	}
	if len(parsed.Columns) == 0 {
		t.Error("expected non-empty columns list")
	}
	if parsed.SampleQuery == "" {
		t.Error("expected non-empty sample_query")
	}
}

func TestDescribeTableNotFound(t *testing.T) {
	s := New(testBackends(), "test")
	req := newCallToolRequest(map[string]any{"name": "nonexistent_table"})

	result, err := s.handleDescribeTable(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result for unknown table")
	}
}

func TestDescribeTableMissingArg(t *testing.T) {
	s := New(testBackends(), "test")
	req := newCallToolRequest(map[string]any{})

	result, err := s.handleDescribeTable(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result for missing name argument")
	}
}

func TestOsqueryQuerySuccess(t *testing.T) {
	s := New(testBackends(), "test")
	req := newCallToolRequest(map[string]any{
		"sql": "SELECT pid, name, path, cmdline, state, cwd, root, uid, gid, on_disk, resident_size, total_size FROM processes",
	})

	result, err := s.handleOsqueryQuery(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	text := result.Content[0].(mcp.TextContent).Text
	var parsed struct {
		Columns     []string            `json:"columns"`
		Rows        []map[string]string `json:"rows"`
		RowCount    int                 `json:"row_count"`
		Backend     string              `json:"backend"`
		QueryTimeMs int64               `json:"query_time_ms"`
	}
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if parsed.RowCount != 2 {
		t.Errorf("expected 2 rows, got %d", parsed.RowCount)
	}
	if len(parsed.Columns) == 0 {
		t.Error("expected non-empty columns")
	}
}

func TestOsqueryQueryEmptyResult(t *testing.T) {
	s := New(testBackends(), "test")
	req := newCallToolRequest(map[string]any{
		"sql": "SELECT pid, name FROM processes WHERE pid = 999999",
	})

	result, err := s.handleOsqueryQuery(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	text := result.Content[0].(mcp.TextContent).Text
	var parsed struct {
		RowCount int `json:"row_count"`
	}
	json.Unmarshal([]byte(text), &parsed)
	if parsed.RowCount != 0 {
		t.Errorf("expected 0 rows, got %d", parsed.RowCount)
	}
}

func TestOsqueryQueryMissingArg(t *testing.T) {
	s := New(testBackends(), "test")
	req := newCallToolRequest(map[string]any{})

	result, err := s.handleOsqueryQuery(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result for missing sql argument")
	}
}
