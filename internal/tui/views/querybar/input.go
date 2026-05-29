// Package querybar provides the SQL query input pane backed by
// Bubble Tea's textarea.Model.
package querybar

import (
	"log/slog"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// RunQueryMsg is emitted when the user presses Enter in the input pane with
// a non-empty query. It is consumed by AppModel.handleRunQueryMsg.
type RunQueryMsg struct {
	SQL string
}

// Model wraps a textarea.Model to provide the query entry interface.
type Model struct {
	Input              textarea.Model // the Bubble Tea text area widget
	selected           bool
	active             bool           // true when in insert mode (accepts Ctrl+A)
	originalText       lipgloss.Style // saved Text style to restore when deselecting
	originalCursorLine lipgloss.Style // saved CursorLine style to restore when deselecting
}

// New creates a new input Model with the default placeholder and focus.
func New() Model {
	ta := textarea.New()
	ta.Placeholder = "SELECT * FROM processes LIMIT 10;"
	ta.ShowLineNumbers = false
	ta.Focus()
	ta.KeyMap.Paste = key.NewBinding(key.WithKeys("ctrl+v", "ctrl+shift+v"))

	slog.Info("Initializing Querybar Model")

	origText := ta.FocusedStyle.Text
	origCursorLine := ta.FocusedStyle.CursorLine
	return Model{Input: ta, originalText: origText, originalCursorLine: origCursorLine}
}

// Init returns the textarea.Blink command to start the cursor animation.
func (m Model) Init() tea.Cmd {
	return textarea.Blink
}

// Focus activates the querybar and focuses the underlying textarea. Any
// previous selection state and custom styles are reset so the input starts
// clean every time insert mode is entered.
func (m *Model) Focus() tea.Cmd {
	m.active = true
	m.selected = false
	m.Input.FocusedStyle.Text = m.originalText
	m.Input.FocusedStyle.CursorLine = m.originalCursorLine
	return m.Input.Focus()
}

// Blur deactivates the querybar, clears any selection visual state, and
// blurs the underlying textarea.
func (m *Model) Blur() {
	m.active = false
	m.selected = false
	m.Input.FocusedStyle.Text = m.originalText
	m.Input.FocusedStyle.CursorLine = m.originalCursorLine
	m.Input.Blur()
}

// Update delegates all messages to the underlying textarea model.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if msg, ok := msg.(tea.WindowSizeMsg); ok {
		m.Input.SetWidth(msg.Width)
		m.Input.SetHeight(msg.Height)
	}

	if msg, ok := msg.(tea.KeyMsg); ok {
		if !m.active {
			// ignore all special handling when not in insert mode
		} else if msg.Type == tea.KeyCtrlA {
			if m.selected {
				m.selected = false
				m.Input.FocusedStyle.Text = m.originalText
				m.Input.FocusedStyle.CursorLine = m.originalCursorLine
				m.Input.Focus()
			} else {
				m.selected = true
				selStyle := lipgloss.NewStyle().
					Background(lipgloss.Color("4")).
					Foreground(lipgloss.Color("15"))
				m.Input.FocusedStyle.Text = selStyle
				m.Input.FocusedStyle.CursorLine = selStyle
				m.Input.Focus()
				m.Input.CursorEnd()
				return m, nil
			}
		}

		if m.selected {
			m.selected = false
			m.Input.FocusedStyle.Text = m.originalText
			m.Input.FocusedStyle.CursorLine = m.originalCursorLine
			m.Input.Focus()

			if msg.Type == tea.KeyBackspace || msg.Type == tea.KeyDelete {
				m.Input.SetValue("")
				return m, nil
			} else if msg.Type == tea.KeyRunes || msg.Type == tea.KeySpace || key.Matches(msg, m.Input.KeyMap.Paste) {
				m.Input.SetValue("")
			}
		}
	}

	var cmd tea.Cmd
	m.Input, cmd = m.Input.Update(msg)
	return m, cmd
}

// View delegates rendering to the underlying textarea model.
func (m Model) View() string {
	return m.Input.View()
}
