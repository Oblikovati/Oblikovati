// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
)

// CN-C8 — the APEX/consumed-apex cone-host corner (OCCT blend/simple/C8, id G3): the corner vertex IS the
// cone apex (0,0,120), so the apex region is consumed entirely and the standard trihedral weld does not
// directly apply. This gate PINS the exact apex-strip topology on the REAL imported fixture driven through
// the full feature path (import → AddBase → AddFilletSets → Recompute), the strongest gate:
//
//   - the cone face is the STRIP between the two canal bands (one geom.Cone);
//   - both radial planes are cut down to bottom SLIVERS (two equal-area geom.Plane, + the untouched bottom
//     disc = three planes total);
//   - the corner ball wraps OVER the top as the ROOF — a geom.Sphere cap of Girard area ≈448.387 (spherical
//     excess 4.48387 sr = 448.387/r²), NOT its ~808 complement;
//   - the two Cone∧Plane RULINGS become two symmetric canal arms (geom.BSplineSurface), whose cone-side
//     rails BOTH end at the SAME pinch T (deduped — no synthesized bridge edge, so the edge count is 15).
//
// The whole solid is watertight (Valid+Closed+HolesContained+IsSolid, every edge exactly 2-incident),
// volume-positive, and EVERY face fold-free (the repo's highest-priority gate). C8 stays area-RED: its
// exact whole-body area 9781.45 is +1.46% over OCCT's 9640.68 (an exact-vs-OCCT deviation — OCCT's own C8
// corner is a non-tangent filled sag; see the DRAWEXE forensic in cnc8-report.md). The per-case override is
// CN6, so this gate asserts C8 is OVER the deps gate on purpose, and pins the exact value so a regression is
// caught. Do-no-harm: the CN4b-2 generic weld already builds this; this file is the verification receipt.

const (
	c8ExactArea = 9781.45 // our EXACT rolling-ball whole-body area (measured, CN-C8)
	c8Oracle    = 9640.68 // OCCT's reference area (embeds OCCT's sagging filled corner)
	c8Girard    = 448.387 // exact corner spherical-triangle Girard area (derivation §3)
	c8Faces     = 8       // strip + bottom disc + 2 slivers + 2 canals + cylinder + sphere
	c8Edges     = 15      // a synthesized pinch bridge would make 16
)

func TestC8ApexStripTopology(t *testing.T) {
	t.Parallel()
	body := caseResultBody(t, "C8")
	assertWatertight(t, "C8", body, c8Faces)
	assertC8FaceInventory(t, body)
	assertC8CornerRoofCap(t, body)
	assertC8FoldFree(t, body)
	assertC8DedupedPinch(t, body)
	assertC8AreaStaysRed(t, body)
}

// assertC8FaceInventory pins the apex-strip role counts and the two symmetries: the two canal arms are
// congruent (equal mesh area) and the two radial-plane slivers are congruent — the fingerprint of a
// symmetric 90° apex wedge. A mis-built strip/sliver (or a lost canal arm) breaks a count or a symmetry.
func assertC8FaceInventory(t *testing.T, body *topo.Body) {
	t.Helper()
	var bsplines, cylinders, spheres, cones int
	var planeAreas []float64
	for _, f := range body.Faces() {
		switch f.Geometry().(type) {
		case geom.BSplineSurface:
			bsplines++
		case geom.Cylinder:
			cylinders++
		case geom.Sphere:
			spheres++
		case geom.Cone:
			cones++
		case geom.Plane:
			planeAreas = append(planeAreas, faceMeshArea2(f))
		}
	}
	if bsplines != 2 || cylinders != 1 || spheres != 1 || cones != 1 || len(planeAreas) != 3 {
		t.Fatalf("C8 face inventory bspline=%d cyl=%d sph=%d cone=%d plane=%d, want 2/1/1/1/3",
			bsplines, cylinders, spheres, cones, len(planeAreas))
	}
	assertC8Symmetries(t, body, planeAreas)
}

