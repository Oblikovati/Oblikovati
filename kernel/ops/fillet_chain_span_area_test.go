// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// The developed-span criterion's acceptance, and the FALSIFICATION of the projected one it replaces.
//
// Every expectation here is a CLOSED FORM of the region the ring bounds on its own host: a cylinder
// patch of angular extent φ, height Δz and radius R has developed area R·φ·Δz, and its Newell (vector)
// area is |∫∫ n dA| = 2·R·Δz·sin(φ/2) — the projection factor 2sin(φ/2)/φ, which is 1 only in the limit
// φ→0 and falls to 0.0952 at φ = 2π·0.955. That factor is per-SPAN, which is exactly why the projected
// criterion can invert a comparison.

// spanAreaCylR / spanAreaCylTheta / … are the falsification host: a radius-10 cylinder whose trimmed
// face is L-SHAPED in its own chart — a wide, short band u ∈ [0, 6] × z ∈ [0, 1] with a narrow, tall
// tower u ∈ [0, 0.3] × z ∈ [0, 8] rising off its left end. Both pieces are ordinary trimmed-cylinder
// geometry (a nearly-closed cylindrical wall with a notch); nothing here is contrived except that the
// two spans have DIFFERENT angular extents, which is the case the projected measure cannot rank.
const (
	spanAreaCylR     = 10.0 // cylinder radius
	spanAreaCylTheta = 6.0  // the face's total angular extent (rad) — 0.955 of a full turn
	spanAreaCylSplit = 0.3  // the tower's angular extent, where the chain's ruling stands
	spanAreaBandTop  = 1.0  // the wide band's height
	spanAreaTowerTop = 8.0  // the tower's height
)

// spanAreaCylinder is the falsification host, with u = 0 pinned to +X so the chart below is the
// surface's own.
func spanAreaCylinder() geom.Cylinder {
	c, err := geom.NewCylinderWithRef(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), spanAreaCylR)
	if err != nil {
		panic(err)
	}
	return c
}

// spanAreaAt is the host's own parametrisation, used to build the ring so the closed forms below are
// statements about the SURFACE, not about a fit.
func spanAreaAt(u, z float64) math.Point3 {
	return math.P3(math.Scalar(spanAreaCylR*stdmath.Cos(u)), math.Scalar(spanAreaCylR*stdmath.Sin(u)), math.Scalar(z))
}

// spanAreaRimArc is a z = const rim arc from u0 to u1, split into pieces no wider than 1.5 rad so the
// ring is built the way a real trimmed face carries its rim.
func spanAreaRimArc(z, u0, u1 float64) []endSeg {
	n := int(stdmath.Ceil(stdmath.Abs(u1-u0)/1.5 - 1e-12))
	out := make([]endSeg, 0, n)
	for i := 0; i < n; i++ {
		a := u0 + (u1-u0)*float64(i)/float64(n)
		b := u0 + (u1-u0)*float64(i+1)/float64(n)
		out = append(out, chainCircleSeg(math.P3(0, 0, math.Scalar(z)), math.V3(1, 0, 0), math.V3(0, 1, 0), spanAreaCylR, a, b))
	}
	return out
}

// spanAreaLRing is the L-shaped face boundary, counter-clockwise in the chart:
// (0,0) → (Θ,0) → (Θ,1) → (0.3,1) → (0.3,8) → (0,8) → close.
func spanAreaLRing() []endSeg {
	ruling := func(u, z0, z1 float64) endSeg {
		return endSeg{from: spanAreaAt(u, z0), to: spanAreaAt(u, z1)}
	}
	ring := spanAreaRimArc(0, 0, spanAreaCylTheta)
	ring = append(ring, ruling(spanAreaCylTheta, 0, spanAreaBandTop))
	ring = append(ring, spanAreaRimArc(spanAreaBandTop, spanAreaCylTheta, spanAreaCylSplit)...)
	ring = append(ring, ruling(spanAreaCylSplit, spanAreaBandTop, spanAreaTowerTop))
	ring = append(ring, spanAreaRimArc(spanAreaTowerTop, spanAreaCylSplit, 0)...)
	return append(ring, ruling(0, spanAreaTowerTop, 0))
}

// spanAreaChain is the bite chain: the ruling at u = 0.3 from the bottom rim up to the band's top, which
// cuts the L into the wide band (right) and the tall tower (left).
func spanAreaChain() []endSeg {
	return []endSeg{{from: spanAreaAt(spanAreaCylSplit, 0), to: spanAreaAt(spanAreaCylSplit, spanAreaBandTop)}}
}

// spanAreaBandArea / spanAreaTowerArea are the two spans' closed-form DEVELOPED areas, R·φ·Δz.
func spanAreaBandArea() float64 {
	return spanAreaCylR * (spanAreaCylTheta - spanAreaCylSplit) * spanAreaBandTop
}

func spanAreaTowerArea() float64 {
	return spanAreaCylR * spanAreaCylSplit * spanAreaTowerTop
}

