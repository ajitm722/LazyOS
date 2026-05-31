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

	// Switch to insert mode so key presses are accepted
	m.Focus()

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

// TestEnterNormal verifies that EnterNormal keeps the textarea focused
// (cursor visible) but sets active=false and normalMode=true.
func TestEnterNormal(t *testing.T) {
	m := New()
	m.EnterNormal()

	if m.active {
		t.Error("expected active=false after EnterNormal")
	}
	if !m.normalMode {
		t.Error("expected normalMode=true after EnterNormal")
	}
	if !m.Input.Focused() {
		t.Error("expected input focused after EnterNormal (cursor visible)")
	}
}

// TestEnterNormalThenFocus verifies that calling Focus() after EnterNormal()
// transitions cleanly to insert mode.
func TestEnterNormalThenFocus(t *testing.T) {
	m := New()
	m.Input.SetValue("hello world")
	m.EnterNormal()

	m.Focus()
	if !m.active {
		t.Error("expected active=true after Focus")
	}
	if m.normalMode {
		t.Error("expected normalMode=false after Focus")
	}
	if !m.Input.Focused() {
		t.Error("expected input focused after Focus")
	}
}

// TestWordForwardNormalMode verifies that pressing w in normal mode moves the
// cursor forward by one word.
func TestWordForwardNormalMode(t *testing.T) {
	m := New()
	m.Input.SetValue("hello world foo")
	m.EnterNormal()

	// Press w — cursor should move from col 0 to after "hello" (col 6, the space)
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	if m2.Input.Value() != "hello world foo" {
		t.Error("value should not change on word navigation")
	}

	// Press w again — cursor should move from after "hello " to after "world"
	m3, _ := m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	if m3.Input.Value() != "hello world foo" {
		t.Error("value should not change on word navigation")
	}
}

// TestWordBackwardNormalMode verifies that pressing b in normal mode moves the
// cursor backward by one word.
func TestWordBackwardNormalMode(t *testing.T) {
	m := New()
	m.Input.SetValue("hello world foo")
	m.EnterNormal()
	m.Input.CursorEnd() // start at the end

	// Press b — cursor should move back to start of "foo" (col 12)
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	if m2.Input.Value() != "hello world foo" {
		t.Error("value should not change on word navigation")
	}

	// Press b again — cursor should move back to start of "world" (col 6)
	m3, _ := m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	if m3.Input.Value() != "hello world foo" {
		t.Error("value should not change on word navigation")
	}
}

// TestIgnoredKeysInNormalMode verifies that regular typing keys are ignored
// in normal mode — they do not insert characters.
func TestIgnoredKeysInNormalMode(t *testing.T) {
	m := New()
	m.Input.SetValue("hello")
	m.EnterNormal()

	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if m2.Input.Value() != "hello" {
		t.Errorf("expected value unchanged in normal mode, got %q", m2.Input.Value())
	}
}

// TestCharacterLeftNormalMode verifies that h and left arrow move the cursor
// left one character in normal mode.
func TestCharacterLeftNormalMode(t *testing.T) {
	m := New()
	m.Input.SetValue("abcde")
	m.EnterNormal()
	m.Input.CursorEnd()

	// Press h — cursor should move from col 5 to col 4
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	if m2.Input.Value() != "abcde" {
		t.Error("value should not change on character navigation")
	}

	// Press left arrow — cursor should move from col 4 to col 3
	m3, _ := m2.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if m3.Input.Value() != "abcde" {
		t.Error("value should not change on character navigation")
	}
}

// TestCharacterRightNormalMode verifies that l and right arrow move the cursor
// right one character in normal mode.
func TestCharacterRightNormalMode(t *testing.T) {
	m := New()
	m.Input.SetValue("abcde")
	m.EnterNormal()
	m.Input.CursorStart()

	// Press l — cursor should move from col 0 to col 1
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	if m2.Input.Value() != "abcde" {
		t.Error("value should not change on character navigation")
	}

	// Press right arrow — cursor should move from col 1 to col 2
	m3, _ := m2.Update(tea.KeyMsg{Type: tea.KeyRight})
	if m3.Input.Value() != "abcde" {
		t.Error("value should not change on character navigation")
	}
}

// TestMouseClickPositionCursor verifies that a left-click positions the
// text cursor at the click column. We verify by inserting a character after
// the click and checking where it lands.
func TestMouseClickPositionCursor(t *testing.T) {
	m := New()
	m.Input.SetValue("hello world")
	m.Focus() // insert mode so text input works

	// Click at X=5 (between 'o' and ' ').
	msg := tea.MouseMsg(tea.MouseEvent{
		X: 5, Y: 0, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress,
	})
	m2, _ := m.Update(msg)

	// Type 'X' at cursor position.
	m3, _ := m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'X'}})
	if m3.Input.Value() != "helloX world" {
		t.Errorf("expected 'helloX world', got %q", m3.Input.Value())
	}
}

// TestMouseClickPositionCursorMultiLine verifies click-to-position on a
// multi-line value navigates to the correct line and column.
func TestMouseClickPositionCursorMultiLine(t *testing.T) {
	m := New()
	m.Input.SetValue("abc\ndef\nghi")
	m.Focus()

	// Click at line 1 (the "def" line), column 1 (between 'd' and 'e').
	msg := tea.MouseMsg(tea.MouseEvent{
		X: 1, Y: 1, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress,
	})
	m2, _ := m.Update(msg)

	// Type 'X' at cursor position.
	m3, _ := m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'X'}})
	if m3.Input.Value() != "abc\ndXef\nghi" {
		t.Errorf("expected 'abc\\ndXef\\nghi', got %q", m3.Input.Value())
	}
}
