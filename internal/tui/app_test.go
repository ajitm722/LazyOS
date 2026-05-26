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
	m := NewApp(map[string]daemons.Queryer{"mock": mq}, config.Keys{}).(AppModel)
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
	m.layout.Querybar.Input.Focus()
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
// It uses a table-driven pattern covering: osquery preferred over others,
// fallback to the first key when osquery is absent, and empty-map edge case.
func TestGetDefaultBackend(t *testing.T) {
	mock1 := &mock.MockQueryer{}
	mock2 := &mock.MockQueryer{}

	tests := []struct {
		name    string
		clients map[string]daemons.Queryer
		want    string
	}{
		{name: "prefers osquery", clients: map[string]daemons.Queryer{"other": mock1, "osquery": mock2}, want: "osquery"},
		{name: "fallback to first", clients: map[string]daemons.Queryer{"custom": mock1}, want: "custom"},
		{name: "empty clients", clients: map[string]daemons.Queryer{}, want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getDefaultBackend(tt.clients); got != tt.want {
				t.Errorf("getDefaultBackend() = %q, want %q", got, tt.want)
			}
		})
	}
}
