package algebra

import (
	"fmt"
	"strings"
)

// OpType represents the standard Relational Algebra operators.
type OpType string

const (
	Projection OpType = "Project (π)"
	Selection  OpType = "Select (σ)"
	Join       OpType = "Join (⨝)"
	Aggregate  OpType = "Aggregate (γ)"
	Relation   OpType = "Table"
)

// RAExpression represents a single node in an RA tree.
type RAExpression struct {
	Type      OpType
	Table     string        // Used when Type == Relation
	Columns   []string      // Used for Projection or Aggregation
	Condition string        // Used for Selection condition or Join ON clause
	Input     *RAExpression // Left (or only) child
	Right     *RAExpression // Right child — used for Join only
}

// String returns a human-readable mathematical RA expression.
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
		if e.Condition == "" {
			return fmt.Sprintf("(%s ⨝ %s)", e.Input.String(), e.Right.String())
		}
		return fmt.Sprintf("(%s ⨝_{%s} %s)", e.Input.String(), e.Condition, e.Right.String())
	default:
		return ""
	}
}