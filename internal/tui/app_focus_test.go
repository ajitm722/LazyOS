package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestFocusNavigation verifies the Ctrl+L/Ctrl+H pane-cycling logic.
//
// The test initialises a bare AppModel, then for each table row it sets the
// starting pane, sends the specified key, and asserts:
//   - The active pane ID matches wantPane.
//   - The query-bar text-input Focused state matches wantFocused
//     (true only when PaneQuery is active).
//
// Ctrl+L moves forward through [sidebar → query → results → sidebar].
// Ctrl+H moves backward through [sidebar → results → query → sidebar].
func TestFocusNavigation(t *testing.T) {
	m := defaultAppModel()

	tests := []struct {
		name        string
		startPane   PaneID
		key         tea.KeyMsg
		wantPane    PaneID
		wantFocused bool
	}{
		{name: "ctrl+l sidebar to query", startPane: PaneSidebar, key: tea.KeyMsg{Type: tea.KeyCtrlL}, wantPane: PaneQuery, wantFocused: true},
		{name: "ctrl+l query to results", startPane: PaneQuery, key: tea.KeyMsg{Type: tea.KeyCtrlL}, wantPane: PaneResults, wantFocused: false},
		{name: "ctrl+l results wrap to sidebar", startPane: PaneResults, key: tea.KeyMsg{Type: tea.KeyCtrlL}, wantPane: PaneSidebar, wantFocused: false},
		{name: "ctrl+h sidebar wrap to results", startPane: PaneSidebar, key: tea.KeyMsg{Type: tea.KeyCtrlH}, wantPane: PaneResults, wantFocused: false},
		{name: "ctrl+h results to query", startPane: PaneResults, key: tea.KeyMsg{Type: tea.KeyCtrlH}, wantPane: PaneQuery, wantFocused: true},
		{name: "ctrl+h query to sidebar", startPane: PaneQuery, key: tea.KeyMsg{Type: tea.KeyCtrlH}, wantPane: PaneSidebar, wantFocused: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m.panes = m.panes.Set(tt.startPane)
			m2, _ := updateApp(m, tt.key)

			if m2.panes.Current() != tt.wantPane {
				t.Errorf("expected pane %v, got %v", tt.wantPane, m2.panes.Current())
			}
			if m2.layout.Querybar.Input.Focused() != tt.wantFocused {
				t.Errorf("expected Focused=%v, got %v", tt.wantFocused, m2.layout.Querybar.Input.Focused())
			}
		})
	}
}

// TestUnhandledMsgFallthrough sends a message type that Update does not
// recognise (tea.MouseMsg) from each pane to exercise the Update fallthrough
// to routeToFocused and all three routeToFocused switch branches.
func TestUnhandledMsgFallthrough(t *testing.T) {
	m := defaultAppModel()

	for _, pane := range []PaneID{PaneSidebar, PaneQuery, PaneResults} {
		m.panes = m.panes.Set(pane)
		m2, cmd := updateApp(m, tea.MouseMsg{})
		if cmd != nil {
			t.Errorf("expected nil cmd for MouseMsg in pane %v, got non-nil", pane)
		}
		if m2.panes.Current() != pane {
			t.Errorf("expected pane %v unchanged after fallthrough, got %v", pane, m2.panes.Current())
		}
	}
}

// TestToggleTable sends the toggle-table binding ('t') twice and checks that
// IsTableMode flips from false → true → false.
func TestToggleTable(t *testing.T) {
	m := defaultAppModel()

	if m.layout.Results.IsTableMode {
		t.Error("expected IsTableMode=false initially")
	}

	m2, _ := updateApp(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	if !m2.layout.Results.IsTableMode {
		t.Error("expected IsTableMode=true after first toggle")
	}

	m3, _ := updateApp(m2, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	if m3.layout.Results.IsTableMode {
		t.Error("expected IsTableMode=false after second toggle")
	}
}
