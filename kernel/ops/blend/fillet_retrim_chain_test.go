// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// The chain-capable loop rebuild's acceptance suite. Every expectation here is a CLOSED FORM derived
// from the corpus body's own dimensions — no oracle, no captured number — so the primitive is measured
// against the geometry it must produce, not against what it happens to produce.
//
// The three consumers it is built for supply the cases:
//   - simple/Y2 (a 100³ box with a 10×10 notch at x∈[90,100], z∈[80,90], filleted r=15 along the
//     y=0 ∧ z=100 edge): the setback line z=85 crosses the notch, so the host plane's substituted
//     tangent point lands 5 PAST its adjacent edge's own end. Host, band, notch wall, notch top and
//     the wall above the notch are all chain-retrim consumers, and their five closed forms sum with
//     the untouched faces to 61050.07 against OCCT's 61050.1.
//   - simple/Y4 (same body, r=25): the setback line z=75 clears the notch entirely and the rebuild
//     consumes the notch's three whole rim edges without any clipping at all.
//   - complex/D8: the far-end trim's terminal section leaves the radius-24 corner round at its u=0
//     ruling; the round's whole far rim arc must disappear (selfcross-trim-report.md §5.1).

// areaSamplesPerSeg is the per-segment sampling of a measured ring. A ring's area is only ever measured
// HERE, in a test, so it is sized to put the sampler's own error below 1e-9 relative on the tightest
// curved span in the suite (a 0.34 rad arc of radius 15) rather than for speed.
const areaSamplesPerSeg = 16384

// retrimChainTol is the model weld the acceptance cases run at: 1e-7 on bodies whose diagonal is ~175,
// i.e. the same relative band the corpus retrim uses.
const retrimChainTol = 1e-7

// ---------------------------------------------------------------------------------------------
// simple/Y2 and simple/Y4 — the setback overrun
// ---------------------------------------------------------------------------------------------

// yHostRing is the Y2/Y4 host plane's ORIGINAL outer boundary (y = 0): the 100×100 face with the
// 10×10 slot notch removed at x ∈ [90,100], z ∈ [80,90].
func yHostRing() []endSeg {
	p := []math.Point3{
		math.P3(100, 0, 80), math.P3(90, 0, 80), math.P3(90, 0, 90), math.P3(100, 0, 90),
		math.P3(100, 0, 100), math.P3(0, 0, 100), math.P3(0, 0, 0), math.P3(100, 0, 0),
	}
	return chainLineRing(p)
}

// closedLineRing turns an ordered point list into a closed ring of straight segments.
func chainLineRing(p []math.Point3) []endSeg {
	out := make([]endSeg, len(p))
	for i := range p {
		out[i] = endSeg{from: p[i], to: p[(i+1)%len(p)]}
	}
	return out
}

// TestChainRetrimLoopRebuildsY2HostPlaneThroughItsNotch is the setback-overrun consumer's acceptance:
// r=15 recedes the host boundary to z=85, which at x ∈ [90,100] is INSIDE the notch, so the substituted
// tangent point (100,0,85) is not on the face at all. The chain-capable rebuild clips the setback line
// at the notch wall and consumes the three whole rim edges the notch contributed, leaving the closed
// form 100·85 − 10·5 = 8450. transformLoop ships 8475 for exactly this face.
func TestChainRetrimLoopRebuildsY2HostPlaneThroughItsNotch(t *testing.T) {
	t.Parallel()
	chain := []endSeg{{from: math.P3(100, 0, 85), to: math.P3(0, 0, 85)}}
	got, ok := chainRetrimLoop(chainPlaneThrough(math.P3(0, 0, 0), math.V3(0, 1, 0)), yHostRing(), chain, retrimChainTol)
	if !ok {
		t.Fatal("chainRetrimLoop declined the Y2 host plane")
	}
	assertChainRingClosed(t, got)
	assertChainArea(t, "Y2 host plane area", chainPlanarArea(got), 8450)
}

