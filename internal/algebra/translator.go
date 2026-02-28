package algebra

import (
	"fmt"
	"strings"
)

// ToSQL is the entry point. It ensures the final output is a complete SELECT statement.
func (e *RAExpression) ToSQL() string {
    if e == nil {
        return ""
    }
    // If we are at a base table, we need to turn it into a query
    if e.Type == Relation {
        return fmt.Sprintf("SELECT * FROM %s", e.Table)
    }
    
    // Otherwise, let the recursive logic handle it
    return e.translate()
}

// translate handles the recursive generation
func (e *RAExpression) translate() string {
    switch e.Type {
    case Relation:
        return e.Table // Just the name, no SELECT prefix here
    case Selection:
        return fmt.Sprintf("SELECT * FROM %s WHERE %s", e.Input.ToSource(), e.Condition)
    case Projection:
        cols := strings.Join(e.Columns, ", ")
        return fmt.Sprintf("SELECT %s FROM %s", cols, e.Input.ToSource())
    case Aggregate:
        cols := strings.Join(e.Columns, ", ")
        return fmt.Sprintf("SELECT %s FROM %s", cols, e.Input.ToSource())
    case Join:
        onClause := ""
        if e.Condition != "" {
            onClause = " ON " + e.Condition
        }
        return fmt.Sprintf("SELECT * FROM %s JOIN %s%s", e.Input.ToSource(), e.Right.ToSource(), onClause)
    default:
        return ""
    }
}

func (e *RAExpression) ToSource() string {
    if e == nil { return "" }
    if e.Type == Relation {
        return e.Table
    }
    // Use translate() so we don't accidentally double-prefix "SELECT *" inside subqueries
    return fmt.Sprintf("(%s)", e.translate())
}