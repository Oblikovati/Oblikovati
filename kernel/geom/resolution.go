// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/math"
)

// Phase 1 of ADR-0042 (Oblikovati/Oblikovati#1242): a single, model-relative source
// of truth for "how close is coincident in this model". Every weld / coincidence /
// on-line / on-plane test should derive its tolerance from the SIZE of the geometry
// it operates on, rather than reading a cm-anchored absolute constant. Absolute
// constants are scale-blind: a part authored at nm/pm scale has all its coordinates
// closer than a 1e-6 cm weld and is merged out of existence, while a km-scale part
// gets needlessly tight tolerances. A size-relative resolution makes the modeller
// faithful across scales.
//
// Resolution lives in geom (not ops) because BOTH the high-level ops welding/CSG path
// AND the delicate kernel/brep planar-arrangement boolean must share one definition —
// and brep cannot import ops (the dependency runs the other way). geom is below both.

// epsRel is the base relative coincidence precision: two points within epsRel × modelSize
// count as the same vertex. 1e-9 is the precedent already set by the bounds-relative
// convex-hull and surface-query tolerances, and sits ~7 orders inside float64's ~16
// significant digits, leaving ample headroom for arithmetic.
const epsRel = 1e-9

// minModelSize floors the model size so a degenerate, single-point or empty operand
// still yields a strictly positive resolution. 1 (one database centimetre) matches the
// long-standing convex-hull fallback.
const minModelSize = 1.0

// Per-purpose tolerance coefficients, as multiples of the base resolution (epsRel × size).
// They differ ON PURPOSE: a vertex weld must be TIGHT — only float-noise-coincident
// points may merge, or a finely-detailed part (a threaded screw whose bbox dwarfs its
// socket) over-merges distinct vertices and tears open — while a default sew gap is
// deliberately generous. The old absolute constants happened to work only because typical
// parts were ~1 cm; expressed relative to size they are far too loose for a 15-unit part,
// so these coefficients are pinned to the proven-robust relative values instead.
const (
	weldCoef  = 1      // vertex / arrangement weld (epsRel × size: the convex-hull precedent)
	planeCoef = 100    // coplanar / on-plane / on-line classification
	sewCoef   = 100000 // default sew gap (deliberately generous, ≈1e-4 cm @ 1cm)
	// stitchCoef is the boolean seam-stitch weld. It must be COARSER than the noise of the SSI
	// producer feeding the stitch: the tracer accepts an on-curve point anywhere within 1e-7 of the
	// trace extent (ssiToleranceFraction), so the same seam point computed from the two operand
	// sides can differ by ~2e-7·size — a finer weld leaves the two copies unmerged and the seam
	// tears open (#1602). 1000 × epsRel = 1e-6·size reproduces the proven absolute 1e-6 stitch grid
	// at the historical ~1 cm part scale and scales with the model.
	stitchCoef = 1000
)

// volCoef is the relative volume tolerance for boolean result classification. Volume
// scales with size³, so this keeps the classification scale-faithful; it reproduces
// boolean.go's 1e-6 cm³ at a 1 cm reference part (size = 1).
const volCoef = 1e-6

// Resolution is a model's size-relative coincidence scale: the single value from which
// all weld / on-line / on-plane / sew / volume tolerances are derived. Build it from the
// geometry an op is about to work on (its bounding-box diagonal) and read the
// purpose-specific tolerance you need.
//
//	res := geom.ResolutionForBox(box)
//	if p.DistanceTo(q) < res.Weld() { /* same vertex */ }
type Resolution struct {
	size float64 // the model size (bounding-box diagonal) this resolution derives from
}

// ResolutionForSize builds a Resolution from an explicit model size (a bounding-box
// diagonal in database units), flooring it at minModelSize so the result is always
// strictly positive.
func ResolutionForSize(size float64) Resolution {
	if stdmath.IsNaN(size) || size < minModelSize {
		size = minModelSize
	}
	return Resolution{size: size}
}

// ResolutionForBox builds a Resolution from a bounding box's diagonal — the entry point
// for any geometry whose extent is already known as a box (a body's RangeBox, a face's
// bounds). An empty box floors to minModelSize.
func ResolutionForBox(box math.Box) Resolution {
	if box.IsEmpty() {
		return ResolutionForSize(minModelSize)
	}
	return ResolutionForSize(float64(box.Diagonal().Length()))
}

// ResolutionForPoints builds a Resolution from a 3D point set's bounding-box diagonal.
func ResolutionForPoints(pts []math.Point3) Resolution {
	box := math.EmptyBox()
	for _, p := range pts {
		box = box.ExtendPoint(p)
	}
	return ResolutionForBox(box)
}

// ResolutionForPoints2D builds a Resolution from a 2D point set's bounding-box diagonal —
// the entry point for planar-arrangement code working in projected face space.
func ResolutionForPoints2D(pts []math.Point2) Resolution {
	if len(pts) == 0 {
		return ResolutionForSize(minModelSize)
	}
	lo, hi := pts[0], pts[0]
	for _, p := range pts {
		lo = math.P2(stdmath.Min(lo.X, p.X), stdmath.Min(lo.Y, p.Y))
		hi = math.P2(stdmath.Max(hi.X, p.X), stdmath.Max(hi.Y, p.Y))
	}
	return ResolutionForSize(float64(lo.DistanceTo(hi)))
}

