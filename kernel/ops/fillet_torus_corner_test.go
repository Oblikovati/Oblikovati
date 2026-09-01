// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// torusCornerExactTol pins the synthetic-fixture centre tightly — sibling of coneCornerExactTol.
const torusCornerExactTol = 1e-7

// TestTorusHostCorner_Recognizes checks torusHostCorner accepts exactly {1 torus, 2 planes}.
func TestTorusHostCorner_Recognizes(t *testing.T) {
	t.Parallel()
	lin := topo.NewLineage(topo.Tok("test", "torus-corner-recognize", 0))
	bld := topo.NewBuilder(true, lin)
	tor, _ := geom.NewTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 50, 20)
	faces := []*topo.Face{
		bld.AddFace(tor, lin),
		bld.AddFace(planeOn(t, math.P3(0, 0, 0), math.V3(0, 0, 1)), lin),
		bld.AddFace(planeOn(t, math.P3(0, 0, 0), math.V3(0, 1, 0)), lin),
	}
	got, torusFace, planes, ok := torusHostCorner(faces)
	if !ok || got.MajorRadius != 50 || got.MinorRadius != 20 || torusFace == nil || planes[0] == nil || planes[1] == nil {
		t.Fatalf("torusHostCorner declined or misidentified a genuine {torus,plane,plane} set: ok=%v", ok)
	}
}

// torusSyntheticFixture builds a hand-solvable torus-host corner: Rm=50, Rt=20 ring torus at the
// origin (axis ẑ), planes z=0 (boss, material z>0 → c.z=-r) and y=0 (boss, material y>0 → c.y=-r),
// r=5. The offset ρ=Rt-r=15 pins radial(c)=50±√(15²-5²)=50±√200 (two branches, EACH with a ±t twin
// since radial depends only on |t|) — up to four real tangent points on this line, a genuine
// multi-root corner. The vertex sits at the (radial=50+√200, t>0) branch so nearest-vertex resolves
// it unambiguously (no cylinder arms are built on this bare synthetic vertex).
func torusSyntheticFixture(t *testing.T) (*topo.Vertex, []*topo.Face, float64, math.Point3) {
	t.Helper()
	const rm, rt, r = 50.0, 20.0, 5.0
	rho := rt - r
	radial := rm + stdmath.Sqrt(rho*rho-r*r) // outer branch
	tx := stdmath.Sqrt(radial*radial - r*r)
	want := math.P3(tx, -r, -r)
	lin := topo.NewLineage(topo.Tok("test", "torus-corner-synthetic", 0))
	bld := topo.NewBuilder(true, lin)
	v := bld.AddVertex(want, lin)
	tor, err := geom.NewTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), rm, rt)
	if err != nil {
		t.Fatalf("torus: %v", err)
	}
	faces := []*topo.Face{
		bld.AddFace(tor, lin),
		bld.AddFace(planeOn(t, math.P3(0, 0, 0), math.V3(0, 0, 1)), lin),
		bld.AddFace(planeOn(t, math.P3(0, 0, 0), math.V3(0, 1, 0)), lin),
	}
	return v, faces, r, want
}

// TestTorusHostCorner_ExactSyntheticCentre solves the hand-built fixture and checks the solved
// centre against the independently-derived expected point (torusSyntheticFixture's `want`, computed
// via the direct radial/axial algebra, not the quartic machinery under test).
func TestTorusHostCorner_ExactSyntheticCentre(t *testing.T) {
	t.Parallel()
	v, faces, r, want := torusSyntheticFixture(t)
	cb, err := solveBlend(nil, v, faces, r)
	if err != nil {
		t.Fatalf("synthetic torus corner declined: %v", err)
	}
	if got := cb.center.DistanceTo(want); float64(got) > torusCornerExactTol {
		t.Fatalf("centre %v, want %v (independent radial/axial solve), Δ=%.3e", cb.center, want, got)
	}
	if cb.sphere.Radius != r {
		t.Fatalf("sphere radius %g, want %g", cb.sphere.Radius, r)
	}
}

