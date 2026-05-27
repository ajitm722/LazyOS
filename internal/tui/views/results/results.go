// Package results provides the Model that owns both the line-mode viewport
// and the table-mode table widget, plus the IsTableMode toggle.
package results

import (
	"log/slog"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	initialContent   = "Awaiting Query...\nPress ctrl+h, ctrl+l to navigate, a to autofill query from sidebar, i to edit query, esc and e to execute, t to toggle." // viewport placeholder before first query
	tableHeaderLines = 2                                                                                                                                            // border + column header height subtracted from pane height
)

var (
	headerBorderColor = lipgloss.Color("240") // muted gray for column header border
	selectedBg        = lipgloss.Color("57")  // deep indigo for focused row background
	selectedFg        = lipgloss.Color("229") // light yellow for focused row text
)

// Model holds the two rendering widgets (viewport and table) and the boolean
// that controls which one is displayed.
type Model struct {
	// View is the scrollable line-mode viewport.
	View viewport.Model
	// Table is the column-based table widget for structured display.
	Table table.Model
	// IsTableMode toggles between line-mode (false) and table-mode (true).
	IsTableMode bool
	// Width of the results pane (set during resize).
	Width int
	// Height of the results pane (set during resize).
	Height int
	// tableStyles holds the cached styles for the table.
	tableStyles table.Styles
}

// New creates a fresh results Model with placeholder text and cached styles.
func New() Model {
	rv := viewport.New(0, 0)
	rv.SetContent(initialContent)

	rt := table.New(table.WithFocused(true))

	return Model{
		View:        rv,
		Table:       rt,
		IsTableMode: false,
		tableStyles: initTableStyles(),
	}
}

// initTableStyles builds and returns the cached table styles.
func initTableStyles() table.Styles {
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(headerBorderColor).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(selectedFg).
		Background(selectedBg).
		Bold(false)
	return s
}

// Init satisfies tea.Model — no-op for the results view.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update delegates to either the viewport or the table depending on
// IsTableMode. In line mode j/k are mapped to scroll down/up.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	m = m.handleWindowResize(msg)

	if km, ok := msg.(tea.KeyMsg); ok && !m.IsTableMode {
		switch km.String() {
		case "j":
			m.View.LineDown(1)
			return m, nil
		case "k":
			m.View.LineUp(1)
			return m, nil
		}
	}

	var cmd tea.Cmd
	if m.IsTableMode {
		m.Table, cmd = m.Table.Update(msg)
	} else {
		m.View, cmd = m.View.Update(msg)
	}
	return m, cmd
}

// handleWindowResize updates pane dimensions and child widget bounds when
// a WindowSizeMsg is received.
func (m Model) handleWindowResize(msg tea.Msg) Model {
	if msg, ok := msg.(tea.WindowSizeMsg); ok {
		m.Width = msg.Width
		m.Height = msg.Height
		m.View.Width = msg.Width
		m.View.Height = msg.Height
		m.Table.SetHeight(max(0, msg.Height-tableHeaderLines))

		slog.Debug("results pane resized", "width", msg.Width, "height", msg.Height)
	}
	return m
}

// ViewStr returns the rendered content of whichever widget is currently
// active (line-mode viewport or table-mode table).
func (m Model) ViewStr() string {
	if m.IsTableMode {
		return m.Table.View()
	}
	return m.View.View()
}
