// Package tui provides the Lipgloss-based layout engine that arranges the
// sidebar, query input, results view, and help menu into a cohesive TUI.
package tui

import (
	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ajitm722/LazyOS/internal/tui/views/querybar"
	"github.com/ajitm722/LazyOS/internal/tui/views/results"
	"github.com/ajitm722/LazyOS/internal/tui/views/sidebar"
)

// Layout dimension constants for the three-pane proportional split.
//
// All values are in terminal characters (width) and lines (height).
// Fractions are applied against the full terminal dimensions; integer values
// (insets, borders) are absolute amounts subtracted from the result.
const (
	helpBarHeight         = 1   // rows reserved for the help menu at the bottom
	sidebarWidthFraction  = 0.3 // fraction of terminal width given to the sidebar
	resultsWidthFraction  = 0.7 // fraction of terminal width given to the results pane
	resultsHeightFraction = 0.8 // fraction of main height given to results (query bar gets the rest)
	paneContentInset      = 4   // total chars subtracted per axis for pane border + gutter
	minRequiredWidth      = 80  // minimum terminal width (characters) before rendering is skipped
	minRequiredHeight     = 24  // minimum terminal height (lines) before rendering is skipped
)

// Terminal colour palette indices (256‑colour ANSI).
const (
	unfocusedBorderColor = "240" // grey; used for inactive pane borders
	focusedBorderColor   = "42"  // green; used for the active pane border
	helpTextColor        = "241" // dim grey; help bar text
	warningTextColor     = "204" // red; used for the too-small-terminal warning
)

var (
	basePaneStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(unfocusedBorderColor))

	focusedPaneStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color(focusedBorderColor))

	helpStyle = lipgloss.NewStyle().MarginTop(0).MarginLeft(1).Foreground(lipgloss.Color(helpTextColor))
)

// Layout encapsulates all child models and handles terminal resizing.
type Layout struct {
	Sidebar  sidebar.Model  // table list pane
	Querybar querybar.Model // SQL query input pane
	Results  results.Model  // query output pane
	Help     help.Model     // keybinding help bar

	termWidth  int  // last-known terminal width (characters), set in Update
	termHeight int  // last-known terminal height (lines), set in Update
	tooSmall   bool // true when terminal is below minRequired dimensions
}

// Init delegates initialization to all child components.
func (l Layout) Init() tea.Cmd {
	return tea.Batch(l.Sidebar.Init(), l.Querybar.Init(), l.Results.Init())
}

// paneBounds holds the inner (content-area) dimensions for each child pane.
type paneBounds struct {
	leftWidth   int // sidebar content width
	leftHeight  int // sidebar content height
	viewWidth   int // results content width
	viewHeight  int // results content height
	queryHeight int // query bar content height
}

// computePaneBounds derives safe content-area boxes clamped to zero.
func computePaneBounds(width, height int) paneBounds {
	mainHeight := max(0, height-helpBarHeight)

	listW := int(float64(width) * sidebarWidthFraction)
	rightW := width - listW
	inputH := int(float64(mainHeight) * (1.0 - resultsHeightFraction))
	viewH := mainHeight - inputH

	return paneBounds{
		leftWidth:   max(0, listW-paneContentInset),
		leftHeight:  max(0, mainHeight-paneContentInset),
		viewWidth:   max(0, rightW-paneContentInset),
		viewHeight:  max(0, viewH-paneContentInset),
		queryHeight: max(0, inputH-paneContentInset),
	}
}

// Update handles WindowSizeMsg and dispatches tailored bounds to children.
func (l Layout) Update(msg tea.WindowSizeMsg) (Layout, tea.Cmd) {
	l.termWidth = msg.Width
	l.termHeight = msg.Height

	if msg.Width < minRequiredWidth || msg.Height < minRequiredHeight {
		l.tooSmall = true
		return l, nil
	}
	l.tooSmall = false
	l.Help.Width = msg.Width

	return l.resizeChildren(computePaneBounds(msg.Width, msg.Height))
}

