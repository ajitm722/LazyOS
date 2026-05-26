package tui

import (
	"reflect"
	"testing"

	"github.com/ajitm722/LazyOS/internal/config"
	"github.com/ajitm722/LazyOS/internal/daemons"
	"github.com/ajitm722/LazyOS/internal/daemons/mock"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// TestKeyBindingOverride exercises the InputHandler key-override mechanism.
//
// The test creates an AppModel with custom key bindings:
//   - ToggleTable remapped to Ctrl+T (default is t).
//   - FocusNext remapped to Ctrl+N (default is ctrl+l).
//
// It then uses a table-driven approach to verify each key matches the
// expected action type via reflect.TypeOf, ensuring the ActionRegistry
// mapping is correct after overrides.
func TestKeyBindingOverride(t *testing.T) {
	m := NewApp(map[string]daemons.Queryer{"mock": &mock.MockQueryer{}}, config.Keys{
		ToggleTable: "ctrl+t",
		FocusNext:   "ctrl+n",
	}).(AppModel)

	tests := []struct {
		name string
		key  tea.KeyMsg
		want AppAction
	}{
		{name: "ctrl+n triggers FocusNext", key: tea.KeyMsg{Type: tea.KeyCtrlN}, want: FocusNextAction{}},
		{name: "ctrl+t triggers ToggleTable", key: tea.KeyMsg{Type: tea.KeyCtrlT}, want: ToggleTableAction{}},
		{name: "ctrl+h unchanged as FocusPrev", key: tea.KeyMsg{Type: tea.KeyCtrlH}, want: FocusPrevAction{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, a := range m.input.Actions {
				if key.Matches(tt.key, a.Binding) {
					if got, want := reflect.TypeOf(a.Action), reflect.TypeOf(tt.want); got != want {
						t.Errorf("expected action %v, got %v", want, got)
					}
					return
				}
			}
			t.Errorf("key %s matched no binding", tt.key.String())
		})
	}
}

// TestUnmatchedKey sends a plain 'z' rune key (which has no binding in the
// ActionRegistry) and asserts that:
//   - No cmd is returned (the key falls through to routeToFocused).
//   - The active pane remains PaneSidebar unchanged.
func TestUnmatchedKey(t *testing.T) {
	m := defaultAppModel()

	m2, cmd := updateApp(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	if cmd != nil {
		t.Errorf("expected nil cmd for unmatched key, got non-nil")
	}
	if m2.panes.Current() != PaneSidebar {
		t.Errorf("expected PaneSidebar unchanged, got %v", m2.panes.Current())
	}
}

// TestShortHelp verifies that ShortHelp returns at least one binding and
// that every binding has a non-empty help key string.
func TestShortHelp(t *testing.T) {
	h := NewInputHandler(config.Keys{})
	bindings := h.ShortHelp()
	if len(bindings) == 0 {
		t.Fatal("expected non-empty ShortHelp bindings")
	}
	for _, b := range bindings {
		if b.Help().Key == "" {
			t.Error("binding has empty key in ShortHelp")
		}
	}
}

// TestFullHelp verifies that FullHelp returns at least one row of bindings.
func TestFullHelp(t *testing.T) {
	h := NewInputHandler(config.Keys{})
	help := h.FullHelp()
	if len(help) == 0 {
		t.Fatal("expected non-empty FullHelp")
	}
}
