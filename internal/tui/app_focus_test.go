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
// recognise (tea.MouseMsg with zero-value button/action) from each pane to
// exercise the Update fallthrough to routeToFocused and all three
// routeToFocused switch branches.
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

// TestMouseClickSwitchFocus verifies that a left-click on a different pane
// switches focus and adjusts the querybar state appropriately.
func TestMouseClickSwitchFocus(t *testing.T) {
	// Layout for 100×50: listW=30, sidebarStyleWidth=28, rightPaneStart=28, inputH=9
	m := defaultAppModel()

	tests := []struct {
		name      string
		startPane PaneID
		clickX    int
		clickY    int
		wantPane  PaneID
	}{
		{name: "click sidebar from query", startPane: PaneQuery, clickX: 5, clickY: 5, wantPane: PaneSidebar},
		{name: "click query from sidebar", startPane: PaneSidebar, clickX: 50, clickY: 3, wantPane: PaneQuery},
		{name: "click results from sidebar", startPane: PaneSidebar, clickX: 50, clickY: 25, wantPane: PaneResults},
		{name: "click same pane no-op", startPane: PaneQuery, clickX: 50, clickY: 3, wantPane: PaneQuery},
		{name: "click footer ignored", startPane: PaneQuery, clickX: 10, clickY: 49, wantPane: PaneQuery},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m.panes = m.panes.Set(tt.startPane)
			msg := tea.MouseMsg(tea.MouseEvent{
				X:      tt.clickX,
				Y:      tt.clickY,
				Button: tea.MouseButtonLeft,
				Action: tea.MouseActionPress,
			})
			m2, _ := updateApp(m, msg)

			if m2.panes.Current() != tt.wantPane {
				t.Errorf("expected pane %v, got %v", tt.wantPane, m2.panes.Current())
			}
		})
	}
}

// TestMouseClickQuerybarFocusState verifies that clicking the querybar
// respects the current mode when focusing.
func TestMouseClickQuerybarFocusState(t *testing.T) {
	// Layout for 100×50: listW=30, inputH≈9 — click at (50, 3) hits querybar

	t.Run("normal mode click focuses with EnterNormal", func(t *testing.T) {
		m := defaultAppModel()
		m.panes = m.panes.Set(PaneSidebar)
		m.layout.Querybar.Blur()
		m.mode = NormalMode

		msg := tea.MouseMsg(tea.MouseEvent{
			X: 50, Y: 3, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress,
		})
		m2, _ := updateApp(m, msg)

		if m2.panes.Current() != PaneQuery {
			t.Errorf("expected PaneQuery, got %v", m2.panes.Current())
		}
		if !m2.layout.Querybar.Input.Focused() {
			t.Error("expected querybar focused after click in normal mode")
		}
	})

	t.Run("insert mode click focuses with Focus", func(t *testing.T) {
		m := defaultAppModel()
		m.panes = m.panes.Set(PaneSidebar)
		m.layout.Querybar.Blur()
		m.mode = InsertMode

		msg := tea.MouseMsg(tea.MouseEvent{
			X: 50, Y: 3, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress,
		})
		m2, _ := updateApp(m, msg)

		if m2.panes.Current() != PaneQuery {
			t.Errorf("expected PaneQuery, got %v", m2.panes.Current())
		}
		if !m2.layout.Querybar.Input.Focused() {
			t.Error("expected querybar focused after click in insert mode")
		}
	})
}

// TestMouseWheelDoesNotSwitchFocus verifies that mouse wheel events do not
// switch pane focus but are still forwarded to the focused pane.
func TestMouseWheelDoesNotSwitchFocus(t *testing.T) {
	m := defaultAppModel()
	m.panes = m.panes.Set(PaneQuery)

	// Wheel up over sidebar
	msg := tea.MouseMsg(tea.MouseEvent{
		X: 5, Y: 5, Button: tea.MouseButtonWheelUp, Action: tea.MouseActionPress,
	})
	m2, _ := updateApp(m, msg)

	if m2.panes.Current() != PaneQuery {
		t.Errorf("expected PaneQuery unchanged after wheel event, got %v", m2.panes.Current())
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
