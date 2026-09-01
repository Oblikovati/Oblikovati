// SPDX-License-Identifier: GPL-2.0-only

package blend

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

// wallPointAt returns the R=50 axis-ẑ (ref x̂) wall point at chart (θ, z).
func wallPointAt(theta, z float64) math.Point3 {
	return math.P3(50*stdmath.Cos(theta), 50*stdmath.Sin(theta), z)
}

// closedWallRect is a genuine CLOSED wall-face loop on the R=50 axis-ẑ cylinder bounding the region
// θ∈[−1,1], z∈[zlo,zhi]: bottom rim (zlo), right axial edge (θ=1), top rim (zhi), left axial edge
// (θ=−1). Unlike a bare pair of rim arcs, this is a real ring, so chartPointInLoop develops it into a
// well-defined interior — the fixture the D9-T1 reflex-corner station guard needs to tell an interior
// far-vertex station (accept) from an off-face one (decline). The θ=0 ruling crosses only the two rims.
func closedWallRect(t *testing.T, zlo, zhi float64) []endSeg {
	t.Helper()
	bottom := rimArcSeg(t, zlo, -1, 2) // θ: −1 → 1 at z=zlo
	top := rimArcSeg(t, zhi, 1, -2)    // θ:  1 → −1 at z=zhi
	right := endSeg{from: wallPointAt(1, zlo), to: wallPointAt(1, zhi)}
	left := endSeg{from: wallPointAt(-1, zhi), to: wallPointAt(-1, zlo)}
	return []endSeg{bottom, right, top, left}
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
	t.Parallel()
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
	t.Parallel()
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

// TestArmRulingEnd_AuthorityRejectsWrongRim proves the far-vertex authority BITES: when the ruling's
// forward loop crossing (top rim z=80) disagrees with the far vertex AND that far vertex lies OFF the
// host face (z=125, above the z=80 boundary of the closed wall region z∈[15,80]), armRulingEnd
// honest-rejects rather than fabricate a rail past the boundary. Post D9-T1 the runout-disagreeing
// crossing falls through to the interior-station fallback (rulingStationOuter), and that fallback
// declines because the station's chart point is OUTSIDE the loop — the authority still bites off-face.
func TestArmRulingEnd_AuthorityRejectsWrongRim(t *testing.T) {
	t.Parallel()
	cyl := mustCylinder(t, math.P3(0, 0, 0), math.V3(0, 0, 1), 50)
	tHost := math.P3(50, 0, 15)
	v := math.P3(50, 0, 10)
	segs := closedWallRect(t, 15, 80) // host face region z∈[15,80]; ruling crosses the top rim z=80
	arm := armSetback{arm: cyl, farVertex: math.P3(50, 0, 125), runoutKnown: true}

	if _, ok := armRulingEnd(hostFaceFor(t, cyl), cyl, arm, tHost, v, segs, 0.02); ok {
		t.Fatalf("authority: far vertex z=125 lies above the host's z=80 boundary (off-face), must decline")
	}
}

// TestArmRulingEnd_InteriorStationAccepted is the D9-T1 reflex-corner accept: when the ruling's forward
// loop crossing (top rim z=80) OVERSHOOTS the far vertex (z=60, interior to the face region z∈[15,80]),
// the true outer end is the interior far-vertex STATION, not the crossing. armRulingEnd falls back to
// rulingStationOuter and returns the station z=60 — the same mechanism that greens D9's cap ruling.
func TestArmRulingEnd_InteriorStationAccepted(t *testing.T) {
	t.Parallel()
	cyl := mustCylinder(t, math.P3(0, 0, 0), math.V3(0, 0, 1), 50)
	tHost := math.P3(50, 0, 15)
	v := math.P3(50, 0, 10)
	segs := closedWallRect(t, 15, 80)
	arm := armSetback{arm: cyl, farVertex: math.P3(50, 0, 60), runoutKnown: true}

	end, ok := armRulingEnd(hostFaceFor(t, cyl), cyl, arm, tHost, v, segs, 0.02)
	if !ok || stdmath.Abs(float64(end.Z)-60) > 0.02 {
		t.Fatalf("interior far vertex z=60 must be accepted at the station; got z=%.4f ok=%v", float64(end.Z), ok)
	}
}

// reentrantWallLoop is a wall-face loop whose region re-enters along the θ=0 ruling: a rectangle θ∈[−1,1],
// z∈[15,80] with a NOTCH (a slot z∈[40,50] removed across θ∈[−1,0.5]). The vertical ruling θ=0 up from
// z=15 crosses the boundary at z=15, 40, 50, 80 — so a station at z=60 sits inside by ray-cast PARITY
// (odd crossings above it) yet the rail from z=15 to z=60 spans the OFF-FACE notch (z∈[40,50]).
func reentrantWallLoop(t *testing.T) []endSeg {
	t.Helper()
	return []endSeg{
		rimArcSeg(t, 15, -1, 2),                                // bottom rim z=15, θ:−1→1
		{from: wallPointAt(1, 15), to: wallPointAt(1, 80)},     // right axial θ=1
		rimArcSeg(t, 80, 1, -2),                                // top rim z=80, θ:1→−1
		{from: wallPointAt(-1, 80), to: wallPointAt(-1, 50)},   // left-upper axial θ=−1
		rimArcSeg(t, 50, -1, 1.5),                              // notch-top rim z=50, θ:−1→0.5
		{from: wallPointAt(0.5, 50), to: wallPointAt(0.5, 40)}, // notch-right axial θ=0.5
		rimArcSeg(t, 40, 0.5, -1.5),                            // notch-bottom rim z=40, θ:0.5→−1
		{from: wallPointAt(-1, 40), to: wallPointAt(-1, 15)},   // left-lower axial θ=−1
	}
}

// TestArmRulingEnd_UndershootStationDeclined is the D9-T2 overshoot-guard regression: on a RE-ENTRANT loop
// the far-vertex station (z=60) lands inside by ray-cast parity, but the ruling first LEAVES the loop at
// the notch (z=40, an UNDERSHOOT: rFirst=25 < the station runout rFar=45), so the rail would span off-face
// territory. armRulingEnd must DECLINE rather than fabricate it — an endpoint-only interiority test would
// wrongly accept. (Reverting stationRunReached re-accepts, re-reddening this.)
func TestArmRulingEnd_UndershootStationDeclined(t *testing.T) {
	t.Parallel()
	cyl := mustCylinder(t, math.P3(0, 0, 0), math.V3(0, 0, 1), 50)
	tHost := math.P3(50, 0, 15)
	v := math.P3(50, 0, 10)
	segs := reentrantWallLoop(t)
	arm := armSetback{arm: cyl, farVertex: math.P3(50, 0, 60), runoutKnown: true}

	if end, ok := armRulingEnd(hostFaceFor(t, cyl), cyl, arm, tHost, v, segs, 0.02); ok {
		t.Fatalf("undershoot station z=60 must be DECLINED (ruling exits the loop at the notch z=40 first); got z=%.4f", float64(end.Z))
	}
}

// TestCylChart_RoundTrip checks the (θ,z) chart's to2/to3 invert on the wall — a point maps to its
// azimuth+height and back to itself. The wall's ref is pinned to x̂ so θ is deterministic (NewCylinder
// picks an arbitrary in-plane ref).
func TestCylChart_RoundTrip(t *testing.T) {
	t.Parallel()
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
