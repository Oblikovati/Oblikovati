// SPDX-License-Identifier: GPL-2.0-only

package validate

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
	// IsSolid records whether the validated body is a solid (vs an open surface/sheet body). Validity
	// itself allows an open surface body; [ValidationReport.ValidSolid] pairs Valid with this so a caller
	// that specifically requires a watertight solid result (every boolean path) has one post-condition.
	IsSolid bool
	// EulerCharacteristic is the body surface's χ from the Euler–Poincaré relation V−E+2F−L (L = total
	// face loops; the +2F−L corrects the naive V−E+F for B-rep seam edges and holed faces). For a closed
	// orientable solid χ = Σ over shells of 2−2·genusₛ, so EulerConsistent reports whether χ is admissible
	// (even, and ≤ 2 per shell). A body can pass the per-edge manifold/closed/orientation checks yet still
	// be a topological impossibility — a dropped or doubled face that keeps every edge used twice — which
	// an odd or too-large χ catches where the volume guard cannot (Oblikovati#1407).
	EulerCharacteristic int
	EulerConsistent     bool
	// HolesContained reports whether every planar face's hole loops lie strictly inside their outer loop
	// (the B-rep invariant that a hole is an interior void). It is a diagnostic flag, NOT folded into Valid
	// yet: a malformed protruding-hole face is invisible to the per-edge manifold/closed checks but poisons
	// the tessellator. See checkHoleContainment. Meaningful only when HoleContainmentChecked is true.
	HolesContained bool
	// HoleContainmentChecked records whether the hole-containment level actually RAN. [ValidateTopology]
	// leaves it false: that level is the per-edge and Euler tests only, and reporting a not-looked-for
	// defect as "none found" is the kind of quiet lie this package exists to prevent.
	HoleContainmentChecked bool
	Issues                 []string
}

// ValidateTopology is validity LEVEL 1, the cheapest of the ordered levels: the per-edge manifold /
// closed / orientation tests and the Euler-Poincare admissibility check, over the body's edge and
// loop counts alone. It reads no surface geometry, projects nothing, and tessellates nothing, so it
// is the level a caller can afford on EVERY result of every recompute — which is what the feature
// engine's post-condition runs (ADR-0061 stage 6). It leaves HoleContainmentChecked false.
//
// [Validate] is this plus the hole-containment level; prefer it whenever the extra work is
// affordable, and this when it is not.
//
//	if !ops.ValidateTopology(b).Valid { /* an invalid body is an error, not a return value */ }
func ValidateTopology(b *topo.Body) ValidationReport {
	r := ValidationReport{Manifold: true, Closed: true, OrientationOK: true, EulerConsistent: true, IsSolid: b.IsSolid()}
	for _, e := range b.Edges() {
		r.checkEdgeUses(e, b.IsSolid())
	}
	r.checkEuler(b)
	// HolesContained is intentionally NOT folded into Valid yet — the fillet trim that stops producing
	// protruding-hole faces must land first, or existing valid-solid assertions would flip red. It is a
	// tripwire flag until then, which is also why level 1 can skip it without weakening Valid.
	r.Valid = r.Manifold && r.OrientationOK && (!b.IsSolid() || (r.Closed && r.EulerConsistent))
	return r
}

// checkEdgeUses applies the per-edge manifold/closed/orientation rules to one edge: an edge of a
// manifold solid must be used by exactly two faces with opposite orientation.
func (r *ValidationReport) checkEdgeUses(e *topo.Edge, solid bool) {
	switch uses := e.Uses(); {
	case len(uses) < 2:
		r.Closed = false
		if solid {
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

// Validate checks a body's topology: every edge of a manifold solid must be used by
// exactly two faces with opposite orientation, a solid must be closed (no boundary
// edges), and its Euler characteristic must be admissible for a closed orientable
// solid. It reports each offending edge precisely (PBI-084) — a surface body is
// allowed to be open.
//
// It is [ValidateTopology] followed by the hole-containment level, which projects every multi-loop
// planar face's loops into the face plane and tests containment analytically. That second level is
// materially more expensive than the first and its verdict is NOT part of Valid, so a caller that
// only needs the Valid bar should call [ValidateTopology] instead.
func Validate(b *topo.Body) ValidationReport {
	r := ValidateTopology(b)
	r.checkHoleContainment(b)
	r.HoleContainmentChecked = true
	return r
}

// ValidSolid is the boolean pipeline's adoption post-condition: the body is a valid, watertight solid.
// It is exactly Valid AND IsSolid — for a solid, Valid already implies Closed and Manifold, and for a
// surface body the IsSolid term rejects it — so the boolean paths gate on this one report method rather
// than re-composing Valid/Closed/Manifold/IsSolid by hand (the deleted validBooleanSolid, #2294).
func (r ValidationReport) ValidSolid() bool {
	return r.Valid && r.IsSolid
}

// checkEuler computes the surface χ = V−E+2F−L (the Euler–Poincaré form, correct across B-rep seams and
// holed faces) and, for a CLOSED solid, verifies it is admissible: even (an odd χ cannot be a closed
// orientable 2-manifold) and at most 2 per shell (χ = Σ over shells of 2−2·genus, so each contributes
// ≤ 2). A violation is a topology defect the per-edge tests can miss, recorded as an issue.
func (r *ValidationReport) checkEuler(b *topo.Body) {
	r.EulerCharacteristic = b.EulerCharacteristic()
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
