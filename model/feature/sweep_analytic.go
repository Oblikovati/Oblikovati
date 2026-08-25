// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// Analytic sweep along a STRAIGHT path (#2164 follow-up). A NormalToPath sweep with no
// taper/twist/scaling/rail/guide is a RIGID sweep: every cross-section is the profile rotated by one
// fixed align (plane normal → path tangent) and translated to the path point. Over a straight run
// (all path points collinear) that is exactly an EXTRUDE along the tangent — so we reuse the extrude
// analytic prism (circle → cylinder, line+arc loop → arc cap edges + partial-cylinder walls) by
// lifting the SAME 2D profile onto a synthetic frame whose normal is the tangent. A swept rod/pipe
// projected onto a sketch then sees real arcs, not the ~48-facet-per-arc chord ring the faceted skin
// produced. Any non-rigid sweep, a bent path, or a hole-bearing profile falls back to the faceted
// section skin (returns nil). Booleans re-facet the analytic body on demand (combine → planarized).

// analyticStraightSweep builds the analytic swept solid for a rigid straight sweep, or nil to signal
// the caller should keep the faceted section skin. The rigid-config gate is the caller's
// (sweepIsRigid); here we only require a straight collinear path and a hole-free profile (holes need
// the drilled path, matching the extrude analytic guard).
func analyticStraightSweep(prof *sketch.Profile, plane sketch.Plane, path []math.Point3, feat string) *topo.Body {
	if len(prof.InnerLoops()) != 0 {
		return nil
	}
	tangent, length, ok := straightPathAxis(path)
	if !ok {
		return nil
	}
	frame, ok := straightSweepFrame(prof, plane, path[0], tangent)
	if !ok {
		return nil
	}
	return analyticPrismOrNil(prof.OuterLoop(), frame, span{near: 0, far: length}, feat)
}

// straightPathAxis returns the unit start→end direction and the run length when every path point is
// collinear (a straight run, so the sweep is a pure translation). ok is false for a zero-length path
// or a bent one — a genuine curve keeps the faceted skin (a circular-arc path is handled analytically
// elsewhere as a torus).
func straightPathAxis(path []math.Point3) (math.UnitVector3, float64, bool) {
	first, last := path[0], path[len(path)-1]
	tangent, err := math.UnitVector3FromVector(first.VectorTo(last))
	if err != nil {
		return math.UnitVector3{}, 0, false // start == end
	}
	length := float64(first.DistanceTo(last))
	tol := straightPathTol(length)
	for _, p := range path[1 : len(path)-1] {
		if offAxisDistance(first, tangent, p) > tol {
			return math.UnitVector3{}, 0, false // a bend: not a straight run
		}
	}
	return tangent, length, true
}

// offAxisDistance is the perpendicular distance of p from the line through base along dir.
func offAxisDistance(base math.Point3, dir math.UnitVector3, p math.Point3) float64 {
	v := base.VectorTo(p)
	along := v.Dot(dir.AsVector())
	perp := v.Sub(dir.AsVector().Scale(along))
	return float64(perp.Length())
}

// straightPathTol is the collinearity tolerance, scaled to the run length so a large sweep is not
// held to an absolute micron while a tiny one is not welded away.
func straightPathTol(length float64) float64 {
	tol := length * 1e-7
	if tol < 1e-7 {
		return 1e-7
	}
	return tol
}

// straightSweepFrame builds the synthetic sketch plane the profile rides on for a rigid straight
// sweep: the profile's sketch frame rotated by align (plane normal → tangent) and translated so the
// profile centroid lands on the path start. ToModel on this frame reproduces the sweep placement
// exactly — path[0] + align·(plane.ToModel(p) − centroid) — so the analytic prism it raises equals
// the faceted skin. ok is false if the rotated axes degenerate (they cannot: align is a rotation).
func straightSweepFrame(prof *sketch.Profile, plane sketch.Plane, start math.Point3, tangent math.UnitVector3) (sketch.Plane, bool) {
	align := math.RotateBetween(plane.Normal(), tangent)
	centroid := centroidOf(modelPolygon(prof, plane))
	origin := start.TranslateBy(align.TransformVector(centroid.VectorTo(plane.Origin())))
	xa, err := math.UnitVector3FromVector(align.TransformVector(plane.XAxis().AsVector()))
	if err != nil {
		return sketch.Plane{}, false
	}
	ya, err := math.UnitVector3FromVector(align.TransformVector(plane.YAxis().AsVector()))
	if err != nil {
		return sketch.Plane{}, false
	}
	frame, err := sketch.NewPlane(origin, xa, ya)
	if err != nil {
		return sketch.Plane{}, false
	}
	return frame, true
}
