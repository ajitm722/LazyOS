package tui

import (
	"strings"
	"testing"

	"github.com/ajitm722/LazyOS/internal/daemons/mock"
)

// TestComputePaneBounds verifies the three-pane fractional math for a range
// of terminal dimensions.
//
// computePaneBounds divides the terminal into sidebar (30% width) and main
// view (70% width), then splits the main view vertically into results (80%
// of height) and query bar (20% of height). Insets and borders are
// subtracted via paneContentInset and helpBarHeight.
//
// The table includes standard dimensions, edge cases (zero, too-small,
// single-row), and a case where the query bar just barely gets 1 unit.
func TestComputePaneBounds(t *testing.T) {
	tests := []struct {
		name   string
		width  int
		height int
		want   paneBounds
	}{
		{
			// mainHeight=49, lW=int(100*0.3)-4=26, lH=49-4=45, vW=int(100*0.7)-4=66, vH=int(49*0.8)-4=35, qH=int(49*0.2)-4=5
			name: "standard terminal 100x50", width: 100, height: 50,
			want: paneBounds{leftWidth: 26, leftHeight: 45, viewWidth: 66, viewHeight: 36, queryHeight: 5},
		},
		{
			// mainHeight=9, lW=int(10*0.3)-4=-1→0, lH=9-4=5, vW=int(10*0.7)-4=3, vH=int(9*0.8)-4=3, qH=int(9*0.2)-4=-3→0
			name: "too small terminal 10x10", width: 10, height: 10,
			want: paneBounds{leftWidth: 0, leftHeight: 5, viewWidth: 3, viewHeight: 4, queryHeight: 0},
		},
		{
			// mainHeight=0, all inputs clamp to 0
			name: "zero dimensions", width: 0, height: 0,
			want: paneBounds{leftWidth: 0, leftHeight: 0, viewWidth: 0, viewHeight: 0, queryHeight: 0},
		},
		{
			// mainHeight=23, lW=int(80*0.3)-4=20, lH=23-4=19, vW=int(80*0.7)-4=52, vH=int(23*0.8)-4=14, qH=int(23*0.2)-4=0
			name: "minimum viable 80x24", width: 80, height: 24,
			want: paneBounds{leftWidth: 20, leftHeight: 19, viewWidth: 52, viewHeight: 15, queryHeight: 0},
		},
		{
			// mainHeight=29, lW=int(80*0.3)-4=20, lH=29-4=25, vW=int(80*0.7)-4=52, vH=int(29*0.8)-4=19, qH=int(29*0.2)-4=1
			name: "tall enough for query bar 80x30", width: 80, height: 30,
			want: paneBounds{leftWidth: 20, leftHeight: 25, viewWidth: 52, viewHeight: 20, queryHeight: 1},
		},
		{
			// mainHeight=0, lW=int(100*0.3)-4=26, lH=0-4=-4→0, vW=int(100*0.7)-4=66, vH=int(0*0.8)-4=-4→0, qH=int(0*0.2)-4=-4→0
			name: "single row height", width: 100, height: 1,
			want: paneBounds{leftWidth: 26, leftHeight: 0, viewWidth: 66, viewHeight: 0, queryHeight: 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computePaneBounds(tt.width, tt.height)
			if got != tt.want {
				t.Errorf("computePaneBounds(%d, %d) = %+v, want %+v", tt.width, tt.height, got, tt.want)
			}
		})
	}
}

// TestLayoutTooSmall verifies that when the terminal is below the minimum
// required dimensions (80×24), the Layout sets tooSmall=true and the View
// renders the user-facing warning string.
func TestLayoutTooSmall(t *testing.T) {
	m := newAppModelSized(&mock.MockQueryer{}, 50, 10)

	if !m.layout.tooSmall {
		t.Error("expected tooSmall=true for 50x10 terminal")
	}

	view := m.View()
	if !strings.Contains(view, "Terminal too small") {
		t.Errorf("expected too-small warning, got:\n%s", view)
	}
}

// TestLayoutViewRenders verifies that the composed View renders correctly
// for all three focus states. For each pane the view must be non-empty, and
// the help bar must include the registered key-binding hints.
//
// This exercises computePaneSizes, paneStylesForFocus, and the full
// Layout.View pipeline.
func TestLayoutViewRenders(t *testing.T) {
	m := newAppModel(&mock.MockQueryer{})

	for _, focus := range []PaneID{PaneSidebar, PaneQuery, PaneResults} {
		view := m.layout.View(focus, m.input, NormalMode)
		if view == "" {
			t.Errorf("expected non-empty view for focus %v", focus)
		}
	}

	view := m.View()
	if !strings.Contains(view, "toggle table") {
		t.Errorf("expected help text in view, got:\n%s", view)
	}
	if !strings.Contains(view, "next pane") {
		t.Errorf("expected 'next pane' in help")
	}
}

