// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// notchedTargetBody builds the r=3 h=10 cylinder already notched by the first oblique cut (plane x+z≤9.5) —
// the already-cut target the corner-junction fixtures cut a second time.
func notchedTargetBody(t *testing.T) *topo.Body {
	t.Helper()
	bare, err := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 10)
	if err != nil {
		t.Fatalf("bare cylinder: %v", err)
	}
	pl, _ := geom.NewPlane(math.P3(1.5, 0, 8), math.V3(1, 0, 1))
	target, err := HalfSpaceCut(bare, pl)
	if err != nil {
		t.Fatalf("notch cut: %v", err)
	}
	return target
}

// Slice A (EPIC #1738, ADR-0048): the exact triple-point solver. The corner-junction fixture is the notched
// r=3 cylinder cut by a rod r=1 axis +x at z=7; the rod's front loop crosses the first-cut plane x+z=9.5 at
// two points that lie on ALL THREE surfaces (target cylinder, rod, notch plane) — the shared vertices the
// coupled faces re-emit to. Hand-derived (Method C): on C_P(u)=(3cos u,3sin u,9.5−3cos u), the rod implicit
// gives cos u=0.95 → (2.85, ±0.9367, 6.65).

func TestCornerJunctionsAreExactTriplePoints(t *testing.T) {
	target := notchedTargetBody(t)
	_, _, _, prior, ok := cutCylinderSideFace(target)
	if !ok {
		t.Fatal("notched target has no recognisable cut cylinder side")
	}
	rodAxisPt, rodAxis, rodR := math.P3(-6, 0, 7), math.V3(1, 0, 0), 1.0
	js := cornerJunctions(prior, rodAxisPt, rodAxis, rodR)
	if len(js) != 2 {
		t.Fatalf("cornerJunctions found %d triple points; want 2 (the front loop crosses the notch twice)", len(js))
	}
	for _, j := range js {
		p := j.point
		if d := rodRadialDist(p, rodAxisPt, rodAxis); stdmath.Abs(d-rodR) > 1e-7 {
			t.Errorf("junction (%.5f,%.5f,%.5f) rod-axis distance %.9f; want %.1f (on rod surface)", p.X, p.Y, p.Z, d, rodR)
		}
		if r := stdmath.Hypot(float64(p.X), float64(p.Y)); stdmath.Abs(r-3) > 1e-7 {
			t.Errorf("junction (%.5f,%.5f,%.5f) target-axis distance %.9f; want 3 (on target cylinder)", p.X, p.Y, p.Z, r)
		}
		if g := float64(p.X) + float64(p.Z) - 9.5; stdmath.Abs(g) > 1e-7 {
			t.Errorf("junction (%.5f,%.5f,%.5f) plane residual x+z-9.5=%.9f; want 0 (on notch plane)", p.X, p.Y, p.Z, g)
		}
		if stdmath.Abs(float64(p.X)-2.85) > 1e-3 || stdmath.Abs(float64(p.Z)-6.65) > 1e-3 || stdmath.Abs(stdmath.Abs(float64(p.Y))-0.9367) > 1e-3 {
			t.Errorf("junction (%.5f,%.5f,%.5f) not at the hand-derived (2.85,±0.9367,6.65)", p.X, p.Y, p.Z)
		}
	}
	// Both crossings are transversal (the rod pierces the notch boundary, not grazes it): both degeneracy
	// sines sit well clear of 0, so the tangency gate keeps this config on the analytic path.
	for _, j := range js {
		surfSurf, curveCurve := junctionDegeneracy(j, prior, math.P3(0, 0, 0), math.V3(0, 0, 1), rodAxisPt, rodAxis)
		if surfSurf < 0.05 {
			t.Errorf("junction surface-surface sine %.4f too small — surfaces near tangent (unexpected here)", surfSurf)
		}
		if curveCurve < 0.05 {
			t.Errorf("junction curve-curve sine %.4f too small — boundaries near graze (unexpected here)", curveCurve)
		}
	}
}