// TestChainRetrimLoopRebuildsY4HostPlaneConsumingTheWholeNotch pins the no-clip half of the primitive:
// r=25 recedes the boundary to z=75, BELOW the notch, so both landings are already on the ring and the
// rebuild's whole job is to consume the notch's three rim edges plus the two above it. Closed form
// 100·75 = 7500; transformLoop ships 7600 and its loop back-tracks along a COLLINEAR sibling edge, which
// is why the self-crossing ratchet cannot see this one at all.
func TestChainRetrimLoopRebuildsY4HostPlaneConsumingTheWholeNotch(t *testing.T) {
	t.Parallel()
	chain := []endSeg{{from: math.P3(100, 0, 75), to: math.P3(0, 0, 75)}}
	got, ok := chainRetrimLoop(chainPlaneThrough(math.P3(0, 0, 0), math.V3(0, 1, 0)), yHostRing(), chain, retrimChainTol)
	if !ok {
		t.Fatal("chainRetrimLoop declined the Y4 host plane")
	}
	assertChainRingClosed(t, got)
	assertChainArea(t, "Y4 host plane area", chainPlanarArea(got), 7500)
}

// y2NotchWallArc is the fillet band's imprint on the notch's inner wall x = 90: the band cylinder's
// cross-section there, from the host tangent line (90,0,85) up to where it leaves the wall at z = 90.
func y2NotchWallArc() endSeg {
	return chainCircleSeg(math.P3(90, 15, 85), math.V3(0, -1, 0), math.V3(0, 0, 1), 15, 0, stdmath.Asin(1.0/3))
}

// TestChainRetrimLoopBitesTheY2NotchWall: the notch's inner wall must lose the circular segment the
// band sweeps out of it, ∫₀⁵ (15 − √(225 − t²)) dt = 1.4130085, off a 10×100 rectangle.
func TestChainRetrimLoopBitesTheY2NotchWall(t *testing.T) {
	t.Parallel()
	ring := chainLineRing([]math.Point3{
		math.P3(90, 100, 80), math.P3(90, 100, 90), math.P3(90, 0, 90), math.P3(90, 0, 80),
	})
	got, ok := chainRetrimLoop(chainPlaneThrough(math.P3(90, 0, 0), math.V3(1, 0, 0)), ring, []endSeg{y2NotchWallArc()}, retrimChainTol)
	if !ok {
		t.Fatal("chainRetrimLoop declined the Y2 notch wall")
	}
	assertChainRingClosed(t, got)
	assertChainArea(t, "Y2 notch wall area", chainPlanarArea(got), 1000-y2WallBite())
}

// y2WallBite is ∫₀⁵ (15 − √(225 − t²)) dt in closed form — the area the r=15 band takes out of the
// notch's inner wall.
func y2WallBite() float64 {
	return 75 - (2.5*stdmath.Sqrt(200) + 112.5*stdmath.Asin(1.0/3))
}

// TestChainRetrimLoopBitesTheY2NotchTop: the notch's top face z = 90 loses the strip the band's own
// ruling there cuts, 10·(15 − √200) = 8.578644, off a 10×100 rectangle.
func TestChainRetrimLoopBitesTheY2NotchTop(t *testing.T) {
	t.Parallel()
	ring := chainLineRing([]math.Point3{
		math.P3(100, 100, 90), math.P3(100, 0, 90), math.P3(90, 0, 90), math.P3(90, 100, 90),
	})
	y := 15 - stdmath.Sqrt(200)
	chain := []endSeg{{from: math.P3(90, math.Scalar(y), 90), to: math.P3(100, math.Scalar(y), 90)}}
	got, ok := chainRetrimLoop(chainPlaneThrough(math.P3(0, 0, 90), math.V3(0, 0, 1)), ring, chain, retrimChainTol)
	if !ok {
		t.Fatal("chainRetrimLoop declined the Y2 notch top")
	}
	assertChainRingClosed(t, got)
	assertChainArea(t, "Y2 notch top area", chainPlanarArea(got), 1000-10*y)
}

