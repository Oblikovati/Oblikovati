// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"

	"oblikovati.org/kernel/topo"
)

// ValidationReport is the outcome of [Validate]: whether the body is a valid
// manifold, plus precise issues for any offending edges.
type ValidationReport struct {
	Valid         bool
	Manifold      bool
	Closed        bool
	OrientationOK bool
	// EulerCharacteristic is the body surface's χ from the Euler–Poincaré relation V−E+2F−L (L = total
	// face loops; the +2F−L corrects the naive V−E+F for B-rep seam edges and holed faces). For a closed
	// orientable solid χ = Σ over shells of 2−2·genusₛ, so EulerConsistent reports whether χ is admissible
	// (even, and ≤ 2 per shell). A body can pass the per-edge manifold/closed/orientation checks yet still
	// be a topological impossibility — a dropped or doubled face that keeps every edge used twice — which
	// an odd or too-large χ catches where the volume guard cannot (Oblikovati#1407).
	EulerCharacteristic int
	EulerConsistent     bool
	Issues              []string
}

// Validate checks a body's topology: every edge of a manifold solid must be used by
// exactly two faces with opposite orientation, a solid must be closed (no boundary
// edges), and its Euler characteristic must be admissible for a closed orientable
// solid. It reports each offending edge precisely (PBI-084) — a surface body is
// allowed to be open.
func Validate(b *topo.Body) ValidationReport {
	r := ValidationReport{Manifold: true, Closed: true, OrientationOK: true, EulerConsistent: true}
	for _, e := range b.Edges() {
		switch uses := e.Uses(); {
		case len(uses) < 2:
			r.Closed = false
			if b.IsSolid() {
				r.Issues = append(r.Issues, fmt.Sprintf("boundary (open) edge %d on a solid", e.ID()))
			}
		case len(uses) > 2:
			r.Manifold = false
			r.Issues = append(r.Issues, fmt.Sprintf("non-manifold edge %d used by %d faces", e.ID(), len(uses)))
		default:
			if uses[0].Reversed() == uses[1].Reversed() {
				r.OrientationOK = false
				r.Issues = append(r.Issues, fmt.Sprintf("inconsistent orientation at edge %d", e.ID()))
			}
		}
	}
	r.checkEuler(b)
	r.Valid = r.Manifold && r.OrientationOK && (!b.IsSolid() || (r.Closed && r.EulerConsistent))
	return r
}

// checkEuler computes the surface χ = V−E+2F−L (the Euler–Poincaré form, correct across B-rep seams and
// holed faces) and, for a CLOSED solid, verifies it is admissible: even (an odd χ cannot be a closed
// orientable 2-manifold) and at most 2 per shell (χ = Σ over shells of 2−2·genus, so each contributes
// ≤ 2). A violation is a topology defect the per-edge tests can miss, recorded as an issue.
func (r *ValidationReport) checkEuler(b *topo.Body) {
	loops := 0
	for _, f := range b.Faces() {
		loops += len(f.Loops())
	}
	r.EulerCharacteristic = len(b.Vertices()) - len(b.Edges()) + 2*len(b.Faces()) - loops
	if !b.IsSolid() || !r.Closed {
		return // χ = 2−2g only constrains a closed orientable solid; an open sheet is unconstrained
	}
	shells := len(b.Shells())
	if !eulerAdmissible(r.EulerCharacteristic, shells) {
		r.EulerConsistent = false
		r.Issues = append(r.Issues, fmt.Sprintf(
			"Euler characteristic V−E+2F−L = %d is inadmissible for a closed solid of %d shell(s) (must be even and ≤ %d)",
			r.EulerCharacteristic, shells, 2*shells))
	}
}

// eulerAdmissible reports whether χ is possible for a CLOSED orientable solid of the given shell count:
// χ = Σ over shells of (2 − 2·genusₛ), so it must be EVEN and at most 2 per shell (genus ≥ 0). An odd or
// too-large χ is a topological impossibility — a defect the per-edge manifold checks can miss (#1407).
func eulerAdmissible(chi, shells int) bool {
	return chi%2 == 0 && chi <= 2*shells
}

// BoundaryEdges returns the open (boundary) edges of a body — those used by fewer
// than two faces. An empty result means the body is closed.
func BoundaryEdges(b *topo.Body) []*topo.Edge {
	var open []*topo.Edge
	for _, e := range b.Edges() {
		if len(e.Uses()) < 2 {
			open = append(open, e)
		}
	}
	return open
}
