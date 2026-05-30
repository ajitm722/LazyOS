// Package tui is the core Bubble Tea application layer. It contains the root
// AppModel, the AppAction interface and its implementations, and the
// InputHandler that routes key bindings to actions via a data-driven
// BoundAction slice.
package tui

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/ajitm722/LazyOS/internal/cache"
	"github.com/ajitm722/LazyOS/internal/config"
	"github.com/ajitm722/LazyOS/internal/daemons"
	"github.com/ajitm722/LazyOS/internal/logger"
	"github.com/ajitm722/LazyOS/internal/tui/views/querybar"
	"github.com/ajitm722/LazyOS/internal/tui/views/results"
	"github.com/ajitm722/LazyOS/internal/tui/views/sidebar"
)

// Mode represents the current input mode for vim-style interaction.
type Mode int

const (
	NormalMode Mode = iota
	InsertMode
)

// AppModel is the root Bubble Tea model. It owns the Layout, input routing,
// pane manager, backend cycling, and backend clients — but delegates all screen
// math and rendering to the Layout struct.
type AppModel struct {
	mode          Mode                       // vim input mode (normal/insert)
	layout        Layout                     // screen layout and rendering
	input         InputHandler               // key bindings and action registry
	panes         PaneManager                // focus order and active pane tracking
	clients       map[string]daemons.Queryer // active backend connections
	backendOrder  []string                   // ordered list of backend names for cycling
	activeIndex   int                        // index into backendOrder
	activeBackend string                     // key into clients for the current backend
}

// getDefaultBackend prefers "kernel" if available, otherwise falls back to
// the first entry in the ordered list.
func getDefaultBackend(clients map[string]daemons.Queryer, backendOrder []string) string {
	for _, k := range backendOrder {
		if _, ok := clients[k]; !ok {
			continue
		}
		if k == "kernel" {
			return k
		}
	}
	for _, k := range backendOrder {
		if _, ok := clients[k]; ok {
			return k
		}
	}
	return ""
}

// NewApp constructs the root AppModel and wires all layout, input routing,
// backend cycling, and background clients together declaratively.
func NewApp(clients map[string]daemons.Queryer, backendOrder []string, cfg config.Keys) tea.Model {
	activeBackend := getDefaultBackend(clients, backendOrder)
	var activeIndex int
	for i, k := range backendOrder {
		if k == activeBackend {
			activeIndex = i
			break
		}
	}

	var backend daemons.Queryer
	if activeBackend != "" {
		backend = clients[activeBackend]
	}

	sidebarModel := sidebar.New(backend)
	sidebarModel.List.Title = activeBackend

	return AppModel{
		layout: Layout{
			Sidebar:  sidebarModel,
			Querybar: querybar.New(),
			Results:  results.New(),
			Help:     help.New(),
		},
		input:         NewInputHandler(cfg),
		panes:         NewPaneManager(),
		clients:       clients,
		backendOrder:  backendOrder,
		activeIndex:   activeIndex,
		activeBackend: activeBackend,
	}
}

// Init satisfies tea.Model by delegating to the Layout's batch init.
func (m AppModel) Init() tea.Cmd {
	return m.layout.Init()
}

// Update is the top-level message dispatcher. It routes tea.KeyMsg,
// tea.WindowSizeMsg, and custom messages to their dedicated handlers.
func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	case tea.WindowSizeMsg:
		return m.handleWindowSizeMsg(msg)
	case AutofillMsg:
		return m.handleAutofillMsg(msg)
	case RunQueryMsg:
		return m.handleRunQueryMsg(msg)
	case RunSourceQueryMsg:
		return m.handleRunSourceQueryMsg(msg)
	case QueryResultMsg:
		return m.handleQueryResultMsg(msg)
	case QueryErrorMsg:
		return m.handleQueryErrorMsg(msg)
	}

	return m.routeToFocused(msg)
}

// handleKeyMsg implements vim-style modal key routing.
//
// In INSERT mode control keys (ctrl+l/h/c) still work globally; Esc returns
// to NORMAL; all other keys route to the focused widget. In NORMAL mode all
// global actions are matched, then mode transitions (i/Esc) are handled, then
// remaining keys fall through to routeToFocused.
func (m AppModel) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	slog.Debug("Key pressed", "key", msg.String(), "focus", m.panes.Current(), "mode", m.mode)

	// INSERT mode: control keys still work globally; Esc exits to NORMAL.
	if m.mode == InsertMode {
		if msg.Type == tea.KeyEsc {
			m.mode = NormalMode
			m.layout.Querybar.EnterNormal()
			slog.Debug("Switched to NORMAL mode")
			return m, nil
		}
		if msg.Type == tea.KeyCtrlL || msg.Type == tea.KeyCtrlH || msg.Type == tea.KeyCtrlC {
			for _, mapping := range m.input.Actions {
				if key.Matches(msg, mapping.Binding) {
					return mapping.Action.Apply(m)
				}
			}
		}
		return m.routeToFocused(msg)
	}

	// NORMAL mode: if the sidebar list is filtering, route keys directly
	// so /, j, k, a, etc. are consumed by the filter rather than global actions.
	if m.panes.Current() == PaneSidebar && m.layout.Sidebar.List.FilterState() == list.Filtering {
		return m.routeToFocused(msg)
	}

	// Global actions from the registry.
	for _, mapping := range m.input.Actions {
		if key.Matches(msg, mapping.Binding) {
			slog.Debug("Key matched action", "action", fmt.Sprintf("%T", mapping.Action))
			return mapping.Action.Apply(m)
		}
	}

	// Mode transition: i enters INSERT mode in the query bar.
	if msg.String() == "i" && m.panes.Current() == PaneQuery {
		m.mode = InsertMode
		m.layout.Querybar.Focus()
		slog.Debug("Switched to INSERT mode")
		return m, nil
	}

	return m.routeToFocused(msg)
}