// TestClipChainToRingTrimsTheY2WallAboveTheNotch is the CLIP half on a curved chain: the wall x = 100
// only exists for z ∈ [90,100], but the fillet's contact arc there runs all the way down to (100,0,85)
// — 5 past the face's own bottom edge. Clipped at z = 90 the face keeps
// 1000 − (150 − ∫₅¹⁵ √(225 − t²) dt) = 953.1276; transformLoop ships 964.0267 and the loop self-crosses.
func TestClipChainToRingTrimsTheY2WallAboveTheNotch(t *testing.T) {
	t.Parallel()
	ring := chainLineRing([]math.Point3{
		math.P3(100, 0, 90), math.P3(100, 100, 90), math.P3(100, 100, 100), math.P3(100, 0, 100),
	})
	chain := []endSeg{chainCircleSeg(math.P3(100, 15, 85), math.V3(0, 0, 1), math.V3(0, -1, 0), 15, 0, stdmath.Pi/2)}
	got, ok := chainRetrimLoop(chainPlaneThrough(math.P3(100, 0, 0), math.V3(1, 0, 0)), ring, chain, retrimChainTol)
	if !ok {
		t.Fatal("chainRetrimLoop declined the Y2 wall above the notch")
	}
	assertChainRingClosed(t, got)
	assertChainArea(t, "Y2 wall-above-notch area", chainPlanarArea(got), 1000-(150-y2UpperIntegral()))
}

// y2UpperIntegral is ∫₅¹⁵ √(225 − t²) dt in closed form.
func y2UpperIntegral() float64 {
	return 176.71458676442586 - (2.5*stdmath.Sqrt(200) + 112.5*stdmath.Asin(1.0/3))
}

// TestChainRetrimLoopInterruptsTheY2Band is the band consumer: the notch interrupts the quarter
// cylinder over x ∈ [90,100] for the sweep below z = 90, so the band keeps
// (π/2)·15·100 − 15·asin(1/3)·10 = 2305.2190. transformLoop ships the uninterrupted 2356.18.
func TestChainRetrimLoopInterruptsTheY2Band(t *testing.T) {
	t.Parallel()
	y := 15 - stdmath.Sqrt(200)
	ring := []endSeg{
		{from: math.P3(0, 0, 85), to: math.P3(100, 0, 85)},
		chainCircleSeg(math.P3(100, 15, 85), math.V3(0, 0, 1), math.V3(0, -1, 0), 15, stdmath.Pi/2, 0),
		{from: math.P3(100, 15, 100), to: math.P3(0, 15, 100)},
		chainCircleSeg(math.P3(0, 15, 85), math.V3(0, 0, 1), math.V3(0, -1, 0), 15, 0, stdmath.Pi/2),
	}
	chain := []endSeg{
		y2NotchWallArc(),
		{from: math.P3(90, math.Scalar(y), 90), to: math.P3(100, math.Scalar(y), 90)},
	}
	got, ok := chainRetrimLoop(y2BandCylinder(), ring, chain, retrimChainTol)
	if !ok {
		t.Fatal("chainRetrimLoop declined the Y2 band")
	}
	assertChainRingClosed(t, got)
	want := stdmath.Pi/2*15*100 - 15*stdmath.Asin(1.0/3)*10
	assertChainArea(t, "Y2 band area", chainDevelopedArea(got, y2BandChart), want)
}

// y2BandChart develops a point of the Y2 fillet band (radius 15 about the line y = 15, z = 85 along x)
// into the cylinder's own METRIC chart: arc length around the section, distance along the axis.
func y2BandChart(p math.Point3) (float64, float64) {
	dy, dz := float64(p.Y)-15, float64(p.Z)-85
	return 15 * stdmath.Atan2(dz, dy), float64(p.X)
}

// ---------------------------------------------------------------------------------------------
// complex/D8 — the far-end trim landing off the stop FACE
// ---------------------------------------------------------------------------------------------

const (
	d8CX = 223.39418029785 // the corner round's axis x
	d8CY = 59.093784332275 // the corner round's axis y
	d8CR = 24.0            // the corner round's radius
	d8BR = 30.0            // the fillet band's radius
	d8BZ = -20.0           // the fillet band's axis z
	d8DX = 6.0             // the round's axis x minus the band's axis x
)

// d8TrimCurve is complex/D8's far-end trim curve in closed form: the intersection of the r=30 band
// (axis along +Y at x = 217.39418029785, z = −20) with the radius-24 corner round (axis along +Z at
// (223.39418029785, 59.093784332275)), parameterised by the ROUND's own angular coordinate u, on which
// the round's own face spans exactly [−π/2, 0].
//
// The shipped edge is a geom.BSplineCurve fit of this curve running u ∈ [−π/2, +0.25268025514] and
// ending at (217.39418, 35.85588, 10) — 0.2527 rad, 6.064 of developed length, PAST the face's own
// u = 0 ruling. Using the exact curve rather than the fit keeps the acceptance a closed form.
type d8TrimCurve struct{ lo, hi float64 }