// TestCornerPreSplitSharesExactVertices is the watertightness precondition (ADR-0048): after the pre-split
// every triple point is an EXACT endpoint of both a prior sub-edge AND an imprint sub-arc, so the arrangement
// re-emits one shared 3D vertex and the weld closes. Without this the two arms re-anchor to a chord point vs
// an on-cylinder point (~2.6e-4 apart) and the junction splits (the χ=1/free=5 bug).
func TestCornerPreSplitSharesExactVertices(t *testing.T) {
	target := notchedTargetBody(t)
	rod, err := SolidCylinder(math.P3(-6, 0, 7), math.V3(1, 0, 0), 1, 12)
	if err != nil {
		t.Fatalf("rod: %v", err)
	}
	loops, ok := require2Loops(partialRimImprint(target, rod, &diag.Recorder{}))
	if !ok {
		t.Fatalf("imprint did not trace as two loops")
	}
	_, _, _, prior, _ := cutCylinderSideFace(target)
	js := cornerJunctions(prior, math.P3(-6, 0, 7), math.V3(1, 0, 0), 1)

	sp := splitPriorAtJunctions(prior, js)
	if len(sp.edges) != len(prior.edges)+len(js) {
		t.Errorf("split prior has %d edges; want %d (+1 per junction)", len(sp.edges), len(prior.edges)+len(js))
	}
	si := splitImprintAtJunctions(loops, js)

	for _, j := range js {
		if !endpointNear(priorEdgeEndpoints(sp.edges), j.point, 1e-9) {
			t.Errorf("triple point (%.5f,%.5f,%.5f) is not an exact prior sub-edge endpoint", j.point.X, j.point.Y, j.point.Z)
		}
		if !endpointNear(imprintEndpoints(si), j.point, 1e-9) {
			t.Errorf("triple point (%.5f,%.5f,%.5f) is not an exact imprint sub-arc endpoint", j.point.X, j.point.Y, j.point.Z)
		}
	}
	// The disjoint back entry loop must survive as ONE closed curve (complete loop ingest, no split).
	// Closedness is asked of the curve, not of its type: the loop is an exact ruled∩quadric section now,
	// and only splits into open sub-arcs when a junction actually lies on it (#3489).
	closed := 0
	for _, cv := range si {
		if geom.CurveIsClosed(cv) {
			closed++
		}
	}
	if closed != 1 {
		t.Errorf("split imprint has %d closed loops; want 1 (the untouched back entry loop)", closed)
	}
}

// TestRodNotchSectionPassesThroughTriplePoints: the shared rod∩notch-plane ellipse (which trims the tunnel
// and bites the notch cap) must pass through BOTH triple points, so all three coupled faces meet there.
func TestRodNotchSectionPassesThroughTriplePoints(t *testing.T) {
	target := notchedTargetBody(t)
	_, _, _, prior, _ := cutCylinderSideFace(target)
	notch, ok := recoverNotchPlane(prior)
	if !ok {
		t.Fatal("could not recover the notch plane from the prior boundary")
	}
	if n := notch.Normal(); stdmath.Abs(float64(n.X)-float64(n.Z)) > 1e-9 {
		t.Errorf("notch normal (%.4f,%.4f,%.4f) not proportional to (1,0,1)", n.X, n.Y, n.Z)
	}
	rod, _ := SolidCylinder(math.P3(-6, 0, 7), math.V3(1, 0, 0), 1, 12)
	rodCyl, _, _, _ := cylinderSolidParams(facesOfAny(rod))
	res := geom.ResolutionForSize(20)
	sec, ok := rodNotchSection(notch, rodCyl, res)
	if !ok {
		t.Fatal("rod∩notch-plane section unavailable")
	}
	// Every section sample lies on BOTH the notch plane and the rod surface — confirms the correct ellipse.
	for i := 0; i <= 256; i++ {
		p := sec.PointAt(float64(i) / 256)
		if g := float64(p.X) + float64(p.Z) - 9.5; stdmath.Abs(g) > 1e-7 {
			t.Fatalf("section sample off the notch plane by %.9f", g)
		}
		if d := rodRadialDist(p, math.P3(-6, 0, 7), math.V3(1, 0, 0)); stdmath.Abs(d-1) > 1e-7 {
			t.Fatalf("section sample off the rod surface by %.9f", d-1)
		}
	}
	// Each triple point lies on the section ellipse — established as "on the notch plane AND on the rod
	// surface" (the ellipse is exactly plane∩rod), the same invariant TestCornerJunctionsAreExactTriplePoints
	// asserts to 1e-7. So the shared section curve and the shared vertices are the same geometry.
	for _, j := range cornerJunctions(prior, math.P3(-6, 0, 7), math.V3(1, 0, 0), 1) {
		onPlane := stdmath.Abs(float64(j.point.X)+float64(j.point.Z)-9.5) < 1e-7
		onRod := stdmath.Abs(rodRadialDist(j.point, math.P3(-6, 0, 7), math.V3(1, 0, 0))-1) < 1e-7
		if !onPlane || !onRod {
			t.Errorf("triple point (%.4f,%.4f,%.4f) not on plane∩rod (onPlane=%v onRod=%v)", j.point.X, j.point.Y, j.point.Z, onPlane, onRod)
		}
	}
}

