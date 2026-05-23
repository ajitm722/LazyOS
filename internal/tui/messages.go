package tui

import (
	"github.com/ajitm722/lazyos/internal/tui/views/querybar"
	"github.com/ajitm722/lazyos/internal/tui/views/sidebar"
)

// RunQueryMsg is a type alias for querybar.RunQueryMsg — it carries the SQL
// string to execute and is emitted when Enter is pressed in the input pane.
type RunQueryMsg = querybar.RunQueryMsg

// AutofillMsg is a type alias for sidebar.AutofillMsg — it carries the selected
// table name and is emitted when Enter is pressed in the sidebar list.
type AutofillMsg = sidebar.AutofillMsg

// QueryResultMsg carries the row data returned by a successful osquery call.
// Columns is populated from the first row's keys when rows are non-empty;
// it may be set explicitly when rows are empty so the UI can show column
// headers in table mode.
type QueryResultMsg struct {
	Rows    []map[string]string
	Columns []string
}

// QueryErrorMsg carries the error from a failed osquery call.
type QueryErrorMsg struct {
	Err error
}