// assertTorusCentreIndependent recomputes, by hand (not calling torusCorePoint/torusCornerConsistent),
// the plane distances and the core-circle distance for centre c, and checks the core-circle distance
// equals Rt∓r on EITHER convex/concave branch (the independent-check style of the cylinder/cone
// sibling tests, which likewise accept either offset branch rather than reproducing the production
// sign read).
func assertTorusCentreIndependent(t *testing.T, c math.Point3, tor geom.Torus, planes [2]*topo.Face, r float64) {
	t.Helper()
	for _, f := range planes {
		pl := f.Geometry().(geom.Plane)
		d := stdmath.Abs(float64(pl.Origin.VectorTo(c).Dot(outwardPlaneNormal(f, pl))))
		if stdmath.Abs(d-r) > 1e-6 {
			t.Fatalf("centre %v is %.9f from a host plane, want %g", c, d, r)
		}
	}
	a := tor.AxisDir.AsVector()
	w := tor.Center.VectorTo(c)
	axial := float64(w.Dot(a))
	radial := stdmath.Sqrt(float64(w.Dot(w)) - axial*axial)
	distCore := stdmath.Sqrt((radial-tor.MajorRadius)*(radial-tor.MajorRadius) + axial*axial)
	if stdmath.Abs(distCore-(tor.MinorRadius-r)) > 1e-6 && stdmath.Abs(distCore-(tor.MinorRadius+r)) > 1e-6 {
		t.Fatalf("centre %v is %.9f from the core circle, want Rt∓r = %g∓%g", c, distCore, tor.MinorRadius, r)
	}
}

// torusImportedCorner drives a REAL corpus fixture through solveBlend and independently certifies
// the resulting centre. Used by all four R4-wave torus cases (E6 E8 F1 F3) — the case still declines
// end-to-end in the corpus scoreboard (fillet_torusarm.go's "spiric canal, unsupported" cap-plane
// arm gate is a separate, unowned file — see the R4 wave report); this certifies the CORNER PATCH
// capability alone, matching the sibling files' "solves the sphere; the weld/arm is a follow-on"
// precedent.
func torusImportedCorner(t *testing.T, rel string, near math.Point3, r float64) {
	t.Helper()
	body := corpusFixture(t, rel)
	v := vertexNearest(t, body, near)
	faces := facesAtVertex(v)
	tor, _, planes, ok := torusHostCorner(faces)
	if !ok {
		t.Fatalf("%s corner at %v is not a [torus,plane,plane] host set", rel, near)
	}
	cb, err := solveBlend(nil, v, faces, r)
	if err != nil {
		t.Fatalf("%s: torus corner declined: %v", rel, err)
	}
	assertTorusCentreIndependent(t, cb.center, tor, planes, r)
}

// TestTorusHostCorner_E6Imported certifies simple/E6 — a HORN torus (Rm=Rt=100): the offset torus
// (ρ=90) stays non-degenerate even though the host itself self-intersects at the axis, per the
// geometry-math-advisor derivation's horn-torus note.
func TestTorusHostCorner_E6Imported(t *testing.T) {
	t.Parallel()
	torusImportedCorner(t, "simple/E6.step", math.P3(200, 0, 0), 10)
}

// TestTorusHostCorner_E8Imported is E6's horn-torus sibling certification (same base solid,
// different corner).
func TestTorusHostCorner_E8Imported(t *testing.T) {
	t.Parallel()
	torusImportedCorner(t, "simple/E8.step", math.P3(170.71067811865, 0, 70.710678118655), 10)
}

// TestTorusHostCorner_F1Imported certifies simple/F1 — a RING torus (Rm=200, Rt=50, Rm>Rt).
func TestTorusHostCorner_F1Imported(t *testing.T) {
	t.Parallel()
	torusImportedCorner(t, "simple/F1.step", math.P3(153.0153689607, 0, 17.101007166283), 10)
}

// TestTorusHostCorner_F3Imported is F1's ring-torus sibling certification.
func TestTorusHostCorner_F3Imported(t *testing.T) {
	t.Parallel()
	torusImportedCorner(t, "simple/F3.step", math.P3(250, 0, 0), 10)
}

// TestTorusCornerConsistent_RejectsPerturbedCentre is the certificate regression fence, sibling of
// TestConeCornerConsistent_RejectsPerturbedCentre.
func TestTorusCornerConsistent_RejectsPerturbedCentre(t *testing.T) {
	t.Parallel()
	v, faces, r, want := torusSyntheticFixture(t)
	tor, _, planes, ok := torusHostCorner(faces)
	if !ok {
		t.Fatalf("recognizer declined the synthetic fixture")
	}
	res := torusCornerResolution(v, tor, planes)
	if !torusCornerConsistent(want, tor, planes, r, 1, res) {
		t.Fatalf("certified centre %v rejected at its own sign s=1; want accept", want)
	}
	offPlane := want.TranslateBy(math.V3(0, 1, 0).Scale(1e-3))
	if torusCornerConsistent(offPlane, tor, planes, r, 1, res) {
		t.Fatalf("centre nudged off a plane accepted; want reject")
	}
	offTorus := want.TranslateBy(math.V3(1, 0, 0).Scale(1))
	if torusCornerConsistent(offTorus, tor, planes, r, 1, res) {
		t.Fatalf("centre nudged off the torus core-circle distance accepted; want reject")
	}
}
