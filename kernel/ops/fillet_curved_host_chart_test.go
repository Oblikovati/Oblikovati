// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// rimArcSeg builds a horizontal rim endSeg on the wall (centre on the axis at height z, radius 50)
// spanning θ∈[start, start+sweep] — the developed chart image is the z=const line the vertical ruling
// crosses. Reuses mustArc (normal ẑ, ref x̂) from fillet_curved_retrim_test.go.
func rimArcSeg(t *testing.T, z, start, sweep float64) endSeg {
	t.Helper()
	arc := mustArc(t, math.P3(0, 0, z), 50, start, sweep)
	return endSeg{from: arc.PointAt(0), to: arc.PointAt(1), curve: arc, mid: arc.PointAt(0.5), arc: true}
}

// notchedWallLoop is the N7 s_4 defect in miniature: a full-height outer rim at z=130 (the GLOBAL axial
// extreme axialExtremeEnd slid to) AND an intermediate notch-top rim at z=80, both spanning θ=0 (the
// ruling azimuth). The vertical ruling up from z=15 must terminate at the FIRST rim it meets — z=80 —
// not the global z=130.
func notchedWallLoop(t *testing.T) []endSeg {
	t.Helper()
	return []endSeg{
		rimArcSeg(t, 80, -0.5, 1.0),  // notch-top rim z=80, θ∈[−0.5,0.5] (contains θ₀=0)
		rimArcSeg(t, 130, -0.5, 1.0), // global outer rim z=130, θ∈[−0.5,0.5]
	}
}

// cleanWallLoop is a single top rim at height z (no notch) — the B3 wall reduction: the first crossing
// IS the global extreme, so armRulingEnd must equal axialExtremeEnd there.
func cleanWallLoop(t *testing.T, z float64) []endSeg {
	t.Helper()
	return []endSeg{rimArcSeg(t, z, -0.5, 1.0)}
}

// hostFaceFor wraps the wall cylinder as a bare host face — armRulingEnd reads only host.Geometry() for
// the chart (the crossing loop is passed as segs), so no boundary loops are needed on the face.
func hostFaceFor(t *testing.T, cyl geom.Cylinder) *topo.Face {
	t.Helper()
	bld := topo.NewBuilder(true, topo.Lineage{})
	return bld.AddFace(cyl, topo.Lineage{})
}

// TestArmRulingEnd_StopsAtNotchTopNotGlobalExtreme is the exact N7 s_4 defect: a cylinder arm's wall
// ruling must terminate at the first forward crossing (the notch-top rim z=80), NOT slide to the loop's
// global axial extreme (z=130) as axialExtremeEnd did. Wall R=50 axis ẑ; ruling θ₀=0 up from z=15.
func TestArmRulingEnd_StopsAtNotchTopNotGlobalExtreme(t *testing.T) {
	cyl := mustCylinder(t, math.P3(0, 0, 0), math.V3(0, 0, 1), 50) // wall R=50, axis ẑ
	tHost := math.P3(50, 0, 15)                                    // s_4 setback foot, θ₀=0
	v := math.P3(50, 0, 10)                                        // bitten corner vertex (below → ruling runs up)
	segs := notchedWallLoop(t)
	arm := armSetback{arm: cyl, farVertex: math.P3(50, 0, 80), runoutKnown: true}
	tol := 0.02 // res.Weld()*r, r=5

	end, ok := armRulingEnd(hostFaceFor(t, cyl), cyl, arm, tHost, v, segs, tol)

	if !ok {
		t.Fatalf("armRulingEnd: expected the z=80 notch-top runout, got ok=false")
	}
	if got := float64(end.Z); stdmath.Abs(got-80) > tol {
		t.Fatalf("armRulingEnd terminated at z=%.4f; want z=80 (notch top), NOT z=130 (global extreme)", got)
	}
}

// TestArmRulingEnd_CleanWallReducesToGlobalExtreme is the B3 wall reduction: on a clean wall the first
// crossing IS the single rim, so armRulingEnd returns exactly what axialExtremeEnd returned.
func TestArmRulingEnd_CleanWallReducesToGlobalExtreme(t *testing.T) {
	cyl := mustCylinder(t, math.P3(0, 0, 0), math.V3(0, 0, 1), 50)
	tHost := math.P3(50, 0, 15)
	v := math.P3(50, 0, 10)
	segs := cleanWallLoop(t, 100) // single top rim at z=100, no notch
	arm := armSetback{arm: cyl, farVertex: math.P3(50, 0, 100), runoutKnown: true}

	end, ok := armRulingEnd(hostFaceFor(t, cyl), cyl, arm, tHost, v, segs, 0.02)

	if !ok || stdmath.Abs(float64(end.Z)-100) > 0.02 {
		t.Fatalf("clean wall: armRulingEnd must equal the global rim z=100 (axialExtremeEnd reduction); got z=%.4f ok=%v", float64(end.Z), ok)
	}
}

