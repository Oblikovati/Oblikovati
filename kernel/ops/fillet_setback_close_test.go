// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// s1SetbackAssembled builds the FULL intact-boss result for S1 and assembles it: every original body
// face transformed as usual (each crossing boss wall left INTACT — footprint SUBDIVIDED by
// buildSetbackFaces' inserts but never split, never emitted; the two host planes RE-CLIPPED via
// set.replace) + the two plain-R wings + the resolved setback patches. It mirrors filletResultFaces'
// composition (transformedBodyFaces + extra) but drives buildSetbackFaces directly — the only way to
// validate closure, since a shell can only be judged by ASSEMBLING it (assembleBody's weld invariant).
func s1SetbackAssembled(t *testing.T) *topo.Body {
	t.Helper()
	ef, body := s1SetbackEdge(t)
	maps := s1RebuildMaps(body, ef)
	res := ResolutionForBody(body)
	b, ok := detectSetbackBands(ef, res)
	if !ok {
		t.Fatal("s1SetbackAssembled: detectSetbackBands ok=false")
	}
	loops, ok := extractSetbackPatches(b, ef, res)
	if !ok {
		t.Fatal("s1SetbackAssembled: extractSetbackPatches ok=false")
	}
	set := runoutSet{replace: map[uint64]filletFace{}}
	if !buildSetbackFaces(&set, ef, b, loops, res, maps) {
		t.Fatal("s1SetbackAssembled: buildSetbackFaces ok=false")
	}
	faces := append(transformedBodyFaces(body, maps, set.replace), set.extra...)
	return assembleBody(faces, "fillet")
}

// s1SetbackEdge resolves S1's front-top edge fillet (R=6, its corpus radius) — the same edgeFillet the
// runout fixture uses, but paired with its body so the closure test can transform the body faces.
func s1SetbackEdge(t *testing.T) (edgeFillet, *topo.Body) {
	t.Helper()
	body := importCorpusSolid(t, "simple/S1")
	e := edgeAtMidpoint(body, math.P3(0, -10, 10))
	if e == nil {
		t.Fatal("s1SetbackEdge: front-top edge (0,-10,10) not found")
	}
	fil, err := computeEdgeFillet(body, filletPick{edge: e, r0: 6, r1: 6},
		map[uint64]*cornerBlend{}, map[uint64]*cornerMiter{}, FillConcaveOutward)
	if err != nil {
		t.Fatalf("s1SetbackEdge: computeEdgeFillet: %v", err)
	}
	return fil, body
}

// s1RebuildMaps builds the same filletRebuildMaps filletResultFaces threads into the rebuild path
// (fillet_faces.go). buildSetbackFaces mutates maps.edgeInserts to subdivide the intact boss-wall
// footprint rims, which transformedBodyFaces then applies — so the maps MUST be the real ones, not a
// zero value.
func s1RebuildMaps(body *topo.Body, ef edgeFillet) filletRebuildMaps {
	abSubst, endCorner, edgeInserts := filletMaps([]edgeFillet{ef})
	fans, fanV := classifyEndCorners([]edgeFillet{ef})
	spreads, _ := buildSpreadMaps(fans, body)
	pruneEndCorners(endCorner, fanV)
	return filletRebuildMaps{abSubst: abSubst, endCorner: endCorner, edgeInserts: edgeInserts, spreads: spreads}
}

// TestBuildSetbackFaces_S1Watertight is the closure gate: the assembled intact-boss S1 body must be a
// watertight, hole-contained solid whose tessellated area is within OCCT's 1% (ref 3662.79, forensics
// §8.2). A partial fill (an open shell) or a surviving hole would fail here.
func TestBuildSetbackFaces_S1Watertight(t *testing.T) {
	body := s1SetbackAssembled(t)
	if !body.IsSolid() {
		t.Fatalf("S1 intact-boss shell is not a solid: %d open edges", len(openEdges(body)))
	}
	r := Validate(body)
	if !r.Valid || !r.HolesContained {
		t.Fatalf("S1 intact-boss shell invalid: Valid=%v HolesContained=%v", r.Valid, r.HolesContained)
	}
	area := BodyGeometryProperties(body, PropertyQuality()).Area
	if area < 3626.2 || area > 3699.4 {
		t.Fatalf("S1 area %.4f outside OCCT 1%% gate [3626.2, 3699.4] (ref 3662.79)", area)
	}
}

// TestBuildSetbackFaces_S1BossWallsIntact pins the do-no-harm invariant that distinguishes this path
// from the old split tiler: each crossing boss wall survives as ONE face at (near) its full analytic
// area — r8 cyl 753.982, r6 cyl 565.487 (forensics §8.2) — never split into sub-faces. The tessellated
// area is a straight-chord polygon of the footprint rim, so a sub-1% band under the analytic value is
// the subdivision, not a lost region.
func TestBuildSetbackFaces_S1BossWallsIntact(t *testing.T) {
	body := s1SetbackAssembled(t)
	if got := countCylFacesNear(body, 753.982, 7.6); got != 1 {
		t.Fatalf("want exactly ONE intact r8 boss wall near 753.982 (un-split), got %d", got)
	}
	if got := countCylFacesNear(body, 565.487, 5.7); got != 1 {
		t.Fatalf("want exactly ONE intact r6 boss wall near 565.487 (un-split), got %d", got)
	}
}

// TestBuildSetbackFaces_S1HostsSingleLoop pins forensics §3/§8.2: every result face is WIRE:1 — the boss
// footprint holes are OPENED into the fillet cut (re-clipped host planes), not preserved as inner loops.
func TestBuildSetbackFaces_S1HostsSingleLoop(t *testing.T) {
	body := s1SetbackAssembled(t)
	for _, f := range body.Faces() {
		if n := len(f.Loops()); n != 1 {
			t.Fatalf("face %T has %d loops, want WIRE:1 (footprint opened into the cut)", f.Geometry(), n)
		}
	}
}

// TestBuildSetbackFaces_HonestReject proves the do-no-harm reject: a non-S1 input (the behind-band
// fixture, whose bosses never reach the receded band) detects no setback bands, so buildSetbackFaces is
// never reached — but resolveSetbackTiling on its empty bands must decline, so the whole edge falls back.
func TestBuildSetbackFaces_HonestReject(t *testing.T) {
	ef, res := runoutFixtureBehindBand(t)
	b, _ := detectSetbackBands(ef, res)
	set := runoutSet{replace: map[uint64]filletFace{}}
	if buildSetbackFaces(&set, ef, b, nil, res, filletRebuildMaps{}) {
		t.Fatal("buildSetbackFaces must reject a non-S1 (behind-band) input, never a partial fill")
	}
}

// countCylFacesNear counts cylinder faces whose tessellated area is within tol of want — the intact-wall
// presence test (exactly one per boss radius proves un-split).
func countCylFacesNear(body *topo.Body, want, tol float64) int {
	n := 0
	for _, f := range body.Faces() {
		if _, ok := f.Geometry().(geom.Cylinder); !ok {
			continue
		}
		if a := tessArea(TessellateFace(f, PropertyQuality())); a >= want-tol && a <= want+tol {
			n++
		}
	}
	return n
}

// tessArea sums a mesh's triangle areas.
func tessArea(m *Mesh) float64 {
	a := 0.0
	for i := 0; i+2 < len(m.Indices); i += 3 {
		p, q, r := m.Positions[m.Indices[i]], m.Positions[m.Indices[i+1]], m.Positions[m.Indices[i+2]]
		a += float64(p.VectorTo(q).Cross(p.VectorTo(r)).Length()) / 2
	}
	return a
}
