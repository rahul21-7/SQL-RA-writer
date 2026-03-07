package algebra

import (
	"fmt"
	"strings"
)

func (e *RAExpression) ToSource() string {
	if e == nil { return "" }
	if e.Type == Relation {
		return e.Table
	}
	// Recursive wrapping: (SELECT ...) AS sub
	return fmt.Sprintf("(%s) AS sub", e.translate())
}

func (e *RAExpression) translate() string {
	if e == nil { return "" }
	switch e.Type {
	case Relation:
		// Return a plain table name if it's the base source
		return "SELECT * FROM " + e.Table
	case Selection:
		// Ensure Condition and Input are strictly separated
		return fmt.Sprintf("SELECT * FROM %s WHERE %s", e.Input.ToSource(), e.Condition)
	case Aggregate, Projection:
		cols := strings.Join(e.Columns, ", ")
		return fmt.Sprintf("SELECT %s FROM %s", cols, e.Input.ToSource())
	default:
		return ""
	}
}

func (e *RAExpression) ToSQL() string {
	// The entry point should return the final query without the outer 'AS sub'
	return e.translate()
}