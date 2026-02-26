package algebra

import (
	"fmt"
	"strings"
)

// OpType represents the standard Relational Algebra operators
type OpType string

const (
	Projection OpType = "Project (π)"
	Selection  OpType = "Select (σ)"
	Join       OpType = "Join (⨝)"
	Aggregate  OpType = "Aggregate (γ)"
	Relation   OpType = "Table"
)

// RAExpression represents a single node in an RA tree
type RAExpression struct {
	Type      OpType
	Table     string       // Used if Type is Relation
	Columns   []string     // Used for Projection or Aggregation
	Condition string       // Used for Selection or Join condition
	Input     *RAExpression // The child operation
}

// String returns a mathematical representation of the RA expression
func (e *RAExpression) String() string {
	if e == nil {
		return ""
	}
	switch e.Type {
	case Relation:
		return e.Table
	case Projection:
		return fmt.Sprintf("π %s (%s)", strings.Join(e.Columns, ", "), e.Input.String())
	case Selection:
		return fmt.Sprintf("σ %s (%s)", e.Condition, e.Input.String())
	case Aggregate:
		return fmt.Sprintf("γ %s (%s)", strings.Join(e.Columns, ", "), e.Input.String())
	case Join:
		return fmt.Sprintf("(%s ⨝ %s)", e.Input.String(), e.Condition)
	default:
		return ""
	}
}

// Simple helper to build a basic Select-From-Where structure
func NewSimpleQuery(table string, condition string, cols []string) RAExpression {
	source := &RAExpression{Type: Relation, Table: table}
	filter := &RAExpression{Type: Selection, Condition: condition, Input: source}
	return RAExpression{Type: Projection, Columns: cols, Input: filter}
}