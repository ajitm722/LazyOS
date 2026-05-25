package tui

import (
	"errors"
	"strings"
	"testing"
	"time"

	"testing/synctest"

	"github.com/ajitm722/LazyOS/internal/daemons"
	"github.com/ajitm722/LazyOS/internal/daemons/mock"
	tea "github.com/charmbracelet/bubbletea"
)

// TestQueryColumnsFromResponse verifies that column headers come from the
// response data (first row's map keys) and NOT from the schema. This ensures
// computed expressions like `size / 1024 AS mb` appear in the output and
// extra schema columns do not leak in.
func TestQueryColumnsFromResponse(t *testing.T) {
	mock := &mock.MockQueryer{
		DefaultResult: []map[string]string{
			{"state": "running", "name": "init", "pid": "1"},
		},
		Schema: mock.MockTables,
	}
	m := newAppModel(mock)
	m = focusQuery(m, "SELECT * FROM processes")

	_, cmd := updateApp(m, tea.KeyMsg{Type: tea.KeyCtrlE})
	runMsg := cmd()
	_, cmd = updateApp(m, runMsg)
	msg := cmd()
	qrMsg, ok := msg.(QueryResultMsg)
	if !ok {
		t.Fatalf("expected QueryResultMsg, got %T", msg)
	}

	// Response has 3 keys: pid, name, state. Schema additionally has path,
	// cmdline, cwd, root, uid, gid, on_disk, resident_size, total_size.
	// Extra schema columns must NOT appear.
	if len(qrMsg.Columns) != 3 {
		t.Errorf("expected 3 columns from response data, got %d: %v", len(qrMsg.Columns), qrMsg.Columns)
	}

	colSet := make(map[string]bool)
	for _, c := range qrMsg.Columns {
		colSet[c] = true
	}
	for _, want := range []string{"pid", "name", "state"} {
		if !colSet[want] {
			t.Errorf("expected response column %q, not found in %v", want, qrMsg.Columns)
		}
	}
	if colSet["path"] || colSet["cmdline"] {
		t.Error("schema-only columns leaked into result")
	}
}

// TestQueryDispatchStandard exercises the full query pipeline:
//
//  1. Focus the query pane and type a SQL string.
//  2. Send Enter — the EnterAction should return a cmd that produces
//     RunQueryMsg carrying the SQL.
//  3. Feed RunQueryMsg through Update — handleRunQueryMsg returns a cmd
//     that calls the backend (the mock returns injected rows).
//  4. Execute the cmd — it should produce QueryResultMsg with 2 rows.
//  5. Feed QueryResultMsg through Update — handleQueryResultMsg shifts
//     focus to PaneResults and populates the viewport with the formatted
//     row data.
//
// Assertions cover the message type at each step, the row count, the final
// pane, and the viewport content containing expected formatted output.
func TestQueryDispatchStandard(t *testing.T) {
	mock := &mock.MockQueryer{
		Results: map[string][]map[string]string{
			"SELECT * FROM test": {
				{"pid": "1", "name": "init"},
				{"pid": "2", "name": "kthreadd"},
			},
		},
	}
	m := newAppModel(mock)
	m = focusQuery(m, "SELECT * FROM test")

	m2, cmd := updateApp(m, tea.KeyMsg{Type: tea.KeyCtrlE})
	if cmd == nil {
		t.Fatal("expected non-nil cmd after Enter in query pane")
	}

	runMsg := cmd()
	if _, ok := runMsg.(RunQueryMsg); !ok {
		t.Fatalf("expected RunQueryMsg, got %T", runMsg)
	}

	m3, cmd := updateApp(m2, runMsg)
	if cmd == nil {
		t.Fatal("expected non-nil cmd after RunQueryMsg")
	}

	queryResult := cmd()
	qrMsg, ok := queryResult.(QueryResultMsg)
	if !ok {
		t.Fatalf("expected QueryResultMsg, got %T", queryResult)
	}
	if len(qrMsg.Rows) != 2 {
		t.Errorf("expected 2 rows, got %d", len(qrMsg.Rows))
	}

	m4, _ := updateApp(m3, qrMsg)

	if m4.panes.Current() != PaneResults {
		t.Errorf("expected PaneResults after query result, got %v", m4.panes.Current())
	}
	if !strings.Contains(m4.layout.Results.View.View(), "--- Row 1 ---") {
		t.Errorf("expected row data in viewport, got:\n%s", m4.layout.Results.View.View())
	}
}

