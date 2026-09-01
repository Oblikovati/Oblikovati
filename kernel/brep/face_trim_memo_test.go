// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Guards for the per-face (u,v) development memo (M48/C3, Oblikovati/Oblikovati#3477). The memo caches
// an INPUT — the loops projected into the surface's parameter domain — never a decision, so the whole
// point is that it is invisible in the answers.

// memoProbePoints returns points on f's surface spanning its parameter domain, plus a few off it, so a
// containment sweep exercises both the inside and the outside of the trim.
func memoProbePoints(t *testing.T, f *topo.Face) []math.Point3 {
	t.Helper()
	s := f.Geometry()
	uLo, uHi := domainOrDefault(s.UDomain())
	vLo, vHi := domainOrDefault(s.VDomain())
	var out []math.Point3
	for i := range 7 {
		for j := range 7 {
			u := uLo + (uHi-uLo)*float64(i)/6
			v := vLo + (vHi-vLo)*float64(j)/6
			out = append(out, s.PointAt(u, v))
		}
	}
	return out
}

// domainOrDefault clamps an unbounded parameter range to a finite sweep window.
func domainOrDefault(lo, hi float64) (float64, float64) {
	if lo < -1e6 || hi > 1e6 {
		return -3, 3
	}
	return lo, hi
}

// memoTestBodies returns the fixture bodies the sweeps run over: a planar solid, a cylinder (periodic
// in u, seamed side face) and a sphere (no exterior cast axis, so it takes the geodesic-winding arm).
func memoTestBodies(t *testing.T) []*topo.Body {
	t.Helper()
	block, err := SolidBlock(math.P3(0, 0, 0), math.P3(2, 3, 4), "block")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	cyl, err := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 5)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	sphere, err := SolidSphere(math.P3(0, 0, 0), 2, "ball")
	if err != nil {
		t.Fatalf("SolidSphere: %v", err)
	}
	return []*topo.Body{block, cyl, sphere}
}

// TestTrimUVMemoChangesNoVerdict is the guard you cannot infer: every containment answer must be the
// same with the memo cold as with it warm. It classifies each probe with the memo cleared, then again
// with it populated, and requires equality — so a stale or mis-keyed development is caught as a
// DIFFERENT ANSWER rather than as a performance change nobody notices.
func TestTrimUVMemoChangesNoVerdict(t *testing.T) {
	t.Parallel()
	checked := 0
	for _, b := range memoTestBodies(t) {
		for _, f := range b.Faces() {
			for _, p := range memoProbePoints(t, f) {
				f.SetTrimUVMemo(nil) // cold: the development is rebuilt for this query
				cold := PointInFaceTrim(f, p)
				warm := PointInFaceTrim(f, p) // warm: the memo built above is reused
				if cold != warm {
					t.Fatalf("face %d at %v: cold=%v warm=%v — the memo changed the verdict", f.ID(), p, cold, warm)
				}
				checked++
			}
		}
	}
	if checked == 0 {
		t.Fatal("the sweep classified no points, so it proved nothing")
	}
}

// TestMemoizedTrimMatchesTheUnmemoizedPath compares the two implementations directly: PointInFaceTrim
// (memoized) against pointInTrimUV on a freshly flattened face (not memoized, the path the curved
// boolean's synthesized faces still take). They must agree on every probe, or the two have drifted.
func TestMemoizedTrimMatchesTheUnmemoizedPath(t *testing.T) {
	t.Parallel()
	for _, b := range memoTestBodies(t) {
		for _, f := range b.Faces() {
			for _, p := range memoProbePoints(t, f) {
				if got, want := PointInFaceTrim(f, p), pointInTrimUV(curvedFaceOf(f), p); got != want {
					t.Fatalf("face %d at %v: memoized=%v un-memoized=%v", f.ID(), p, got, want)
				}
			}
		}
	}
}

// TestTrimUVMemoIsBuiltOncePerFace pins the memo's whole reason for existing: the development is built
// on the first query and REUSED, not rebuilt. Falsify by dropping the SetTrimUVMemo store — the second
// query then hands back a different development and this goes red.
func TestTrimUVMemoIsBuiltOncePerFace(t *testing.T) {
	t.Parallel()
	body, err := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 5)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	f := body.Faces()[0]
	if f.TrimUVMemo() != nil {
		t.Fatal("a fresh face must carry no development yet")
	}
	first := faceTrimUVOf(f)
	if second := faceTrimUVOf(f); second != first {
		t.Error("the second query must reuse the first query's development, not rebuild it")
	}
	if f.TrimUVMemo() == nil {
		t.Error("the development must be stored on the face, or every caller pays to rebuild it")
	}
}

// TestTrimUVMemoDevelopsEveryLoop covers the development itself: a face with a hole must develop BOTH
// rings, since the even-odd count needs the hole to cancel the outer loop.
func TestTrimUVMemoDevelopsEveryLoop(t *testing.T) {
	t.Parallel()
	slab, err := SolidBlock(math.P3(0, 0, 0), math.P3(10, 10, 2), "slab")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	drilled, err := CutCylindricalHole(slab, math.P3(5, 5, 0), math.V3(0, 0, 1), 2)
	if err != nil {
		t.Fatalf("CutCylindricalHole: %v", err)
	}
	for _, f := range drilled.Faces() {
		if _, planar := f.Geometry().(geom.Plane); !planar || len(f.Loops()) < 2 {
			continue
		}
		m := faceTrimUVOf(f)
		if len(m.rings) != len(f.Loops()) {
			t.Errorf("face %d has %d loops but developed %d rings", f.ID(), len(f.Loops()), len(m.rings))
		}
		return
	}
	t.Fatal("the drilled slab produced no planar face with a hole, so the assertion proved nothing")
}