// TestCornerJunctionGateAcceptsTransversalDeclinesGrazing pins the scale-invariant tangency gate (Slice B,
// ADR-0048): a clean transversal crossing (rod at z=7, its front loop piercing the notch) is accepted, while
// a rod that only GRAZES the notch floor (z=5.5, its top tangent to the section) declines — the bias-toward-
// decline that keeps a shallow crossing off the analytic path.
func TestCornerJunctionGateAcceptsTransversalDeclinesGrazing(t *testing.T) {
	for _, tc := range []struct {
		z    float64
		want bool
	}{{7.0, true}, {5.5, false}} {
		target := notchedTargetBody(t)
		rod, err := SolidCylinder(math.P3(-6, 0, math.Scalar(tc.z)), math.V3(1, 0, 0), 1, 12)
		if err != nil {
			t.Fatalf("rod z=%.1f: %v", tc.z, err)
		}
		_, cyl, _, prior, _ := cutCylinderSideFace(target)
		rodCyl, _, _, _ := cylinderSolidParams(facesOfAny(rod))
		js := cornerJunctions(prior, rodCyl.Origin, rodCyl.AxisDir.AsVector(), rodCyl.Radius)
		if got := cornerJunctionTransversal(js, prior, cyl, rodCyl); got != tc.want {
			t.Errorf("z=%.1f: transversal gate = %v (%d junctions); want %v", tc.z, got, len(js), tc.want)
		}
	}
}

// TestCornerJunctionScaleInvariantAngle confirms the gate measures the TRUE geometric crossing angle, equal
// on cylinders of vastly different radii — the first-fundamental-form quotient (ADR-0048). The same relative
// crossing geometry, scaled ×1000 in radius, yields the same curve-curve sine (no R shear).
func TestCornerJunctionScaleInvariantAngle(t *testing.T) {
	sines := make([]float64, 0, 2)
	for _, s := range []float64{1, 1000} {
		bare, _ := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), math.Scalar(3*s), math.Scalar(10*s))
		pl, _ := geom.NewPlane(math.P3(math.Scalar(1.5*s), 0, math.Scalar(8*s)), math.V3(1, 0, 1))
		target, err := HalfSpaceCut(bare, pl)
		if err != nil {
			t.Fatalf("scale %.0f notch: %v", s, err)
		}
		_, cyl, _, prior, ok := cutCylinderSideFace(target)
		if !ok {
			t.Fatalf("scale %.0f: no cut side", s)
		}
		rodPt, rodAxis, rodR := math.P3(math.Scalar(-6*s), 0, math.Scalar(7*s)), math.V3(1, 0, 0), 1*s
		js := cornerJunctions(prior, rodPt, rodAxis, rodR)
		if len(js) != 2 {
			t.Fatalf("scale %.0f: %d junctions; want 2", s, len(js))
		}
		_, cc := junctionDegeneracy(js[0], prior, cyl.Origin, cyl.AxisDir.AsVector(), rodPt, rodAxis)
		sines = append(sines, cc)
	}
	if stdmath.Abs(sines[0]-sines[1]) > 1e-6 {
		t.Errorf("curve-curve sine drifted with radius: R=1 gave %.8f, R=1000 gave %.8f (want scale-invariant)", sines[0], sines[1])
	}
}

func priorEdgeEndpoints(edges []loopEdge) []math.Point3 {
	pts := make([]math.Point3, 0, 2*len(edges))
	for _, e := range edges {
		pts = append(pts, e.start(), e.end())
	}
	return pts
}

func imprintEndpoints(curves []geom.Curve3) []math.Point3 {
	var pts []math.Point3
	for _, c := range curves {
		pts = append(pts, c.PointAt(0), c.PointAt(1))
	}
	return pts
}

func endpointNear(pts []math.Point3, target math.Point3, tol float64) bool {
	for _, p := range pts {
		if float64(p.DistanceTo(target)) < tol {
			return true
		}
	}
	return false
}
