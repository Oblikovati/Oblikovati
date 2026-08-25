// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
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

// analyticRigidSweep builds the analytic swept solid for a rigid (no taper/twist/scaling/rail/guide,
// NormalToPath) sweep, or nil to fall back to the faceted section skin. A straight run reuses the
// extrude analytic prism; a full circular path with a circle profile is a torus. A partial circular
// path (a pipe elbow) still facets for now — it needs the partial surface-of-revolution builder.
func analyticRigidSweep(prof *sketch.Profile, plane sketch.Plane, path *sketch.Path3D, feat string) *topo.Body {
	if body := analyticStraightSweep(prof, plane, path.Points(), feat); body != nil {
		return body
	}
	return analyticTorusSweep(prof, path, feat)
}

// analyticTorusSweep builds a full torus when a single circle profile is swept along a FULL circular
// path: the circle rides the path (its centroid is the path circle), so the swept tube is the torus
// whose major radius is the path circle radius and minor radius the profile circle radius. nil for a
// non-circle profile, a hole-bearing profile, an open (partial) path, a non-circular path, or a minor
// radius that reaches the major (a self-intersecting spindle torus we do not build).
func analyticTorusSweep(prof *sketch.Profile, path *sketch.Path3D, feat string) *topo.Body {
	if !path.IsClosed() || len(prof.InnerLoops()) != 0 {
		return nil
	}
	circle := circleLoop(prof.OuterLoop())
	if circle == nil {
		return nil
	}
	fit, ok := fitPathCircle(path.Points())
	if !ok {
		return nil
	}
	minor := float64(circle.CurveRadius())
	if minor <= 0 || minor >= fit.Radius {
		return nil
	}
	body, err := brep.SolidTorus(fit.Center, fit.Normal.AsVector(), fit.Radius, minor, feat)
	if err != nil {
		return nil
	}
	return body
}

// fitPathCircle fits the circle through three well-separated path points and confirms every point
// lies on it (in radius AND in the circle's plane). ok is false for fewer than three points, a
// collinear triple, or any point off the circle — the caller then keeps the faceted skin.
func fitPathCircle(pts []math.Point3) (geom.Circle, bool) {
	if len(pts) < 3 {
		return geom.Circle{}, false
	}
	fit, err := geom.CircleByThreePoints(pts[0], pts[len(pts)/3], pts[2*len(pts)/3])
	if err != nil {
		return geom.Circle{}, false
	}
	tol := fit.Radius * 1e-6
	if tol < 1e-7 {
		tol = 1e-7
	}
	for _, p := range pts {
		if stdmath.Abs(float64(fit.Center.DistanceTo(p))-fit.Radius) > tol {
			return geom.Circle{}, false // off the circle radius
		}
		if stdmath.Abs(float64(fit.Center.VectorTo(p).Dot(fit.Normal.AsVector()))) > tol {
			return geom.Circle{}, false // out of the circle's plane
		}
	}
	return fit, true
}

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
