// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// The trim classifier must not read a loop's traversal HANDEDNESS (Oblikovati/Oblikovati#3477).
// Handedness about S_u×S_v orients a shell only up to one global sign — orient_consistent.go fixes that
// sign from the whole body's signed volume, precisely because a geometrically valid body may carry the
// coherent but inverted choice. An OCCT-parity fillet result (corpus simple/B3) is exactly such a body,
// and while [faceTrimUV.contains] read handedness per face, every torus and sphere patch of it claimed
// the far side of its own surface. ops.SelfIntersections then reported three "crossings" at witness
// points sitting outside both faces' range boxes, and the body was refused as not watertight.
//
// So the region is the rings' even-odd interior in the covering (u, v) plane — the same orientation-free
// decision every open-domain surface already took — and [curvedFace.outerless], a TOPOLOGICAL datum the
// builder records and topo carries on the loop, names the one case where the face is the complement.

// yRimCapFace builds the radius-5 sphere trimmed by the circle at y = 4, walked in the given direction.
// The rim is a circle about +Y while the sphere's own chart runs about +Z, so the ring CLOSES in the
// covering plane (it circles no pole) and encloses the small y > 4 cap there.
func yRimCapFace(t *testing.T, forward bool) curvedFace {
	t.Helper()
	f := capRegionFixture(t, 5, 4, !forward)
	if got := len(f.loops[0].edges); got != 1 {
		t.Fatalf("the fixture cap has %d edges, want 1 (the rim)", got)
	}
	return f
}

// TestTrimVerdictIgnoresRingHandedness: the two windings of one rim are the SAME region to the trim
// classifier. Before the fix they were complements of each other, so a body whose closed-surface loops
// were coherently inverted had every such face read inside-out.
func TestTrimVerdictIgnoresRingHandedness(t *testing.T) {
	t.Parallel()
	for _, forward := range []bool{true, false} {
		f := yRimCapFace(t, forward)
		for _, c := range []struct {
			y    float64
			want bool
		}{{4.6, true}, {4.9, true}, {3.5, false}, {0, false}, {-4.5, false}} {
			p := meridianPoint(5, c.y, 1.1)
			if got := pointInTrimUV(f, p); got != c.want {
				t.Errorf("rim walked forward=%v claims %v (y=%g) = %v, want %v", forward, p, c.y, got, c.want)
			}
		}
	}
}

// TestOuterlessFaceOwnsTheRingComplement: the complement reading is what [curvedFace.outerless] names,
// and it too is independent of how the rim is walked.
func TestOuterlessFaceOwnsTheRingComplement(t *testing.T) {
	t.Parallel()
	for _, forward := range []bool{true, false} {
		f := yRimCapFace(t, forward)
		f.outerless = true
		for _, c := range []struct {
			y    float64
			want bool
		}{{4.6, false}, {3.5, true}, {0, true}, {-4.5, true}} {
			p := meridianPoint(5, c.y, 1.1)
			if got := pointInTrimUV(f, p); got != c.want {
				t.Errorf("outerless rim forward=%v claims %v (y=%g) = %v, want %v", forward, p, c.y, got, c.want)
			}
		}
	}
}

// TestDisjointRingsKeepTheFootClassifier: two rims that do NOT nest bound the belt between them and the
// two caps beyond them equally, so the rings alone name no region and the winding still has to. A sphere
// zone is exactly that face, and ops' region integral gates the convention on it
// (TestSphereZoneRegionIntegralNamesTheBelt).
func TestDisjointRingsKeepTheFootClassifier(t *testing.T) {
	t.Parallel()
	belt := capRegionFixture(t, 5, 4, true)
	belt.loops = append(belt.loops, capRegionFixture(t, 5, -1, false).loops...)
	if m := developFaceTrim(belt); m.ringsBound {
		t.Fatal("two disjoint rims must not be read as bounding one region")
	}
	for _, c := range []struct {
		y    float64
		want bool
	}{{4.5, false}, {2, true}, {0, true}, {-2, false}} {
		p := meridianPoint(5, c.y, 0.6)
		if got := pointInTrimUV(belt, p); got != c.want {
			t.Errorf("the belt claims %v (y=%g) = %v, want %v", p, c.y, got, c.want)
		}
	}
}

// TestWrappingRingKeepsTheFootClassifier: a rim that circles the surface's own periodic axis — a
// latitude circle about the sphere's chart axis — bounds NO region of the covering plane, so its ring
// cannot be entered by an even-odd ray and the nearest-foot classifier still decides. Handedness names
// the region there, and must keep doing so.
func TestWrappingRingKeepsTheFootClassifier(t *testing.T) {
	t.Parallel()
	sphere, err := geom.NewSphere(math.P3(0, 0, 0), 5)
	if err != nil {
		t.Fatalf("sphere: %v", err)
	}
	rim, err := geom.NewCircle(math.P3(0, 0, 3), math.V3(0, 0, 1), 4)
	if err != nil {
		t.Fatalf("latitude rim: %v", err)
	}
	f := curvedFace{surface: sphere, loops: []curvedLoop{{edges: []loopEdge{{curve: rim, t0: 0, t1: 1}}}}}
	if m := developFaceTrim(f); m.ringsBound {
		t.Fatal("a latitude ring about the chart axis must be seen to WRAP, not to bound a region")
	}
	for _, c := range []struct {
		z    float64
		want bool
	}{{4.5, true}, {3.5, true}, {2, false}, {-4.5, false}} {
		p := sphere.PointAt(1.1, stdmath.Asin(c.z/5))
		if got := pointInTrimUV(f, p); got != c.want {
			t.Errorf("the latitude cap claims %v (z=%g) = %v, want %v", p, c.z, got, c.want)
		}
	}
}
