package tui

import (
	"log/slog"

	"github.com/charmbracelet/bubbles/list"

	"github.com/ajitm722/LazyOS/internal/tui/views/sidebar"
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
	if m.panes.Current() != PaneQuery {
		m.layout.Querybar.Blur()
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
	if m.panes.Current() != PaneQuery {
		m.layout.Querybar.Blur()
	}
	return m, nil
}

// NextBackendAction cycles to the next backend in the ordered list, rebuilding
// the sidebar with the new backend's schema.
type NextBackendAction struct{}

func (a NextBackendAction) Apply(m AppModel) (AppModel, tea.Cmd) {
	if len(m.backendOrder) <= 1 {
		return m, nil
	}
	m.activeIndex = (m.activeIndex + 1) % len(m.backendOrder)
	m.activeBackend = m.backendOrder[m.activeIndex]

	m.layout.Sidebar = sidebar.New(m.clients[m.activeBackend])

	bounds := computePaneBounds(m.layout.termWidth, m.layout.termHeight)
	var cmd tea.Cmd
	m.layout.Sidebar, cmd = m.layout.Sidebar.Update(
		tea.WindowSizeMsg{Width: bounds.leftWidth, Height: bounds.leftHeight},
	)

	slog.Info("Switched backend", "backend", m.activeBackend)
	return m, cmd
}

// AutofillAction fills the query bar with a SELECT query for the selected
// sidebar table. Only active when the sidebar is focused and not in filter mode.
type AutofillAction struct{}

func (a AutofillAction) Apply(m AppModel) (AppModel, tea.Cmd) {
	if m.panes.Current() != PaneSidebar {
		return m, nil
	}
	if m.layout.Sidebar.List.FilterState() == list.Filtering {
		return m, nil
	}
	if i, ok := m.layout.Sidebar.List.SelectedItem().(sidebar.TableItem); ok {
		slog.Debug("autofill triggered via AutofillAction", "table", i.Schema.Name)
		return m, func() tea.Msg { return AutofillMsg{TableName: i.Schema.Name} }
	}
	return m, nil
}

// ExecuteAction runs the SQL in the query bar and shifts focus to results.
// Only active when the query bar is focused.
type ExecuteAction struct{}

func (a ExecuteAction) Apply(m AppModel) (AppModel, tea.Cmd) {
	if m.panes.Current() != PaneQuery {
		return m, nil
	}
	v := m.layout.Querybar.Input.Value()
	if v != "" {
		slog.Debug("execute triggered via ExecuteAction", "query", v)
		return m, func() tea.Msg { return RunQueryMsg{SQL: v} }
	}
	return m, nil
}