// TestArmRulingEnd_AuthorityRejectsWrongRim proves the far-vertex authority BITES: when the first
// chart crossing (z=80) disagrees with the filleted edge's far vertex (here z=125, an interior weld /
// wrong-edge scenario), armRulingEnd honest-rejects rather than fabricate a rail.
func TestArmRulingEnd_AuthorityRejectsWrongRim(t *testing.T) {
	cyl := mustCylinder(t, math.P3(0, 0, 0), math.V3(0, 0, 1), 50)
	tHost := math.P3(50, 0, 15)
	v := math.P3(50, 0, 10)
	segs := notchedWallLoop(t) // first crossing at z=80
	arm := armSetback{arm: cyl, farVertex: math.P3(50, 0, 125), runoutKnown: true}

	if _, ok := armRulingEnd(hostFaceFor(t, cyl), cyl, arm, tHost, v, segs, 0.02); ok {
		t.Fatalf("authority: crossing z=80 disagrees with far vertex z=125 by 45 ≫ tol, must decline")
	}
}

// xEqual50PlaneChart builds the x=50 corner-host plane's chart with a DETERMINISTIC (u,v)=(y,z) basis
// (NewPlaneFromAxes, not NewPlane's arbitrary in-plane pick), so the vertex coordinates in the N7 x=50
// tests below read directly as (y,z).
func xEqual50PlaneChart(t *testing.T) planeChart {
	t.Helper()
	pl, err := geom.NewPlaneFromAxes(math.P3(50, 0, 0), math.V3(0, 1, 0), math.V3(0, 0, 1))
	if err != nil {
		t.Fatalf("build x=50 plane: %v", err)
	}
	return planeChart{pl}
}

// openNotchLoop is the N7 x=50 plane host's OWN bitten (opened) loop near the wall/plane junction
// vertex vtx: a single dangling edge running from vtx outward on the +y side only. This mirrors the
// real post-bite topology gap — the loop is opened at the corner, so nothing covers the OTHER side of
// vtx. Confirmed empirically (probe): an exact-vertex ray already succeeds against this chain (u=0 is
// always in [0,1] at an edge's OWN endpoint), but a ray whose independently-computed target (the arm's
// far-vertex authority, ~thousandths off the loop's own vertex — same provenance gap runoutAgrees
// tolerates elsewhere) drifts toward the UNCOVERED side finds no interior hit on any edge at all: that
// is the genuine N7 "no valid exit" mechanism, not a boundary-inclusivity bug (u∈[0,1] here is already
// inclusive) — the brief's "strict u∈(0,1)" framing was pre-N2; what's actually missing is a candidate
// that doesn't depend on which side of vtx an edge happens to extend.
func openNotchLoop(t *testing.T, vtx math.Point3) []endSeg {
	t.Helper()
	far := vtx.TranslateBy(math.V3(0, 40, 0))
	corner := far.TranslateBy(math.V3(0, 0, -70))
	return []endSeg{
		{from: vtx, to: far},
		{from: far, to: corner},
	}
}

// TestChartRulingExit_LandsOnLoopVertex is the concrete N7 x=50 plane decline: the s_4 ruling's target
// (tHost≈(50,0,10), independently-computed far vertex near (50,0,80)) drifts a few thousandths toward
// the side openNotchLoop's single dangling edge does NOT cover. Today's interior-only scan finds no
// crossing (see the probe evidence in the task report) → chartRulingExit must ALSO try each edge's
// endpoints as candidates and snap the winner onto the loop's own vertex.
func TestChartRulingExit_LandsOnLoopVertex(t *testing.T) {
	ch := xEqual50PlaneChart(t)
	vtx := math.P3(50, 0, 80)
	segs := openNotchLoop(t, vtx)
	tHost := math.P3(50, 0, 10) // s_4 setback foot on the x=50 plane
	o2 := ch.to2(tHost)
	target := math.P3(50, -0.005, 80.01) // arm authority's independently-computed near-vertex, off-side
	d2 := ch.to2(target).AsVector().Sub(o2.AsVector())
	tol := 0.02 // res.Weld()*r, r=5

	end, ok := chartRulingExit(ch, segs, o2, d2, tol)

	if !ok {
		t.Fatalf("chartRulingExit rejected a ruling landing within tol of loop vertex %v (the N7 x=50 decline)", vtx)
	}
	if got := float64(end.DistanceTo(vtx)); got > tol {
		t.Fatalf("chartRulingExit landed at %v; want the loop vertex %v (snap, no split); dist=%.6f > tol=%.4f", end, vtx, got, tol)
	}
}

