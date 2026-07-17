// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"fmt"

	stdmath "math"

	"oblikovati.org/math"
)

// loftCanal builds the constant-radius rolling-ball CORNER PATCH: the pipe (canal) surface swept
// by the radius-`radius` cross-section arc as its center walks the spine. It is emitted as a
// degree-2-rational-in-u × spline-in-v NURBS (exactly OCCT's 2×9, 3×10 u-rational form) by lofting
// the three arc control-curves — the fa foot-locus (weight 1), the shoulder locus (weight cos β,
// VARYING with v), the fb foot-locus (weight 1) — in HOMOGENEOUS coordinates (Piegl & Tiller §10.3;
// canal-corner-math.md §4). The parametrization is the watertight crux (advisory §3): u = the arc
// parameter (u=0 → fa, u=1 → fb) and v = chord-length along the spine, so ALL FOUR boundary
// isoparms are rails — v=0/v=1 the two end cross-section arcs, u=0/u=1 the two foot-loci — which
// C3 emits Loops on for a watertight weld.
//
// The spine is MARCHED (canalSpine C1), so its exact vertices (SpineVertices, on-host to ~2e-9) are
// lofted with an INTERPOLATING v-spline (advisory §4: exact-loft when closed-form, interpolate the
// stations when marched). It reads the spine VERTICES, never PointAt(t) — a polyline's off-vertex
// points ride the chord sag (C1 contract). res gates the do-no-harm feet-at-radius check.
//
//	surf, err := loftCanal(spine, [2]Surface{wall, s10}, 5, res)
func loftCanal(spine Curve3, hosts [2]Surface, radius float64, res Resolution) (BSplineSurface, error) {
	verts, ok := SpineVertices(spine)
	if !ok {
		return BSplineSurface{}, fmt.Errorf("loftCanal: spine %T exposes no vertices; need a marched canal spine", spine)
	}
	if len(verts) < 2 {
		return BSplineSurface{}, fmt.Errorf("loftCanal: spine has %d vertices, need >= 2 to loft", len(verts))
	}
	cols, err := stationColumns(verts, hosts, radius, res)
	if err != nil {
		return BSplineSurface{}, err
	}
	vParams, err := alphaParams(coords3(verts), 1) // chord-length spine parametrization (P&T §9.2.1)
	if err != nil {
		return BSplineSurface{}, fmt.Errorf("loftCanal: spine chord-length params: %w", err)
	}
	vDeg, _ := fitDegree(len(verts)) // len(verts) >= 2 guaranteed above → no error
	return assembleCanalLoft(cols, vParams, vDeg)
}

// arcStation is one v-station's cross-section arc control triple + shoulder weight: the column of
// the 3×N control net the loft assembles.
type arcStation struct {
	fa, shoulder, fb math.Point3
	weight           float64
}

// stationColumns computes the cross-section arc control triple (fa, shoulder, fb) and weight at
// every spine vertex: the two rolling-ball feet (closest point on each host) plus the arc controls
// between them. It errors (do-no-harm floor) when a foot is not at ball distance from its host, or
// the feet graze — the caller (canalProvider) then falls through to coons4.
func stationColumns(verts []math.Point3, hosts [2]Surface, radius float64, res Resolution) ([]arcStation, error) {
	tol := res.Weld()
	cols := make([]arcStation, len(verts))
	for j, m := range verts {
		fa, err := ballFoot(hosts[0], m, radius, tol)
		if err != nil {
			return nil, fmt.Errorf("loftCanal station %d (%v): host A: %w", j, m, err)
		}
		fb, err := ballFoot(hosts[1], m, radius, tol)
		if err != nil {
			return nil, fmt.Errorf("loftCanal station %d (%v): host B: %w", j, m, err)
		}
		shoulder, weight, err := arcControls(m, fa, fb, radius)
		if err != nil {
			return nil, fmt.Errorf("loftCanal station %d (%v): %w", j, m, err)
		}
		cols[j] = arcStation{fa: fa, shoulder: shoulder, fb: fb, weight: weight}
	}
	return cols, nil
}

// ballFoot returns the rolling-ball contact point on host from center m (its closest point), which
// for a valid canal spine sits exactly `radius` away — the ball touches. It errors when that
// distance strays from radius by more than tol (the center is not at ball distance from this host).
func ballFoot(host Surface, m math.Point3, radius, tol float64) (math.Point3, error) {
	_, _, foot := ClosestPointOnSurface(host, m)
	if d := stdmath.Abs(float64(foot.DistanceTo(m)) - radius); d > tol {
		return math.Point3{}, fmt.Errorf(
			"ball center %v is %g off radius %g from host %T (tol %g)", m, d, radius, host, tol)
	}
	return foot, nil
}

