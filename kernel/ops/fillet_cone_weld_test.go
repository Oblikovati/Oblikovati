// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/validate"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// CN4b-2 — the cone-host corner WELD greens OCCT blend/simple/{C2,C6} and builds the exact D1 snout + C8
// apex solids. These drive the REAL imported fixtures through the full weld (computeCorners → computeFillets →
// weldCurvedArmOrFloor) and pin: watertight (Valid+HolesContained+IsSolid), volume-positive, per-face
// FOLD-FREE (the brief's highest-priority gate), and the exact corner spherical-triangle Girard area.
// D1's whole-body area is the EXACT rolling-ball fillet (canal band verified against direct integration in
// the derivation), +1.39% over OCCT's snout APPROXIMATION — an honest exact-vs-OCCT deviation like C8,
// forensically confirmed against DRAWEXE per-face (torus/cylinder/plane match to <0.01%; OCCT's canal is
// 3.6% under the exact envelope). So D1's TOTAL is asserted as the exact value, not OCCT's approximation.
// C8 (CN-C8) joins them: its exact apex-strip solid is +1.46% over OCCT, whose C8 corner is a non-tangent
// filled sag (345.038 vs our exact ball cap 448.387) — the same exact-vs-OCCT posture, CN6 overrides both.

// coneWeldFixture is a cone-host rim/snout corner and its DRAWEXE-pinned corner centre + Girard area.
type coneWeldFixture struct {
	name        string
	corner      math.Point3
	girard      float64 // exact corner spherical-triangle area (derivation §3)
	wholeArea   float64 // the EXACT rolling-ball whole-body area (ours), pinned so a regression is caught
	oracleArea  float64 // OCCT's reference area (the corpus oracle)
	exactVsOCCT bool    // true ⇒ our exact area honestly exceeds OCCT's approximation (D1); C2/C6 match OCCT
}

func coneWeldFixtures() []coneWeldFixture {
	return []coneWeldFixture{
		{"C2", math.P3(90, 0, 0), 206.879, 40666.13, 40663.6, false},
		{"C6", math.P3(0, -40, 150), 144.309, 89364.89, 89366.1, false},
		{"D1", math.P3(50, 0, 0), 238.485, 10078.84, 9940.87, true},
		// C8 — the APEX/consumed-apex corner (corner vertex IS the cone apex (0,0,120), CN-C8). Its two
		// filleted edges are the two Cone∧Plane RULINGS (both canal arms end at the SAME pinch T, deduped —
		// no bridge), the apex wedge opens into the cut, so the corner ball wraps OVER the top as the ROOF
		// (Girard 448.387, spherical excess 4.48387 sr — far bigger than a rim corner). Whole area 9781.45 is
		// the EXACT rolling-ball fillet, +1.46% over OCCT's 9640.68: OCCT's own C8 corner is a non-tangent
		// filled BSpline sag (area 345.038, base rail z=60.1934 vs our exact ball centre z=60.0589, 0.052 off
		// cone tangency — DRAWEXE forensic in cnc8-report.md). An exact-vs-OCCT deviation like D1; CN6 overrides.
		{"C8", math.P3(0, 0, 120), 448.387, 9781.45, 9640.68, true},
	}
}

// weldConeCornerFixture imports a fixture, fillets the three edges meeting at its corner vertex, and
// returns the welded body (or fails). It mirrors the corpus feature path at the ops layer so the weld can
// be asserted face-by-face.
func weldConeCornerFixture(t *testing.T, fx coneWeldFixture) *topo.Body {
	t.Helper()
	body := importSimpleFixture(t, fx.name)
	v := nearestCornerVertex(body, fx.corner)
	picks := make([]filletPick, 0, 3)
	for _, e := range v.Edges() {
		picks = append(picks, filletPick{edge: e, r0: coneArmR, r1: coneArmR})
	}
	blends, miters, err := computeCorners(body, picks)
	if err != nil {
		t.Fatalf("%s: computeCorners: %v", fx.name, err)
	}
	fils, err := computeFillets(body, picks, blends, miters, FillConcaveOutward, nil)
	if err != nil {
		t.Fatalf("%s: computeFillets: %v", fx.name, err)
	}
	welded, err := weldCurvedArmOrFloor(body, fils, blends, miters)
	if err != nil {
		t.Fatalf("%s: weld: %v", fx.name, err)
	}
	return welded
}

// nearestCornerVertex returns the body vertex closest to p — the picked trihedral corner.
func nearestCornerVertex(b *topo.Body, p math.Point3) *topo.Vertex {
	var best *topo.Vertex
	bd := stdmath.Inf(1)
	for _, v := range b.Vertices() {
		if d := float64(v.Point().DistanceTo(p)); d < bd {
			bd, best = d, v
		}
	}
	return best
}

