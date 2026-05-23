package tui

// PaneID provides a clear string identifier for our views
type PaneID string

const (
	PaneSidebar PaneID = "sidebar"
	PaneQuery   PaneID = "query"
	PaneResults PaneID = "results"
)

// PaneManager handles the active pane state immutably.
type PaneManager struct {
	Order []PaneID // pane cycle order
	Index int      // index of the currently focused pane
}

// NewPaneManager encapsulates the default layout order and starting pane.
func NewPaneManager() PaneManager {
	return PaneManager{
		Order: []PaneID{PaneSidebar, PaneQuery, PaneResults},
		Index: 0,
	}
}

// Next returns a new PaneManager shifted forward
func (f PaneManager) Next() PaneManager {
	f.Index = (f.Index + 1) % len(f.Order)
	return f
}

// Prev returns a new PaneManager shifted backward
func (f PaneManager) Prev() PaneManager {
	f.Index = (f.Index - 1 + len(f.Order)) % len(f.Order)
	return f
}

// Current returns the active PaneID
func (f PaneManager) Current() PaneID {
	return f.Order[f.Index]
}

// Set explicitly jumps to a specific pane
func (f PaneManager) Set(id PaneID) PaneManager {
	for i, p := range f.Order {
		if p == id {
			f.Index = i
			break
		}
	}
	return f
}