func (c d8TrimCurve) Domain() (float64, float64) { return 0, 1 }

func (c d8TrimCurve) PointAt(t float64) math.Point3 {
	u := c.lo + t*(c.hi-c.lo)
	dx := d8DX - d8CR*stdmath.Sin(u)
	return math.P3(math.Scalar(d8CX-d8CR*stdmath.Sin(u)), math.Scalar(d8CY-d8CR*stdmath.Cos(u)),
		math.Scalar(d8BZ+stdmath.Sqrt(d8BR*d8BR-dx*dx)))
}

func (c d8TrimCurve) TangentAt(t float64) math.Vector3 {
	const h = 1e-7
	a, b := c.PointAt(stdmath.Max(0, t-h)), c.PointAt(stdmath.Min(1, t+h))
	return a.VectorTo(b)
}

// d8RoundRing is the corner round's ORIGINAL boundary, read from the imported body: two full-height
// rulings at u = 0 and u = −π/2 and the two rim arcs at z = −90 and z = +10.
func d8RoundRing() []endSeg {
	u0, u90 := math.P3(d8CX, d8CY-d8CR, 0), math.P3(d8CX+d8CR, d8CY, 0)
	at := func(p math.Point3, z float64) math.Point3 { return math.P3(p.X, p.Y, math.Scalar(z)) }
	return []endSeg{
		chainCircleSeg(math.P3(d8CX, d8CY, -90), math.V3(0, -1, 0), math.V3(-1, 0, 0), d8CR, 0, -stdmath.Pi/2),
		{from: at(u90, -90), to: at(u90, 10)},
		chainCircleSeg(math.P3(d8CX, d8CY, 10), math.V3(0, -1, 0), math.V3(-1, 0, 0), d8CR, -stdmath.Pi/2, 0),
		{from: at(u0, 10), to: at(u0, -90)},
	}
}

// TestChainRetrimLoopStopsD8sFarEndTrimOnItsOwnFace is the far-end-trim consumer's acceptance. The trim
// curve overshoots the round's u = 0 ruling to u = +0.2527; clipped at the ruling it ends at
// (u=0, v=−99.393877) and the rebuild consumes the round's WHOLE far rim arc, which is exactly
// selfcross-trim-report.md §5.1's target geometry. The remaining face is the closed form
// 24·∫_{−π/2}^{0} (70 + √(900 − (6 − 24 sin u)²)) du = 3307.1168 — the number the shipped loop's
// shoelace (3305.9057) misses by the 1.2111 lobe it pinches off.
func TestChainRetrimLoopStopsD8sFarEndTrimOnItsOwnFace(t *testing.T) {
	t.Parallel()
	chain := []endSeg{d8TrimSeg()}
	got, ok := chainRetrimLoop(d8RoundCylinder(), d8RoundRing(), chain, 1e-6)
	if !ok {
		t.Fatal("chainRetrimLoop declined D8's corner round")
	}
	assertChainRingClosed(t, got)
	assertChainArea(t, "D8 stop wall area", chainDevelopedArea(got, d8Chart), d8StopWallArea())
}

// TestClipChainToRingCutsD8sOvershootAtTheRuling pins the clip point itself, independently of the
// splice: the analytic crossing is (6)² + (v+70)² = 900 ⇒ v = −99.393877, i.e. z = 9.393877 on the
// round's u = 0 ruling.
func TestClipChainToRingCutsD8sOvershootAtTheRuling(t *testing.T) {
	t.Parallel()
	got, ok := clipChainToRing(d8RoundRing(), []endSeg{d8TrimSeg()}, 1e-6)
	if !ok {
		t.Fatal("clipChainToRing declined D8's overshoot")
	}
	want := math.P3(d8CX, d8CY-d8CR, math.Scalar(d8BZ+stdmath.Sqrt(d8BR*d8BR-d8DX*d8DX)))
	if d := float64(got[len(got)-1].to.DistanceTo(want)); d > 1e-9 {
		t.Errorf("clip landed %.6g from the u=0 ruling crossing: got %+v want %+v", d, got[len(got)-1].to, want)
	}
}

