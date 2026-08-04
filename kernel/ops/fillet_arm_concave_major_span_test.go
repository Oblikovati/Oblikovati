// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	m "oblikovati.org/math"
)

// m8BossRimParent is the M8 fixture's own boss top rim, written from the FIXTURE, not from the
// code: `simple/M8.step` is a 100³ box carrying a boss of radius 25 centred on (60,50), cut back to
// three quadrants by the planes x=60 and y=50, so the boss's top rim at z=150 is a 270° arc running
// counter-clockwise from (85,50,150) — angle 0 — to (60,25,150) — angle 270°.
func m8BossRimParent() geom.Arc3d {
	up, _ := m.UnitVector3FromVector(m.V3(0, 0, 1))
	ref, _ := m.UnitVector3FromVector(m.V3(1, 0, 0))
	return geom.Arc3d{
		Center: m.P3(60, 50, 150), Normal: up, RefDir: ref,
		Radius: 25, StartAngle: 0, SweepAngle: 1.5 * stdmath.Pi,
	}
}

// m8RimContactAngle is the angle at which the r=5 fillet on the boss∧(x=60) edge touches the boss
// wall, derived from the fixture in closed form and NOT read off any built body: the rolling ball
// sits inside the material, so its centre is 25−5 = 20 from the boss axis and 5 from the plane
// x=60, i.e. at (55, 50−√(20²−5²)) = (55, 50−√375). The wall contact is that centre direction
// scaled to the boss radius, so its in-plane offset is 25·(−5,−√375)/20 → cos = −1/4. The rim
// therefore survives from angle 0 counter-clockwise to 2π − arccos(−1/4) = 4.459708725 rad
// (255.5225°): a MAJOR span, whose complement is the 104.4775° arc that must never be built.
func m8RimContactAngle() float64 { return 2*stdmath.Pi - stdmath.Acos(-0.25) }

// TestReterminatedRimSegKeepsAMajorSpanMajor drives the SHIPPED re-termination entry points — the
// three functions weldChainRuns/reterminateBothEnds actually call when a corner chain is spliced
// into a host ring — with M8's own boss rim and its own fillet contact point, and requires the
// rebuilt segment to be the 255.52° span the solid really retains, not its 104.48° COMPLEMENT.
//
// ★ This is the M8 defect. rebuildArcSeg used to re-fit the sub-arc through arcMidBetween's
// SHORTER-arc midpoint, which makes Arc3dByThreePoints return the complement for any span past a
// semicircle (the N7 whole-curve-sub-span lesson that subSeg and retainedRimCurve already carry).
// The complement has the SAME endpoints and lies on the SAME circle, so nothing but its EXTENT
// separates it — hence the midpoint assertion below, which is the only one that fails when the
// major-span arm is removed. M8 then handed the edge catalog a curve 22.01 from the one its top
// plane offered for the same welded edge, and which shipped was decided by build order alone.
func TestReterminatedRimSegKeepsAMajorSpanMajor(t *testing.T) {
	parent := m8BossRimParent()
	start := parent.PointAt(0)                                // (85,50,150), the rim's own start
	contact := pointOnRimAtAngle(parent, m8RimContactAngle()) // (53.75, 25.793854086, 150)
	tol := 1e-9 * parent.Radius

	seg := endSeg{from: start, to: parent.PointAt(1), curve: parent, arc: true}
	for _, tc := range []struct {
		name string
		got  func() (endSeg, bool)
	}{
		{"reterminateSegTo", func() (endSeg, bool) { return reterminateSegTo(seg, contact, tol) }},
		{"reterminateSegFrom", func() (endSeg, bool) {
			rev := endSeg{from: parent.PointAt(1), to: start, curve: parent, arc: true}
			return reterminateSegFrom(rev, contact, tol)
		}},
		{"reterminateBothEnds", func() (endSeg, bool) { return reterminateBothEnds(seg, start, contact, tol) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, ok := tc.got()
			if !ok {
				t.Fatalf("%s declined M8's own rim span — both endpoints are exactly on the boss rim", tc.name)
			}
			assertSpansM8sMajorRim(t, tc.name, out, parent)
		})
	}
}