// Size is the model size (bounding-box diagonal) this resolution derives from.
func (r Resolution) Size() float64 { return r.size }

// Weld is the (tight) vertex coincidence tolerance: two points closer than this are the
// same vertex. Used for every coincidence weld — CSG output, point welder, hole/planar
// arrangement, loop closure, convex-hull dedup. It must stay tight; see the coefficient
// note above.
func (r Resolution) Weld() float64 { return weldCoef * epsRel * r.size }

// Plane is the coplanar / on-plane / on-line classification tolerance: how far a point may
// sit from a cutting plane (BSP), a segment (T-junction) or an arrangement edge and still
// count as on it.
func (r Resolution) Plane() float64 { return planeCoef * epsRel * r.size }

// Sew is the default gap a Sew with tolerance 0 closes — deliberately generous.
func (r Resolution) Sew() float64 { return sewCoef * epsRel * r.size }

// Stitch is the boolean seam-stitch weld: how close two independently computed copies of the same
// seam point may sit and still merge into one vertex. Deliberately coarser than Weld — stitched
// points carry SSI-tracer noise (1e-7 of the trace extent), not float noise; see stitchCoef (#1602).
func (r Resolution) Stitch() float64 { return stitchCoef * epsRel * r.size }

// Area is the relative area / cross-product tolerance for degenerate-triangle and
// turn-direction tests in the planar arrangement: areas scale with size², so this keeps
// a "near-zero area" classification scale-faithful (reproduces the old 1e-9 at size 1).
func (r Resolution) Area() float64 { return epsRel * r.size * r.size }

// Volume is the relative volume tolerance for boolean result classification. It scales
// with size³.
func (r Resolution) Volume() float64 { return volCoef * r.size * r.size * r.size }

// How geom primitives relate to Resolution (ADR-0042, #1504).
//
// The geometry primitives in this package — line/plane/curve intersection, point inversion,
// B-spline/NURBS knot removal and degree reduction, curve stroking, arc-length integration —
// take a SCALAR `tol float64`, not a Resolution. That is deliberate: they are general,
// reusable math kernels, and the RIGHT tolerance depends on the caller's purpose, which only
// the caller knows. A model-coincidence caller (the brep/ops booleans and welders) passes a
// size-relative length from a Resolution — res.Weld() for a vertex coincidence, res.Plane()
// for an on-line/on-plane test — so the size-relative tolerance flows all the way down. Other
// callers pass a tolerance of a DIFFERENT kind entirely: a fitting/approximation bound (knot
// removal, degree reduction), a relative integration-error target (arc length), or a chord
// deflection (stroking). Threading a typed Resolution into these primitives would be wrong —
// it would erase the caller's choice of WHICH tolerance and couple pure math to model scale.
//
// What ADR-0042 actually requires of the low layer is that no model-coincidence decision is
// gated on a cm-anchored ABSOLUTE epsilon. That is enforced directly, not through the type
// system: TestNoUnjustifiedAbsoluteEpsilons (kernel/ops) scans this package's tolerance hot
// paths for bare 1e-N length literals; each must be relativised via a Resolution or annotated
// `// tol:<kind>` (angular / parametric / numeric / area / volume / calibrated) to declare it
// is not a model length. So the `tol float64` parameters stay, and the guard — not a signature
// rewrite — is what keeps the bottom of the stack scale-faithful.

// spanCeilingOrders is the usable dynamic range of one model in float64: ~15.95 significant
// decimal digits, so a feature more than ~10^15 smaller than the model extent is below the
// representable floor and would be silently merged (ADR-0042 §Phase 2). We use 15 as the
// conservative, round ceiling.
const spanCeilingOrders = 15

// FeatureResolvable reports whether a feature of size featureSize is distinguishable in a model
// whose extent is modelBox — i.e. it is at least the model's coincidence resolution. A false
// result means the feature is below the single-model span ceiling (float64 ~15 orders) and would
// be welded away; the caller should surface SpanCeilingWarning rather than build silently.
func FeatureResolvable(modelBox math.Box, featureSize float64) bool {
	return featureSize >= ResolutionForBox(modelBox).Weld()
}

// SpanCeilingWarning returns a human-readable diagnostic when a feature of size featureSize
// cannot be resolved in a model bounded by modelBox, and "" when it can. float64 caps a single
// model at ~15 orders of magnitude (ADR-0042 §Phase 2): a pm feature on a km part exceeds that,
// so this lets the UI/API warn the user instead of silently merging the feature. The message
// names the offending size, the model's resolution, and the working-scale remedy.
func SpanCeilingWarning(modelBox math.Box, featureSize float64) string {
	res := ResolutionForBox(modelBox)
	if featureSize <= 0 || featureSize >= res.Weld() {
		return ""
	}
	return fmt.Sprintf("feature size %g is below this model's resolution %g (extent %g spans more than "+
		"the ~%d orders of magnitude float64 can represent in one model); it would be merged away — "+
		"model at a working unit closer to the feature scale, or split the design across documents",
		featureSize, res.Weld(), res.Size(), spanCeilingOrders)
}