// d8TrimSeg is the far-end trim as the retrim layer carries it: one segment from the band's tangency on
// the x = 247.394 wall to its overshot landing past the round's own u = 0 ruling.
func d8TrimSeg() endSeg {
	c := d8TrimCurve{lo: -stdmath.Pi / 2, hi: stdmath.Asin(0.25)}
	return endSeg{from: c.PointAt(0), to: c.PointAt(1), curve: c, mid: c.PointAt(0.5)}
}

// d8Chart develops a point of the corner round into its own metric chart (arc length around, height).
func d8Chart(p math.Point3) (float64, float64) {
	return d8CR * stdmath.Atan2(d8CX-float64(p.X), d8CY-float64(p.Y)), float64(p.Z)
}

// d8StopWallArea is 24·∫_{−π/2}^{0} (70 + √(900 − (6 − 24 sin u)²)) du by Simpson on 4096 panels — the
// integrand is smooth and LINEAR at the u = −π/2 endpoint (the radicand vanishes quadratically there),
// so composite Simpson converges at its full fourth order and is exact to ~1e-12 here.
func d8StopWallArea() float64 {
	f := func(u float64) float64 {
		dx := d8DX - d8CR*stdmath.Sin(u)
		return 70 + stdmath.Sqrt(d8BR*d8BR-dx*dx)
	}
	return d8CR * chainSimpson(f, -stdmath.Pi/2, 0, 4096)
}

// simpson is composite Simpson's rule on n (even) panels.
func chainSimpson(f func(float64) float64, a, b float64, n int) float64 {
	h := (b - a) / float64(n)
	sum := f(a) + f(b)
	for i := 1; i < n; i++ {
		w := 2.0
		if i%2 == 1 {
			w = 4
		}
		sum += w * f(a+float64(i)*h)
	}
	return sum * h / 3
}

// ---------------------------------------------------------------------------------------------
// the strangler guard and the invariant
// ---------------------------------------------------------------------------------------------

// TestChainRetrimLoopIsTheExistingSpliceWhenNothingOverruns is the ADOPTION guard: on a chain whose two
// extremes already lie on the ring — every splice the retrim layer performs today — the chain-capable
// rebuild returns spliceCornerBiteChain's result POINT FOR POINT. Routing an existing caller through the
// primitive therefore cannot move a corpus green; only a genuinely overrunning chain takes the new path.
func TestChainRetrimLoopIsTheExistingSpliceWhenNothingOverruns(t *testing.T) {
	t.Parallel()
	chain := []endSeg{{from: math.P3(100, 0, 75), to: math.P3(0, 0, 75)}}
	host := chainPlaneThrough(math.P3(0, 0, 0), math.V3(0, 1, 0))
	want, ok := spliceCornerBiteChain(host, yHostRing(), chain, retrimChainTol)
	if !ok {
		t.Fatal("spliceCornerBiteChain declined the reference case")
	}
	got, ok := chainRetrimLoop(host, yHostRing(), chain, retrimChainTol)
	if !ok {
		t.Fatal("chainRetrimLoop declined the reference case")
	}
	assertSameChainRing(t, got, want)
}

// TestClipChainToRingDeclinesAChainThatNeverReachesTheRing is the invariant's negative side: a chain
// running entirely off the face has no landing to splice at, and the primitive must decline rather than
// snap it to the nearest boundary point. A guessed landing is a silently wrong solid; a decline is the
// do-no-harm floor the rest of the retrim layer already falls to.
func TestClipChainToRingDeclinesAChainThatNeverReachesTheRing(t *testing.T) {
	t.Parallel()
	away := []endSeg{{from: math.P3(500, 0, 500), to: math.P3(600, 0, 500)}}
	if got, ok := clipChainToRing(yHostRing(), away, retrimChainTol); ok {
		t.Errorf("clipChainToRing accepted a chain that never reaches the ring: %+v", got)
	}
}

// ---------------------------------------------------------------------------------------------
// measurement helpers
// ---------------------------------------------------------------------------------------------

