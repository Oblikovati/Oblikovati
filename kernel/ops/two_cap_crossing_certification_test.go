// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/ops/validate"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Two-cap-exit certification (EPIC Oblikovati/Oblikovati#1724, ADR-0046). A steeper oblique cylinder tool
// enters one target cap and exits the OTHER, staying inside the wall the whole way (an angled through-hole)
// — the sibling the slice-1 recognizer deferred. The result is the wall kept WHOLE, each cap holed by its
// exit ellipse, and the tool wall between the two cap planes reversed into the cavity (the tunnel). Certified
// against OpenCASCADE (gmsh occ.cut getMass) plus a point-membership audit vs the analytic CSG predicate.

// OCC ground truth for the two-cap fixture (base -2.416, 20 deg tool), from gmsh OCC occ.cut + getMass.
const (
	occTwoCapVol   = 266.3615948
	occTwoCapArea  = 288.5728612
	occTwoCapCx    = -0.0197008
	occTwoCapCy    = 0.0
	occTwoCapCz    = 5.0
	occTwoCapFaces = 4 // whole wall + top cap (ellipse hole) + bottom cap (ellipse hole) + tunnel
)

func twoCapTool() (base math.Point3, dir math.Vector3, r, h float64) {
	th := 20.0 * stdmath.Pi / 180
	ux, uz := stdmath.Sin(th), stdmath.Cos(th)
	return math.P3(-2.416, 0, -2.518), math.V3(math.Scalar(ux), 0, math.Scalar(uz)), 0.7, 16
}

func twoCapFixture(t *testing.T) (target, tool *topo.Body) {
	t.Helper()
	tg, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 10)
	if err != nil {
		t.Fatalf("target: %v", err)
	}
	b, d, r, h := twoCapTool()
	tl, err := brep.SolidCylinder(b, d, r, h)
	if err != nil {
		t.Fatalf("tool: %v", err)
	}
	return tg, tl
}

func TestTwoCapCrossingCutIsWatertightAndValid(t *testing.T) {
	t.Parallel()
	target, tool := twoCapFixture(t)
	res, err := Boolean(Cut, target, tool)
	if err != nil {
		t.Fatalf("Boolean(Cut): %v", err)
	}
	if r := Validate(res); !r.Valid {
		t.Errorf("two-cap cut is not a valid solid: manifold=%v closed=%v orient=%v euler=%v issues=%v",
			r.Manifold, r.Closed, r.OrientationOK, r.EulerConsistent, r.Issues)
	}
	for _, gq := range certGateQualities() {
		mesh, _ := tessellate.TessellateBody(res, gq.q)
		if free := freeEdgeCount(mesh); free != 0 {
			t.Errorf("%s quality: two-cap cut tessellated with %d free edges; want 0 — a cross-face crack at a cap ellipse", gq.name, free)
		}
		if folds := validate.FoldEdgeCount(mesh); folds != 0 {
			t.Errorf("%s quality: two-cap cut mesh has %d fold edges; want 0", gq.name, folds)
		}
	}
}

func TestTwoCapCrossingCutMomentsMatchOCC(t *testing.T) {
	t.Parallel()
	target, tool := twoCapFixture(t)
	res, err := Boolean(Cut, target, tool)
	if err != nil {
		t.Fatalf("Boolean(Cut): %v", err)
	}
	if n := len(res.Faces()); n != occTwoCapFaces {
		t.Errorf("two-cap cut has %d faces; want %d (whole wall + 2 holed caps + tunnel)", n, occTwoCapFaces)
	}
	gp := BodyGeometryProperties(res, capCertQuality())
	if rel := stdmath.Abs(gp.Volume-occTwoCapVol) / occTwoCapVol; rel > 0.006 {
		t.Errorf("volume %.4f vs OCC %.4f (rel %.4f > 0.006)", gp.Volume, occTwoCapVol, rel)
	}
	if rel := stdmath.Abs(gp.Area-occTwoCapArea) / occTwoCapArea; rel > 0.004 {
		t.Errorf("area %.4f vs OCC %.4f (rel %.4f > 0.004)", gp.Area, occTwoCapArea, rel)
	}
	occCentroid := math.P3(occTwoCapCx, occTwoCapCy, occTwoCapCz)
	if d := float64(gp.Centroid.DistanceTo(occCentroid)); d > 0.05 {
		t.Errorf("centroid (%.4f,%.4f,%.4f) vs OCC (%.4f,%.4f,%.4f): distance %.4f > 0.05",
			gp.Centroid.X, gp.Centroid.Y, gp.Centroid.Z, occTwoCapCx, occTwoCapCy, occTwoCapCz, d)
	}
}

// TestTwoCapCrossingCutMembershipMatchesCSG samples an interior grid and asserts inside/outside agrees with
// the analytic CSG predicate target\tool — so a right-volume-wrong-shape build still fails.
func TestTwoCapCrossingCutMembershipMatchesCSG(t *testing.T) {
	t.Parallel()
	target, tool := twoCapFixture(t)
	res, err := Boolean(Cut, target, tool)
	if err != nil {
		t.Fatalf("Boolean(Cut): %v", err)
	}
	tb, td, tr, tl := twoCapTool()
	inTarget := func(p math.Point3) bool {
		return stdmath.Hypot(float64(p.X), float64(p.Y)) <= 3 && p.Z >= 0 && p.Z <= 10
	}
	axialOf := func(p math.Point3) float64 { return float64(tb.VectorTo(p).Dot(td)) }
	inTool := func(p math.Point3) bool {
		ax := axialOf(p)
		if ax < 0 || ax > tl {
			return false
		}
		w := tb.VectorTo(p)
		return float64(w.Sub(td.Scale(math.Scalar(ax))).Length()) <= tr
	}
	const shell = 0.15
	nearSurface := func(p math.Point3) bool {
		rWall := stdmath.Abs(stdmath.Hypot(float64(p.X), float64(p.Y)) - 3)
		ax := axialOf(p)
		rTool := stdmath.Abs(float64(tb.VectorTo(p).Sub(td.Scale(math.Scalar(ax))).Length()) - tr)
		zCap := stdmath.Min(stdmath.Abs(float64(p.Z)), stdmath.Abs(float64(p.Z)-10))
		return rWall < shell || rTool < shell || zCap < shell
	}
	mesh, _ := tessellate.TessellateBody(res, DefaultQuality())
	mismatches := 0
	const n = 60
	for i := range n {
		for j := range n {
			for k := range n {
				p := math.P3(math.Scalar(-3+6*(float64(i)+0.5)/n), math.Scalar(-3+6*(float64(j)+0.5)/n), math.Scalar(10*(float64(k)+0.5)/n))
				if nearSurface(p) {
					continue
				}
				if pointInMesh(mesh, p) != (inTarget(p) && !inTool(p)) {
					mismatches++
				}
			}
		}
	}
	if mismatches > 0 {
		t.Errorf("membership audit: %d interior points disagree with target\\tool (#1724 two-cap)", mismatches)
	}
}
