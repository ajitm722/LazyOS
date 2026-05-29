package tui

import (
	"testing"

	"github.com/ajitm722/LazyOS/internal/config"
	"github.com/ajitm722/LazyOS/internal/daemons"
	"github.com/ajitm722/LazyOS/internal/daemons/mock"
	tea "github.com/charmbracelet/bubbletea"
)

// newAppModel builds an AppModel wired to a single mock backend named "mock",
// with a default 100×50 terminal.
func newAppModel(mq *mock.MockQueryer) AppModel {
	return newAppModelSized(mq, 100, 50)
}

// newAppModelSized builds an AppModel with a custom terminal size.
func newAppModelSized(mq *mock.MockQueryer, width, height int) AppModel {
	m := NewApp(map[string]daemons.Queryer{"mock": mq}, []string{"mock"}, config.Keys{}).(AppModel)
	m2, _ := m.Update(tea.WindowSizeMsg{Width: width, Height: height})
	return m2.(AppModel)
}

// defaultAppModel is a shorthand for tests that only need a plain AppModel
// with a zero-value mock backend (no custom expectations).
func defaultAppModel() AppModel {
	return newAppModel(&mock.MockQueryer{})
}

// focusQuery sets the pane to PaneQuery, fills the input with sql, and focuses
// the text input. It returns the mutated model.
func focusQuery(m AppModel, sql string) AppModel {
	m.panes = m.panes.Set(PaneQuery)
	m.layout.Querybar.Input.SetValue(sql)
	m.layout.Querybar.Focus()
	return m
}

// updateApp is a thin wrapper around AppModel.Update that peels off the
// tea.Model interface assertion, saving repeated result.(AppModel) calls.
func updateApp(m AppModel, msg tea.Msg) (AppModel, tea.Cmd) {
	r, cmd := m.Update(msg)
	return r.(AppModel), cmd
}

// ---------------------------------------------------------------------------
// Core lifecycle tests.
// ---------------------------------------------------------------------------

// TestAppInit verifies that Init() returns a non-nil tea.Cmd, proving that
// the Layout batch-init fires properly.
func TestAppInit(t *testing.T) {
	m := defaultAppModel()
	cmd := m.Init()
	if cmd == nil {
		t.Error("expected non-nil cmd from AppModel.Init()")
	}
}

// TestQuitAction sends Ctrl+C through the Update loop and asserts that the
// returned command produces tea.QuitMsg when executed.
func TestQuitAction(t *testing.T) {
	m := defaultAppModel()
	_, cmd := updateApp(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("expected non-nil cmd for quit")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("expected tea.QuitMsg, got %T", msg)
	}
}

// TestGetDefaultBackend exercises the startup-backend-preference logic.
// It verifies that "osquery-kernel" is preferred when present, then falls
// back to the first entry in the ordered list.
func TestGetDefaultBackend(t *testing.T) {
	mock1 := &mock.MockQueryer{}
	mock2 := &mock.MockQueryer{}

	tests := []struct {
		name         string
		clients      map[string]daemons.Queryer
		backendOrder []string
		want         string
	}{
		{
			name:         "prefers kernel",
			clients:      map[string]daemons.Queryer{"aws": mock1, "kernel": mock2},
			backendOrder: []string{"aws", "kernel"},
			want:         "kernel",
		},
		{
			name:         "fallback to first in order",
			clients:      map[string]daemons.Queryer{"custom": mock1},
			backendOrder: []string{"custom"},
			want:         "custom",
		},
		{
			name:         "empty order",
			clients:      map[string]daemons.Queryer{},
			backendOrder: []string{},
			want:         "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getDefaultBackend(tt.clients, tt.backendOrder); got != tt.want {
				t.Errorf("getDefaultBackend() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestExecuteSourceAction verifies that pressing E in the query bar produces
// a RunSourceQueryMsg, and pressing it elsewhere produces no cmd.
func TestExecuteSourceAction(t *testing.T) {
	m := defaultAppModel()

	m = focusQuery(m, "SELECT 1")
	m.panes = m.panes.Set(PaneQuery)

	Ekey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'E'}}
	m2, cmd := updateApp(m, Ekey)
	if cmd == nil {
		t.Fatal("expected non-nil cmd for E when query bar focused")
	}
	msg := cmd()
	if _, ok := msg.(RunSourceQueryMsg); !ok {
		t.Errorf("expected RunSourceQueryMsg, got %T", msg)
	}
	_ = m2

	m3 := focusQuery(m, "SELECT 1")
	m3.panes = m3.panes.Set(PaneSidebar)
	_, cmd2 := updateApp(m3, Ekey)
	if cmd2 != nil {
		t.Error("expected nil cmd for E when sidebar focused")
	}
}

// TestNextBackendAction verifies that Shift+B cycles to the next backend and
// wraps around. Uses a two-backend config.
func TestNextBackendAction(t *testing.T) {
	mq1 := &mock.MockQueryer{
		Schema: []daemons.TableSchema{{Name: "table1"}},
	}
	mq2 := &mock.MockQueryer{
		Schema: []daemons.TableSchema{{Name: "table2"}},
	}
	clients := map[string]daemons.Queryer{"b1": mq1, "b2": mq2}
	am := NewApp(clients, []string{"b1", "b2"}, config.Keys{}).(AppModel)
	am2, _ := am.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	m := am2.(AppModel)

	if m.activeBackend != "b1" {
		t.Fatalf("expected initial backend b1, got %s", m.activeBackend)
	}

	Bkey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'B'}}
	m3, cmd := updateApp(m, Bkey)
	if cmd != nil {
		t.Fatal("expected nil cmd for backend cycle")
	}
	if m3.activeBackend != "b2" {
		t.Errorf("expected b2 after first B, got %s", m3.activeBackend)
	}

	m4, _ := updateApp(m3, Bkey)
	if m4.activeBackend != "b1" {
		t.Errorf("expected b1 after second B (wrap), got %s", m4.activeBackend)
	}
}

