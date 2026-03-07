package algebra

import (
	"strings"
)

func ParseRA(input string) *RAExpression {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil
	}

	// 1. Determine Operator and Find structural Split
	var op rune
	opIdx := -1
	for i, r := range input {
		if r == 'γ' || r == 'σ' || r == 'π' {
			op = r
			opIdx = i
			break
		}
	}

	// Base Case: No operator found, must be a table
	if opIdx == -1 {
		return &RAExpression{
			Type:  Relation,
			Table: strings.Trim(input, "() "),
		}
	}

	// Find the FIRST parenthesis AFTER the operator - this starts the body
	firstParen := strings.Index(input[opIdx:], "(")
	if firstParen == -1 {
		return &RAExpression{Type: Relation, Table: strings.Trim(input, "() ")}
	}
	firstParen += opIdx // Adjust to absolute index

	lastParen := strings.LastIndex(input, ")")
	if lastParen <= firstParen {
		return &RAExpression{Type: Relation, Table: strings.Trim(input, "() ")}
	}

	// 2. Extract Header (Operator + Args) and Body (Inner Logic)
	header := strings.TrimSpace(input[:firstParen])
	body := strings.TrimSpace(input[firstParen+1 : lastParen])

	// 3. Recursive Parsing
	switch op {
	case 'γ':
		args := strings.TrimSpace(strings.TrimPrefix(header, "γ"))
		return &RAExpression{Type: Aggregate, Columns: splitColumns(args), Input: ParseRA(body)}
	case 'σ':
		cond := strings.TrimSpace(strings.TrimPrefix(header, "σ"))
		return &RAExpression{Type: Selection, Condition: cond, Input: ParseRA(body)}
	case 'π':
		args := strings.TrimSpace(strings.TrimPrefix(header, "π"))
		return &RAExpression{Type: Projection, Columns: splitColumns(args), Input: ParseRA(body)}
	}

	return &RAExpression{Type: Relation, Table: strings.Trim(input, "() ")}
}

func splitColumns(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" || s == "*" || strings.Contains(strings.ToLower(s), "count") {
		return []string{"COUNT(*)"}
	}
	return []string{s}
}