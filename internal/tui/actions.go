package tui

import (
	"log/slog"

	"github.com/ajitm722/lazyos/internal/tui/views/sidebar"
	tea "github.com/charmbracelet/bubbletea"
)

// AppAction is the Command Pattern interface. Every state mutation is
// extracted into a struct that implements Apply, which receives the current
// AppModel and returns the updated model plus an optional tea.Cmd.
type AppAction interface {
	Apply(m AppModel) (AppModel, tea.Cmd)
}

// QuitAction terminates the Bubble Tea program via tea.Quit.
type QuitAction struct{}

func (a QuitAction) Apply(m AppModel) (AppModel, tea.Cmd) {
	return m, tea.Quit
}

// ToggleTableAction flips IsTableMode on the results view, switching the
// display between line-mode and table-mode rendering.
type ToggleTableAction struct{}

func (a ToggleTableAction) Apply(m AppModel) (AppModel, tea.Cmd) {
	m.layout.Results.IsTableMode = !m.layout.Results.IsTableMode
	slog.Debug("Toggled table mode", "isTableMode", m.layout.Results.IsTableMode)
	return m, nil
}

// FocusNextAction advances activeFocus forward through the three panes
// (list, input, view) and manages the text input's Focus/Blur state.
type FocusNextAction struct{}

func (a FocusNextAction) Apply(m AppModel) (AppModel, tea.Cmd) {
	oldFocus := m.panes.Current()
	m.panes = m.panes.Next()
	slog.Debug("Focused next", "from", oldFocus, "to", m.panes.Current())
	if m.panes.Current() == PaneQuery {
		m.layout.Querybar.Input.Focus()
	} else {
		m.layout.Querybar.Input.Blur()
	}
	return m, nil
}

// FocusPrevAction advances activeFocus backward through the three panes
// (list, input, view) and manages the text input's Focus/Blur state.
type FocusPrevAction struct{}

func (a FocusPrevAction) Apply(m AppModel) (AppModel, tea.Cmd) {
	oldFocus := m.panes.Current()
	m.panes = m.panes.Prev()
	slog.Debug("Focused prev", "from", oldFocus, "to", m.panes.Current())
	if m.panes.Current() == PaneQuery {
		m.layout.Querybar.Input.Focus()
	} else {
		m.layout.Querybar.Input.Blur()
	}
	return m, nil
}

// EnterAction handles the enter key contextually based on the active pane.
type EnterAction struct{}

func (a EnterAction) Apply(m AppModel) (AppModel, tea.Cmd) {
	switch m.panes.Current() {
	case PaneSidebar:
		if i, ok := m.layout.Sidebar.List.SelectedItem().(sidebar.TableItem); ok {
			slog.Debug("enter key pressed on table via EnterAction", "table", i.Schema.Name)
			return m, func() tea.Msg { return AutofillMsg{TableName: i.Schema.Name} }
		}
	case PaneQuery:
		v := m.layout.Querybar.Input.Value()
		if v != "" {
			slog.Debug("enter key pressed in querybar via EnterAction", "query", v)
			return m, func() tea.Msg { return RunQueryMsg{SQL: v} }
		}
	}
	return m, nil
}
