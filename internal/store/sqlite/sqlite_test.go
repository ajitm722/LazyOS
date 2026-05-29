//go:build integration

package sqlite

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenAndClose(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	st, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	if err := st.Health(context.Background()); err != nil {
		t.Fatalf("Health check failed: %v", err)
	}

	if err := st.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Fatal("database file was not created")
	}
}

func TestSyncTableAndQuery(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	st, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer st.Close()

	columns := []string{"id", "name", "value"}
	rows := []map[string]string{
		{"id": "1", "name": "alpha", "value": "100"},
		{"id": "2", "name": "beta", "value": "200"},
	}

	if err := st.SyncTable("test_table", columns, rows); err != nil {
		t.Fatalf("SyncTable failed: %v", err)
	}

	if !st.HasTable("test_table") {
		t.Fatal("HasTable returned false after SyncTable")
	}

	if st.HasTable("nonexistent") {
		t.Fatal("HasTable returned true for nonexistent table")
	}

	gotRows, gotCols, err := st.Query(context.Background(), "SELECT * FROM test_table")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(gotRows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(gotRows))
	}

	if len(gotCols) != 3 {
		t.Fatalf("expected 3 columns, got %d", len(gotCols))
	}

	if gotRows[0]["name"] != "alpha" && gotRows[1]["name"] != "alpha" {
		t.Fatal("expected row with name=alpha")
	}
}

func TestSyncTableReplaces(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	st, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer st.Close()

	columns1 := []string{"col"}
	rows1 := []map[string]string{{"col": "old"}}
	if err := st.SyncTable("t", columns1, rows1); err != nil {
		t.Fatalf("first SyncTable failed: %v", err)
	}

	columns2 := []string{"col"}
	rows2 := []map[string]string{{"col": "new"}, {"col": "also_new"}}
	if err := st.SyncTable("t", columns2, rows2); err != nil {
		t.Fatalf("second SyncTable failed: %v", err)
	}

	gotRows, _, err := st.Query(context.Background(), "SELECT * FROM t")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(gotRows) != 2 {
		t.Fatalf("expected 2 rows after replace, got %d", len(gotRows))
	}
}

func TestQueryEmptyResult(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	st, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer st.Close()

	if err := st.SyncTable("empty", []string{"col"}, nil); err != nil {
		t.Fatalf("SyncTable with no rows failed: %v", err)
	}

	gotRows, gotCols, err := st.Query(context.Background(), "SELECT * FROM empty")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(gotRows) != 0 {
		t.Fatalf("expected 0 rows, got %d", len(gotRows))
	}

	if len(gotCols) != 1 {
		t.Fatalf("expected 1 column, got %d", len(gotCols))
	}
}

func TestQueryWithFilterAndLimit(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	st, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer st.Close()

	columns := []string{"pid", "name", "state"}
	rows := []map[string]string{
		{"pid": "1", "name": "nginx", "state": "S"},
		{"pid": "2", "name": "sshd", "state": "S"},
		{"pid": "3", "name": "bash", "state": "R"},
	}

	if err := st.SyncTable("processes", columns, rows); err != nil {
		t.Fatalf("SyncTable failed: %v", err)
	}

	gotRows, _, err := st.Query(context.Background(), "SELECT name FROM processes WHERE state = 'R' LIMIT 5")
	if err != nil {
		t.Fatalf("Query with filter failed: %v", err)
	}

	if len(gotRows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(gotRows))
	}
	if gotRows[0]["name"] != "bash" {
		t.Fatalf("expected name=bash, got %s", gotRows[0]["name"])
	}
}

func TestSyncTableSpecialChars(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	st, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer st.Close()

	columns := []string{"key-name", "value field"}
	rows := []map[string]string{
		{"key-name": "k1", "value field": "v1"},
	}

	if err := st.SyncTable("weird-table", columns, rows); err != nil {
		t.Fatalf("SyncTable with special chars failed: %v", err)
	}

	gotRows, _, err := st.Query(context.Background(), `SELECT "key-name" FROM "weird-table"`)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(gotRows) != 1 || gotRows[0]["key-name"] != "k1" {
		t.Fatal("unexpected result for special char columns")
	}
}

// TestQueryInvalidSQL verifies that Query returns an error for invalid SQL.
func TestQueryInvalidSQL(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	st, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer st.Close()

	if err := st.SyncTable("t", []string{"col"}, []map[string]string{{"col": "v"}}); err != nil {
		t.Fatalf("SyncTable failed: %v", err)
	}

	_, _, err = st.Query(context.Background(), "SELECT invalid SQL !!")
	if err == nil {
		t.Fatal("expected error for invalid SQL")
	}
}

// TestHasTableClosedStore verifies that HasTable handles a closed store gracefully.
func TestHasTableClosedStore(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	st, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	st.Close()

	if st.HasTable("anything") {
		t.Fatal("expected false for closed store")
	}
}

// TestSyncTableErrorPath verifies SyncTable fails gracefully on a closed store.
func TestSyncTableErrorPath(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	st, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	st.Close()

	err = st.SyncTable("t", []string{"c"}, []map[string]string{{"c": "v"}})
	if err == nil {
		t.Fatal("expected error from SyncTable on closed store")
	}
}

// TestHealthClosedStore verifies Health check on a closed store.
func TestHealthClosedStore(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	st, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	st.Close()

	if err := st.Health(context.Background()); err == nil {
		t.Fatal("expected error from Health on closed store")
	}
}
