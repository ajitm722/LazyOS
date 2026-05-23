package daemons

import "strings"

// ExtractColumnNames parses a schema Columns string (e.g. "pid BIGINT, name TEXT")
// into an ordered list of column names.
func ExtractColumnNames(columnsStr string) []string {
	parts := strings.Split(columnsStr, ",")
	cols := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if idx := strings.IndexByte(p, ' '); idx > 0 {
			cols = append(cols, p[:idx])
		} else {
			cols = append(cols, p)
		}
	}
	return cols
}

// AutofillColumns returns a comma-separated column list for the given table
// from the schema, or "*" if the table is not found. Used by the TUI to build
// the initial SELECT query when the user picks a table from the sidebar.
func AutofillColumns(tableName string, schema []TableSchema) string {
	for _, t := range schema {
		if strings.EqualFold(t.Name, tableName) {
			return strings.Join(ExtractColumnNames(t.Columns), ", ")
		}
	}
	return "*"
}

// DeriveColumnsFromSchema extracts column names for the given SQL by looking up
// the queried table in the backend's schema. Returns nil if unresolvable.
func DeriveColumnsFromSchema(sql string, schema []TableSchema) []string {
	sqlUpper := strings.ToUpper(sql)
	fromIdx := strings.Index(sqlUpper, "FROM")
	if fromIdx < 0 {
		return nil
	}

	// Advance past "FROM" (4 characters) and trim leading space.
	const fromLen = 4
	rest := strings.TrimSpace(sql[fromIdx+fromLen:])

	// Find the first delimiter marking the end of the table name.
	end := strings.IndexAny(rest, " \t\n\r;")
	if end < 0 {
		end = len(rest)
	}

	tableName := rest[:end]
	tableName = strings.Trim(tableName, "\"'`")

	for _, t := range schema {
		if strings.EqualFold(t.Name, tableName) {
			return ExtractColumnNames(t.Columns)
		}
	}
	return nil
}
