// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// PreciseRangeBox is the tight axis-aligned box over the body's ANALYTIC B-rep — face interiors
// included, unlike the topology RangeBox, which only reads vertices and edge curves (a sphere's
// equator bulge needs this).
//
// It reads no tessellation (M48/C3, Oblikovati/Oblikovati#3421): a facet chord lies INSIDE the true
// surface, so a mesh box under-measures every convex bulge by the sagitta and any classification or
// culling keyed on it inherits that error — and the error moves with the display Quality. q survives
// only for the per-face fallback below.
//
// Example: box := ops.PreciseRangeBox(sphereBody, ops.DefaultQuality()) // spans the full diameter
func PreciseRangeBox(b *topo.Body, q Quality) math.Box {
	box := math.EmptyBox()
	for _, f := range b.Faces() {
		box = unionNonEmpty(box, analyticFaceBox(f, q))
	}
	for _, w := range b.Wires() {
		box = wireExtend(box, w)
	}
	return box
}

// analyticFaceBox bounds one trimmed face from its analytic geometry. A coordinate's extremum over
// the face is attained either on the boundary curves — bounded exactly by faceBoundaryBox — or at an
// interior point where the surface normal is parallel to that axis, which
// geom.SurfaceAxisCriticalPoints enumerates in closed form; a critical point counts only when it
// lies inside THIS face's trim. A boundary-less face (a whole sphere) has no boundary curves, so its
// surface's own box is the answer. A surface kind whose interior extrema cannot be enumerated as
// isolated points (a torus about a world axis, a NURBS patch) falls back to the face tessellation at
// q — a named fallback, not a silent degrade.
func analyticFaceBox(f *topo.Face, q Quality) math.Box {
	crit, ok := geom.SurfaceAxisCriticalPoints(f.Geometry())
	if !ok {
		return tessellatedFaceBox(f, q)
	}
	if len(f.Loops()) == 0 {
		return geom.SurfaceRangeBox(f.Geometry())
	}
	box := faceBoundaryBox(f)
	for _, p := range crit {
		if brep.PointInFaceTrim(f, p) {
			box = box.ExtendPoint(p)
		}
	}
	return box
}

// faceBoundaryBox bounds a face's boundary curves exactly, taking each analytic curve's per-axis
// extrema in closed form (geom.CurveRangeBox3) rather than sampling them.
func faceBoundaryBox(f *topo.Face) math.Box {
	box := math.EmptyBox()
	for _, e := range f.Edges() {
		box = unionNonEmpty(box, edgeCurveBox(e))
	}
	return box
}

// edgeCurveBox bounds one edge's curve. An edge on an UNBOUNDED curve (a line, a parabola) has no
// finite closed-form curve box, so it is bounded by walking its own parameter domain — the curve
// discretizer, which is a derived view of the exact curve and not a face mesh.
func edgeCurveBox(e *topo.Edge) math.Box {
	if box := geom.CurveRangeBox3(e.Geometry()); !boxUnbounded(box) {
		return box
	}
	lo, hi := e.Geometry().Domain()
	if stdmath.IsInf(lo, 0) || stdmath.IsInf(hi, 0) {
		return math.BoxFromPoints(e.StartVertex().Point(), e.EndVertex().Point())
	}
	return math.BoxFromPoints(geom.SampleCurve3(e.Geometry(), preciseCurveSamples)...)
}

// preciseCurveSamples is the walk count for an edge whose curve has no finite closed-form box.
const preciseCurveSamples = 32

// tessellatedFaceBox is the fallback box for a face whose surface kind has interior extrema that
// cannot be enumerated in closed form. It under-measures a bulge by the facet sagitta; it is used
// only where nothing exact exists yet.
func tessellatedFaceBox(f *topo.Face, q Quality) math.Box {
	return math.BoxFromPoints(TessellateFace(f, q).Positions...)
}

// boxUnbounded reports whether any of the box's corners runs off to infinity — the marker
// geom.CurveRangeBox3 returns for an unbounded curve.
func boxUnbounded(box math.Box) bool {
	return stdmath.IsInf(float64(box.Min.X), 0) || stdmath.IsInf(float64(box.Min.Y), 0) ||
		stdmath.IsInf(float64(box.Min.Z), 0) || stdmath.IsInf(float64(box.Max.X), 0) ||
		stdmath.IsInf(float64(box.Max.Y), 0) || stdmath.IsInf(float64(box.Max.Z), 0)
}

// unionNonEmpty extends box by o, skipping an EMPTY o: math.Box.Union extends by o's corners, and
// the empty box's corners are ±Inf, which would make the union unbounded.
func unionNonEmpty(box, o math.Box) math.Box {
	if o.IsEmpty() {
		return box
	}
	return box.Union(o)
}
