// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// Link 3 of the torus∩cylinder far-runout chain: the geom.TorusCylinderArc section curve. These tests pin
// that the feet-bracketed Newton continuation is EXACT on the torus (by construction — PointAt returns
// Torus.PointAt) and on the capping cylinder to well below the far-runout floor (5.8e-8), and that the fold
// guard honest-rejects a COAXIAL capping cylinder (g′(u)≡0 everywhere — the whole section is latitude
// circles, not a single u(v)-graph). Fixtures are synthetic parallel-axis tori/cylinders with exact feet.

// parallelAxisSection is a named fixture: a torus (centre O, axis ẑ, R′=50, r=10) and a capping cylinder
// coaxial-in-direction but offset (axis ẑ through (30,0,·), R₂=40). Its axes are PARALLEL, so each v-slice
// is a benign circle∩circle with the exact azimuth cos u_v = (ρ(v)² − 700)/(60·ρ(v)), ρ(v)=50+10·cos v.
type parallelAxisSection struct {
	torus Torus
	cyl   Cylinder
}

func newParallelAxisSection(t *testing.T) parallelAxisSection {
	t.Helper()
	tor, err := NewTorusWithRef(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), 50, 10)
	if err != nil {
		t.Fatalf("fixture torus: %v", err)
	}
	cyl, err := NewCylinder(math.P3(30, 0, 0), math.V3(0, 0, 1), 40)
	if err != nil {
		t.Fatalf("fixture capping cylinder: %v", err)
	}
	return parallelAxisSection{torus: tor, cyl: cyl}
}

// footAzimuth returns the exact on-cylinder azimuth u at tube angle v for the fixture (arccos of the slice's
// circle∩circle solution); the +arccos branch is the +y-side foot.
func (s parallelAxisSection) footAzimuth(v float64) float64 {
	rho := s.torus.MajorRadius + s.torus.MinorRadius*stdmath.Cos(v)
	return stdmath.Acos((rho*rho - 700) / (60 * rho))
}

// cylRadialResidual is the point's signed distance to the capping cylinder (|(p−O₂)⊥â₂| − R₂).
func (s parallelAxisSection) cylRadialResidual(p math.Point3) float64 {
	a2 := s.cyl.AxisDir.AsVector()
	e := s.cyl.Origin.VectorTo(p)
	perp := e.Sub(a2.Scale(e.Dot(a2)))
	return stdmath.Abs(float64(perp.Length()) - s.cyl.Radius)
}

// torusRadialResidual is the point's exact distance to the torus surface (|hypot(radial−R′, axial) − r|).
func torusRadialResidual(tr Torus, p math.Point3) float64 {
	axis := tr.AxisDir.AsVector()
	q := tr.Center.VectorTo(p)
	za := float64(q.Dot(axis))
	radial := float64(q.Sub(axis.Scale(math.Scalar(za))).Length())
	return stdmath.Abs(stdmath.Hypot(radial-tr.MajorRadius, za) - tr.MinorRadius)
}

// TestTorusCylinderArc_SectionResidual: the continuation section is exact on the torus and on the capping
// cylinder across the whole arc, from feet at v=0 (ρ=60) to v=π/2 (ρ=50).
func TestTorusCylinderArc_SectionResidual(t *testing.T) {
	s := newParallelAxisSection(t)
	v0, v1 := 0.0, stdmath.Pi/2
	u0, u1 := s.footAzimuth(v0), s.footAzimuth(v1)
	weld := 1e-7
	arc, ok := NewTorusCylinderArc(s.torus, s.cyl, v0, u0, v1, u1, weld)
	if !ok {
		t.Fatal("NewTorusCylinderArc declined a benign parallel-axis section")
	}
	for i := 0; i <= 40; i++ {
		p := arc.PointAt(float64(i) / 40)
		if d := torusRadialResidual(s.torus, p); d > 1e-9 {
			t.Fatalf("sample %d off the torus by %.3e (must be exact-on-arm)", i, d)
		}
		if d := s.cylRadialResidual(p); d > 5.8e-8 {
			t.Fatalf("sample %d off the capping cylinder by %.3e (> 5.8e-8 far-runout floor)", i, d)
		}
	}
	// Endpoints land on the feet.
	if d := float64(arc.PointAt(0).DistanceTo(s.torus.PointAt(u0, v0))); d > 1e-9 {
		t.Fatalf("PointAt(0) off foot0 by %.3e", d)
	}
	if d := float64(arc.PointAt(1).DistanceTo(s.torus.PointAt(u1, v1))); d > 1e-9 {
		t.Fatalf("PointAt(1) off foot1 by %.3e", d)
	}
}

// TestTorusCylinderArc_FoldRejectsCoaxialCapping: a capping cylinder COAXIAL with the torus makes g′(u)≡0
// (∂P/∂u ⟂ E for an on-axis capping), so the section is latitude circles, not a u(v)-graph — the fold guard
// must honest-reject rather than fabricate a branch.
func TestTorusCylinderArc_FoldRejectsCoaxialCapping(t *testing.T) {
	tor, err := NewTorusWithRef(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), 50, 10)
	if err != nil {
		t.Fatalf("torus: %v", err)
	}
	coaxial, err := NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 55) // coaxial: cuts the tube at cos v=0.5
	if err != nil {
		t.Fatalf("coaxial cylinder: %v", err)
	}
	v := stdmath.Acos(0.5) // a real crossing latitude (ρ=55) — but g′≡0 there, so the arc must decline
	if _, ok := NewTorusCylinderArc(tor, coaxial, v, 0.3, v+0.2, 0.3, 1e-7); ok {
		t.Fatal("NewTorusCylinderArc must fold-reject a coaxial capping cylinder (g′≡0), not build an arc")
	}
}
