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
	Table     string        // Used if Type is Relation
	Columns   []string      // Used for Projection or Aggregation
	Condition string        // Used for Selection or Join condition (ON clause)
	Input     *RAExpression // The child operation (Left input)
	Right     *RAExpression // Specifically for binary ops like Join
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
		// Now uses both Input (Left) and Right
		return fmt.Sprintf("(%s ⨝_%s %s)", e.Input.String(), e.Condition, e.Right.String())
	default:
		return ""
	}
}