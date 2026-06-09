// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"

	"oblikovati.org/math"
)

// SuggestionKind names a constraint the inference engine proposes while sketching.
type SuggestionKind uint8

const (
	SuggestHorizontal SuggestionKind = iota
	SuggestVertical
	SuggestCoincident
	SuggestParallel
	SuggestPerpendicular
)

// Suggestion is one inferred constraint with a priority (higher wins). It carries
// enough to apply the constraint on commit; the glyph that visualizes it is drawn
// by the interaction-graphics overlay (UI, deferred) — inference itself is a pure,
// headless heuristic separate from the solver (modeling/00).
type Suggestion struct {
	Kind     SuggestionKind
	Priority int
	// Target is an existing point a Coincident suggestion would snap to (nil otherwise).
	Target *Point
}

// Inference proposes likely constraints from in-progress geometry. Thresholds are
// in sketch units / radians. It is a ranked heuristic, never authoritative.
type Inference struct {
	// AngleTolerance: how close to axis-aligned (radians) infers horizontal/vertical.
	AngleTolerance float64
	// SnapDistance: how close a free endpoint must be to an existing point to infer
	// coincidence.
	SnapDistance float64
}

// NewInference returns an inference engine with sensible default thresholds.
func NewInference() *Inference {
	return &Inference{AngleTolerance: 3 * stdmath.Pi / 180, SnapDistance: 1e-3}
}

// InferSegment proposes constraints for a segment being drawn from a to b: a
// horizontal/vertical suggestion when it is nearly axis-aligned. The result is
// sorted by descending priority.
func (in *Inference) InferSegment(a, b math.Point2) []Suggestion {
	dx, dy := b.X-a.X, b.Y-a.Y
	if dx == 0 && dy == 0 {
		return nil
	}
	angle := stdmath.Atan2(stdmath.Abs(dy), stdmath.Abs(dx)) // 0 = horizontal, pi/2 = vertical
	var out []Suggestion
	if angle <= in.AngleTolerance {
		out = append(out, Suggestion{Kind: SuggestHorizontal, Priority: priorityFor(angle, 0, in.AngleTolerance)})
	}
	if stdmath.Pi/2-angle <= in.AngleTolerance {
		out = append(out, Suggestion{Kind: SuggestVertical, Priority: priorityFor(angle, stdmath.Pi/2, in.AngleTolerance)})
	}
	return out
}

// InferSnap proposes a coincidence when free is within SnapDistance of one of the
// candidate points, choosing the nearest. It returns a single suggestion or nil.
func (in *Inference) InferSnap(free math.Point2, candidates []*Point) []Suggestion {
	var best *Point
	bestDist := in.SnapDistance
	for _, c := range candidates {
		d := free.DistanceTo(c.Position())
		if d <= bestDist {
			best, bestDist = c, d
		}
	}
	if best == nil {
		return nil
	}
	return []Suggestion{{Kind: SuggestCoincident, Priority: 100, Target: best}}
}

// priorityFor scores how close the angle is to the ideal within tolerance: a closer
// match scores higher (0..100).
func priorityFor(angle, ideal, tol float64) int {
	miss := stdmath.Abs(angle - ideal)
	return int(100 * (1 - miss/tol))
}
