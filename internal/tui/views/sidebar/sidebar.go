// Package sidebar provides the table list pane backed by Bubble
// Tea's list.Model.
package sidebar

import (
	"io"
	"log/slog"
	"time"

	"github.com/ajitm722/LazyOS/internal/daemons"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/reflow/wordwrap"
)

// DelegateRowHeight is the height in terminal rows for each sidebar list
// item. Five rows provides enough vertical space to display a table name,
// its description, and a sample of its columns without overwhelming the
// pane — it's the smallest value that comfortably fits all three lines.
const delegateRowHeight = 5

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

// Model wraps a list.Model to display table names.
type Model struct {
	List           list.Model // the Bubble Tea list widget
	lastClickTime  time.Time  // for double-click detection
	lastClickIndex int        // item index of last click
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
	d.SetHeight(delegateRowHeight)

	l := list.New(items, customDelegate{d}, 0, 0)
	l.Title = "Tables"

	slog.Info("Initializing Sidebar Model")

	return Model{List: l}
}

// Init satisfies tea.Model — no-op for the sidebar.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update delegates all messages to the underlying list.Model. Mouse left-click
// selects the item under the cursor; double-click (same item within 500ms)
// emits an AutofillMsg. Wheel events scroll the list up/down.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if msg, ok := msg.(tea.WindowSizeMsg); ok {
		m.List.SetSize(msg.Width, msg.Height)
	}

	if msg, ok := msg.(tea.MouseMsg); ok {
		if msg.Action == tea.MouseActionPress {
			switch msg.Button {
			case tea.MouseButtonWheelDown:
				return m.scrollBy(3), nil
			case tea.MouseButtonWheelUp:
				return m.scrollBy(-3), nil
			case tea.MouseButtonLeft:
				idx := m.itemIndexAt(msg.Y)
				if idx >= 0 && idx < len(m.List.Items()) {
					m.List.Select(idx)
					now := time.Now()
					if idx == m.lastClickIndex && now.Sub(m.lastClickTime) < 500*time.Millisecond {
						if item, ok := m.List.SelectedItem().(TableItem); ok {
							m.lastClickTime = time.Time{} // reset
							m.lastClickIndex = -1
							return m, func() tea.Msg { return AutofillMsg{TableName: item.Schema.Name} }
						}
					}
					m.lastClickTime = now
					m.lastClickIndex = idx
					return m, nil
				}
			}
		}
	}

	var cmd tea.Cmd
	m.List, cmd = m.List.Update(msg)
	return m, cmd
}

// scrollBy moves the list cursor by delta items, clamping to valid range.
func (m Model) scrollBy(delta int) Model {
	idx := m.List.Index() + delta
	items := m.List.Items()
	if idx < 0 {
		idx = 0
	}
	if idx >= len(items) {
		idx = len(items) - 1
	}
	m.List.Select(idx)
	return m
}

// itemIndexAt maps a mouse Y coordinate (relative to the sidebar content
// area) to a global item index. Returns -1 if the coordinate falls outside
// the item area.
func (m Model) itemIndexAt(y int) int {
	titleHeight := 2 // title text + title bar padding
	if m.List.FilterState() == list.Filtering {
		titleHeight = 1 // filter input replaces title
	}

	localY := y - titleHeight
	if localY < 0 {
		return -1
	}

	perPage := m.List.Paginator.PerPage
	if perPage <= 0 {
		return -1
	}

	localIdx := localY / delegateRowHeight
	if localIdx >= perPage {
		return -1
	}

	return m.List.Paginator.Page*perPage + localIdx
}

// View delegates rendering to the underlying list.Model.
func (m Model) View() string {
	return m.List.View()
}