// TestDevelopedSpanAreaIsExactOnACylinderPatch pins the criterion against the closed form on the host
// family it exists for. A cylinder's metric is u-independent, so the Green quadrature's every integrand
// is inside its rule's exactness degree — the value is exact to rounding, NOT merely convergent, and it
// is exact at the ring's coarse production sampling because a z = const arc's chart image is a straight
// line whatever the sample count.
func TestDevelopedSpanAreaIsExactOnACylinderPatch(t *testing.T) {
	ring := append(spanAreaRimArc(0, 0, 2.0),
		endSeg{from: spanAreaAt(2.0, 0), to: spanAreaAt(2.0, 3.0)})
	ring = append(ring, spanAreaRimArc(3.0, 2.0, 0)...)
	ring = append(ring, endSeg{from: spanAreaAt(0, 3.0), to: spanAreaAt(0, 0)})
	got := developedSpanArea(spanAreaCylinder(), segPolyline(ring))
	want := spanAreaCylR * 2.0 * 3.0
	if rel := stdmath.Abs(got-want) / want; rel > 1e-12 {
		t.Errorf("cylinder patch developed area: got %.12f, closed form %.12f (rel %.3g)", got, want, rel)
	}
}

// TestDevelopedSpanAreaIsTheNewellAreaOnAPlane is the no-regression half: on a planar host the criterion
// must be the SAME NUMBER the projected one produced, bit for bit, so no planar corpus pick can move.
func TestDevelopedSpanAreaIsTheNewellAreaOnAPlane(t *testing.T) {
	ring := []math.Point3{math.P3(0, 0, 0), math.P3(10, 0, 0), math.P3(10, 0, 4), math.P3(3, 0, 4)}
	plane := chainPlaneThrough(math.P3(0, 0, 0), math.V3(0, 1, 0))
	newell := float64(newellNormal(ring).Length()) / 2
	if got := developedSpanArea(plane, ring); got != newell {
		t.Errorf("planar host: developed %.17g must be the Newell area %.17g bit for bit", got, newell)
	}
}

// ★ TestProjectedSpanAreaPicksTheWRONGCornerWhereTheDevelopedOnePicksRight is the FALSIFICATION the
// developed criterion exists for, in both directions on ONE case.
//
// The L-shaped host is cut by a ruling into a wide short band (φ = 5.7 rad, Δz = 1) and a narrow tall
// tower (φ = 0.3 rad, Δz = 8). Their DEVELOPED areas are 57 and 24 — the tower is the smaller corner,
// and the splice must remove it. Their PROJECTED areas are 2·10·1·sin(2.85) = 5.748 and
// 2·10·8·sin(0.15) = 23.910 — the ranking is INVERTED, because the band's 5.7 rad of wrap costs it 90 %
// of its width in projection while the tower's 0.3 rad costs it 0.4 %. A projected criterion therefore
// removes the band: it keeps 24 of a 81-unit face and throws away 57.
func TestProjectedSpanAreaPicksTheWRONGCornerWhereTheDevelopedOnePicksRight(t *testing.T) {
	host, ring, chain := spanAreaCylinder(), spanAreaLRing(), spanAreaChain()
	band, tower := spanAreaSpans(t, ring, chain)

	devBand, devTower := developedSpanArea(host, band), developedSpanArea(host, tower)
	assertChainArea(t, "band span, developed", devBand, spanAreaBandArea())
	assertChainArea(t, "tower span, developed", devTower, spanAreaTowerArea())
	if devTower >= devBand {
		t.Fatalf("premise broken: the tower (%.4f) must be the SMALLER span (band %.4f)", devTower, devBand)
	}

	projBand, projTower := spanAreaNewell(band), spanAreaNewell(tower)
	if projBand >= projTower {
		t.Fatalf("falsification is inert: the projected areas (band %.4f, tower %.4f) do not invert the ranking",
			projBand, projTower)
	}
	t.Logf("projected: band=%.4f tower=%.4f (ranks the band smaller — WRONG)", projBand, projTower)
	t.Logf("developed: band=%.4f tower=%.4f (ranks the tower smaller — RIGHT)", devBand, devTower)

	got, ok := spliceCornerBiteChain(host, ring, chain, 1e-9)
	if !ok {
		t.Fatal("spliceCornerBiteChain declined the L-shaped host")
	}
	assertChainRingClosed(t, got)
	assertChainArea(t, "spliced face, developed", developedSpanArea(host, segPolyline(got)), spanAreaBandArea())
}

// spanAreaSpans splits the ring at the chain's two extremes and returns the two spans' sampled point
// rings, closed by the chain — the very rings spliceCornerBiteChain's criterion is handed.
func spanAreaSpans(t *testing.T, ring, chain []endSeg) (band, tower []math.Point3) {
	t.Helper()
	p0, p1 := chain[0].from, chain[len(chain)-1].to
	split := insertSplits(ring, []math.Point3{p0, p1}, 1e-9)
	i, j := indexOfSegFrom(split, p0, 1e-9), indexOfSegFrom(split, p1, 1e-9)
	if i < 0 || j < 0 {
		t.Fatalf("chain extremes are not on the ring: i=%d j=%d", i, j)
	}
	return spanAreaClosedRing(segsForward(split, i, j)), spanAreaClosedRing(segsForward(split, j, i))
}

// spanAreaClosedRing samples a span and appends its far endpoint, matching chainBiteArea's own ring
// assembly for a chain that carries no interior bulge (the ruling here is straight).
func spanAreaClosedRing(span []endSeg) []math.Point3 {
	return append(segPolyline(span), span[len(span)-1].to)
}

// spanAreaNewell is the criterion this change REPLACES — |newellNormal|/2, the projected area.
func spanAreaNewell(ring []math.Point3) float64 {
	return float64(newellNormal(ring).Length()) / 2
}