// assertC8Symmetries checks the two canal-arm BSpline faces are congruent and two of the three planes (the
// radial slivers) are congruent — both required by the 90°-wedge mirror symmetry.
func assertC8Symmetries(t *testing.T, body *topo.Body, planeAreas []float64) {
	t.Helper()
	var canals []float64
	for _, f := range body.Faces() {
		if _, ok := f.Geometry().(geom.BSplineSurface); ok {
			canals = append(canals, faceMeshArea2(f))
		}
	}
	// Guard the two-arm assumption before indexing/dividing: a topology regression that dropped a
	// canal arm would otherwise panic (index) or divide by a zero area (NaN) instead of failing clean.
	if len(canals) != 2 || canals[0] == 0 {
		t.Fatalf("C8 expects exactly two non-degenerate canal-arm BSpline faces, got %d areas %v", len(canals), canals)
	}
	if rel := stdmath.Abs(canals[0]-canals[1]) / canals[0]; rel > 0.01 {
		t.Fatalf("C8 canal arms not congruent: %.3f vs %.3f (rel %.3f%% > 1%%)", canals[0], canals[1], rel*100)
	}
	if !twoEqualPlanes(planeAreas) {
		t.Fatalf("C8 radial slivers not congruent among plane areas %v", planeAreas)
	}
}

// twoEqualPlanes reports whether at least two of the plane areas match within 1% (the mirror-symmetric
// radial slivers), the third being the larger untouched bottom disc.
func twoEqualPlanes(areas []float64) bool {
	for i := range areas {
		for j := i + 1; j < len(areas); j++ {
			if stdmath.Abs(areas[i]-areas[j])/areas[i] <= 0.01 {
				return true
			}
		}
	}
	return false
}

// assertC8CornerRoofCap checks the corner geom.Sphere face meshes to the sub-hemisphere Girard CAP
// (≈448.387), not its ~808 complement — the ball is the ROOF of the solid. A uniform-flip regression that
// meshed the complement reads 4πr²−448 ≈ 808, far outside 1%.
func assertC8CornerRoofCap(t *testing.T, body *topo.Body) {
	t.Helper()
	f := cornerBlendSphereFace(body)
	if f == nil {
		t.Fatal("C8 carries no corner-blend sphere face")
	}
	if a := faceMeshArea2(f); stdmath.Abs(a-c8Girard) > 0.01*c8Girard {
		t.Fatalf("C8 corner cap area %.3f, want the exact Girard %.3f within 1%% (a complement-fill reads ~808)", a, c8Girard)
	}
}

// assertC8FoldFree gates the brief's highest priority: every face of the apex solid meshes fold-free.
func assertC8FoldFree(t *testing.T, body *topo.Body) {
	t.Helper()
	for _, f := range body.Faces() {
		assertFaceFoldFreeAtEveryQuality(t, "C8", f, nil)
	}
}

// assertC8DedupedPinch checks both canal cone-side rails end at the SAME pinch T without a synthesized
// bridge: the edge count is exactly 15 (a bridge would add one) and no edge is a zero-length sliver.
func assertC8DedupedPinch(t *testing.T, body *topo.Body) {
	t.Helper()
	if got := len(body.Edges()); got != c8Edges {
		t.Fatalf("C8 has %d edges, want %d (a synthesized pinch bridge adds one)", got, c8Edges)
	}
	band := 1e-6 * boundingDiag(body)
	for _, e := range body.Edges() {
		if l := float64(e.StartVertex().Point().DistanceTo(e.EndVertex().Point())); l < band {
			if !edgeIsClosedLoop(e) {
				t.Fatalf("C8 edge %d has coincident endpoints (len %.3g < %.3g) — a collapsed bridge sliver", e.ID(), l, band)
			}
		}
	}
}

// edgeIsClosedLoop reports whether an edge is a legitimate closed curve (start==end by design, e.g. a full
// circle), so its coincident endpoints are not a degenerate bridge.
func edgeIsClosedLoop(e *topo.Edge) bool {
	return e.StartVertex() == e.EndVertex()
}

// assertC8AreaStaysRed pins C8's EXACT whole-body area and confirms it is deliberately OVER the deps gate
// (the exact rolling-ball corner is larger than OCCT's sag) — C8 is NOT greened here; CN6 overrides it.
func assertC8AreaStaysRed(t *testing.T, body *topo.Body) {
	t.Helper()
	area := query.BodyGeometryProperties(body, ops.PropertyQuality()).Area
	if stdmath.Abs(area-c8ExactArea) > 0.005*c8ExactArea {
		t.Fatalf("C8 whole-body area %.3f drifted from the pinned exact %.3f (>0.5%%)", area, c8ExactArea)
	}
	if area <= c8Oracle*1.01 {
		t.Fatalf("C8 exact area %.3f should EXCEED OCCT's %.3f by >1%% (the exact ball cap over the sag)", area, c8Oracle)
	}
}
