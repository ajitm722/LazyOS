package sidebar

import (
	"bytes"
	"context"
	"testing"

	"github.com/ajitm722/lazyos/internal/daemons"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type mockQueryer struct{}

func (m *mockQueryer) Query(ctx context.Context, sql string) ([]map[string]string, []string, error) {
	return nil, nil, nil
}
func (m *mockQueryer) GetSchema() []daemons.TableSchema {
	return []daemons.TableSchema{
		{Name: "test_table", Description: "A test table", Columns: "id"},
	}
}
func (m *mockQueryer) Close() error { return nil }

// TestNewSidebar verifies that New() creates a sidebar with the correct title
// ("Tables") and populates the list with items derived from the backend's
// schema, checking the Title and FilterValue of the first item.
func TestNewSidebar(t *testing.T) {
	m := New(&mockQueryer{})
	if m.List.Title != "Tables" {
		t.Errorf("expected Title to be 'Tables', got %s", m.List.Title)
	}
	if len(m.List.Items()) != 1 {
		t.Fatalf("expected 1 item, got %d", len(m.List.Items()))
	}

	item := m.List.Items()[0].(TableItem)
	if item.Title() != "test_table" {
		t.Errorf("expected item title 'test_table', got '%s'", item.Title())
	}
	if item.FilterValue() != "test_table" {
		t.Errorf("expected filter value 'test_table', got '%s'", item.FilterValue())
	}
}

// TestInitSidebar verifies that the sidebar model's Init() returns nil,
// confirming no initial command is emitted.
func TestInitSidebar(t *testing.T) {
	m := New(nil)
	if cmd := m.Init(); cmd != nil {
		t.Error("expected Init() to return nil")
	}
}

// TestUpdateSidebar verifies that the sidebar model correctly handles
// WindowSizeMsg (updating the list width) and renders a non-empty view
// without panicking.
func TestUpdateSidebar(t *testing.T) {
	m := New(nil)

	// Test WindowSizeMsg
	m, _ = m.Update(tea.WindowSizeMsg{Width: 30, Height: 20})
	if m.List.Width() != 30 {
		t.Errorf("expected width 30, got %d", m.List.Width())
	}

	// Test View does not panic
	view := m.View()
	if view == "" {
		t.Error("expected non-empty view")
	}
}

// TestCustomDelegateRender verifies that the custom delegate produces
// non-empty rendered output for a TableItem and that the wrappedItem helper
// correctly surfaces the overridden description.
func TestCustomDelegateRender(t *testing.T) {
	d := list.NewDefaultDelegate()
	cd := customDelegate{d}

	m := list.New([]list.Item{}, d, 20, 10)
	item := TableItem{Schema: daemons.TableSchema{Name: "t", Description: "desc", Columns: "c"}}

	var buf bytes.Buffer
	cd.Render(&buf, m, 0, item)

	output := buf.String()
	if output == "" {
		t.Error("expected delegate render output to be non-empty")
	}

	// Also test the wrappedItem directly
	wi := wrappedItem{TableItem: item, desc: "wrapped desc"}
	if wi.Description() != "wrapped desc" {
		t.Errorf("expected wrapped desc, got '%s'", wi.Description())
	}
}

// TestCustomDelegateRenderSmallWidth verifies that the custom delegate
// produces non-empty output even when the list width is extremely small
// (1 unit), exercising the textwidth <= 0 edge case.
func TestCustomDelegateRenderSmallWidth(t *testing.T) {
	d := list.NewDefaultDelegate()
	cd := customDelegate{d}

	// Make width extremely small to trigger textwidth <= 0
	m := list.New([]list.Item{}, d, 1, 10)
	item := TableItem{Schema: daemons.TableSchema{Name: "t", Description: "desc", Columns: "c"}}

	var buf bytes.Buffer
	cd.Render(&buf, m, 0, item)

	output := buf.String()
	if output == "" {
		t.Error("expected delegate render output to be non-empty even with small width")
	}
}

type dummyItem struct{}

func (d dummyItem) Title() string       { return "dummy" }
func (d dummyItem) Description() string { return "dummy" }
func (d dummyItem) FilterValue() string { return "dummy" }

// TestCustomDelegateRenderNonTableItem verifies that the custom delegate
// produces non-empty output when rendering a list item that is not a
// TableItem, verifying the fallback rendering path.
func TestCustomDelegateRenderNonTableItem(t *testing.T) {
	d := list.NewDefaultDelegate()
	cd := customDelegate{d}

	m := list.New([]list.Item{}, d, 20, 10)
	item := dummyItem{}

	var buf bytes.Buffer
	cd.Render(&buf, m, 0, item)

	output := buf.String()
	if output == "" {
		t.Error("expected delegate render output to be non-empty for non-TableItem")
	}
}

// TestAutofillMsg verifies that the AutofillMsg struct correctly stores and
// exposes the TableName field.
func TestAutofillMsg(t *testing.T) {
	msg := AutofillMsg{TableName: "users"}
	if msg.TableName != "users" {
		t.Errorf("expected users, got %s", msg.TableName)
	}
}
