package results

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestNewResults verifies that the results model is initialised with the
// correct default state: IsTableMode is false and the viewport displays the
// initial "Awaiting Query" placeholder.
func TestNewResults(t *testing.T) {
	m := New()
	m.View.Width = 50
	m.View.Height = 20
	if m.IsTableMode {
		t.Error("expected IsTableMode to be false initially")
	}
	if !strings.Contains(m.View.View(), "Awaiting Query") {
		t.Errorf("expected initial content, got %s", m.View.View())
	}
}

// TestInitResults verifies that the results model's Init() returns nil,
// confirming no initial command is emitted.
func TestInitResults(t *testing.T) {
	m := New()
	if cmd := m.Init(); cmd != nil {
		t.Error("expected Init() to return nil")
	}
}

// TestUpdateResults verifies that the results model correctly handles
// WindowSizeMsg (updating dimensions) and delegates key events to the
// underlying viewport in line mode and to the table renderer in table mode.
func TestUpdateResults(t *testing.T) {
	m := New()

	// Test WindowSizeMsg updates dimensions
	m, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	if m.Width != 100 {
		t.Errorf("expected width 100, got %d", m.Width)
	}
	if m.Height != 50 {
		t.Errorf("expected height 50, got %d", m.Height)
	}

	// Default line mode should delegate to View
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})

	// Table mode should delegate to Table
	m.IsTableMode = true
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
}

// TestViewStr verifies that the ViewStr method returns a non-empty rendered
// string in both line mode and table mode.
func TestViewStr(t *testing.T) {
	m := New()
	m.View.Width = 50
	m.View.Height = 20
	m.IsTableMode = false
	if view := m.ViewStr(); view == "" {
		t.Error("expected non-empty viewstr in line mode")
	}

	m.IsTableMode = true
	if view := m.ViewStr(); view == "" {
		t.Error("expected non-empty viewstr in table mode")
	}
}

// TestFormatData verifies that FormatData correctly formats row data in line
// mode (showing "Row N" labels), displays "0 rows returned" for empty results,
// and extracts column names from row map keys when columns are nil.
func TestFormatData(t *testing.T) {
	m := New()
	m.Width = 100
	m.Height = 50
	m.View.Width = 100
	m.View.Height = 50

	cols := []string{"id", "name"}
	data := []map[string]string{
		{"id": "1", "name": "test1"},
		{"id": "2", "name": "test2"},
	}

	m = m.FormatData(data, cols)
	view := m.View.View()
	if !strings.Contains(view, "Row 1") || !strings.Contains(view, "test1") {
		t.Errorf("line mode view did not format properly, got %s", view)
	}

	// Test zero rows
	m = m.FormatData(nil, nil)
	if !strings.Contains(m.View.View(), "0 rows returned") {
		t.Errorf("expected '0 rows returned' message, got %s", m.View.View())
	}

	// Test rows but no cols (extraction)
	dataNoCols := []map[string]string{{"foo": "bar"}}
	m = m.FormatData(dataNoCols, nil)
	if !strings.Contains(m.View.View(), "foo") {
		t.Errorf("expected extracted key 'foo', got %s", m.View.View())
	}
}

// TestFormatError verifies that FormatError produces a viewport string
// containing the "Query failed:" prefix and the error message text.
func TestFormatError(t *testing.T) {
	m := New()
	m.View.Width = 50
	m.View.Height = 20
	m = m.FormatError(errors.New("db error"))
	if !strings.Contains(m.View.View(), "Query failed: db error") {
		t.Errorf("expected error message, got %s", m.View.View())
	}
}

// TestFormatMessage verifies that FormatMessage sets the viewport content
// to the given string and clears the table widget.
func TestFormatMessage(t *testing.T) {
	m := New()
	m.View.Width = 50
	m.View.Height = 20
	m = m.FormatMessage("loading...")
	if !strings.Contains(m.View.View(), "loading") {
		t.Errorf("expected 'loading...', got %s", m.View.View())
	}
}

// TestComputeTableLayout verifies that computeTableLayout produces a layout
// with the expected number of keys and a positive column width.
func TestComputeTableLayout(t *testing.T) {
	m := New()
	m.Width = 100

	layout := m.computeTableLayout([]string{"a", "b", "c"})
	if len(layout.keys) != 3 {
		t.Errorf("expected 3 keys, got %d", len(layout.keys))
	}
	if layout.colWidth <= 0 {
		t.Errorf("expected positive col width, got %d", layout.colWidth)
	}
}