// assembleCanalLoft lofts the three pole-rows into the degree-2-rational-u × degree-vDeg-spline-v
// surface: the two foot-loci as weight-1 interpolants, the shoulder locus in homogeneous 4-D so the
// varying cos-β weight stays exact (pitfall 5). All three share one chord-length v-knot vector
// (identical params + degree → identical averaged knots), so the net is rectangular.
func assembleCanalLoft(cols []arcStation, vParams []float64, vDeg int) (BSplineSurface, error) {
	faRow, vKnots, err := interpEuclideanRow(footRow(cols, true), vDeg, vParams)
	if err != nil {
		return BSplineSurface{}, fmt.Errorf("loftCanal: fa foot-locus interpolation: %w", err)
	}
	fbRow, _, err := interpEuclideanRow(footRow(cols, false), vDeg, vParams)
	if err != nil {
		return BSplineSurface{}, fmt.Errorf("loftCanal: fb foot-locus interpolation: %w", err)
	}
	shRow, shWeights, err := interpHomogeneousRow(cols, vDeg, vParams)
	if err != nil {
		return BSplineSurface{}, fmt.Errorf("loftCanal: shoulder-locus interpolation: %w", err)
	}
	ones := onesRow(len(cols))
	ctrl := [][]math.Point3{faRow, shRow, fbRow}
	weights := [][]float64{ones, shWeights, ones}
	return NewBSplineSurface(2, vDeg, ctrl, weights, []float64{0, 0, 0, 1, 1, 1}, vKnots)
}

// interpEuclideanRow globally interpolates a weight-1 pole row (a foot-locus) through pts at the
// shared chord-length v-params, returning the control points and the shared v-knot vector.
func interpEuclideanRow(pts []math.Point3, vDeg int, vParams []float64) ([]math.Point3, []float64, error) {
	ctrl, knots, err := fitInterpolationAt(coords3(pts), vDeg, vParams)
	if err != nil {
		return nil, nil, err
	}
	return points3(ctrl), knots, nil
}

// interpHomogeneousRow interpolates the shoulder pole-row in HOMOGENEOUS 4-D coordinates
// (w·x, w·y, w·z, w) so the rational skin stays exact along v (P&T §10.3; pitfall 5 — never
// interpolate positions and weights separately). It projects the interpolated homogeneous controls
// back to (point, weight). The knots equal interpEuclideanRow's (same params + degree), so discarded.
func interpHomogeneousRow(cols []arcStation, vDeg int, vParams []float64) ([]math.Point3, []float64, error) {
	hom, _, err := fitInterpolationAt(shoulderHomogeneous(cols), vDeg, vParams)
	if err != nil {
		return nil, nil, err
	}
	return projectHomogeneous(hom)
}

// shoulderHomogeneous lifts each station's shoulder to its weighted homogeneous 4-tuple.
func shoulderHomogeneous(cols []arcStation) [][]float64 {
	out := make([][]float64, len(cols))
	for i, c := range cols {
		w := c.weight
		out[i] = []float64{w * float64(c.shoulder.X), w * float64(c.shoulder.Y), w * float64(c.shoulder.Z), w}
	}
	return out
}

// projectHomogeneous divides each interpolated homogeneous control (X, Y, Z, W) by W back to a
// Euclidean control point + weight W. It errors on a zero weight (a projected point at infinity).
func projectHomogeneous(hom [][]float64) ([]math.Point3, []float64, error) {
	pts := make([]math.Point3, len(hom))
	weights := make([]float64, len(hom))
	for i, r := range hom {
		w := r[3]
		if w == 0 {
			return nil, nil, fmt.Errorf("loftCanal: homogeneous shoulder weight 0 at control %d (point at infinity)", i)
		}
		pts[i] = math.P3(math.Scalar(r[0]/w), math.Scalar(r[1]/w), math.Scalar(r[2]/w))
		weights[i] = w
	}
	return pts, weights, nil
}

// footRow extracts one foot-locus pole row: the fa column when first, else the fb column.
func footRow(cols []arcStation, first bool) []math.Point3 {
	out := make([]math.Point3, len(cols))
	for i, c := range cols {
		if first {
			out[i] = c.fa
		} else {
			out[i] = c.fb
		}
	}
	return out
}

// onesRow returns n unit weights (the weight-1 foot-locus pole rows).
func onesRow(n int) []float64 {
	out := make([]float64, n)
	for i := range out {
		out[i] = 1
	}
	return out
}