// assertSpansM8sMajorRim checks the re-terminated segment carries the MAJOR 255.5225° rim span: the
// right endpoints, the parent's own circle, the closed-form sweep, and — the check that actually
// separates the arc from its complement — a midpoint on the retained side of the circle.
func assertSpansM8sMajorRim(t *testing.T, name string, out endSeg, parent geom.Arc3d) {
	t.Helper()
	want := m8RimContactAngle()
	arc, isArc := out.curve.(geom.Arc3d)
	if !isArc {
		t.Fatalf("%s: rebuilt segment carries %T, want a geom.Arc3d sub-arc of the boss rim", name, out.curve)
	}
	if got := stdmath.Abs(arc.SweepAngle); stdmath.Abs(got-want) > 1e-9 {
		t.Errorf("%s: sub-arc sweeps %.9f rad, want M8's closed-form retained span %.9f rad "+
			"(%.9f is its COMPLEMENT)", name, got, want, 2*stdmath.Pi-want)
	}
	lo, hi := arc.Domain()
	mid := arc.PointAt((lo + hi) / 2)
	wantMid := pointOnRimAtAngle(parent, want/2)
	badMid := pointOnRimAtAngle(parent, want+(2*stdmath.Pi-want)/2)
	if d := float64(mid.DistanceTo(wantMid)); d > 1e-6*parent.Radius {
		t.Errorf("%s: sub-arc midpoint %v is %.6g from the retained rim's own midpoint %v (the complement's is %v) "+
			"— endpoints and circle alone do NOT separate an arc from its complement", name, mid, d, wantMid, badMid)
	}
}

// pointOnRimAtAngle is the point on parent's circle at the given angle from its RefDir — the
// fixture-side evaluation, written independently of the production sub-arc algebra.
func pointOnRimAtAngle(parent geom.Arc3d, angle float64) m.Point3 {
	bin := parent.Normal.Cross(parent.RefDir)
	off := parent.RefDir.AsVector().Scale(m.Scalar(parent.Radius * stdmath.Cos(angle))).
		Add(bin.Scale(m.Scalar(parent.Radius * stdmath.Sin(angle))))
	return parent.Center.TranslateBy(off)
}

// TestReterminatedRimSegCarriesAnExactSemicircle pins π — the major-span guard's OWN edge, and the one
// span the shorter-arc re-fit underneath it cannot build at all.
//
// At exactly a semicircle the two endpoints are antipodal, so arcMidBetween's bisector from̂+tô is the
// null vector; it degrades to the chord midpoint, which is the circle's CENTRE, and Arc3dByThreePoints
// then sees three collinear points and errors — the whole re-termination declines a span the parent
// could carry exactly. subArcMajor therefore takes >= π, not > π. The three spans below bracket the
// edge: a hair under (the minor re-fit), exactly π, and a hair over (the major parent-parameter trim).
func TestReterminatedRimSegCarriesAnExactSemicircle(t *testing.T) {
	parent := m8BossRimParent() // r=25 at (60,50,150), sweeping 1.5π from angle 0
	tol := 1e-9 * parent.Radius
	seg := endSeg{from: parent.PointAt(0), to: parent.PointAt(1), curve: parent, arc: true}
	for _, want := range []float64{stdmath.Pi - 1e-6, stdmath.Pi, stdmath.Pi + 1e-6} {
		out, ok := reterminateSegTo(seg, pointOnRimAtAngle(parent, want), tol)
		if !ok {
			t.Fatalf("re-termination declined a %.9f rad span of the parent's own circle — both endpoints "+
				"are exactly on it, and at π the three-point re-fit's bisector midpoint is undefined", want)
		}
		arc, isArc := out.curve.(geom.Arc3d)
		if !isArc {
			t.Fatalf("%.9f rad span carries %T, want a geom.Arc3d sub-arc", want, out.curve)
		}
		if got := stdmath.Abs(arc.SweepAngle); stdmath.Abs(got-want) > 1e-6 {
			t.Errorf("sub-arc sweeps %.9f rad, want %.9f (%.9f is its COMPLEMENT)", got, want, 2*stdmath.Pi-want)
		}
	}
}
