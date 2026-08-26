// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/math"
)

// Exact rational representation of a ruled blend between two circular arcs (#1606, audit A10).
//
// A linear-taper variable fillet on a straight edge sweeps a circular profile arc whose radius
// grows linearly along the edge. The blend it traces is an OBLIQUE circular cone: the profile
// planes stay parallel (perpendicular to the edge) but the arc centres walk the dihedral
// bisector as the radius grows, so no right-circular geom.Cone represents it. It IS exactly
// representable as a NURBS surface: the profile arc is a rational quadratic (Piegl & Tiller
// ch. 7), and ruling two arcs with the SAME sweep in parallel frames is degree 1 in v with
// column-wise identical weights — so the ruled surface reproduces the blend exactly, not as the
// C0 polyhedral strip it used to facet into (~11° creases).

// arcNurbsSegments returns the number of ≤90° rational-quadratic segments an arc of the given
// sweep needs (P&T A7.1's quadrant split).
func arcNurbsSegments(sweep float64) int {
	return int(stdmath.Ceil(sweep/(stdmath.Pi/2) - 1e-12)) // tol:parametric — quadrant count rounding
}

// arcNurbsBasis returns the unit-circle control directions and weights of a rational quadratic
// arc from angle 0 through sweep, in the plane spanned by (xdir, ydir): dir(θ) = cosθ·xdir +
// sinθ·ydir. The caller scales/translates the directions per profile (centre + radius·dir), so
// two arcs sharing one sweep share this basis — the key to an exact ruled blend.
func arcNurbsBasis(xdir, ydir math.Vector3, sweep float64) (dirs []math.Vector3, weights []float64, knots []float64) {
	segs := arcNurbsSegments(sweep)
	d := sweep / float64(segs)
	w := stdmath.Cos(d / 2)
	at := func(theta float64) math.Vector3 {
		return xdir.Scale(stdmath.Cos(theta)).Add(ydir.Scale(stdmath.Sin(theta)))
	}
	dirs = append(dirs, at(0))
	weights = append(weights, 1)
	knots = []float64{0, 0, 0}
	for s := range segs {
		mid := float64(s)*d + d/2
		// The interior control direction sits on the tangent intersection: dir(mid)/cos(d/2).
		dirs = append(dirs, at(mid).Scale(1/w), at(float64(s+1)*d))
		weights = append(weights, w, 1)
		u := float64(s+1) / float64(segs)
		if s+1 < segs {
			knots = append(knots, u, u)
		} else {
			knots = append(knots, 1, 1, 1)
		}
	}
	return dirs, weights, knots
}

// NewRuledArcBlend builds the exact rational ruled surface between two circular arcs that share
// one angular frame: arc v=0 is c0 + r0·dir(θ), arc v=1 is c1 + r1·dir(θ), θ from 0 through
// sweep in the (xdir, ydir) plane. r1 (or r0) may be zero — the column collapses to the apex
// point and the surface is the exact oblique cone of a fillet run-out.
//
//	blend, err := geom.NewRuledArcBlend(c0, r0, c1, r1, xdir, ydir, math.Pi/2)
func NewRuledArcBlend(c0 math.Point3, r0 float64, c1 math.Point3, r1 float64, xdir, ydir math.Vector3, sweep float64) (BSplineSurface, error) {
	if sweep <= 0 || sweep > 2*stdmath.Pi {
		return BSplineSurface{}, fmt.Errorf("geom: ruled arc blend sweep %g out of (0, 2π]", sweep)
	}
	if r0 < 0 || r1 < 0 || (r0 == 0 && r1 == 0) {
		return BSplineSurface{}, fmt.Errorf("geom: ruled arc blend radii (%g, %g) must be ≥ 0 and not both zero", r0, r1)
	}
	dirs, w, uKnots := arcNurbsBasis(xdir, ydir, sweep)
	ctrl := make([][]math.Point3, len(dirs))
	weights := make([][]float64, len(dirs))
	for i, d := range dirs {
		ctrl[i] = []math.Point3{c0.TranslateBy(d.Scale(r0)), c1.TranslateBy(d.Scale(r1))}
		weights[i] = []float64{w[i], w[i]}
	}
	return NewBSplineSurface(2, 1, ctrl, weights, uKnots, []float64{0, 0, 1, 1})
}

// NewConicSectionCurve is the single-segment rational quadratic p0–s–p2 with interior (shoulder)
// weight w — the exact conic every fillet cross-section that is an arc (w = cos of the half
// wedge) or a rho conic (w = rho/(1−rho)) reduces to. It satisfies Curve3, so a blend face's
// boundary edge carries the SAME geometry as the blend surface's end isoline.
func NewConicSectionCurve(p0, s, p2 math.Point3, w float64) (BSplineCurve, error) {
	return NewBSplineCurve(2, []math.Point3{p0, s, p2}, []float64{1, w, 1}, []float64{0, 0, 0, 1, 1, 1})
}

// NewRuledSectionBlend rules two conic sections (p0–s–p2 control triangles sharing shoulder
// weight w) into the exact degree (2,1) rational blend surface — the variable fillet's span
// between two cross-section profiles (#1606). One section may be fully degenerate (all three
// points equal): a fillet run-out's apex column.
func NewRuledSectionBlend(sec0, sec1 [3]math.Point3, w float64) (BSplineSurface, error) {
	if w <= 0 {
		return BSplineSurface{}, fmt.Errorf("geom: ruled section blend shoulder weight %g must be > 0", w)
	}
	weights := []float64{1, w, 1}
	ctrl := make([][]math.Point3, 3)
	wts := make([][]float64, 3)
	for i := range 3 {
		ctrl[i] = []math.Point3{sec0[i], sec1[i]}
		wts[i] = []float64{weights[i], weights[i]}
	}
	return NewBSplineSurface(2, 1, ctrl, wts, []float64{0, 0, 0, 1, 1, 1}, []float64{0, 0, 1, 1})
}
