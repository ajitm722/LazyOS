package cache

import "strings"

// extractTableNames extracts all table names referenced in a SQL query.
// It handles FROM and JOIN clauses, comma-separated table lists, and
// skips SQL keywords and subqueries.
func extractTableNames(sql string) []string {
	lower := keywordNormalize(sql)

	var tables []string
	seen := make(map[string]bool)

	i := 0
	n := len(lower)
	for i < n {
		for i < n && (lower[i] == ' ' || lower[i] == '\t' || lower[i] == '\n' || lower[i] == '\r' || lower[i] == ',' || lower[i] == ';') {
			i++
		}
		if i >= n {
			break
		}

		switch {
		case lower[i] == '*':
			i++
		case lower[i] == '(':
			depth := 1
			i++
			for i < n && depth > 0 {
				if lower[i] == '(' {
					depth++
				} else if lower[i] == ')' {
					depth--
				}
				i++
			}
		case isWordChar(lower[i]):
			word := ""
			for i < n && isWordChar(lower[i]) {
				word += string(lower[i])
				i++
			}

			if word == "from" || word == "join" {
				parseFromClause(lower, &i, &tables, seen)
			}
		default:
			i++
		}
	}

	return tables
}

// parseFromClause reads one or more table names after a FROM or JOIN keyword.
// It stops at query clause keywords like WHERE, ORDER BY, LIMIT, and skips
// join modifiers (INNER, LEFT, ON, etc.) and comma separators.
func parseFromClause(lower string, i *int, tables *[]string, seen map[string]bool) {
	n := len(lower)

	for *i < n {
		for *i < n && (lower[*i] == ' ' || lower[*i] == '\t' || lower[*i] == '\n' || lower[*i] == '\r') {
			*i++
		}
		if *i >= n {
			return
		}

		if lower[*i] == ',' {
			*i++
			continue
		}

		if !isWordChar(lower[*i]) {
			return
		}

		word := ""
		for *i < n && isWordChar(lower[*i]) {
			word += string(lower[*i])
			*i++
		}

		if isJoinWord(word) {
			continue
		}

		if isQueryKeyword(word) {
			return
		}

		if !seen[word] {
			*tables = append(*tables, word)
			seen[word] = true
		}
	}
}

// isJoinWord returns true when word is a SQL join modifier keyword that
// should be skipped when collecting table names from a FROM/JOIN clause.
func isJoinWord(word string) bool {
	switch word {
	case "join", "inner", "outer", "left", "right", "full", "cross", "natural", "on":
		return true
	}
	return false
}

// isQueryKeyword returns true when word is a SQL clause keyword that
// terminates the table name collection in a FROM/JOIN clause.
func isQueryKeyword(word string) bool {
	switch word {
	case "select", "from", "where", "group", "having", "order", "limit",
		"offset", "union", "intersect", "except", "insert", "update", "delete",
		"create", "drop", "alter", "into", "values", "set":
		return true
	}
	return false
}

// isWordChar returns true when c is a lowercase letter, digit, or underscore.
// Used to tokenize SQL while skipping operators and punctuation.
func isWordChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_'
}

// keywordNormalize lowercases all ASCII letters in an SQL string so that
// keyword matching is case-insensitive without copying through strings.Map.
func keywordNormalize(sql string) string {
	var s strings.Builder
	for _, r := range sql {
		if r >= 'A' && r <= 'Z' {
			s.WriteRune(r + ('a' - 'A'))
		} else {
			s.WriteRune(r)
		}
	}
	return s.String()
}
