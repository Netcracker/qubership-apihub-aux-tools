package model

import "strings"

// NormKey normalizes an identifier for cross-source matching: trims whitespace,
// lowercases and removes "_", "-" and inner spaces, so that DDL snake_case
// (ledger_moniker), workbook camelCase (ledgerMoniker) and UPPERCASE variants
// all collapse to one key.
func NormKey(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case '_', '-', ' ':
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// ColKey builds the matching key of a column within a table.
func ColKey(table, column string) string {
	return NormKey(table) + "|" + NormKey(column)
}
