package cache

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/ajitm722/LazyOS/internal/daemons"
	"github.com/ajitm722/LazyOS/internal/daemons/mock"
	"github.com/ajitm722/LazyOS/internal/store/sqlite"
)

func TestExtractTableNames(t *testing.T) {
	tests := []struct {
		sql    string
		tables []string
	}{
		{"SELECT * FROM processes", []string{"processes"}},
		{"select * from aws_ec2_instance", []string{"aws_ec2_instance"}},
		{"SELECT * FROM processes;", []string{"processes"}},
		{"SELECT name FROM processes WHERE pid = 1234", []string{"processes"}},
		{"SELECT * FROM processes LIMIT 5", []string{"processes"}},
		{"  SELECT * FROM processes  ", []string{"processes"}},
		{"SELECT a, b FROM my_table ;", []string{"my_table"}},
		{"SELECT * FROM aws_ec2_instance WHERE instances_state = 'running'", []string{"aws_ec2_instance"}},
		{"SELECT * FROM t1, t2", []string{"t1", "t2"}},
		{"SELECT * FROM t1 JOIN t2 ON t1.id = t2.id", []string{"t1", "t2"}},
		{"SELECT * FROM t1 INNER JOIN t2 ON t1.id = t2.id", []string{"t1", "t2"}},
		{"SELECT * FROM t1 LEFT JOIN t2 ON t1.id = t2.id", []string{"t1", "t2"}},
		{"SELECT * from tbl", []string{"tbl"}},
		{"", nil},
		{"SELECT COUNT(*) FROM t", []string{"t"}},
		{"SELECT 1", nil},
	}

	for _, tt := range tests {
		got := extractTableNames(tt.sql)
		if !reflect.DeepEqual(got, tt.tables) {
			t.Errorf("extractTableNames(%q) = %v, want %v", tt.sql, got, tt.tables)
		}
	}
}

func TestCachedQueryer_Query(t *testing.T) {
	mq := &mock.MockQueryer{
		DefaultResult: []map[string]string{
			{"pid": "1", "name": "test"},
		},
	}
	cq := NewCachedQueryer(mq, nil)

	rows, cols, err := cq.Query(context.Background(), "SELECT * FROM test")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if cols == nil {
		t.Fatal("expected non-nil columns")
	}
}

func TestCachedQueryer_Query_NilStore(t *testing.T) {
	mq := &mock.MockQueryer{
		DefaultResult: []map[string]string{
			{"pid": "1", "name": "test"},
		},
	}
	cq := NewCachedQueryer(mq, nil)

	rows, cols, err := cq.Query(context.Background(), "SELECT * FROM test")
	if err != nil {
		t.Fatalf("Query with nil store failed: %v", err)
	}
	if len(rows) != 1 || rows[0]["pid"] != "1" {
		t.Fatal("expected result from upstream fallthrough")
	}
	if cols == nil {
		t.Fatal("expected non-nil columns")
	}
}

