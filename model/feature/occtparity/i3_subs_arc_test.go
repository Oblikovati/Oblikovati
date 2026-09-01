// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/topo"
)

// TestI3SubsArcGreen is the whole-body gate on I3 — a straight LineSegment fillet between a planar ANNULAR
// SECTOR host (two concentric arcs r=200/r=300 + two radial edges) and a triangular end-cap. transformLoop's
// `subs` branch (a fillet A/B-corner pulled back to its tangent point) used to CHORD the leaving survivor
// edge — the sector's r=300 outer arc — collapsing the host plane 38270→14084 (−63%) and folding the
// neighbour cone (conformCylConeFaces), inflating the whole body to +2.55% (i3-recon-rootcause.md). This
// asserts, on the REAL STEP body, that the subs-branch survivor-arc carry (fillet_survivor_rim.go) plus the
// conformance do-no-harm guard green it: a watertight fold-free solid whose whole-body area matches OCCT
// within deps AND whose annular-sector host plane has RECOVERED its area (≈38270, not the chorded ≈14084).
func TestI3SubsArcGreen(t *testing.T) {
	t.Parallel()
	body := caseResultBody(t, "I3")
	assertWatertight(t, "I3", body, 6)
	assertFoldFreeFaces(t, "I3", body)
	assertWholeBodyArea(t, "I3", body, 99301.8)
	assertSectorPlaneRecovered(t, body)
}

// assertSectorPlaneRecovered fails unless the large annular-sector host plane (the one the subs branch used
// to chord) tessellates to its true ~38270 area — a floor set ABOVE the ~14084 chord-collapse yet below the
// true value, so a regression that re-chords the outer rim fails loud. The chord bite is ~24000, so any
// value above 30000 proves the rim arc was carried, not straightened.
func assertSectorPlaneRecovered(t *testing.T, body *topo.Body) {
	t.Helper()
	best := 0.0
	for _, f := range body.Faces() {
		if _, ok := f.Geometry().(geom.Plane); !ok {
			continue
		}
		if a := faceMeshArea2(f); a > best {
			best = a
		}
	}
	if best < 30000 || stdmath.Abs(best-38270) > 0.02*38270 {
		t.Fatalf("I3 annular-sector host plane area %.1f, want ~38270 (a re-chorded outer rim collapses it to ~14084)", best)
	}
}

// TestI3ConeFoldFree guards the OTHER half of the defect: the two host cones. Before the fix the plane/cone
// boundary disagreed on the shared rim, so conformCylConeFaces over-extended and FOLDED the cone
// (30514→41730, fold=1); the crude arc-carry made it worse (fold=4). The carry restores the shared rim AND
// the conformance guard keeps each cone's fold-free mesh, so every cone here meshes fold-free and positive.
func TestI3ConeFoldFree(t *testing.T) {
	t.Parallel()
	body := caseResultBody(t, "I3")
	cones := 0
	for _, f := range body.Faces() {
		if _, ok := f.Geometry().(geom.Cone); !ok {
			continue
		}
		cones++
		m := tessellate.TessellateFace(f, ops.PropertyQuality())
		if a := ops.MeshArea(m); a <= 0 || stdmath.IsInf(a, 0) || stdmath.IsNaN(a) {
			t.Fatalf("I3 cone face meshed to %.4f, want a finite positive area", a)
		}
		assertFaceFoldFreeAtEveryQuality(t, "I3 cone", f, m)
	}
	if cones != 2 {
		t.Fatalf("I3 has %d cone faces, want 2", cones)
	}
}