// TestQueryTimeout uses testing/synctest to verify that the mock's internal
// timeout is enforced when simulating a slow query.
//
// The mock has InternalTimeout (3s) shorter than SlowDuration (5s).
// Inside the synctest bubble the mock wraps ctx with context.WithTimeout,
// then selects on the derived context's Done() vs time.After(5s).
// The 3s deadline fires first and the mock returns ErrQueryTimeout,
// proving the backend-level timeout is enforced even when the TUI passes
// a plain context — exactly as the real osquery Client behaves.
//
// Real elapsed time for this test is near-zero; both timers are entirely
// virtual.
func TestQueryTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		mock := &mock.MockQueryer{
			SimulateSlowQuery: true,
			SlowDuration:      5 * time.Second,
			InternalTimeout:   3 * time.Second,
		}
		m := newAppModel(mock)
		m = focusQuery(m, "SELECT * FROM test")

		m2, cmd := updateApp(m, tea.KeyMsg{Type: tea.KeyCtrlE})
		if cmd == nil {
			t.Fatal("expected non-nil cmd after Enter in query pane")
		}
		runMsg := cmd()

		m3, cmd := updateApp(m2, runMsg)
		if cmd == nil {
			t.Fatal("expected non-nil cmd after RunQueryMsg")
		}

		msg := cmd()

		qeMsg, ok := msg.(QueryErrorMsg)
		if !ok {
			t.Fatalf("expected QueryErrorMsg, got %T", msg)
		}
		if !errors.Is(qeMsg.Err, daemons.ErrQueryTimeout) {
			t.Errorf("expected ErrQueryTimeout, got %v", qeMsg.Err)
		}

		m4, _ := updateApp(m3, qeMsg)
		if !strings.Contains(m4.layout.Results.View.View(), "query timed out") {
			t.Errorf("expected timeout message, got:\n%s", m4.layout.Results.View.View())
		}
	})
}

// TestQueryErrors consolidates the error-display tests into a table-driven
// test covering both sentinel errors recognised by handleQueryErrorMsg and
// generic errors that fall through to raw display.
//
// For each case:
//   - mock.DefaultErr is set to the test's error value.
//   - The query pipeline runs and produces QueryErrorMsg.
//   - If wantErr is non-nil, we verify errors.Is matches the sentinel.
//   - The viewport content is checked for the expected wantView substring.
func TestQueryErrors(t *testing.T) {
	tests := []struct {
		name     string
		mockErr  error
		wantErr  error
		wantView string
	}{
		{
			name:     "query failed sentinel",
			mockErr:  daemons.ErrQueryFailed,
			wantErr:  daemons.ErrQueryFailed,
			wantView: "query failed",
		},
		{
			name:     "generic error fallthrough",
			mockErr:  errors.New("unexpected database failure"),
			wantErr:  nil,
			wantView: "unexpected database failure",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mock.MockQueryer{DefaultErr: tt.mockErr}
			m := newAppModel(mock)
			m = focusQuery(m, "SELECT 1")

			_, cmd := updateApp(m, tea.KeyMsg{Type: tea.KeyCtrlE})
			runMsg := cmd()

			_, cmd = updateApp(m, runMsg)
			msg := cmd()

			qeMsg, ok := msg.(QueryErrorMsg)
			if !ok {
				t.Fatalf("expected QueryErrorMsg, got %T", msg)
			}
			if tt.wantErr != nil && !errors.Is(qeMsg.Err, tt.wantErr) {
				t.Errorf("expected error %v, got %v", tt.wantErr, qeMsg.Err)
			}

			m2, _ := updateApp(m, qeMsg)
			view := m2.layout.Results.View.View()
			if !strings.Contains(view, tt.wantView) {
				t.Errorf("expected view to contain %q, got:\n%s", tt.wantView, view)
			}
		})
	}
}

// TestEmptyQueryEnter verifies that pressing Enter in the query pane when
// the input is empty produces nil cmd — the EnterAction falls through
// without emitting RunQueryMsg.
func TestEmptyQueryEnter(t *testing.T) {
	m := defaultAppModel()
	m.panes = m.panes.Set(PaneQuery)

	_, cmd := updateApp(m, tea.KeyMsg{Type: tea.KeyCtrlE})
	if cmd != nil {
		t.Error("expected nil cmd when query input is empty")
	}
}

