// Package merge matches the comments workbook against the parsed DDL model and
// produces the set of comments/FKs to generate plus all merge warnings.
package merge

import "strings"

// CanonIsPK canonicalizes the IsPK flag: Y/YES/1/TRUE (any case) → "Y",
// empty/N/NO/0/FALSE → "". ok=false for anything else.
func CanonIsPK(s string) (string, bool) {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "":
		return "", true
	case "Y", "YES", "1", "TRUE":
		return "Y", true
	case "N", "NO", "0", "FALSE":
		return "", true
	}
	return "", false
}

// CanonConstraint canonicalizes the Constraint flag to PK/FK/PFK/"".
// ok=false for anything else.
func CanonConstraint(s string) (string, bool) {
	switch v := strings.ToUpper(strings.TrimSpace(s)); v {
	case "":
		return "", true
	case "PK", "FK", "PFK":
		return v, true
	}
	return "", false
}
