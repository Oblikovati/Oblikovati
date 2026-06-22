// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Phase 1 of ADR-0042 (Oblikovati/Oblikovati#1242): a single, model-relative
// source of truth for "how close is coincident in this model". Every weld /
// coincidence / on-line / on-plane test should derive its tolerance from the
// SIZE of the geometry it operates on, rather than reading a cm-anchored absolute
// constant. Absolute constants are scale-blind: a part authored at nm/pm scale has
// all its coordinates closer than a 1e-6 cm weld and is merged out of existence,
// while a km-scale part gets needlessly tight tolerances. A size-relative
// resolution makes the modeller faithful across scales.
//
// This file only introduces the concept and the derived tolerances; the kernel
// ops are migrated onto it in #1243, and the pm→km scale-invariance is the
// acceptance gate in #1244. Nothing here changes behaviour until call sites adopt
// it.

// epsRel is the base relative coincidence precision: two points within
// epsRel × modelSize count as the same vertex. 1e-9 is the precedent already set
// by the bounds-relative convex-hull (convexhull.go) and surface-query
// (geom/evaluator_surface_query.go) tolerances, and sits ~7 orders inside
// float64's ~16 significant digits, leaving ample headroom for arithmetic.
const epsRel = 1e-9

// minModelSize floors the model size so a degenerate, single-point or empty
// operand still yields a strictly positive resolution. 1 (one database
// centimetre) matches the long-standing convex-hull fallback (boundsDiagonal
// returns 1 for a zero-extent point set), so adopting Resolution there is a no-op.
const minModelSize = 1.0

// Per-purpose tolerance coefficients, as multiples of the base resolution
// (epsRel × size). They differ ON PURPOSE — a vertex weld is tight, a default sew
// gap is deliberately generous — and are calibrated so that at a 1 cm reference
// part (size = 1) each reproduces the historical absolute constant it replaces in
// #1243. This preserves today's behaviour at typical (~1 cm) scale while making
// every tolerance scale with the model.
const (
	weldCoef  = 1      // vertex / arrangement weld   (≈1e-9 cm @ 1cm: arrWeld, weldPointTol)
	planeCoef = 100    // coplanar / on-plane test    (≈1e-7 cm @ 1cm: csgEps)
	gridCoef  = 1000   // weld grid / on-line test    (≈1e-6 cm @ 1cm: weldGrid, onLineTol)
	sewCoef   = 100000 // default sew gap            (≈1e-4 cm @ 1cm: defaultSewTolerance)
)

// volCoef is the relative volume tolerance for boolean result classification,
// reproducing boolean.go's 1e-6 cm³ at a 1 cm reference part (size = 1). Volume
// scales with size³, so this keeps the classification scale-faithful.
const volCoef = 1e-6

// Resolution is a model's size-relative coincidence scale: the single value from
// which all weld / on-line / on-plane / sew / volume tolerances are derived. Build
// it from the geometry an op is about to work on (its bounding-box diagonal) and
// read the purpose-specific tolerance you need.
//
//	res := ResolutionForBody(b)
//	if p.DistanceTo(q) < res.Weld() { /* same vertex */ }
type Resolution struct {
	size float64 // the model size (bounding-box diagonal) this resolution derives from
}

// ResolutionForSize builds a Resolution from an explicit model size (a
// bounding-box diagonal in database units), flooring it at minModelSize so the
// result is always strictly positive.
func ResolutionForSize(size float64) Resolution {
	if stdmath.IsNaN(size) || size < minModelSize {
		size = minModelSize
	}
	return Resolution{size: size}
}

// ResolutionForPoints builds a Resolution from a point set's bounding-box
// diagonal — the entry point for point-based ops (convex hull, arrangements).
func ResolutionForPoints(pts []math.Point3) Resolution {
	if len(pts) == 0 {
		return ResolutionForSize(minModelSize)
	}
	return ResolutionForSize(boundsDiagonal(pts))
}

// ResolutionForBody builds a Resolution from a body's true bounding-box diagonal
// (curved edges included, via RangeBox) — the entry point for B-rep ops (boolean,
// CSG, sew, weld).
func ResolutionForBody(b *topo.Body) Resolution {
	if b == nil {
		return ResolutionForSize(minModelSize)
	}
	box := b.RangeBox()
	if box.IsEmpty() {
		return ResolutionForSize(minModelSize)
	}
	return ResolutionForSize(float64(box.Diagonal().Length()))
}

// Size is the model size (bounding-box diagonal) this resolution derives from.
func (r Resolution) Size() float64 { return r.size }

// Weld is the vertex/arrangement coincidence tolerance: two points closer than
// this are the same vertex (replaces arrWeld, weldPointTol).
func (r Resolution) Weld() float64 { return weldCoef * epsRel * r.size }

// Plane is the coplanar / on-plane classification tolerance (replaces csgEps).
func (r Resolution) Plane() float64 { return planeCoef * epsRel * r.size }

// Grid is the weld-grid / on-line tolerance for CSG output and segment-interior
// tests (replaces weldGrid, onLineTol).
func (r Resolution) Grid() float64 { return gridCoef * epsRel * r.size }

// Sew is the default gap a Sew with tolerance 0 closes — deliberately generous
// (replaces defaultSewTolerance).
func (r Resolution) Sew() float64 { return sewCoef * epsRel * r.size }

// Volume is the relative volume tolerance for boolean result classification
// (replaces boolean.go's absolute 1e-6 cm³). It scales with size³.
func (r Resolution) Volume() float64 { return volCoef * r.size * r.size * r.size }