// TestRenderTooSmallWarning verifies the pure renderTooSmallWarning function
// produces output containing the warning string regardless of dimensions.
func TestRenderTooSmallWarning(t *testing.T) {
	warning := renderTooSmallWarning(80, 24)
	if !strings.Contains(warning, "Terminal too small") {
		t.Errorf("expected too-small warning, got: %s", warning)
	}
}

// TestComputePaneSizes performs a basic sanity check on the outer-dimension
// calculation: for a 100×50 terminal all three panes must have positive
// width (height may be zero at the edges, but width should always be >0 for
// a sufficiently wide terminal).
func TestComputePaneSizes(t *testing.T) {
	sizes := computePaneSizes(100, 50)
	if sizes.listWidth <= 0 {
		t.Errorf("expected positive listWidth, got %d", sizes.listWidth)
	}
	if sizes.viewWidth <= 0 {
		t.Errorf("expected positive viewWidth, got %d", sizes.viewWidth)
	}
	if sizes.inputWidth <= 0 {
		t.Errorf("expected positive inputWidth, got %d", sizes.inputWidth)
	}
}

// TestLayoutViewInsertMode verifies that the View function renders the
// INSERT mode indicator when called with InsertMode.
// TestLayoutViewInsertMode verifies that the View function renders the
// INSERT mode indicator when called with InsertMode.
func TestLayoutViewInsertMode(t *testing.T) {
	m := newAppModel(&mock.MockQueryer{})
	m.mode = InsertMode
	view := m.View()
	if !strings.Contains(view, "INSERT") {
		t.Errorf("expected INSERT mode indicator, got:\n%s", view)
	}
}

// TestPaneAt verifies that PaneAt returns the correct PaneID for screen
// coordinates across a 100×50 terminal layout.
func TestPaneAt(t *testing.T) {
	m := newAppModel(&mock.MockQueryer{})
	// Layout: listW=30, mainHeight=49, inputH=9

	tests := []struct {
		name string
		x, y int
		want PaneID
	}{
		{name: "sidebar top-left", x: 5, y: 5, want: PaneSidebar},
		{name: "sidebar right-edge", x: 27, y: 20, want: PaneSidebar},
		{name: "querybar top", x: 50, y: 3, want: PaneQuery},
		{name: "querybar bottom", x: 50, y: 8, want: PaneQuery},
		{name: "results top", x: 50, y: 10, want: PaneResults},
		{name: "results middle", x: 50, y: 25, want: PaneResults},
		{name: "results bottom", x: 50, y: 48, want: PaneResults},
		{name: "footer line", x: 10, y: 49, want: ""},
		{name: "negative Y", x: 10, y: -1, want: ""},
		{name: "right-pane starts at sidebar boundary", x: 28, y: 3, want: PaneQuery},
		{name: "right-pane starts at sidebar boundary results", x: 28, y: 15, want: PaneResults},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := m.layout.PaneAt(tt.x, tt.y)
			if got != tt.want {
				t.Errorf("PaneAt(%d, %d) = %q, want %q", tt.x, tt.y, got, tt.want)
			}
		})
	}
}

// TestMouseOffset verifies that MouseOffset returns correct content-area
// offsets for each pane in a 100×50 terminal.
func TestMouseOffset(t *testing.T) {
	m := newAppModel(&mock.MockQueryer{})
	// Layout: listW=30, mainHeight=49, inputH=9

	tests := []struct {
		name     string
		pane     PaneID
		wantOffX int
		wantOffY int
	}{
		{name: "sidebar", pane: PaneSidebar, wantOffX: 1, wantOffY: 1},
		{name: "querybar", pane: PaneQuery, wantOffX: 33, wantOffY: 1},  // listW(30)+3
		{name: "results", pane: PaneResults, wantOffX: 33, wantOffY: 10}, // listW(30)+3, inputH(9)+1
		{name: "unknown pane", pane: PaneID("unknown"), wantOffX: 0, wantOffY: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotX, gotY := m.layout.MouseOffset(tt.pane)
			if gotX != tt.wantOffX || gotY != tt.wantOffY {
				t.Errorf("MouseOffset(%q) = (%d, %d), want (%d, %d)",
					tt.pane, gotX, gotY, tt.wantOffX, tt.wantOffY)
			}
		})
	}
}