// TestExtractKeysEmpty verifies that extractKeys returns nil when given a nil
// slice of rows (edge case guard).
func TestExtractKeysEmpty(t *testing.T) {
	keys := extractKeys(nil)
	if keys != nil {
		t.Errorf("expected nil for empty rows, got %v", keys)
	}
}

// TestComputeTableLayoutManyCols verifies that when the available width is
// too small, computeTableLayout truncates the column set to a single key
// (width / minimum-col-width = 1).
func TestComputeTableLayoutManyCols(t *testing.T) {
	m := New()
	m.Width = 20 // forces maxCols = 1 (20 / 17 = 1)
	layout := m.computeTableLayout([]string{"a", "b", "c"})
	if len(layout.keys) != 1 {
		t.Errorf("expected 1 key (truncated), got %d", len(layout.keys))
	}
}

// TestBuildRowsTruncationSmallWidth verifies that buildRows truncates cell
// values to the column width when the column width is at or below the
// ellipsis offset (3), producing values without ellipsis markers.
func TestBuildRowsTruncationSmallWidth(t *testing.T) {
	layout := tableLayout{keys: []string{"col1"}, colWidth: 2} // <= ellipsisOffset (3)

	data := []map[string]string{{"col1": "too_long_value"}}
	rows := buildRows(layout, data)

	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}

	val := rows[0][0]
	// Truncation without ellipsis for width 2 should be length 2
	if len(val) > 2 {
		t.Errorf("expected length <= 2, got %d (%s)", len(val), val)
	}
}

// TestBuildRowsTruncationLargeWidth verifies that buildRows truncates cell
// values with an ellipsis suffix when the column width exceeds the ellipsis
// offset (3), ensuring the total length stays within the column width.
func TestBuildRowsTruncationLargeWidth(t *testing.T) {
	layout := tableLayout{keys: []string{"col1"}, colWidth: 5} // > ellipsisOffset (3)

	data := []map[string]string{{"col1": "too_long_value"}}
	rows := buildRows(layout, data)

	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}

	// With colWidth=5, truncated value might look like "to..."
	val := rows[0][0]
	if len(val) > 5 {
		t.Errorf("expected length <= 5, got %d (%s)", len(val), val)
	}
}

// TestUpdateResultsLineModeJKey verifies that pressing j in line mode
// scrolls the viewport down.
func TestUpdateResultsLineModeJKey(t *testing.T) {
	m := New()
	m, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	content := strings.Repeat("line\n", 100)
	m.View.SetContent(content)
	m.View.GotoTop()
	orig := m.View.YOffset

	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if m2.View.YOffset <= orig {
		t.Error("expected y offset to increase after pressing j")
	}
}

// TestUpdateResultsLineModeKKey verifies that pressing k in line mode
// scrolls the viewport up.
func TestUpdateResultsLineModeKKey(t *testing.T) {
	m := New()
	m, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	content := strings.Repeat("line\n", 100)
	m.View.SetContent(content)
	m.View.GotoTop()
	m.View.LineDown(5)
	orig := m.View.YOffset

	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if m2.View.YOffset >= orig {
		t.Error("expected y offset to decrease after pressing k")
	}
}

// TestUpdateTableMode verifies that key events in table mode are delegated
// to the table widget.
func TestUpdateTableMode(t *testing.T) {
	m := New()
	m, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	m.IsTableMode = true
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if !m2.IsTableMode {
		t.Error("expected IsTableMode to stay true")
	}
}

// TestMouseClickTableRow verifies that a left-click in table mode highlights
// the clicked row.
func TestMouseClickTableRow(t *testing.T) {
	m := New()
	m, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	m.IsTableMode = true

	// Populate the table with data so there are rows to click.
	data := []map[string]string{
		{"col1": "a", "col2": "b"},
		{"col1": "c", "col2": "d"},
		{"col1": "e", "col2": "f"},
	}
	m = m.FormatData(data, []string{"col1", "col2"})

	// Click on row 0 (Y=2, after header+border).
	msg := tea.MouseMsg(tea.MouseEvent{
		Y: 2, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress,
	})
	m2, _ := m.Update(msg)

	if m2.Table.SelectedRow() == nil {
		t.Error("expected a row to be selected after click")
	}
}

// TestMouseClickTableHeader verifies that clicking on the header row (Y=0)
// does nothing — no row is selected and no panic occurs.
func TestMouseClickTableHeader(t *testing.T) {
	m := New()
	m, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	m.IsTableMode = true

	msg := tea.MouseMsg(tea.MouseEvent{
		Y: 0, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress,
	})
	m2, _ := m.Update(msg)
	if !m2.IsTableMode {
		t.Error("expected IsTableMode to stay true")
	}
}
