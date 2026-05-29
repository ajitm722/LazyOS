package querybar

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// Helper that constructs a new active model with existing content set.
func modelWithValue(v string) Model {
	m := New()
	m.Input.SetValue(v)
	m.Focus()
	return m
}

// TestNewQuerybar verifies that New() creates a query bar with the correct
// placeholder text and that the text input starts in a focused state.
func TestNewQuerybar(t *testing.T) {
	m := New()
	if m.Input.Placeholder != "SELECT * FROM processes LIMIT 10;" {
		t.Errorf("unexpected placeholder: %s", m.Input.Placeholder)
	}
	if !m.Input.Focused() {
		t.Error("expected input to be focused on creation")
	}
}

// TestInitQuerybar verifies that the query bar's Init() returns a non-nil
// tea.Cmd, confirming the cursor blink command is emitted.
func TestInitQuerybar(t *testing.T) {
	m := New()
	cmd := m.Init()
	if cmd == nil {
		t.Error("expected non-nil tea.Cmd for blink on Init()")
	}
}

// TestUpdateQuerybar verifies that the query bar correctly handles
// WindowSizeMsg (setting the input width) and processes key rune presses
// by updating its value string.
func TestUpdateQuerybar(t *testing.T) {
	m := New()

	// Test WindowSizeMsg sets width
	m, _ = m.Update(tea.WindowSizeMsg{Width: 50, Height: 10})
	if m.Input.Width() <= 0 {
		t.Errorf("expected positive width, got %d", m.Input.Width())
	}

	// Test key press
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s', 'e', 'l'}})
	if m.Input.Value() != "sel" {
		t.Errorf("expected value 'sel', got '%s'", m.Input.Value())
	}
}

// TestViewQuerybar verifies that the query bar's View() returns a non-empty
// string when the input has a value.
func TestViewQuerybar(t *testing.T) {
	m := New()
	m.Input.SetValue("SELECT *")
	view := m.View()
	if view == "" {
		t.Error("expected non-empty view string")
	}
}

// TestRunQueryMsg verifies that the RunQueryMsg struct correctly stores and
// exposes the SQL field.
func TestRunQueryMsg(t *testing.T) {
	msg := RunQueryMsg{SQL: "test"}
	if msg.SQL != "test" {
		t.Errorf("expected test, got %s", msg.SQL)
	}
}

// TestSelectAll verifies that Ctrl+A selects all text and does not return a cmd.
func TestSelectAll(t *testing.T) {
	m := modelWithValue("some query")

	m2, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlA})
	if cmd != nil {
		t.Error("expected nil cmd from Ctrl+A")
	}
	if !m2.selected {
		t.Error("expected selected to be true after Ctrl+A")
	}
}

// TestSelectAllThenBackspace verifies that selecting all and pressing
// Backspace clears the input and returns no cmd.
func TestSelectAllThenBackspace(t *testing.T) {
	m := modelWithValue("some query")

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlA})
	m2, cmd := m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if cmd != nil {
		t.Error("expected nil cmd")
	}
	if m2.Input.Value() != "" {
		t.Errorf("expected empty value, got %q", m2.Input.Value())
	}
}

// TestSelectAllThenDelete verifies that selecting all and pressing Delete
// clears the input.
func TestSelectAllThenDelete(t *testing.T) {
	m := modelWithValue("some query")

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlA})
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyDelete})
	if m2.Input.Value() != "" {
		t.Errorf("expected empty value, got %q", m2.Input.Value())
	}
}

// TestSelectAllThenTypeCharacter verifies that selecting all and then
// typing a character replaces the existing content.
func TestSelectAllThenTypeCharacter(t *testing.T) {
	m := modelWithValue("some query")

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlA})
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'X'}})
	if m2.Input.Value() != "X" {
		t.Errorf("expected 'X', got %q", m2.Input.Value())
	}
}

// TestSelectAllThenSpace verifies that selecting all and then pressing
// space clears the content and inserts a space.
func TestSelectAllThenSpace(t *testing.T) {
	m := modelWithValue("some query")

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlA})
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeySpace})
	if m2.Input.Value() == "some query" {
		t.Error("expected value to change after select-all and space")
	}
}

// TestSelectAllThenPaste verifies that selecting all and then pasting
// (Ctrl+V) clears the input and attempts a paste.
func TestSelectAllThenPaste(t *testing.T) {
	m := modelWithValue("some query")

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlA})
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlV})
	if m2.Input.Value() == "some query" {
		t.Error("expected value to be cleared before paste")
	}
	// The clipboard may be empty in test environments, so we only
	// assert the original value is gone, not the final content.
}

// TestSelectAllToggleOff verifies that pressing Ctrl+A when already selected
// toggles selection off and does not clear the value.
func TestSelectAllToggleOff(t *testing.T) {
	m := modelWithValue("some query")

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlA})
	if !m.selected {
		t.Fatal("expected selected after first Ctrl+A")
	}

	m2, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlA})
	if cmd == nil {
		t.Error("expected non-nil cmd from Ctrl+A toggle-off (fallthrough to textarea.Update)")
	}
	if m2.selected {
		t.Error("expected selected to be false after second Ctrl+A")
	}
	if m2.Input.Value() != "some query" {
		t.Errorf("expected value to be preserved, got %q", m2.Input.Value())
	}
}

// TestSelectAllThenNonMatchingKey verifies that selecting all and pressing
// a non-matching key (e.g. arrow key) deselects without clearing the value.
func TestSelectAllThenNonMatchingKey(t *testing.T) {
	m := modelWithValue("some query")

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlA})
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if m2.selected {
		t.Error("expected selected to be false after non-matching key")
	}
	if m2.Input.Value() != "some query" {
		t.Errorf("expected value to be preserved, got %q", m2.Input.Value())
	}
}

// TestBlur verifies that Blur deactivates the querybar, marks it as not
// focused, and sets selected to false.
func TestBlur(t *testing.T) {
	m := modelWithValue("some query")
	if !m.active {
		t.Fatal("expected active after Focus()")
	}

	m.Blur()
	if m.active {
		t.Error("expected active=false after Blur")
	}
	if m.selected {
		t.Error("expected selected=false after Blur")
	}
	if m.Input.Focused() {
		t.Error("expected input not focused after Blur")
	}
}
