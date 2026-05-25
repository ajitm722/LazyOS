//go:build integration

package osquery

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ajitm722/LazyOS/internal/daemons"
)

const (
	startupTimeout = 5 * time.Second
	queryTimeout   = 10 * time.Second
)

func socketPath(t *testing.T) string {
	t.Helper()
	p := os.Getenv("LAZYOS_TEST_SOCKET")
	if p == "" {
		t.Fatal("LAZYOS_TEST_SOCKET not set; integration tests require a live osqueryd socket")
	}
	return p
}

func TestIntegrationNewClient_ValidSocket(t *testing.T) {
	c, err := NewClient(socketPath(t), startupTimeout, queryTimeout)
	if err != nil {
		t.Fatalf("NewClient with valid socket returned error: %v", err)
	}
	if c == nil {
		t.Fatal("NewClient returned nil client")
	}
	if err := c.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
}

func TestIntegrationNewClient_InvalidSocket(t *testing.T) {
	_, err := NewClient("/nonexistent/socket.em", 0, 0)
	if err == nil {
		t.Fatal("NewClient with invalid socket expected error, got nil")
	}
}

func TestIntegrationGetSchema(t *testing.T) {
	c, err := NewClient(socketPath(t), startupTimeout, queryTimeout)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer c.Close()

	schema := c.GetSchema()
	if len(schema) != len(CoreTables) {
		t.Errorf("GetSchema returned %d tables, want %d", len(schema), len(CoreTables))
	}
	for _, s := range schema {
		found := false
		for _, ct := range CoreTables {
			if ct.Name == s.Name && ct.Columns == s.Columns {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("GetSchema returned unexpected table: %+v", s)
		}
	}
}

func TestIntegrationQuery_Basic(t *testing.T) {
	c, err := NewClient(socketPath(t), startupTimeout, queryTimeout)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer c.Close()

	rows, cols, err := c.Query(context.Background(), "SELECT pid, name FROM processes LIMIT 1")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(rows) == 0 {
		t.Fatal("expected at least one row from processes")
	}
	for _, col := range []string{"pid", "name"} {
		found := false
		for _, c := range cols {
			if c == col {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected column %q in result, got %v", col, cols)
		}
	}
	row := rows[0]
	if row["pid"] == "" {
		t.Error("expected non-empty pid")
	}
	if row["name"] == "" {
		t.Error("expected non-empty name")
	}
}

func TestIntegrationQuery_Timeout(t *testing.T) {
	c, err := NewClient(socketPath(t), startupTimeout, queryTimeout)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 0)
	defer cancel()

	_, _, err = c.Query(ctx, "SELECT * FROM processes")
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

func TestIntegrationQuery_InvalidSQL(t *testing.T) {
	c, err := NewClient(socketPath(t), startupTimeout, queryTimeout)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer c.Close()

	_, _, err = c.Query(context.Background(), "SELECT invalid")
	if err == nil {
		t.Fatal("expected error for invalid SQL, got nil")
	}
}

func TestIntegrationQuery_EmptyResult(t *testing.T) {
	c, err := NewClient(socketPath(t), startupTimeout, queryTimeout)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer c.Close()

	rows, cols, err := c.Query(context.Background(), "SELECT * FROM processes WHERE pid = -1")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("expected 0 rows, got %d", len(rows))
	}
	if len(cols) == 0 {
		t.Fatal("expected non-empty columns even for empty result")
	}
}

// actualColumnsForTable uses PRAGMA table_info to retrieve the real column
// names that osquery exposes for the given table.
func actualColumnsForTable(t *testing.T, c *Client, tableName string) map[string]struct{} {
	t.Helper()
	rows, _, err := c.Query(context.Background(), "PRAGMA table_info("+tableName+")")
	if err != nil {
		t.Fatalf("PRAGMA table_info(%s) failed: %v", tableName, err)
	}
	cols := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		cols[row["name"]] = struct{}{}
	}
	return cols
}

func TestIntegrationAllTableSchemas(t *testing.T) {
	c, err := NewClient(socketPath(t), startupTimeout, queryTimeout)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer c.Close()

	for _, table := range CoreTables {
		t.Run(table.Name, func(t *testing.T) {
			actualCols := actualColumnsForTable(t, c, table.Name)

			declaredCols := daemons.ExtractColumnNames(table.Columns)
			for _, col := range declaredCols {
				if _, ok := actualCols[col]; !ok {
					t.Errorf("table %s: declared column %q does not exist in actual osquery schema", table.Name, col)
				}
			}

			rows, queryCols, err := c.Query(context.Background(), "SELECT * FROM "+table.Name+" LIMIT 1")
			if err != nil {
				// Some tables may not have data or may not be queryable on this system
				t.Logf("SELECT * FROM %s LIMIT 1 returned error (may be acceptable): %v", table.Name, err)
				return
			}
			for _, col := range declaredCols {
				found := false
				for _, qc := range queryCols {
					if qc == col {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("table %s: declared column %q not found in query result columns %v", table.Name, col, queryCols)
				}
			}
			if len(rows) > 0 {
				row := rows[0]
				for _, col := range declaredCols {
					if _, ok := row[col]; !ok {
						t.Errorf("table %s: declared column %q not found in result row", table.Name, col)
					}
				}
			}
		})
	}
}

func TestIntegrationCoreTablesAreQueryable(t *testing.T) {
	c, err := NewClient(socketPath(t), startupTimeout, queryTimeout)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer c.Close()

	for _, table := range CoreTables {
		t.Run(table.Name, func(t *testing.T) {
			rows, _, err := c.Query(context.Background(), "SELECT COUNT(*) AS cnt FROM "+table.Name)
			if err != nil {
				t.Errorf("table %s: COUNT query failed: %v", table.Name, err)
				return
			}
			if len(rows) == 0 {
				t.Errorf("table %s: COUNT returned 0 rows", table.Name)
				return
			}
			t.Logf("table %s: COUNT = %s", table.Name, rows[0]["cnt"])
		})
	}
}

func TestIntegrationDeriveColumnsConsistency(t *testing.T) {
	for _, table := range CoreTables {
		t.Run(table.Name, func(t *testing.T) {
			cols := daemons.DeriveColumnsFromSchema("SELECT * FROM "+table.Name, CoreTables)
			expected := daemons.ExtractColumnNames(table.Columns)
			if len(cols) != len(expected) {
				t.Errorf("DeriveColumnsFromSchema returned %d columns, want %d", len(cols), len(expected))
				return
			}
			for i := range cols {
				if cols[i] != expected[i] {
					t.Errorf("column[%d] = %q, want %q", i, cols[i], expected[i])
				}
			}
		})
	}
}

func TestIntegrationTableHasDescription(t *testing.T) {
	for _, table := range CoreTables {
		if strings.TrimSpace(table.Description) == "" {
			t.Errorf("table %q has empty description", table.Name)
		}
	}
}

func TestIntegrationTableHasName(t *testing.T) {
	for _, table := range CoreTables {
		if strings.TrimSpace(table.Name) == "" {
			t.Errorf("table with index has empty name")
		}
	}
}
