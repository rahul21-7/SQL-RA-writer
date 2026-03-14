package algebra

import (
	"regexp"
	"strings"
)

func clean(s string) string {
	// Allow letters, digits, spaces, comparison operators, dots, parens, commas, ⨝, and *
	// Note: Go regexp does not support \uXXXX — use the literal character.
	reg := regexp.MustCompile(`[^a-zA-Z0-9\s><=!\*\._\(\),⨝]`)
	return strings.TrimSpace(reg.ReplaceAllString(s, ""))
}

// ParseRA parses a Relational Algebra string into an RAExpression tree.
func ParseRA(input string) *RAExpression {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil
	}

	// Find the first RA operator (γ, σ, π)
	var op rune
	opIdx := -1
	for i, r := range input {
		if r == 'γ' || r == 'σ' || r == 'π' {
			op = r
			opIdx = i
			break
		}
	}

	// Base Case: No unary operator found — could be a table name, or a join expression.
	if opIdx == -1 {
		return parseTableOrJoin(input)
	}

	// Find the OUTERMOST opening paren after the operator.
	// We must skip parens that belong to function calls in the header (e.g. count(*), sum(x)).
	// Strategy: scan forward from opIdx, track depth. The first paren at depth=0 that is
	// followed eventually by the rest of the RA body is the structural paren.
	// Simpler reliable approach: find the last '(' before the first RA-operator-like content,
	// by scanning for the paren that, when we look at what's inside, contains RA operators
	// or table names — i.e. it's the structural wrapper, not a function call paren.
	//
	// Reliable rule: the structural paren is the LAST '(' in the header+body region
	// that has a matching ')' at the very end of the expression.
	structuralParen := findStructuralParen(input, opIdx)
	if structuralParen == -1 {
		return &RAExpression{Type: Relation, Table: formatTableSource(input)}
	}

	lastParen := findMatchingParen(input, structuralParen)
	if lastParen == -1 {
		lastParen = strings.LastIndex(input, ")")
	}
	if lastParen == -1 {
		return &RAExpression{Type: Relation, Table: formatTableSource(input)}
	}

	rawHeader := strings.TrimSpace(input[opIdx+1 : structuralParen])
	header := clean(rawHeader)
	body := strings.TrimSpace(input[structuralParen+1 : lastParen])

	switch op {
	case 'γ':
		return &RAExpression{Type: Aggregate, Columns: splitColumns(header), Input: ParseRA(body)}
	case 'σ':
		if header == "" || header == "*" {
			header = "1=1"
		}
		return &RAExpression{Type: Selection, Condition: header, Input: ParseRA(body)}
	case 'π':
		return &RAExpression{Type: Projection, Columns: splitColumns(header), Input: ParseRA(body)}
	}

	return &RAExpression{Type: Relation, Table: formatTableSource(input)}
}

// findStructuralParen finds the opening paren that wraps the RA body (not a function-call paren).
// It scans from the operator forward and returns the index of the paren whose matching close
// is the last ')' in the entire expression — that is the structural wrapper.
func findStructuralParen(s string, fromIdx int) int {
	lastClose := strings.LastIndex(s, ")")
	if lastClose == -1 {
		return -1
	}
	// Walk backwards from lastClose's matching open.
	// Try each '(' from left to right; return the one whose match == lastClose.
	for i := fromIdx; i < len(s); i++ {
		if s[i] == '(' {
			match := findMatchingParen(s, i)
			if match == lastClose {
				return i
			}
		}
	}
	return -1
}

// parseTableOrJoin handles the base case: either a plain table name or a join expression.
// Supports: "table1 ⨝ table2" and "table1 ⨝_{condition} table2"
func parseTableOrJoin(input string) *RAExpression {
	input = strings.Trim(input, "() ")

	joinIdx := strings.Index(input, "⨝")
	if joinIdx == -1 {
		return &RAExpression{Type: Relation, Table: strings.TrimSpace(input)}
	}

	leftPart := strings.TrimSpace(input[:joinIdx])
	rest := strings.TrimSpace(input[joinIdx+len("⨝"):])

	condition := ""
	rightPart := rest
	if strings.HasPrefix(rest, "_") || strings.HasPrefix(rest, "{") {
		condStart := strings.Index(rest, "{")
		condEnd := strings.Index(rest, "}")
		if condStart != -1 && condEnd != -1 {
			condition = strings.TrimSpace(rest[condStart+1 : condEnd])
			rightPart = strings.TrimSpace(rest[condEnd+1:])
		} else {
			parts := strings.SplitN(rest, " ", 2)
			if len(parts) == 2 {
				condition = strings.Trim(parts[0], "_{}")
				rightPart = strings.TrimSpace(parts[1])
			}
		}
	}

	return &RAExpression{
		Type:      Join,
		Condition: condition,
		Input:     ParseRA(leftPart),
		Right:     ParseRA(rightPart),
	}
}

// findMatchingParen finds the closing paren index matching the opening paren at openIdx.
func findMatchingParen(s string, openIdx int) int {
	depth := 0
	for i, r := range s[openIdx:] {
		if r == '(' {
			depth++
		} else if r == ')' {
			depth--
			if depth == 0 {
				return openIdx + i
			}
		}
	}
	return -1
}

// formatTableSource converts a raw table string (possibly containing ⨝) to SQL JOIN syntax.
func formatTableSource(s string) string {
	s = strings.Trim(s, "() ")
	s = strings.ReplaceAll(s, "⨝", " NATURAL JOIN ")
	return strings.TrimSpace(s)
}

// splitColumns splits a column-list string respecting parentheses,
// so "count(*), dept" splits into ["count(*)", "dept"] not ["count(*", " dept"].
func splitColumns(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" || s == "*" {
		return []string{"*"}
	}

	var parts []string
	depth := 0
	start := 0
	for i, r := range s {
		switch r {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				parts = append(parts, strings.TrimSpace(s[start:i]))
				start = i + 1
			}
		}
	}
	parts = append(parts, strings.TrimSpace(s[start:]))
	return parts
}