// TestRunSourceQueryMsgHandler verifies that sending a RunSourceQueryMsg
// through Update dispatches it to handleRunSourceQueryMsg and produces a
// non-nil cmd that eventually yields a QueryResultMsg or QueryErrorMsg.
func TestRunSourceQueryMsgHandler(t *testing.T) {
	mq := &mock.MockQueryer{
		DefaultResult: []map[string]string{
			{"result": "ok"},
		},
	}
	m := newAppModel(mq)

	msg := RunSourceQueryMsg{SQL: "SELECT 1"}
	m2, cmd := updateApp(m, msg)
	if cmd == nil {
		t.Fatal("expected non-nil cmd from RunSourceQueryMsg")
	}

	res := cmd()
	if _, ok := res.(QueryResultMsg); !ok {
		t.Errorf("expected QueryResultMsg, got %T", res)
	}
	_ = m2
}

// TestRunSourceQueryMsgError verifies that RunSourceQueryMsg handler produces
// QueryErrorMsg when the upstream query fails.
func TestRunSourceQueryMsgError(t *testing.T) {
	mq := &mock.MockQueryer{
		DefaultErr: daemons.ErrQueryFailed,
	}
	m := newAppModel(mq)

	msg := RunSourceQueryMsg{SQL: "SELECT 1"}
	m2, cmd := updateApp(m, msg)
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}

	res := cmd()
	if _, ok := res.(QueryErrorMsg); !ok {
		t.Errorf("expected QueryErrorMsg, got %T", res)
	}
	_ = m2
}

// TestGetDefaultBackendNoKernel verifies fallback when kernel is absent.
func TestGetDefaultBackendNoKernel(t *testing.T) {
	clients := map[string]daemons.Queryer{"custom": &mock.MockQueryer{}}
	got := getDefaultBackend(clients, []string{"custom"})
	if got != "custom" {
		t.Errorf("expected custom, got %s", got)
	}
}

// TestGetDefaultBackendMissingEntry verifies that entries with no client are
// skipped during the preference scan.
func TestGetDefaultBackendMissingEntry(t *testing.T) {
	clients := map[string]daemons.Queryer{"real": &mock.MockQueryer{}}
	got := getDefaultBackend(clients, []string{"missing", "real"})
	if got != "real" {
		t.Errorf("expected real, got %s", got)
	}
}

// TestHandleKeyMsgInsertModeEsc verifies that pressing Esc in INSERT mode
// returns to NORMAL mode and blurs the query bar.
func TestHandleKeyMsgInsertModeEsc(t *testing.T) {
	m := defaultAppModel()
	m.mode = InsertMode
	m.panes = m.panes.Set(PaneQuery)
	m.layout.Querybar.Focus()

	m2, cmd := updateApp(m, tea.KeyMsg{Type: tea.KeyEsc})
	if cmd != nil {
		t.Error("expected nil cmd from Esc in INSERT mode")
	}
	if m2.mode != NormalMode {
		t.Error("expected NormalMode after Esc")
	}
}

// TestHandleKeyMsgInsertModeCtrlL verifies that Ctrl+L works in INSERT mode.
func TestHandleKeyMsgInsertModeCtrlL(t *testing.T) {
	m := defaultAppModel()
	m.mode = InsertMode
	initialPane := m.panes.Current()

	m2, cmd := updateApp(m, tea.KeyMsg{Type: tea.KeyCtrlL})
	if cmd != nil {
		t.Fatal("expected nil cmd from Ctrl+L in INSERT mode")
	}
	if m2.panes.Current() == initialPane {
		t.Error("expected pane focus to change after Ctrl+L in INSERT mode")
	}
}

// TestHandleKeyMsgInsertModeRouteVerifies that non-control keys in INSERT mode
// route to the focused widget without triggering global actions.
func TestHandleKeyMsgInsertModeRoute(t *testing.T) {
	m := defaultAppModel()
	m.mode = InsertMode

	m2, _ := updateApp(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	if m2.mode != InsertMode {
		t.Error("expected INSERT mode to persist after key in INSERT mode")
	}
}

// TestHandleKeyMsgNormalModeI enters INSERT mode by pressing i when query
// bar is focused, verifying the mode transition.
func TestHandleKeyMsgNormalModeI(t *testing.T) {
	m := defaultAppModel()
	m.panes = m.panes.Set(PaneQuery)
	m.mode = NormalMode

	m2, cmd := updateApp(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	if cmd != nil {
		t.Error("expected nil cmd from i in query pane")
	}
	if m2.mode != InsertMode {
		t.Error("expected InsertMode after pressing i")
	}
}

// TestAutofillActionWrongPane verifies that AutofillAction does nothing
// when the sidebar is not focused.
func TestAutofillActionWrongPane(t *testing.T) {
	m := defaultAppModel()
	m.panes = m.panes.Set(PaneQuery)
	akey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
	_, cmd := updateApp(m, akey)
	if cmd != nil {
		t.Error("expected nil cmd for autofill from query pane")
	}
}
