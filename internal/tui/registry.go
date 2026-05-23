package tui

import (
	"github.com/ajitm722/lazyos/internal/config"
	"github.com/charmbracelet/bubbles/key"
)

// BoundAction tightly couples a key binding to the action it triggers.
// It is now impossible to define a key without its corresponding action.
type BoundAction struct {
	Binding         key.Binding
	Action          AppAction
	ShowInShortHelp bool
}

// InputHandler stores the single source of truth for global keybindings.
// It implements the help.KeyMap interface directly.
type InputHandler struct {
	Actions []BoundAction // binding-to-action mapping
}

// NewInputHandler creates an InputHandler, applying user-supplied Viper config overrides.
func NewInputHandler(cfg config.Keys) InputHandler {
	return InputHandler{
		Actions: []BoundAction{
			{
				Binding:         bind("ctrl+n", cfg.ToggleTable, "toggle results"),
				Action:          ToggleTableAction{},
				ShowInShortHelp: true,
			},
			{
				Binding:         bind("tab", cfg.FocusNext, "focus next"),
				Action:          FocusNextAction{},
				ShowInShortHelp: true,
			},
			{
				Binding:         bind("shift+tab", cfg.FocusPrev, "focus prev"),
				Action:          FocusPrevAction{},
				ShowInShortHelp: true,
			},
			{
				Binding:         bind("ctrl+e", cfg.Enter, "execute/autofill"),
				Action:          EnterAction{},
				ShowInShortHelp: true,
			},
			{
				Binding:         bind("ctrl+c", cfg.Quit, "quit"),
				Action:          QuitAction{},
				ShowInShortHelp: true,
			},
		},
	}
}

// bind is a helper that prefers overrideKey if non-empty, falling back to defaultKey.
func bind(defaultKey, overrideKey, helpText string) key.Binding {
	targetKey := defaultKey
	if overrideKey != "" {
		targetKey = overrideKey
	}
	return key.NewBinding(
		key.WithKeys(targetKey),
		key.WithHelp(targetKey, helpText),
	)
}

// ShortHelp returns a single-row binding list for the compact help line.
// This satisfies the help.KeyMap interface.
func (h InputHandler) ShortHelp() []key.Binding {
	var bindings []key.Binding
	for _, action := range h.Actions {
		if action.ShowInShortHelp {
			bindings = append(bindings, action.Binding)
		}
	}
	return bindings
}

// FullHelp returns a multi-row binding list for the expanded help view.
// This satisfies the help.KeyMap interface.
func (h InputHandler) FullHelp() [][]key.Binding {
	return [][]key.Binding{h.ShortHelp()}
}
