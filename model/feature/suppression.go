// SPDX-License-Identifier: GPL-2.0-only

package feature

import "github.com/Oblikovati/oblikovati/model/param"

// ComparisonType is the operator of a conditional-suppression test — the
// ComparisonTypeEnum.
type ComparisonType uint8

const (
	Equal ComparisonType = iota
	NotEqual
	LessThan
	GreaterThan
	LessOrEqual
	GreaterOrEqual
)

// suppressionCondition suppresses a feature when a parameter's model value compares
// to a threshold per the operator (e.g. "suppress when count < 2").
type suppressionCondition struct {
	paramName string
	cmp       ComparisonType
	threshold float64
}

// holds reports whether the condition is currently true (⇒ the feature is
// suppressed). A missing parameter makes the condition false (no suppression).
func (c *suppressionCondition) holds(params *param.Parameters) bool {
	if params == nil {
		return false
	}
	p, ok := params.ByName(c.paramName)
	if !ok {
		return false
	}
	return compare(p.ModelValue(), c.cmp, c.threshold)
}

func compare(v float64, cmp ComparisonType, threshold float64) bool {
	switch cmp {
	case Equal:
		return v == threshold
	case NotEqual:
		return v != threshold
	case LessThan:
		return v < threshold
	case GreaterThan:
		return v > threshold
	case LessOrEqual:
		return v <= threshold
	default: // GreaterOrEqual
		return v >= threshold
	}
}