// TestConeCornerWeldWatertight pins the whole-body health gate: every cone-host corner welds to a valid,
// hole-contained, volume-positive solid (C2/C6/D1). The exact same gate the corpus harness applies.
func TestConeCornerWeldWatertight(t *testing.T) {
	if testing.Short() {
		t.Skip("corpus tier (~5s): `make test-corpus`")
	}
	t.Parallel()
	for _, fx := range coneWeldFixtures() {
		t.Run(fx.name, func(t *testing.T) {
			welded := weldConeCornerFixture(t, fx)
			rep := Validate(welded)
			if !rep.Valid || !rep.HolesContained || !welded.IsSolid() {
				t.Fatalf("%s: not watertight (valid=%v holes=%v solid=%v)", fx.name, rep.Valid, rep.HolesContained, welded.IsSolid())
			}
			if vol := BodyGeometryProperties(welded, PropertyQuality()).Volume; vol <= 0 {
				t.Fatalf("%s: volume not positive (%g)", fx.name, vol)
			}
			if n := everyEdgeTwoIncident(welded); n != 0 {
				t.Fatalf("%s: %d edges not exactly 2-incident (open/non-manifold shell)", fx.name, n)
			}
		})
	}
}

// everyEdgeTwoIncident counts edges NOT bordered by exactly two faces — a watertight solid has none.
func everyEdgeTwoIncident(b *topo.Body) int {
	bad := 0
	for _, e := range b.Edges() {
		if len(e.Faces()) != 2 {
			bad++
		}
	}
	return bad
}

// TestConeCornerWeldFoldFree is the brief's HIGHEST-priority gate: every welded face (the re-lofted canal
// arm band, the corner spherical triangle, the retrimmed cone/plane hosts) meshes to its true area with NO
// folds. A folded mesh over-counts area and corrupts every downstream consumer, so validate.FoldEdgeCount must be 0
// on every face of C2/C6/D1.
func TestConeCornerWeldFoldFree(t *testing.T) {
	if testing.Short() {
		t.Skip("corpus tier (~3s): `make test-corpus`")
	}
	t.Parallel()
	for _, fx := range coneWeldFixtures() {
		t.Run(fx.name, func(t *testing.T) {
			welded := weldConeCornerFixture(t, fx)
			for _, f := range welded.Faces() {
				if folds := validate.FoldEdgeCount(TessellateFace(f, PropertyQuality())); folds != 0 {
					t.Fatalf("%s: %T face has %d fold edges (mesh over-counts its area)", fx.name, f.Geometry(), folds)
				}
			}
		})
	}
}

// TestConeCornerGirardArea pins the corner spherical-triangle's exact Girard area (derivation §3): the
// corner blend is the exact equal-r ball's triangle, NOT OCCT's sagging filled patch. This also proves the
// corner blend meshes the sub-hemisphere CAP (not the complement) — the assembleCornerBlendBody uniform-flip
// fix — since the complement area would be 4πr² − girard ≈ 1000+, far outside the 1% tolerance.
func TestConeCornerGirardArea(t *testing.T) {
	if testing.Short() {
		t.Skip("corpus tier (~2s): `make test-corpus`")
	}
	t.Parallel()
	for _, fx := range coneWeldFixtures() {
		t.Run(fx.name, func(t *testing.T) {
			welded := weldConeCornerFixture(t, fx)
			_, mesh, ok := cornerSphereMesh(welded)
			if !ok {
				t.Fatalf("%s: no corner sphere face", fx.name)
			}
			area := meshGeometryProperties(mesh).Area
			if stdmath.Abs(area-fx.girard) > 0.01*fx.girard {
				t.Fatalf("%s: corner Girard area %.3f != exact %.3f (>1%%) — cap/complement mis-mesh?", fx.name, area, fx.girard)
			}
		})
	}
}

// cornerSphereMesh returns the corner blend sphere face's surface + mesh (the only spherical face of a
// cone-host corner).
func cornerSphereMesh(b *topo.Body) (geom.Sphere, *Mesh, bool) {
	for _, f := range b.Faces() {
		if s, ok := f.Geometry().(geom.Sphere); ok {
			return s, TessellateFace(f, PropertyQuality()), true
		}
	}
	return geom.Sphere{}, nil, false
}

// TestConeCornerWholeArea pins each welded solid's whole-body area. C2/C6 match OCCT to <0.01% (no snout,
// exact geometry = OCCT geometry). D1's exact rolling-ball area (10078.84) is +1.39% over OCCT's snout
// APPROXIMATION (9940.87) — asserted against the EXACT value, with the OCCT gap documented as an honest
// exact-vs-OCCT deviation (like C8), forensically confirmed against DRAWEXE.
func TestConeCornerWholeArea(t *testing.T) {
	if testing.Short() {
		t.Skip("corpus tier (~5s): `make test-corpus`")
	}
	t.Parallel()
	for _, fx := range coneWeldFixtures() {
		t.Run(fx.name, func(t *testing.T) {
			welded := weldConeCornerFixture(t, fx)
			area := BodyGeometryProperties(welded, PropertyQuality()).Area
			if stdmath.Abs(area-fx.wholeArea) > 0.005*fx.wholeArea {
				t.Fatalf("%s: whole-body area %.3f drifted from the pinned exact %.3f (>0.5%%)", fx.name, area, fx.wholeArea)
			}
			if !fx.exactVsOCCT && stdmath.Abs(area-fx.oracleArea) > 0.01*fx.oracleArea {
				t.Fatalf("%s: area %.3f not within 1%% of OCCT %.3f", fx.name, area, fx.oracleArea)
			}
			if fx.exactVsOCCT && area < fx.oracleArea {
				t.Fatalf("%s: exact area %.3f should EXCEED OCCT's approximation %.3f (snout under-approximation)", fx.name, area, fx.oracleArea)
			}
		})
	}
}
