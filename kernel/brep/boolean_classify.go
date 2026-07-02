// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// insideSolidWindingThreshold is the generalized-winding-number cutoff for "inside", in raw
// steradians: the winding number w = Σ/4π is ~1 strictly inside an outward-oriented closed
// boundary and ~0 strictly outside, so thresholding the sum at 2π (w = 0.5) is the maximally
// robust split — a fragment sample point near a shared edge perturbs the sum by ≪2π, where the
// retired single-ray parity count flipped outright (#1599).
const insideSolidWindingThreshold = 2 * stdmath.Pi

// insideSolid reports whether p is inside the solid b, by thresholding the generalized winding
// number of b's planar faces at p (Jacobson, Kavan & Sorkine, SIGGRAPH 2013). It replaces a
// parity count along one fixed skewed ray (#1599): parity had no reselection when the ray grazed
// an edge/vertex or ran near-parallel to a face, so fragments whose sample point sat near a
// shared boundary — or models with faces near the magic direction — misclassified
// deterministically. The winding number integrates the WHOLE boundary; it has no direction to
// graze. Multi-query callers (the boolean's classification pass) should build one [solidProbe]
// instead of calling this per point.
func insideSolid(b *topo.Body, p math.Point3) bool {
	return newSolidProbe(b).inside(p)
}

// solidProbe caches the per-body inputs insideSolid re-derived on EVERY query — the flattened
// planar faces and the model-relative on-plane tolerance (facesOf + RangeBox dominated
// classification time on the #1607 pocket-chain profile). Pure hoisting, no culling: the
// faces, the tolerance and the summation order are exactly insideSolid's, so every verdict is
// bit-identical.
type solidProbe struct {
	faces   []planarFace
	onPlane float64
	planar  bool
}

// newSolidProbe flattens b once for repeated inside queries.
func newSolidProbe(b *topo.Body) *solidProbe {
	faces, ok := facesOf(b)
	if !ok {
		return &solidProbe{}
	}
	return &solidProbe{faces: faces, onPlane: geom.ResolutionForBox(b.RangeBox()).Plane(), planar: true}
}

// inside thresholds the generalized winding number of the cached faces at p (see insideSolid).
func (sp *solidProbe) inside(p math.Point3) bool {
	if !sp.planar {
		return false
	}
	sum := 0.0
	for _, f := range sp.faces {
		sum += faceSolidAngle(f, p, sp.onPlane)
	}
	// |sum|: membership of the CLOSED REGION is orientation-independent, and legacy builders can
	// emit a consistently inside-out body (loops wound opposite their outward normals — the
	// buildPrism CW-poly class found while fixing #1600). The magnitude reads ~4π inside such a
	// body either way; a signed threshold would write its entire interior off as outside, which
	// the (orientation-blind) parity ray never did.
	return stdmath.Abs(sum) > insideSolidWindingThreshold
}

// faceSolidAngle sums the signed solid angle every loop of the planar face subtends at p, fanning
// each ring from its first vertex. A fan is winding-exact for any simple ring, convex or not (the
// folded triangles of a reflex fan cancel by sign), and a hole ring — wound opposite the outer —
// subtracts its own contribution.
//
// A p ON the face's plane (within onPlaneTol) contributes exactly 0: a flat polygon subtends no
// solid angle at any coplanar exterior point, and evaluating the fan there instead trips the
// atan2 formula's sign-of-zero degeneracy (num = ±0 with den < 0 fabricates a spurious ±2π —
// the beltb regression, where the operands share their top/bottom planes). The coplanar-INTERIOR
// point — genuinely on the solid's boundary — never reaches this classifier: classifySubFace
// resolves it through the coplanarCover/ON-ON table first.
func faceSolidAngle(f planarFace, p math.Point3, onPlaneTol float64) float64 {
	if stdmath.Abs(float64(f.normal.Dot(f.plane.Origin.VectorTo(p)))) < onPlaneTol {
		return 0
	}
	sum := 0.0
	for _, ring := range f.loops {
		for i := 1; i+1 < len(ring); i++ {
			sum += geom.SignedSolidAngle(p, ring[0], ring[i], ring[i+1])
		}
	}
	return sum
}
