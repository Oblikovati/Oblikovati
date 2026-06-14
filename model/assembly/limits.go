// SPDX-License-Identifier: GPL-2.0-only

package assembly

import "oblikovati.org/math"

// limits bounds a constraint's driven value (offset/angle/ratio). Each bound is
// independently optional; an absent bound does not clamp. It is the host implementation
// of contract.ConstraintLimits and the reference API's ConstraintLimits.
type limits struct {
	min, max, resting          math.Scalar
	hasMin, hasMax, hasResting bool
}

// NewLimits builds a limits value; each bound is included only when its has flag is set.
// It returns nil when no bound is set (an unbounded constraint carries no limits object).
func NewLimits(min math.Scalar, hasMin bool, max math.Scalar, hasMax bool, resting math.Scalar, hasResting bool) *limits {
	if !hasMin && !hasMax && !hasResting {
		return nil
	}
	return &limits{min: min, hasMin: hasMin, max: max, hasMax: hasMax, resting: resting, hasResting: hasResting}
}

// Minimum returns the lower bound and whether it is set.
func (l *limits) Minimum() (float64, bool) { return l.min, l.hasMin }

// Maximum returns the upper bound and whether it is set.
func (l *limits) Maximum() (float64, bool) { return l.max, l.hasMax }

// Resting returns the value a drive returns to when released and whether it is set.
func (l *limits) Resting() (float64, bool) { return l.resting, l.hasResting }

// clamp constrains v to the set bounds, returning the clamped value. An unset bound does
// not constrain that side.
func (l *limits) clamp(v math.Scalar) math.Scalar {
	if l.hasMin && v < l.min {
		v = l.min
	}
	if l.hasMax && v > l.max {
		v = l.max
	}
	return v
}
