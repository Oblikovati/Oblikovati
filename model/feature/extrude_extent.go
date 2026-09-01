// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"errors"
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// span is the pair of signed offsets (along the sketch-plane normal) the extrude sweeps
// between: from near to far. Every extent mode reduces to a span, which the prism builder
// then turns into geometry.
type span struct{ near, far float64 }

// depth returns the swept length (always non-negative).
func (s span) depth() float64 { return stdmath.Abs(s.far - s.near) }

// throughAllMargin pads a through-all span beyond the material so the boolean fully cuts.
const throughAllMargin = 1.0

// resolveSpan computes the extrude's span from its extent type, the running bodies (for
// through-all / to-next), and any referenced work planes (for to-face / from-to).
func (e *ExtrudeFeature) resolveSpan(bodies []*topo.Body, plane sketch.Plane, polys [][]math.Point2) (span, error) {
	ex := e.def.Extent
	switch ex.Type {
	case DistanceExtent:
		return distanceSpan(ex)
	case ThroughAllExtent:
		return throughAllSpan(ex, bodies, plane)
	case ToNextExtent:
		return toNextSpan(bodies, plane, polys)
	case ToFaceExtent:
		return toPlaneSpan(ex, plane)
	case FromToExtent:
		return fromToSpan(ex, plane)
	case DistanceFromFaceExtent:
		return distanceFromFaceSpan(ex, plane)
	default:
		return span{}, fmt.Errorf("extrude: unsupported extent type %d", ex.Type)
	}
}

// distanceSpan turns a distance extent into a span, honoring the direction and the
// optional asymmetric second distance.
func distanceSpan(ex Extent) (span, error) {
	d := ex.distance()
	if d == 0 {
		return span{}, errors.New("extrude distance is zero")
	}
	if ex.isAsymmetric() {
		return span{near: -ex.distance2(), far: d}, nil
	}
	switch ex.Direction {
	case NegativeDir:
		return span{near: -d, far: 0}, nil
	case SymmetricDir:
		return span{near: -d / 2, far: d / 2}, nil
	default:
		return span{near: 0, far: d}, nil
	}
}

// throughAllSpan spans the existing material along the normal (plus a margin) on the
// side(s) the direction selects, so a cut/join goes through everything.
func throughAllSpan(ex Extent, bodies []*topo.Body, plane sketch.Plane) (span, error) {
	if len(bodies) == 0 {
		return span{}, errors.New("extrude: through-all needs existing material")
	}
	lo, hi := normalExtent(bodies, plane)
	switch ex.Direction {
	case NegativeDir:
		return span{near: lo - throughAllMargin, far: 0}, nil
	case SymmetricDir:
		return span{near: lo - throughAllMargin, far: hi + throughAllMargin}, nil
	default:
		return span{near: 0, far: hi + throughAllMargin}, nil
	}
}

// toPlaneSpan extrudes from the sketch plane to the (parallel) target work plane. A nil target is
// an unresolved to-face selector (its face is absent on the current body — e.g. a geometric target
// naming a face an earlier feature under-built); it recomputes to a clear unhealthy reason rather
// than a hard error, so the caller degrades gracefully.
func toPlaneSpan(ex Extent, plane sketch.Plane) (span, error) {
	if ex.ToPlane == nil {
		return span{}, errors.New("extrude: to-face target face was not found on the current body")
	}
	d, err := signedDistanceToPlane(ex.ToPlane, plane, "to-face")
	if err != nil {
		return span{}, err
	}
	return orderedSpan(0, d), nil
}

// fromToSpan extrudes between two (parallel) target work planes.
func fromToSpan(ex Extent, plane sketch.Plane) (span, error) {
	from, err := signedDistanceToPlane(ex.FromPlane, plane, "from-to start")
	if err != nil {
		return span{}, err
	}
	to, err := signedDistanceToPlane(ex.ToPlane, plane, "from-to end")
	if err != nil {
		return span{}, err
	}
	return orderedSpan(from, to), nil
}

// distanceFromFaceSpan extrudes a distance whose far end is measured from a (parallel)
// target work plane.
func distanceFromFaceSpan(ex Extent, plane sketch.Plane) (span, error) {
	base, err := signedDistanceToPlane(ex.ToPlane, plane, "distance-from-face")
	if err != nil {
		return span{}, err
	}
	d := ex.distance()
	if d == 0 {
		return span{}, errors.New("extrude distance is zero")
	}
	if ex.Direction == NegativeDir {
		d = -d
	}
	return orderedSpan(0, base+d), nil
}

// toNextSpan extrudes up to the next face the profile meets along the normal, found by
// ray-casting each profile vertex; the nearest forward hit sets the far offset.
func toNextSpan(bodies []*topo.Body, plane sketch.Plane, polys [][]math.Point2) (span, error) {
	if len(bodies) == 0 {
		return span{}, errors.New("extrude: to-next needs existing material")
	}
	dir := plane.Normal().AsVector()
	best := stdmath.Inf(1)
	for _, poly := range polys {
		for _, p := range poly {
			origin := plane.ToModel(p)
			for _, b := range bodies {
				if _, t, ok := query.RayCastFaces(b, origin, dir, ops.DefaultQuality()); ok && t > math.DefaultTolerance && t < best {
					best = t
				}
			}
		}
	}
	if stdmath.IsInf(best, 1) {
		return span{}, errors.New("extrude: to-next found no face ahead of the profile")
	}
	return span{near: 0, far: best}, nil
}

// signedDistanceToPlane returns the distance from the sketch plane to a target work plane
// measured along the sketch normal. The target must be parallel to the sketch plane
// (a flat cap); angled/curved targets are kernel phase C.
func signedDistanceToPlane(target *WorkPlane, plane sketch.Plane, what string) (float64, error) {
	if target == nil {
		return 0, fmt.Errorf("extrude: %s has no target plane", what)
	}
	n := plane.Normal().AsVector()
	if !target.Plane().Normal().AsVector().IsParallelTo(n, math.DefaultTolerance) {
		return 0, fmt.Errorf("extrude: %s target must be parallel to the sketch plane (angled trim is not supported yet)", what)
	}
	return plane.Origin().VectorTo(target.Plane().Origin()).Dot(n), nil
}

// orderedSpan returns a span with near ≤ far so the prism builder always sweeps upward.
func orderedSpan(a, b float64) span {
	if a <= b {
		return span{near: a, far: b}
	}
	return span{near: b, far: a}
}

// normalExtent returns the min and max projection of the bodies onto the sketch normal, measured
// from the sketch-plane origin — the material's reach along the extrude (for a through-all cut).
//
// It measures the RANGE BOX corners, not the vertices: an ANALYTIC body (a true cylinder / surface
// of revolution) has only a couple of seam vertices that don't span its swept surface, so a
// vertex-based extent collapsed a through-all cut to a near-zero-depth slab that barely cut the body
// (Oblikovati/Oblikovati#129). The range box is computed from the actual surfaces, so its
// axis-aligned corners bound the body — slightly conservative, which only lengthens a through cut.
func normalExtent(bodies []*topo.Body, plane sketch.Plane) (lo, hi float64) {
	n := plane.Normal().AsVector()
	o := plane.Origin()
	lo, hi = stdmath.Inf(1), stdmath.Inf(-1)
	for _, b := range bodies {
		for _, c := range boxCorners(b.RangeBox()) {
			t := o.VectorTo(c).Dot(n)
			lo, hi = stdmath.Min(lo, t), stdmath.Max(hi, t)
		}
	}
	return lo, hi
}