// TestAutofillTrigger verifies that Enter on a sidebar item (which has a
// schema backing it) produces an AutofillMsg.
//
// The mock provides a single table schema so the sidebar has a selectable
// item. Pressing Enter in the sidebar triggers EnterAction, which type-
// asserts the selected item to sidebar.TableItem and returns a cmd that
// yields AutofillMsg with the table name.
func TestAutofillTrigger(t *testing.T) {
	mock := &mock.MockQueryer{
		Schema: []daemons.TableSchema{
			{Name: "processes", Description: "Running processes", Columns: "pid, name, state"},
		},
	}
	m := newAppModelSized(mock, 100, 50)

	if m.panes.Current() != PaneSidebar {
		t.Fatalf("expected PaneSidebar, got %v", m.panes.Current())
	}

	_, cmd := updateApp(m, tea.KeyMsg{Type: tea.KeyCtrlE})
	if cmd == nil {
		t.Fatal("expected non-nil cmd for Enter on sidebar item")
	}

	msg := cmd()
	if _, ok := msg.(AutofillMsg); !ok {
		t.Fatalf("expected AutofillMsg, got %T", msg)
	}
}

// TestAutofillHandler feeds an AutofillMsg directly through Update and
// verifies that handleAutofillMsg populates the query bar with a SELECT
// query using schema column names, moves focus to PaneQuery, and focuses
// the text input. When the backend has no schema it falls back to SELECT *.
func TestAutofillHandler(t *testing.T) {
	t.Run("with schema", func(t *testing.T) {
		mock := &mock.MockQueryer{
			Schema: []daemons.TableSchema{
				{Name: "processes", Description: "test", Columns: "pid, name, path"},
			},
		}
		m := newAppModel(mock)

		m2, cmd := updateApp(m, AutofillMsg{TableName: "processes"})
		if cmd != nil {
			t.Error("expected nil cmd from handleAutofillMsg")
		}

		if m2.panes.Current() != PaneQuery {
			t.Errorf("expected PaneQuery after autofill, got %v", m2.panes.Current())
		}
		if !m2.layout.Querybar.Input.Focused() {
			t.Error("expected query input focused after autofill")
		}
		want := "SELECT pid, name, path FROM processes LIMIT 10;"
		if got := m2.layout.Querybar.Input.Value(); got != want {
			t.Errorf("expected query %q, got %q", want, got)
		}
	})

	t.Run("fallback no schema", func(t *testing.T) {
		m := defaultAppModel()

		m2, cmd := updateApp(m, AutofillMsg{TableName: "processes"})
		if cmd != nil {
			t.Error("expected nil cmd from handleAutofillMsg")
		}

		if m2.panes.Current() != PaneQuery {
			t.Errorf("expected PaneQuery after autofill, got %v", m2.panes.Current())
		}
		if !m2.layout.Querybar.Input.Focused() {
			t.Error("expected query input focused after autofill")
		}
		want := "SELECT * FROM processes LIMIT 10;"
		if got := m2.layout.Querybar.Input.Value(); got != want {
			t.Errorf("expected query %q, got %q", want, got)
		}
	})
}

// TestQueryZeroRows verifies that a query returning no results shows
// "0 rows returned." in line mode while still displaying column headers
// in table mode, when the backend provides column metadata for empty results.
func TestQueryZeroRows(t *testing.T) {
	mock := &mock.MockQueryer{
		DefaultResult: []map[string]string{},
		Schema:        mock.MockTables,
	}
	m := newAppModel(mock)
	m = focusQuery(m, "SELECT * FROM empty")

	// Run through the full query pipeline.
	_, cmd := updateApp(m, tea.KeyMsg{Type: tea.KeyCtrlE})
	runMsg := cmd()
	_, cmd = updateApp(m, runMsg)
	msg := cmd()
	qrMsg, ok := msg.(QueryResultMsg)
	if !ok {
		t.Fatalf("expected QueryResultMsg, got %T", msg)
	}
	if len(qrMsg.Rows) != 0 {
		t.Fatalf("expected 0 rows, got %d", len(qrMsg.Rows))
	}

	m2, _ := updateApp(m, qrMsg)

	// Line mode should show the empty message.
	lineView := m2.layout.Results.View.View()
	if !strings.Contains(lineView, "0 rows returned.") {
		t.Errorf("expected '0 rows returned.', got:\n%s", lineView)
	}

	// Switch to table mode — column headers should be visible.
	m2.layout.Results.IsTableMode = true
	tableView := m2.layout.Results.ViewStr()
	if tableView == "" {
		t.Error("expected non-empty table view even with 0 rows")
	}
	if !strings.Contains(tableView, "pid") || !strings.Contains(tableView, "name") {
		t.Errorf("table mode should show column headers, got:\n%s", tableView)
	}
	// No data rows should exist.
	if strings.Contains(tableView, "Row 1") {
		t.Error("table mode should not show data rows for empty result")
	}
}