// handleWindowSizeMsg delegates all sizing math and child updates to Layout.
func (m AppModel) handleWindowSizeMsg(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	slog.Info("Window resized", "width", msg.Width, "height", msg.Height)
	var cmd tea.Cmd
	m.layout, cmd = m.layout.Update(msg)
	return m, cmd
}

// handleAutofillMsg is triggered when a user presses Enter on a sidebar
// item. It populates the query bar with a SELECT <columns> FROM <table>
// query derived from the backend schema and moves focus to the input pane.
func (m AppModel) handleAutofillMsg(msg AutofillMsg) (tea.Model, tea.Cmd) {
	slog.Info("Autofilling query", "table", msg.TableName)

	colsStr := "*"
	if client := m.clients[m.activeBackend]; client != nil {
		colsStr = daemons.AutofillColumns(msg.TableName, client.GetSchema())
	}
	query := fmt.Sprintf("SELECT %s FROM %s LIMIT 10;", colsStr, msg.TableName)
	m.layout.Querybar.Input.SetValue(query)

	slog.Debug("Focus shifting", "from", m.panes.Current(), "to", PaneQuery)
	m.panes = m.panes.Set(PaneQuery)
	m.mode = NormalMode
	m.layout.Querybar.EnterNormal()
	return m, nil
}

// handleRunQueryMsg spawns a goroutine that calls client.Query and returns
// either a QueryResultMsg or QueryErrorMsg via the tea.Cmd channel.
func (m AppModel) handleRunQueryMsg(msg RunQueryMsg) (tea.Model, tea.Cmd) {
	sql := msg.SQL
	slog.Info("Executing cached query", "sql", sql)

	m.layout.Results = m.layout.Results.FormatMessage("Fetching data...")

	cmd := func() tea.Msg {
		ctx := logger.WithLogger(context.Background(), slog.Default())
		rows, cols, err := m.clients[m.activeBackend].Query(ctx, sql)
		if err != nil {
			slog.Error("Query execution failed", "sql", sql, "error", err)
			return QueryErrorMsg{Err: err}
		}
		slog.Info("Query executed successfully", "sql", sql, "rows", len(rows))
		return QueryResultMsg{Rows: rows, Columns: cols}
	}
	return m, cmd
}

// handleRunSourceQueryMsg spawns a goroutine that calls the upstream source
// and updates the local cache, or falls back to a regular query when no
// cache is configured.
func (m AppModel) handleRunSourceQueryMsg(msg RunSourceQueryMsg) (tea.Model, tea.Cmd) {
	sql := msg.SQL
	slog.Info("Executing source query", "sql", sql)

	m.layout.Results = m.layout.Results.FormatMessage("Fetching from source...")

	cmd := func() tea.Msg {
		ctx := logger.WithLogger(context.Background(), slog.Default())

		var rows []map[string]string
		var cols []string
		var err error

		if cq, ok := m.clients[m.activeBackend].(*cache.CachedQueryer); ok {
			rows, cols, err = cq.QuerySource(ctx, sql)
		} else {
			rows, cols, err = m.clients[m.activeBackend].Query(ctx, sql)
		}

		if err != nil {
			slog.Error("Source query execution failed", "sql", sql, "error", err)
			return QueryErrorMsg{Err: err}
		}
		slog.Info("Source query executed successfully", "sql", sql, "rows", len(rows))
		return QueryResultMsg{Rows: rows, Columns: cols}
	}
	return m, cmd
}

// handleQueryResultMsg formats the returned rows into both line-mode and
// table-mode views, then shifts focus to the results pane.
func (m AppModel) handleQueryResultMsg(msg QueryResultMsg) (tea.Model, tea.Cmd) {
	slog.Debug("Formatting query results", "rows", len(msg.Rows))
	m.layout.Results = m.layout.Results.FormatData(msg.Rows, msg.Columns)

	slog.Debug("Focus shifting", "from", m.panes.Current(), "to", PaneResults)
	m.panes = m.panes.Set(PaneResults)
	m.layout.Querybar.Blur()
	return m, nil
}

// handleQueryErrorMsg displays the error string inside the results viewport.
func (m AppModel) handleQueryErrorMsg(msg QueryErrorMsg) (tea.Model, tea.Cmd) {
	var displayErr error
	if errors.Is(msg.Err, daemons.ErrQueryTimeout) {
		displayErr = fmt.Errorf("query timed out\n\nsuggestion: the daemon took too long to respond. you can try a more specific query or increase the query timeout in your configuration.\n\ndetails: %v", msg.Err)
	} else if errors.Is(msg.Err, daemons.ErrQueryFailed) {
		displayErr = fmt.Errorf("query failed\n\nplease check your SQL syntax and table names.\n\ndetails: %v", msg.Err)
	} else {
		displayErr = msg.Err
	}

	m.layout.Results = m.layout.Results.FormatError(displayErr)
	return m, nil
}

// routeToFocused dispatches the message to whichever child model currently
// holds active focus (sidebar, querybar, or results).
func (m AppModel) routeToFocused(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd
	switch m.panes.Current() {
	case PaneSidebar:
		m.layout.Sidebar, cmd = m.layout.Sidebar.Update(msg)
		cmds = append(cmds, cmd)
	case PaneQuery:
		m.layout.Querybar, cmd = m.layout.Querybar.Update(msg)
		cmds = append(cmds, cmd)
	case PaneResults:
		m.layout.Results, cmd = m.layout.Results.Update(msg)
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

// View satisfies tea.Model by delegating to the Layout's composition logic.
func (m AppModel) View() string {
	return m.layout.View(m.panes.Current(), m.input, m.mode)
}