func TestCachedQueryer_QuerySource_NilStore(t *testing.T) {
	mq := &mock.MockQueryer{
		DefaultResult: []map[string]string{
			{"pid": "1", "name": "test"},
		},
	}
	cq := NewCachedQueryer(mq, nil)

	rows, cols, err := cq.QuerySource(context.Background(), "SELECT * FROM test")
	if err != nil {
		t.Fatalf("QuerySource with nil store failed: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if cols == nil {
		t.Fatal("expected non-nil columns")
	}
}

func TestCachedQueryer_QuerySource_Error(t *testing.T) {
	mq := &mock.MockQueryer{
		DefaultErr: daemons.ErrQueryFailed,
	}
	cq := NewCachedQueryer(mq, nil)

	_, _, err := cq.QuerySource(context.Background(), "SELECT * FROM t")
	if err == nil {
		t.Fatal("expected error from QuerySource")
	}
}

func TestCachedQueryer_WithSchema(t *testing.T) {
	mq := &mock.MockQueryer{
		Schema: []daemons.TableSchema{
			{Name: "test_table", Columns: "col1, col2"},
		},
	}

	cq := NewCachedQueryer(mq, nil)

	schema := cq.GetSchema()
	if len(schema) != 1 || schema[0].Name != "test_table" {
		t.Fatal("unexpected schema from CachedQueryer")
	}
}

func TestCachedQueryer_Close(t *testing.T) {
	mq := &mock.MockQueryer{
		DefaultResult: []map[string]string{
			{"pid": "1", "name": "test"},
		},
	}
	cq := NewCachedQueryer(mq, nil)
	if err := cq.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

// TestCachedQueryer_Query_WithStore verifies that Query lazy-loads an
// uncached table from upstream and persists it in the store.
func TestCachedQueryer_Query_WithStore(t *testing.T) {
	dir, _ := os.MkdirTemp("", "cache-test")
	defer os.RemoveAll(dir)
	st, err := sqlite.Open(filepath.Join(dir, "cache.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	upstreamRows := []map[string]string{
		{"col1": "a", "col2": "b"},
	}
	mq := &mock.MockQueryer{
		DefaultResult: upstreamRows,
	}
	cq := NewCachedQueryer(mq, st)

	rows, cols, err := cq.Query(context.Background(), "SELECT * FROM my_table")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(rows) != 1 || rows[0]["col1"] != "a" {
		t.Fatal("unexpected query result")
	}
	if len(cols) == 0 {
		t.Fatal("expected columns")
	}

	if !st.HasTable("my_table") {
		t.Fatal("expected my_table to be cached after query")
	}
}

// TestCachedQueryer_Query_AlreadyCached verifies that a second query against
// the same table reads from the store without calling upstream.
func TestCachedQueryer_Query_AlreadyCached(t *testing.T) {
	dir, _ := os.MkdirTemp("", "cache-test")
	defer os.RemoveAll(dir)
	st, err := sqlite.Open(filepath.Join(dir, "cache.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	firstResult := []map[string]string{{"col1": "first"}}
	mq := &mock.MockQueryer{
		Results: map[string][]map[string]string{
			"SELECT * FROM t": firstResult,
		},
	}
	cq := NewCachedQueryer(mq, st)

	_, _, err = cq.Query(context.Background(), "SELECT * FROM t")
	if err != nil {
		t.Fatalf("first query: %v", err)
	}

	mq.DefaultResult = []map[string]string{{"col1": "second"}}

	rows2, _, err := cq.Query(context.Background(), "SELECT * FROM t")
	if err != nil {
		t.Fatalf("second query: %v", err)
	}
	if rows2[0]["col1"] != "first" {
		t.Fatal("expected cached (first) result, not upstream (second)")
	}
}

// TestCachedQueryer_QuerySource_WithStore verifies that QuerySource always
// refreshes from upstream and overwrites the local cache.
func TestCachedQueryer_QuerySource_WithStore(t *testing.T) {
	dir, _ := os.MkdirTemp("", "cache-test")
	defer os.RemoveAll(dir)
	st, err := sqlite.Open(filepath.Join(dir, "cache.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	mq := &mock.MockQueryer{
		Results: map[string][]map[string]string{
			"SELECT * FROM t": {{"col1": "fresh"}},
		},
	}
	cq := NewCachedQueryer(mq, st)

	if err := st.SyncTable("t", []string{"col1"}, []map[string]string{{"col1": "stale"}}); err != nil {
		t.Fatalf("seed store: %v", err)
	}

	rows, _, err := cq.QuerySource(context.Background(), "SELECT * FROM t")
	if err != nil {
		t.Fatalf("QuerySource failed: %v", err)
	}
	if rows[0]["col1"] != "fresh" {
		t.Fatal("expected refreshed (fresh) result, not stale")
	}
}

// TestCachedQueryer_Query_UpstreamError verifies that Query propagates
// errors when lazy-loading a table from upstream fails.
func TestCachedQueryer_Query_UpstreamError(t *testing.T) {
	dir, _ := os.MkdirTemp("", "cache-test")
	defer os.RemoveAll(dir)
	st, err := sqlite.Open(filepath.Join(dir, "cache.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	mq := &mock.MockQueryer{
		DefaultErr: daemons.ErrQueryFailed,
	}
	cq := NewCachedQueryer(mq, st)

	_, _, err = cq.Query(context.Background(), "SELECT * FROM bad_table")
	if err == nil {
		t.Fatal("expected error for upstream failure")
	}
}

// TestCachedQueryer_QuerySource_UpstreamError verifies that QuerySource
// propagates errors when the upstream fetch fails.
func TestCachedQueryer_QuerySource_UpstreamError(t *testing.T) {
	dir, _ := os.MkdirTemp("", "cache-test")
	defer os.RemoveAll(dir)
	st, err := sqlite.Open(filepath.Join(dir, "cache.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	mq := &mock.MockQueryer{
		DefaultErr: daemons.ErrQueryFailed,
	}
	cq := NewCachedQueryer(mq, st)

	_, _, err = cq.QuerySource(context.Background(), "SELECT * FROM bad_table")
	if err == nil {
		t.Fatal("expected error for upstream failure in QuerySource")
	}
}
