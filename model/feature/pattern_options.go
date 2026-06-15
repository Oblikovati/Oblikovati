// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"oblikovati.org/api/types"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Pattern definition option surface (M20-F18, #652). Beyond a count and spacing, the
// reference rectangular/circular pattern definitions carry: how a direction's spacing
// value is interpreted, how each occurrence is computed and oriented, the positioning
// method, and an optional boundary that clips the run. Compute/Orientation/Positioning
// are carried here so they round-trip and are addressable; the spacing interpretation and
// the boundary clip change the produced geometry today. The zero value of every field
// reproduces the legacy behavior, so an existing pattern is unaffected.

// PatternOptions is the shared option block both pattern definitions embed.
type PatternOptions struct {
	Spacing     types.PatternSpacingType
	Compute     types.PatternComputeType
	Orientation types.PatternOrientation
	Positioning types.PatternPositioningMethod
	Boundary    *PatternBoundary
}

// PatternBoundary clips a pattern to a closed loop: an occurrence is kept only when its
// test point (the running body's centre, transformed into place) lies inside the polygon,
// projected into the boundary plane. Inclusion records which point the reference tests by;
// all three reduce to the same centre-point probe in this first implementation.
type PatternBoundary struct {
	Plane     geom.Plane
	Polygon   []math.Point2
	Inclusion types.PatternBoundaryInclusion
}

// rectStep returns the per-occurrence offset for a direction given its raw step vector and
// the occurrence count: a "fitted" pattern treats the step as the TOTAL span and divides it
// across the gaps, every other spacing keeps it as the gap itself.
func (o PatternOptions) rectStep(step math.Vector3, count int) math.Vector3 {
	if o.Spacing == types.SpacingFitted && count > 1 {
		return step.Scale(1 / float64(count-1))
	}
	return step
}

// circIncrement returns the angle between adjacent circular occurrences: "between" treats
// the angle as the per-occurrence increment, "fitted" spreads count occurrences across the
// angle inclusive of both ends, and the unset default divides the angle by the count (the
// legacy full-sweep behavior).
func (o PatternOptions) circIncrement(angle float64, count int) float64 {
	if count <= 1 {
		return 0
	}
	switch o.Spacing {
	case types.SpacingBetween:
		return angle
	case types.SpacingFitted:
		return angle / float64(count-1)
	default:
		return angle / float64(count)
	}
}

// clippedOccurrences returns the occurrence indices the boundary excludes. Occurrence 0
// (the seed) is never clipped; a nil/degenerate boundary or a missing seed clips nothing.
func (o PatternOptions) clippedOccurrences(transforms []math.Matrix4, seed math.Point3, hasSeed bool) map[int]bool {
	clipped := map[int]bool{}
	if !hasSeed || o.Boundary == nil || len(o.Boundary.Polygon) < 3 {
		return clipped
	}
	for k := 1; k < len(transforms); k++ {
		if !o.Boundary.contains(transforms[k].TransformPoint(seed)) {
			clipped[k] = true
		}
	}
	return clipped
}

// NewPatternBoundary builds a clipping boundary from a closed loop of 3D points and the
// plane (origin + normal) they are projected into. The polygon is stored in the plane's
// (u,v) frame; callers pass model-space points and need not know the basis.
func NewPatternBoundary(origin math.Point3, normal math.Vector3, polygon []math.Point3, inclusion types.PatternBoundaryInclusion) (*PatternBoundary, error) {
	plane, err := geom.NewPlane(origin, normal)
	if err != nil {
		return nil, err
	}
	poly := make([]math.Point2, len(polygon))
	for i, p := range polygon {
		poly[i] = planeProject2D(plane, p)
	}
	return &PatternBoundary{Plane: plane, Polygon: poly, Inclusion: inclusion}, nil
}

// contains reports whether p, projected into the boundary plane, is inside the polygon.
func (b *PatternBoundary) contains(p math.Point3) bool {
	return pointInsidePolygon2(planeProject2D(b.Plane, p), b.Polygon)
}

// planeProject2D drops a model point into a plane's (u,v) coordinates.
func planeProject2D(pl geom.Plane, p math.Point3) math.Point2 {
	d := pl.Origin.VectorTo(p)
	return math.P2(d.Dot(pl.UAxis.AsVector()), d.Dot(pl.VAxis.AsVector()))
}

// pointInsidePolygon2 is an even–odd ray-cast point-in-polygon test.
func pointInsidePolygon2(p math.Point2, poly []math.Point2) bool {
	in := false
	for i, n := 0, len(poly); i < n; i++ {
		a, b := poly[i], poly[(i+1)%n]
		if (a.Y > p.Y) != (b.Y > p.Y) {
			x := a.X + (p.Y-a.Y)/(b.Y-a.Y)*(b.X-a.X)
			if p.X < x {
				in = !in
			}
		}
	}
	return in
}

// seedCentre returns the centre of the first running body's range box — the representative
// point boundary clipping moves into each occurrence position. ok is false with no bodies.
func seedCentre(bodies []*topo.Body) (math.Point3, bool) {
	for _, b := range bodies {
		if b != nil {
			return b.RangeBox().Center(), true
		}
	}
	return math.Point3{}, false
}