// resizeChildren dispatches tailored WindowSizeMsgs to each child pane.
func (l Layout) resizeChildren(bounds paneBounds) (Layout, tea.Cmd) {
	var cmdSidebar, cmdQuerybar, cmdResults tea.Cmd

	l.Sidebar, cmdSidebar = l.Sidebar.Update(tea.WindowSizeMsg{Width: bounds.leftWidth, Height: bounds.leftHeight})
	l.Querybar, cmdQuerybar = l.Querybar.Update(tea.WindowSizeMsg{Width: bounds.viewWidth, Height: bounds.queryHeight})
	l.Results, cmdResults = l.Results.Update(tea.WindowSizeMsg{Width: bounds.viewWidth, Height: bounds.viewHeight})

	return l, tea.Batch(cmdSidebar, cmdQuerybar, cmdResults)
}

// paneSizes holds outer (style) dimensions for the three Lipgloss panes.
type paneSizes struct {
	listWidth   int // sidebar pane style width
	listHeight  int // sidebar pane style height
	inputWidth  int // query bar pane style width
	inputHeight int // query bar pane style height
	viewWidth   int // results pane style width
	viewHeight  int // results pane style height
}

// computePaneSizes derives outer pane dimensions (the -2 is the RoundedBorder).
func computePaneSizes(width, height int) paneSizes {
	mainHeight := height - helpBarHeight

	listW := int(float64(width) * sidebarWidthFraction)
	rightW := width - listW
	inputH := int(float64(mainHeight) * (1.0 - resultsHeightFraction))
	viewH := mainHeight - inputH

	return paneSizes{
		listWidth:   listW - 2,
		listHeight:  mainHeight - 2,
		inputWidth:  rightW - 2,
		inputHeight: inputH - 2,
		viewWidth:   rightW - 2,
		viewHeight:  viewH - 2,
	}
}

// paneStylesForFocus returns styles with borders styled for the active pane.
func paneStylesForFocus(activeFocus PaneID) (list, input, view lipgloss.Style) {
	list = basePaneStyle
	input = basePaneStyle
	view = basePaneStyle

	switch activeFocus {
	case PaneSidebar:
		list = focusedPaneStyle
	case PaneQuery:
		input = focusedPaneStyle
	case PaneResults:
		view = focusedPaneStyle
	}
	return
}

// renderTooSmallWarning centers a terminal-size warning in the available space.
func renderTooSmallWarning(width, height int) string {
	warning := "Terminal too small.\nMinimum size is 80x24."
	return lipgloss.Place(width, height,
		lipgloss.Center, lipgloss.Center,
		lipgloss.NewStyle().Foreground(lipgloss.Color(warningTextColor)).Render(warning))
}

// View composes the three panes and help menu into a single rendered string.
func (l Layout) View(activeFocus PaneID, keys help.KeyMap, mode Mode) string {
	if l.tooSmall {
		return renderTooSmallWarning(l.termWidth, l.termHeight)
	}

	listStyle, inputStyle, viewStyle := paneStylesForFocus(activeFocus)

	sizes := computePaneSizes(l.termWidth, l.termHeight)
	listStyle = listStyle.Width(sizes.listWidth).Height(sizes.listHeight)
	inputStyle = inputStyle.Width(sizes.inputWidth).Height(sizes.inputHeight)
	viewStyle = viewStyle.Width(sizes.viewWidth).Height(sizes.viewHeight)

	leftPane := listStyle.Render(l.Sidebar.View())
	topRightPane := inputStyle.Render(l.Querybar.View())
	bottomRightPane := viewStyle.Render(l.Results.ViewStr())

	rightPane := lipgloss.JoinVertical(lipgloss.Left, topRightPane, bottomRightPane)
	mainGrid := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)
	helpMenu := helpStyle.Render(l.Help.View(keys))

	modeStr := "NORMAL"
	if mode == InsertMode {
		modeStr = "INSERT"
	}
	modeIndicator := lipgloss.NewStyle().
		Foreground(lipgloss.Color(helpTextColor)).
		Render("-- " + modeStr + " --")

	footer := lipgloss.JoinHorizontal(lipgloss.Left, modeIndicator, helpMenu)
	return lipgloss.JoinVertical(lipgloss.Left, mainGrid, footer)
}
