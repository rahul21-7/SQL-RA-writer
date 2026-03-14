package algebra

import (
	"fmt"
	"regexp"
	"strings"
)
func (e *RAExpression) ToSource() string {
	if e == nil {
		return ""
	}
	if e.Type == Relation {
		return e.Table
	}
	if e.Type == Join {
		return e.translateJoin()
	}
	return fmt.Sprintf("(%s) AS sub", e.translate())
}

// translateJoin produces the SQL JOIN clause for a Join node.
func (e *RAExpression) translateJoin() string {
	left := e.Input.ToSource()
	right := e.Right.ToSource()
	if e.Condition == "" {
		return fmt.Sprintf("%s NATURAL JOIN %s", left, right)
	}
	return fmt.Sprintf("%s JOIN %s ON %s", left, right, stripTableAliases(e.Condition))
}

// translate is the core recursive SQL generator.
func (e *RAExpression) translate() string {
	if e == nil {
		return ""
	}
	switch e.Type {
	case Relation:
		return "SELECT * FROM " + e.Table

	case Join:
		return "SELECT * FROM " + e.translateJoin()

	case Selection:
		// Strip Spider-style aliases (T1., T2.) from the WHERE condition.
		condition := stripTableAliases(e.Condition)
		return fmt.Sprintf("SELECT * FROM %s WHERE %s", e.Input.ToSource(), condition)

	case Projection:
		cols := strings.Join(e.Columns, ", ")
		return fmt.Sprintf("SELECT %s FROM %s", cols, e.Input.ToSource())

	case Aggregate:
		return e.translateAggregate()

	default:
		return ""
	}
}

func (e *RAExpression) translateAggregate() string {
	var aggCols []string   // e.g. COUNT(*), SUM(salary)
	var groupCols []string // e.g. dept_name, year

	aggFns := []string{"count", "sum", "avg", "max", "min"}

	for _, col := range e.Columns {
		colLower := strings.ToLower(strings.TrimSpace(col))
		isAgg := false
		for _, fn := range aggFns {
			if strings.Contains(colLower, fn) {
				isAgg = true
				break
			}
		}
		if isAgg {
			aggCols = append(aggCols, strings.TrimSpace(col))
		} else {
			groupCols = append(groupCols, stripTableAliases(strings.TrimSpace(col)))
		}
	}

	allSelect := append(groupCols, aggCols...)
	selectClause := strings.Join(allSelect, ", ")
	if selectClause == "" {
		selectClause = "COUNT(*)"
	}

	q := fmt.Sprintf("SELECT %s FROM %s", selectClause, e.Input.ToSource())

	if len(groupCols) > 0 {
		q += " GROUP BY " + strings.Join(groupCols, ", ")
	}

	return q
}

func (e *RAExpression) ToSQL() string {
	return e.translate()
}

func stripTableAliases(s string) string {
	reg := regexp.MustCompile(`(?i)\bT\d+\.`)
	return reg.ReplaceAllString(s, "")
}