// Package results provides the Model that owns both the line-mode viewport
// and the table-mode table widget, plus the IsTableMode toggle.
package results

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/muesli/reflow/truncate"
)

const (
	minViewWidth   = 20
	paddingPerCol  = 2
	minColWidth    = 15 + paddingPerCol
	ellipsisOffset = 3
)

// FormatData populates both the line-mode viewport and the table-mode table
// from the given row data and column names. It is called once per successful
// query response. When rowsData is empty the viewport shows a message but the
// table is still populated with column headers so table mode isn't blank.
func (m Model) FormatData(rowsData []map[string]string, columns []string) Model {
	keys := columns
	if len(keys) == 0 && len(rowsData) > 0 {
		keys = extractKeys(rowsData)
	}

	if len(rowsData) == 0 {
		m.View.SetContent("0 rows returned.")
		m.View.GotoTop()
		m.Table = m.buildTableMode(keys, rowsData)
		return m
	}

	m.View.SetContent(formatLineMode(keys, rowsData))
	m.View.GotoTop()

	m.Table = m.buildTableMode(keys, rowsData)
	return m
}

// extractKeys collects the column names from the first row of data.
func extractKeys(rowsData []map[string]string) []string {
	if len(rowsData) == 0 {
		return nil
	}
	keys := make([]string, 0, len(rowsData[0]))
	for key := range rowsData[0] {
		keys = append(keys, key)
	}
	return keys
}

// formatLineMode renders each row as "--- Row N ---\nkey = value" with
// aligned padding. This is the default, schema-agnostic view.
func formatLineMode(keys []string, rowsData []map[string]string) string {
	var sb strings.Builder
	sb.Grow(len(rowsData) * 50)

	maxLen := 0
	for _, key := range keys {
		maxLen = max(maxLen, len(key))
	}

	for i, rowMap := range rowsData {
		fmt.Fprintf(&sb, "--- Row %d ---\n", i+1)
		for _, key := range keys {
			fmt.Fprintf(&sb, "%-*s = %s\n", maxLen, key, rowMap[key])
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// tableLayout holds the computed column configuration — which keys fit
// on screen and how wide each column should be — derived from the pane width.
type tableLayout struct {
	keys     []string // visible column keys (rightmost dropped if too wide)
	colWidth int      // uniform width per column
}

// computeTableLayout determines which columns fit and at what width based
// on the current pane width and the configured constants.
// When keys is empty the returned layout has nil keys and colWidth = 0,
// producing an empty table — this happens when a query returns no rows
// and no column metadata was provided.
func (m Model) computeTableLayout(keys []string) tableLayout {
	if len(keys) == 0 {
		return tableLayout{}
	}
	viewWidth := max(m.Width, minViewWidth)
	maxCols := max(viewWidth/minColWidth, 1)
	if len(keys) > maxCols {
		keys = keys[:maxCols]
	}
	colWidth := max((viewWidth/len(keys))-paddingPerCol, 1)
	return tableLayout{keys: keys, colWidth: colWidth}
}

// buildColumns converts a tableLayout into a []table.Column slice.
// Each column gets an identical width; only the title differs.
func buildColumns(layout tableLayout) []table.Column {
	cols := make([]table.Column, len(layout.keys))
	for i, key := range layout.keys {
		cols[i] = table.Column{Title: key, Width: layout.colWidth}
	}
	return cols
}

// buildRows converts raw row maps into a []table.Row slice, truncating any
// cell value that exceeds the column width. Truncation uses a safe UTF-8
// aware ellipsis via the reflow/truncate package.
func buildRows(layout tableLayout, rowsData []map[string]string) []table.Row {
	rows := make([]table.Row, len(rowsData))
	for i, rowMap := range rowsData {
		row := make(table.Row, len(layout.keys))
		for j, key := range layout.keys {
			val := rowMap[key]
			if len(val) > layout.colWidth {
				if layout.colWidth > ellipsisOffset {
					val = truncate.StringWithTail(val, uint(layout.colWidth), "...")
				} else {
					val = truncate.StringWithTail(val, uint(layout.colWidth), "")
				}
			}
			row[j] = val
		}
		rows[i] = row
	}
	return rows
}

// buildTableMode is the top-level orchestrator for table construction.
// It computes the layout, builds columns and rows, then assembles a fully
// styled table.Model with the cached table styles applied.
func (m Model) buildTableMode(keys []string, rowsData []map[string]string) table.Model {
	layout := m.computeTableLayout(keys)

	newTable := table.New(
		table.WithColumns(buildColumns(layout)),
		table.WithRows(buildRows(layout, rowsData)),
		table.WithFocused(true),
		table.WithHeight(m.Height),
	)
	newTable.SetStyles(m.tableStyles)

	return newTable
}

// FormatError sets the viewport content to a human-readable error message
// and clears the table so stale results from the previous query do not
// persist in table mode.
func (m Model) FormatError(err error) Model {
	m.View.SetContent(fmt.Sprintf("Query failed: %v", err))
	m.View.GotoTop()

	newTable := table.New(table.WithFocused(true))
	newTable.SetStyles(m.tableStyles)
	m.Table = newTable
	return m
}
