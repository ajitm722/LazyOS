// Package sidebar provides the osquery table list pane backed by Bubble
// Tea's list.Model.
package sidebar

import (
	"io"
	"log/slog"

	"github.com/ajitm722/LazyOS/internal/daemons"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/reflow/wordwrap"
)

// AutofillMsg is emitted when the user presses Enter on a sidebar item. It
// carries the selected table name and is consumed by AppModel.handleAutofillMsg.
type AutofillMsg struct {
	TableName string
}

// TableItem implements the list.Item interface for the sidebar list.
type TableItem struct {
	Schema daemons.TableSchema
}

func (i TableItem) Title() string { return i.Schema.Name }
func (i TableItem) Description() string {
	return "-" + i.Schema.Description + "\n-" + i.Schema.Columns
}
func (i TableItem) FilterValue() string { return i.Schema.Name }

// Model wraps a list.Model to display osquery table names.
type Model struct {
	List list.Model // the Bubble Tea list widget
}

// wrappedItem is used internally to provide a dynamically word-wrapped description
// to the list's default delegate.
type wrappedItem struct {
	TableItem
	desc string
}

func (w wrappedItem) Description() string {
	return w.desc
}

// customDelegate wraps the default list delegate to enable multi-line, word-wrapped descriptions.
type customDelegate struct {
	list.DefaultDelegate
}

// Render intercepts the rendering process to dynamically word-wrap the description
// based on the current width of the list before passing it to the default renderer.
func (d customDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	if i, ok := item.(TableItem); ok {
		textwidth := m.Width() - d.Styles.NormalTitle.GetPaddingLeft() - d.Styles.NormalTitle.GetPaddingRight()
		if textwidth <= 0 {
			textwidth = 1
		}
		desc := wordwrap.String(i.Description(), textwidth)
		d.DefaultDelegate.Render(w, m, index, wrappedItem{TableItem: i, desc: desc})
		return
	}
	d.DefaultDelegate.Render(w, m, index, item)
}

// New creates a new sidebar Model populated with the schema from the provided backend.
func New(backend daemons.Queryer) Model {
	var items []list.Item

	if backend != nil {
		for _, schema := range backend.GetSchema() {
			items = append(items, TableItem{Schema: schema})
		}
	}

	d := list.NewDefaultDelegate()
	d.SetHeight(8)

	l := list.New(items, customDelegate{d}, 0, 0)
	l.Title = "Tables"

	slog.Info("Initializing Sidebar Model")

	return Model{List: l}
}

// Init satisfies tea.Model — no-op for the sidebar.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update delegates all messages to the underlying list.Model.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if msg, ok := msg.(tea.WindowSizeMsg); ok {
		m.List.SetSize(msg.Width, msg.Height)
	}

	var cmd tea.Cmd
	m.List, cmd = m.List.Update(msg)
	return m, cmd
}

// View delegates rendering to the underlying list.Model.
func (m Model) View() string {
	return m.List.View()
}
