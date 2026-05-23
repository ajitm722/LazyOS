package daemons

import (
	"testing"
)

// TestExtractColumnNames verifies that ExtractColumnNames correctly parses
// comma-separated column definitions with optional type annotations into a
// slice of column-name strings. It covers standard, type-less, single-column,
// and empty-string inputs.
func TestExtractColumnNames(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{name: "standard", in: "pid, name, state", want: []string{"pid", "name", "state"}},
		{name: "with types (compat)", in: "pid BIGINT, name TEXT", want: []string{"pid", "name"}},
		{name: "single column", in: "hostname", want: []string{"hostname"}},
		{name: "empty", in: "", want: []string{""}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractColumnNames(tt.in)
			if len(got) != len(tt.want) {
				t.Fatalf("ExtractColumnNames(%q) = %v, want %v", tt.in, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("ExtractColumnNames(%q)[%d] = %q, want %q", tt.in, i, got[i], tt.want[i])
				}
			}
		})
	}
}

// TestAutofillColumns verifies that AutofillColumns returns a comma-separated
// column string when the table is found in schema, "*" when it is not found,
// and matches case-insensitively.
func TestAutofillColumns(t *testing.T) {
	schema := []TableSchema{
		{Name: "processes", Columns: "pid, name, state"},
		{Name: "users", Columns: "uid, username, shell"},
	}

	t.Run("table found", func(t *testing.T) {
		got := AutofillColumns("processes", schema)
		want := "pid, name, state"
		if got != want {
			t.Errorf("AutofillColumns() = %q, want %q", got, want)
		}
	})

	t.Run("table not found", func(t *testing.T) {
		got := AutofillColumns("unknown", schema)
		if got != "*" {
			t.Errorf("AutofillColumns() = %q, want %q", got, "*")
		}
	})

	t.Run("case insensitive", func(t *testing.T) {
		got := AutofillColumns("Processes", schema)
		want := "pid, name, state"
		if got != want {
			t.Errorf("AutofillColumns() = %q, want %q", got, want)
		}
	})

	t.Run("empty schema", func(t *testing.T) {
		got := AutofillColumns("processes", nil)
		if got != "*" {
			t.Errorf("AutofillColumns() = %q, want %q", got, "*")
		}
	})
}

// TestDeriveColumnsFromSchema verifies that DeriveColumnsFromSchema extracts
// the correct column names from CoreTables for a given SQL statement. It
// tests simple SELECTs, WHERE clauses, case-insensitive table lookups, missing
// FROM clauses, unknown tables, empty schema, and columns without type info.
func TestDeriveColumnsFromSchema(t *testing.T) {
	schema := []TableSchema{
		{Name: "processes", Columns: "pid, name, state"},
		{Name: "users", Columns: "uid, username, shell"},
	}

	tests := []struct {
		name string
		sql  string
		want []string
	}{
		{name: "simple select", sql: "SELECT * FROM processes", want: []string{"pid", "name", "state"}},
		{name: "with where clause", sql: "SELECT pid, name FROM processes WHERE pid = 1", want: []string{"pid", "name", "state"}},
		{name: "lowercase table", sql: "select * from users", want: []string{"uid", "username", "shell"}},
		{name: "no from clause", sql: "SELECT 1", want: nil},
		{name: "unknown table", sql: "SELECT * FROM unknown_table", want: nil},
		{name: "empty schema", sql: "SELECT * FROM processes", want: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sch := schema
			if tt.name == "empty schema" {
				sch = nil
			}
			got := DeriveColumnsFromSchema(tt.sql, sch)
			if len(got) != len(tt.want) {
				t.Errorf("DeriveColumnsFromSchema(%q) = %v, want %v", tt.sql, got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("DeriveColumnsFromSchema(%q)[%d] = %q, want %q", tt.sql, i, got[i], tt.want[i])
				}
			}
		})
	}
}