// circleSeg builds an arc boundary segment on the circle (center, radius) spanning [a0, a1] radians in
// the orthonormal in-plane frame (u, v).
func chainCircleSeg(center math.Point3, u, v math.Vector3, r, a0, a1 float64) endSeg {
	at := func(a float64) math.Point3 {
		return center.TranslateBy(u.Scale(math.Scalar(r * stdmath.Cos(a))).Add(v.Scale(math.Scalar(r * stdmath.Sin(a)))))
	}
	arc, err := geom.Arc3dByThreePoints(at(a0), at((a0+a1)/2), at(a1))
	if err != nil {
		panic(err)
	}
	return endSeg{from: at(a0), to: at(a1), curve: arc, mid: at((a0 + a1) / 2), arc: true}
}

// densePolyline develops a closed ring into points, sampling every carried curve finely enough that a
// measured area is limited by the geometry rather than by the sampler.
func chainRingPolyline(segs []endSeg) []math.Point3 {
	pts := make([]math.Point3, 0, len(segs)*areaSamplesPerSeg)
	for _, s := range segs {
		for i := range areaSamplesPerSeg {
			pts = append(pts, segPointAt(s, float64(i)/areaSamplesPerSeg))
		}
	}
	return pts
}

// planarRingArea is the enclosed area of a ring that lies in one plane.
func chainPlanarArea(segs []endSeg) float64 {
	return float64(newellNormal(chainRingPolyline(segs)).Length()) / 2
}

// developedArea is the enclosed area of a ring developed into a surface's own metric chart.
func chainDevelopedArea(segs []endSeg, chart func(math.Point3) (float64, float64)) float64 {
	pts := chainRingPolyline(segs)
	sum := 0.0
	for i := range pts {
		x0, y0 := chart(pts[i])
		x1, y1 := chart(pts[(i+1)%len(pts)])
		sum += x0*y1 - x1*y0
	}
	return stdmath.Abs(sum) / 2
}

// assertNear fails when got differs from the closed form by more than 1e-9 relative.
func assertChainArea(t *testing.T, what string, got, want float64) {
	t.Helper()
	if rel := stdmath.Abs(got-want) / stdmath.Abs(want); rel > 1e-9 {
		t.Errorf("%s: got %.10f, closed form %.10f (rel %.3g)", what, got, want, rel)
	}
}

// assertRingClosed fails when the rebuilt ring is not a closed cycle (each segment's `to` is the next
// segment's `from`) — the structural half of the primitive's postcondition.
func assertChainRingClosed(t *testing.T, ring []endSeg) {
	t.Helper()
	for i, s := range ring {
		if d := float64(s.to.DistanceTo(ring[(i+1)%len(ring)].from)); d > 1e-9 {
			t.Fatalf("ring is open at segment %d: to %+v does not meet next from %+v (gap %.6g)",
				i, s.to, ring[(i+1)%len(ring)].from, d)
		}
	}
}

// assertSameRing fails when two rings differ in length or in any vertex.
func assertSameChainRing(t *testing.T, got, want []endSeg) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("ring length %d, want %d", len(got), len(want))
	}
	for i := range got {
		if got[i].from != want[i].from || got[i].to != want[i].to {
			t.Errorf("segment %d: got %+v→%+v, want %+v→%+v", i, got[i].from, got[i].to, want[i].from, want[i].to)
		}
	}
}

// chainPlaneThrough is the host surface of a planar acceptance case — the face's own plane, so the
// splice's smaller-span pick is measured where the face actually lives. On a plane the developed
// measure IS the Newell one, so these cases are unchanged by the criterion.
func chainPlaneThrough(origin math.Point3, normal math.Vector3) geom.Plane {
	pl, err := geom.NewPlane(origin, normal)
	if err != nil {
		panic(err)
	}
	return pl
}

// y2BandCylinder is the Y2 fillet band's own surface: radius 15 about the line y = 15, z = 85 along x.
func y2BandCylinder() geom.Cylinder {
	c, err := geom.NewCylinder(math.P3(0, 15, 85), math.V3(1, 0, 0), 15)
	if err != nil {
		panic(err)
	}
	return c
}

// d8RoundCylinder is complex/D8's radius-24 corner round — the CURVED host whose span pick the
// developed criterion exists for.
func d8RoundCylinder() geom.Cylinder {
	c, err := geom.NewCylinder(math.P3(d8CX, d8CY, 0), math.V3(0, 0, 1), d8CR)
	if err != nil {
		panic(err)
	}
	return c
}