// TestQueryZeroRowsDirect verifies the same behavior when feeding
// QueryResultMsg directly with explicit Columns (bypassing the mock).
func TestQueryZeroRowsDirect(t *testing.T) {
	m := defaultAppModel()

	qrMsg := QueryResultMsg{Rows: []map[string]string{}, Columns: []string{"pid", "name", "state"}}
	m2, _ := updateApp(m, qrMsg)

	if !strings.Contains(m2.layout.Results.View.View(), "0 rows returned.") {
		t.Error("expected '0 rows returned.'")
	}
	m2.layout.Results.IsTableMode = true
	tableView := m2.layout.Results.ViewStr()
	if !strings.Contains(tableView, "pid") {
		t.Error("table mode should show column headers for direct feed")
	}
}

// TestQueryZeroRowsNoColumns verifies that a query returning no results
// without column metadata still works (graceful fallback).
func TestQueryZeroRowsNoColumns(t *testing.T) {
	m := defaultAppModel()

	qrMsg := QueryResultMsg{Rows: []map[string]string{}}
	m2, _ := updateApp(m, qrMsg)

	if !strings.Contains(m2.layout.Results.View.View(), "0 rows returned.") {
		t.Error("expected '0 rows returned.' with no columns")
	}
	m2.layout.Results.IsTableMode = true
	if m2.layout.Results.ViewStr() == "" {
		t.Error("expected non-empty table view even without columns")
	}
}

// TestQueryZeroRowsFallbackNilColumns verifies that when the backend returns
// 0 rows with nil columns (no schema available), the TUI still handles it
// gracefully without panicking.
func TestQueryZeroRowsFallbackNilColumns(t *testing.T) {
	mock := &mock.MockQueryer{
		DefaultResult: []map[string]string{},
	}
	m := newAppModel(mock)
	m = focusQuery(m, "SELECT * FROM unknown")

	_, cmd := updateApp(m, tea.KeyMsg{Type: tea.KeyCtrlE})
	runMsg := cmd()
	_, cmd = updateApp(m, runMsg)
	msg := cmd()
	qrMsg, ok := msg.(QueryResultMsg)
	if !ok {
		t.Fatalf("expected QueryResultMsg, got %T", msg)
	}
	// Columns should be nil because neither mock schema nor rows provided them,
	// then the TUI fallback also fails (unknown table), so remain nil.
	if qrMsg.Columns != nil {
		t.Logf("columns=%v (nil is also fine)", qrMsg.Columns)
	}
	m2, _ := updateApp(m, qrMsg)
	if !strings.Contains(m2.layout.Results.View.View(), "0 rows returned.") {
		t.Error("expected '0 rows returned.' in line mode")
	}
	m2.layout.Results.IsTableMode = true
	if m2.layout.Results.ViewStr() == "" {
		t.Error("expected non-empty table view even with nil columns")
	}
}

// TestQueryZeroRowsFromSchema verifies that when the backend returns 0 rows
// without column metadata but has a schema, the TUI derives columns from the
// schema table definition.
func TestQueryZeroRowsFromSchema(t *testing.T) {
	mock := &mock.MockQueryer{
		DefaultResult: []map[string]string{},
		// No DefaultColumns — force schema fallback.
		Schema: []daemons.TableSchema{
			{Name: "processes", Description: "test", Columns: "pid, name, state"},
		},
	}
	m := newAppModel(mock)
	m = focusQuery(m, "SELECT * FROM processes")

	_, cmd := updateApp(m, tea.KeyMsg{Type: tea.KeyCtrlE})
	runMsg := cmd()
	_, cmd = updateApp(m, runMsg)
	msg := cmd()
	qrMsg, ok := msg.(QueryResultMsg)
	if !ok {
		t.Fatalf("expected QueryResultMsg, got %T", msg)
	}
	if len(qrMsg.Columns) == 0 {
		t.Fatal("expected columns derived from schema")
	}

	m2, _ := updateApp(m, qrMsg)
	m2.layout.Results.IsTableMode = true
	tableView := m2.layout.Results.ViewStr()
	if !strings.Contains(tableView, "pid") || !strings.Contains(tableView, "name") {
		t.Errorf("table mode should show column headers from schema, got:\n%s", tableView)
	}

	// Verify that pid appears before name (schema order), not reversed.
	pidIdx := strings.Index(tableView, "pid")
	nameIdx := strings.Index(tableView, "name")
	if pidIdx < 0 || nameIdx < 0 || pidIdx > nameIdx {
		t.Errorf("expected 'pid' before 'name' in table headers (schema order), got:\n%s", tableView)
	}
}