// TestChartRulingExit_GrazingRulingDeclinesNotPanics is the collinear/grazing floor: a ruling direction
// nearly parallel to a loop edge must not divide by ~0 in raySegment2d's line solve. The edge here is
// offset 5 chart-units from the ruling's own line (well beyond tol) at BOTH ends, so neither the
// (declined) interior crossing nor an endpoint candidate can land near the ray: chartRulingExit must
// cleanly decline — never panic, never fabricate a crossing — falling through to C1's far-vertex
// authority in the real armRulingEnd caller.
func TestChartRulingExit_GrazingRulingDeclinesNotPanics(t *testing.T) {
	ch := xEqual50PlaneChart(t)
	segs := []endSeg{{from: math.P3(50, 5, 10), to: math.P3(50, 5+1e-9, 130)}} // ~parallel to d2, offset 5 from it
	o2 := ch.to2(math.P3(50, 0, 10))
	d2 := math.V2(0, 1) // pure +z ruling — nearly collinear with the lone edge above

	end, ok := chartRulingExit(ch, segs, o2, d2, 0.02)

	if ok {
		t.Fatalf("grazing ruling: expected no crossing (collinear edge, offset endpoints), got end=%v", end)
	}
}

// TestSnapToVertex_ReusesExistingCorner: a landing 0.005 off the true vertex (within tol=0.02) snaps to
// the loop's own stored vertex exactly — so the far-path splice reuses it (no zero-length sliver split).
func TestSnapToVertex_ReusesExistingCorner(t *testing.T) {
	vtx := math.P3(50, 0, 80)
	segs := openNotchLoop(t, vtx)
	p := math.P3(50, 0, 80.005) // within tol of the vertex

	v, ok := snapToVertex(p, segs, 0.02)

	if !ok || float64(v.DistanceTo(vtx)) > 1e-9 {
		t.Fatalf("snapToVertex must return the exact loop vertex %v; got %v ok=%v", vtx, v, ok)
	}
}

// TestSnapToVertex_NoNearbyVertexDeclines: a point far from every loop vertex must not spuriously snap
// (the far-path splice must still split there, not silently jump to an unrelated corner).
func TestSnapToVertex_NoNearbyVertexDeclines(t *testing.T) {
	segs := openNotchLoop(t, math.P3(50, 0, 80))
	p := math.P3(50, 20, 45) // mid-edge-ish, nowhere near vtx=(50,0,80) or far=(50,40,80)

	if v, ok := snapToVertex(p, segs, 0.02); ok {
		t.Fatalf("snapToVertex: expected no vertex within tol of %v, got %v", p, v)
	}
}

// TestCylChart_RoundTrip checks the (θ,z) chart's to2/to3 invert on the wall — a point maps to its
// azimuth+height and back to itself. The wall's ref is pinned to x̂ so θ is deterministic (NewCylinder
// picks an arbitrary in-plane ref).
func TestCylChart_RoundTrip(t *testing.T) {
	wall, err := geom.NewCylinderWithRef(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), 50)
	if err != nil {
		t.Fatalf("build wall with ref x̂: %v", err)
	}
	ch := newCylChart(wall)
	p := math.P3(0, 50, 37) // θ=+π/2 (about ẑ from x̂ toward ŷ), z=37 on the wall
	q := ch.to2(p)
	if stdmath.Abs(float64(q.X)-stdmath.Pi/2) > 1e-9 || stdmath.Abs(float64(q.Y)-37) > 1e-9 {
		t.Fatalf("cylChart.to2(%v) = (θ=%.6f, z=%.6f), want (π/2, 37)", p, float64(q.X), float64(q.Y))
	}
	if back := ch.to3(q); float64(back.DistanceTo(p)) > 1e-9 {
		t.Fatalf("cylChart round trip: to3(to2(%v)) = %v, want the identity", p, back)
	}
}
