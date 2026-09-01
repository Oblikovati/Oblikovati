// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// A plane PARALLEL to a torus axis, offset between the inner and outer tube radii, slices off a single
// outer-tube oval cap: one analytic torus face (the surface inside the spiric oval) plus one planar oval
// lid, watertight, no CSG — the first exact OBLIQUE torus cut (Oblikovati/Oblikovati#1375).
func TestHalfSpaceCutTorusAxisParallelOvalCap(t *testing.T) {
	t.Parallel()
	tor, err := SolidTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 2, "torus")
	if err != nil {
		t.Fatalf("SolidTorus: %v", err)
	}
	// Keep the cap y≥6: HalfSpaceCut keeps the negative side, so the cut normal points −y (into the torus).
	plane, _ := geom.NewPlane(math.P3(0, 6, 0), math.V3(0, -1, 0))
	res, err := HalfSpaceCut(tor, plane)
	if err != nil {
		t.Fatalf("HalfSpaceCut: %v", err)
	}
	assertWatertight(t, res)
	tori, planes := 0, 0
	for _, f := range res.Faces() {
		switch f.Geometry().(type) {
		case geom.Torus:
			tori++
		case geom.Plane:
			planes++
		}
	}
	if tori != 1 || planes != 1 {
		t.Errorf("cap has %d torus + %d plane faces, want exactly 1 + 1 (analytic cap + oval lid)", tori, planes)
	}
	// The cap is one bigon loop (two spiric branches meeting at the two oval pinch vertices).
	if e := len(res.Edges()); e != 2 {
		t.Errorf("cap has %d edges, want 2 (the two spiric branches)", e)
	}
	if vtx := len(res.Vertices()); vtx != 2 {
		t.Errorf("cap has %d vertices, want 2 (the oval's v-extreme pinches)", vtx)
	}
}

// The same plane keeping the OTHER side leaves the genus-1 COMPLEMENT: the full torus minus the oval cap,
// kept as one torus face carrying the oval as a HOLE (no outer loop, it wraps the whole surface) plus the
// oval lid, watertight, no CSG (Oblikovati/Oblikovati#1375).
func TestHalfSpaceCutTorusAxisParallelComplement(t *testing.T) {
	t.Parallel()
	tor, _ := SolidTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 2, "torus")
	// Keep y≤6 (the big complement): the cut normal points +y, leaving the rest of the torus on its negative side.
	plane, _ := geom.NewPlane(math.P3(0, 6, 0), math.V3(0, 1, 0))
	res, err := HalfSpaceCut(tor, plane)
	if err != nil {
		t.Fatalf("HalfSpaceCut: %v", err)
	}
	assertWatertight(t, res)
	tori, planes := 0, 0
	var torusFace = -1
	for i, f := range res.Faces() {
		switch f.Geometry().(type) {
		case geom.Torus:
			tori++
			torusFace = i
		case geom.Plane:
			planes++
		}
	}
	if tori != 1 || planes != 1 {
		t.Fatalf("complement has %d torus + %d plane faces, want exactly 1 + 1", tori, planes)
	}
	// The complement torus face carries the oval as a HOLE (an inner loop), with no outer loop.
	for _, l := range res.Faces()[torusFace].Loops() {
		if l.IsOuter() {
			t.Error("complement torus face has an outer loop; the oval must be a hole (so the face wraps the whole torus)")
		}
	}
}

// The cap/complement predicate ladder (torusSingleOvalCap/Complement, torusAxisParallelOval) and the clampUnitF
// helper were removed when the single oval migrated to the unified (u,v)-arrangement trimmer (#1406); the
// integration tests above (TestHalfSpaceCutTorusAxisParallelOvalCap/Complement) now exercise that path.
