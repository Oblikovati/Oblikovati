// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	opstol "oblikovati.org/kernel/ops/internal/tol"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/ops/validate"
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
	return setbackAssembled(t, ef, body)
}

// setbackAssembled drives one runout edge through the full intact-boss path (detectSetbackBands →
// extractSetbackPatches → buildSetbackFaces) and assembles the result: every original body face
// transformed as usual (each crossing boss wall left INTACT — footprint SUBDIVIDED by buildSetbackFaces'
// inserts but never split, never emitted; the two host planes RE-CLIPPED via set.replace) + the two
// plain-R wings + the resolved setback patches. It mirrors filletResultFaces' composition
// (transformedBodyFaces + extra) but drives buildSetbackFaces directly — the only way to validate
// closure, since a shell can only be judged by ASSEMBLING it (assembleBody's weld invariant). Shared by
// the S1 (two cylinders) and S4 (cone + cylinder) closure gates so both exercise the identical path.
func setbackAssembled(t *testing.T, ef edgeFillet, body *topo.Body) *topo.Body {
	t.Helper()
	maps := s1RebuildMaps(body, ef)
	res := opstol.ForBody(body)
	b, ok := detectSetbackBands(ef, res)
	if !ok {
		t.Fatal("setbackAssembled: detectSetbackBands ok=false")
	}
	loops, ok := extractSetbackPatches(b, ef, res)
	if !ok {
		t.Fatal("setbackAssembled: extractSetbackPatches ok=false (an Adjacent boss wall did not resolve)")
	}
	set := runoutSet{replace: map[uint64]filletFace{}}
	if !buildSetbackFaces(&set, ef, b, loops, res, maps) {
		t.Fatal("setbackAssembled: buildSetbackFaces ok=false")
	}
	faces := append(transformedBodyFaces(body, maps, set.replace), set.extra...)
	return assembleBody(faces)
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
	return filletRebuildMaps{abSubst: abSubst, endCorner: endCorner, edgeInserts: edgeInserts,
		insertCurves: map[*topo.Face]map[uint64][]geom.Curve3{}, spreads: spreads}
}

// TestBuildSetbackFaces_S1Watertight is the closure gate: the assembled intact-boss S1 body must be a
// watertight, hole-contained solid whose tessellated area is within OCCT's 1% (ref 3662.79, forensics
// §8.2). A partial fill (an open shell) or a surviving hole would fail here.
func TestBuildSetbackFaces_S1Watertight(t *testing.T) {
	t.Parallel()
	body := s1SetbackAssembled(t)
	if !body.IsSolid() {
		t.Fatalf("S1 intact-boss shell is not a solid: %d open edges", len(openEdges(body)))
	}
	r := validate.Validate(body)
	if !r.Valid || !r.HolesContained {
		t.Fatalf("S1 intact-boss shell invalid: Valid=%v HolesContained=%v", r.Valid, r.HolesContained)
	}
	area := tessellate.MeshGeometryProperties(mustTessellate(t, body)).Area
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	return countSurfaceFacesNear[geom.Cylinder](body, want, tol)
}

// countConeFacesNear is countCylFacesNear for a cone wall (S4's r13→r5 boss) — the surface-type-agnostic
// intact-wall proof the forensics §8.3/§8.4 calls for (a cone is kept whole exactly like a cylinder).
func countConeFacesNear(body *topo.Body, want, tol float64) int {
	return countSurfaceFacesNear[geom.Cone](body, want, tol)
}

// countSurfaceFacesNear counts faces of concrete surface type S whose tessellated area is within tol of
// want. Shared by the cylinder/cone intact-wall gates so a new boss surface type needs no new counter.
func countSurfaceFacesNear[S geom.Surface](body *topo.Body, want, tol float64) int {
	n := 0
	for _, f := range body.Faces() {
		if _, ok := f.Geometry().(S); !ok {
			continue
		}
		if a := tessArea(tessellate.TessellateFace(f, PropertyQuality())); a >= want-tol && a <= want+tol {
			n++
		}
	}
	return n
}

// s4SetbackAssembled is s1SetbackAssembled for S4 (a CONE boss r13→r5 + an r10 cylinder boss): it builds
// the FULL intact-boss result through the new path and assembles it, the only way to validate closure of
// a cone-Adjacent runout (forensics §8.3). S4 is the case that proves the intact-boss path is surface-
// type-agnostic — the cone wall is kept whole and G1-faired exactly like a cylinder.
func s4SetbackAssembled(t *testing.T) *topo.Body {
	t.Helper()
	ef, body := s4SetbackEdge(t)
	return setbackAssembled(t, ef, body)
}

// s4SetbackEdge resolves S4's front-top edge fillet (R=8, its corpus radius, edge mid (0,-15,0)) — the
// same pick the corpus gate uses, paired with its body so the closure test can transform the body faces.
func s4SetbackEdge(t *testing.T) (edgeFillet, *topo.Body) {
	t.Helper()
	body := importCorpusSolid(t, "simple/S4")
	e := edgeAtMidpoint(body, math.P3(0, -15, 0))
	if e == nil {
		t.Fatal("s4SetbackEdge: front-top edge (0,-15,0) not found")
	}
	fil, err := computeEdgeFillet(body, filletPick{edge: e, r0: 8, r1: 8},
		map[uint64]*cornerBlend{}, map[uint64]*cornerMiter{}, FillConcaveOutward)
	if err != nil {
		t.Fatalf("s4SetbackEdge: computeEdgeFillet: %v", err)
	}
	return fil, body
}

// TestBuildSetbackFaces_S4Watertight is the S4 closure gate: the assembled intact-boss (cone+cyl) body
// must be a watertight, hole-contained solid whose tessellated area is within OCCT's 1% (ref 7004.23,
// forensics §8.3, window [6934.2, 7074.3]). The cone Adjacent must resolve for extractSetbackPatches.
func TestBuildSetbackFaces_S4Watertight(t *testing.T) {
	t.Parallel()
	body := s4SetbackAssembled(t)
	if !body.IsSolid() {
		t.Fatalf("S4 intact-boss shell is not a solid: %d open edges", len(openEdges(body)))
	}
	r := validate.Validate(body)
	if !r.Valid || !r.HolesContained {
		t.Fatalf("S4 intact-boss shell invalid: Valid=%v HolesContained=%v", r.Valid, r.HolesContained)
	}
	area := tessellate.MeshGeometryProperties(mustTessellate(t, body)).Area
	if area < 6934.2 || area > 7074.3 {
		t.Fatalf("S4 area %.4f outside OCCT 1%% gate [6934.2, 7074.3] (ref 7004.23)", area)
	}
}

// TestBuildSetbackFaces_S4BossWallsIntact pins the faithfulness invariant: S4's cone boss survives as ONE
// cone face near its analytic 1218.1 and the r10 cylinder as ONE face near 942.478 (forensics §8.3) —
// never split into sub-faces. This is what the old split path violated (area-coincidental green).
func TestBuildSetbackFaces_S4BossWallsIntact(t *testing.T) {
	t.Parallel()
	body := s4SetbackAssembled(t)
	// The tessellated area is a straight-chord polygon of the footprint rim, so it sits a sub-2% band
	// UNDER the analytic value (the intact-boss subdivision, not a lost region): cone 1203.45 vs 1218.1,
	// r10 cyl 937.42 vs 942.478. The tolerances bracket that undershoot yet exclude a SPLIT wall (a
	// half-cone ≈609, a half-cylinder ≈471 — nowhere near these bands), so exactly ONE proves un-split.
	if got := countConeFacesNear(body, 1218.1, 18); got != 1 {
		t.Fatalf("want exactly ONE intact cone boss wall near 1218.1 (un-split), got %d", got)
	}
	if got := countCylFacesNear(body, 942.478, 8); got != 1 {
		t.Fatalf("want exactly ONE intact r10 boss wall near 942.478 (un-split), got %d", got)
	}
}

// TestBuildSetbackFaces_S4HostsSingleLoop pins forensics §8.3: every S4 result face is WIRE:1 — both boss
// footprint holes (cone base + r10) are opened into the fillet cut, never preserved as inner loops.
func TestBuildSetbackFaces_S4HostsSingleLoop(t *testing.T) {
	t.Parallel()
	body := s4SetbackAssembled(t)
	for _, f := range body.Faces() {
		if n := len(f.Loops()); n != 1 {
			t.Fatalf("face %T has %d loops, want WIRE:1 (footprint opened into the cut)", f.Geometry(), n)
		}
	}
}

// TestFilletEdges_S1WallsIntact is the WIRED-path faithfulness proof for S1: the FULL fillet op
// (FilletEdges → collectRunouts → runoutFacesFor → the intact-boss path) must leave both cylinder
// bosses INTACT — r6 565.487, r8 753.982 (forensics §8.2) — not split. Total area alone (the corpus
// gate) can be right by coincidence; one intact face per boss is the topology-faithful proof.
func TestFilletEdges_S1WallsIntact(t *testing.T) {
	t.Parallel()
	body := filletedCorpusEdge(t, "simple/S1", math.P3(0, -10, 10), 6)
	if got := countCylFacesNear(body, 753.982, 8); got != 1 {
		t.Fatalf("wired S1: want ONE intact r8 boss wall near 753.982 (un-split), got %d", got)
	}
	if got := countCylFacesNear(body, 565.487, 6); got != 1 {
		t.Fatalf("wired S1: want ONE intact r6 boss wall near 565.487 (un-split), got %d", got)
	}
}

// TestFilletEdges_S4WallsIntact is TestFilletEdges_S1WallsIntact for S4: the wired op keeps the cone
// boss (≈1218.1) and the r10 cylinder (≈942.478) each as ONE intact face (forensics §8.3) — the proof
// that the cone routes through the new path faithfully, not the old area-coincidental split.
func TestFilletEdges_S4WallsIntact(t *testing.T) {
	t.Parallel()
	body := filletedCorpusEdge(t, "simple/S4", math.P3(0, -15, 0), 8)
	if got := countConeFacesNear(body, 1218.1, 18); got != 1 {
		t.Fatalf("wired S4: want ONE intact cone boss wall near 1218.1 (un-split), got %d", got)
	}
	if got := countCylFacesNear(body, 942.478, 8); got != 1 {
		t.Fatalf("wired S4: want ONE intact r10 boss wall near 942.478 (un-split), got %d", got)
	}
}

// countEllipCylFacesNear counts elliptical-cylinder faces whose tessellated area is within tol of want —
// the intact-wall proof for the oblique elliptical-cylinder boss of T7 (its SurfaceOfLinearExtrusion of an
// ellipse elementarises to geom.EllipticalCylinder on import), surface-type-agnostic like the cyl/cone gates.
func countEllipCylFacesNear(body *topo.Body, want, tol float64) int {
	return countSurfaceFacesNear[geom.EllipticalCylinder](body, want, tol)
}

// TestFilletEdges_T7WallsIntact is TestFilletEdges_S4WallsIntact for T7 (an OBLIQUE ELLIPTICAL-CYLINDER
// boss + an r8 cylinder): the wired op (FilletEdges → collectRunouts → runoutFacesFor → the intact-boss
// path) must keep the elliptical-cylinder wall (analytic 2381.68, forensics §2) and the r8 cylinder
// (603.186, §1) each as ONE intact face, not split. The elliptical wall's tessellated area sits a sub-1%
// band under its analytic value (the straight-chord subdivision of its footprint rim), so ±24 brackets that
// undershoot yet excludes a split wall (a half-wall ≈1191, nowhere near). This is the topology-faithful
// proof that the ellipse footprint (geom.EllipseFull) rails to the intact wall — total area alone (the
// corpus gate) can be right by coincidence.
func TestFilletEdges_T7WallsIntact(t *testing.T) {
	t.Parallel()
	body := filletedCorpusEdge(t, "simple/T7", math.P3(0, -13, 0), 6)
	if got := countEllipCylFacesNear(body, 2381.68, 24); got != 1 {
		t.Fatalf("wired T7: want ONE intact oblique-ellipse boss wall near 2381.68 (un-split), got %d", got)
	}
	if got := countCylFacesNear(body, 603.186, 6); got != 1 {
		t.Fatalf("wired T7: want ONE intact r8 boss wall near 603.186 (un-split), got %d", got)
	}
}

// filletedCorpusEdge runs the real FilletEdges op on the corpus fixture's edge at mid, radius r, and
// returns the result body — the end-to-end path the corpus area gate drives, so a wall-intact assertion
// on its output proves the WIRED runout path (not the test-only setbackAssembled shortcut) is faithful.
func filletedCorpusEdge(t *testing.T, rel string, mid math.Point3, r float64) *topo.Body {
	t.Helper()
	b := importCorpusSolid(t, rel)
	e := edgeAtMidpoint(b, mid)
	if e == nil {
		t.Fatalf("filletedCorpusEdge: edge %v not found on %s", mid, rel)
	}
	res, err := FilletEdges(b, [][]byte{e.ReferenceKey()}, r)
	if err != nil {
		t.Fatalf("filletedCorpusEdge: FilletEdges(%s, r=%v): %v", rel, r, err)
	}
	return res
}

// TestFilletEdges_T1TorusIntact is TestFilletEdges_S4WallsIntact for T1 (a crossing TORUS boss + an r6
// cylinder): the wired op (FilletEdges → collectRunouts → runoutFacesFor → the intact-boss path) must keep
// the torus wall as ONE intact face near its band area 1144.04 (m4-spike.md §(d): pre-fillet 1143.986,
// OCCT parity) — NOT the 3947.68 full donut the old boss-splitting path produced — and the r6 cylinder
// (565.487) as ONE intact face. The torus tessellates as a chorded band (band_ring_chain.go), not the full
// parametric donut; a single face near 1144 proves both the intact-boss topology AND the band mesher.
func TestFilletEdges_T1TorusIntact(t *testing.T) {
	t.Parallel()
	body := filletedCorpusEdge(t, "simple/T1", math.P3(0, -30, 0), 8)
	if got := countSurfaceFacesNear[geom.Torus](body, 1144.04, 11); got != 1 {
		t.Fatalf("wired T1: want ONE intact torus boss wall near 1144.04 (NOT the 3947.68 donut), got %d", got)
	}
	if got := countCylFacesNear(body, 565.487, 6); got != 1 {
		t.Fatalf("wired T1: want ONE intact r6 boss wall near 565.487 (un-split), got %d", got)
	}
}

// TestFilletEdges_T4TorusIntact is TestFilletEdges_T1TorusIntact for T4 (a larger crossing TORUS boss
// R35/r10 + an r10 cylinder): the wired intact-boss op must keep the torus as ONE face near its band area
// 2826.04 (m4-spike.md §(d): pre-fillet 2825.957) — NOT the 13816.88 full donut — and the r10 cylinder
// (628.30, r10×h10) intact. T4 was baseline-green before; this proves it now routes the torus through the
// intact setback path FAITHFULLY, not by the do-no-harm area coincidence. (The task brief's "942.478" is an
// S4 copy-paste; T4's cylinder is r10×h10 = 628.30, measured pre-fillet and confirmed intact post-fillet.)
func TestFilletEdges_T4TorusIntact(t *testing.T) {
	t.Parallel()
	body := filletedCorpusEdge(t, "simple/T4", math.P3(0, -30, 0), 8)
	if got := countSurfaceFacesNear[geom.Torus](body, 2826.04, 28); got != 1 {
		t.Fatalf("wired T4: want ONE intact torus boss wall near 2826.04 (NOT the 13816.88 donut), got %d", got)
	}
	if got := countCylFacesNear(body, 628.30, 6); got != 1 {
		t.Fatalf("wired T4: want ONE intact r10 boss wall near 628.30 (un-split), got %d", got)
	}
}

// TestFilletEdges_T1SetbackSeamWatertight crosses the setback↔intact-torus seam no other test drives: the
// full FilletEdges op on T1 (box + crossing torus boss) through the intact setback path must weld to a
// watertight, hole-contained SOLID whose torus wall meshes to its band area ≈1144 (band_ring_chain.go),
// NOT the 3947 full donut. The old split path left this seam a full-domain-grid donut and could not weld;
// the intact path keeps the torus whole and welds the setback patches to its subdivided footprint rim.
func TestFilletEdges_T1SetbackSeamWatertight(t *testing.T) {
	t.Parallel()
	body := filletedCorpusEdge(t, "simple/T1", math.P3(0, -30, 0), 8)
	if !body.IsSolid() {
		t.Fatalf("T1 intact-torus setback shell is not a solid: %d open edges", len(openEdges(body)))
	}
	if r := validate.Validate(body); !r.Valid || !r.HolesContained {
		t.Fatalf("T1 intact-torus setback shell invalid: Valid=%v HolesContained=%v", r.Valid, r.HolesContained)
	}
	if got := countSurfaceFacesNear[geom.Torus](body, 1144.04, 11); got != 1 {
		t.Fatalf("T1 torus wall must mesh to its band area ≈1144 (NOT 3947 donut), got %d faces in band", got)
	}
}

// TestFilletEdges_S7SphereIntact is the per-type closure proof for the SPHERE boss (S7), admitted to the
// intact-boss setback path once M4's σ-partition rim (fillet_setback_partition.go) + chorded-band mesher
// (band_ring_chain.go) landed. The wired op (FilletEdges → collectRunouts → runoutFacesFor → intact-boss
// path) must keep the R=13 hemisphere cap as ONE intact geom.Sphere face near its area 2πR²=1061.86 (NOT
// the ~0 pole-collapse the pre-M4 path produced, NOT a 2123.7 full-sphere blow-up) and weld to a
// watertight, hole-contained SOLID. The pre-M4 baseline left both boss footprints PIERCING the host planes
// (HolesContained=false) — a topologically-broken solid that passed only by area tolerance; this proves the
// sphere now closes faithfully like S1/S4/T1. (m5-s7-spike.md: OCCT keeps the hemisphere intact 1061.86;
// our path meshes it 1069.75, whole body +0.732% in-gate, census matching OCCT.)
func TestFilletEdges_S7SphereIntact(t *testing.T) {
	t.Parallel()
	body := filletedCorpusEdge(t, "simple/S7", math.P3(0, -15, 0), 3)
	if !body.IsSolid() {
		t.Fatalf("S7 intact-sphere setback shell is not a solid: %d open edges", len(openEdges(body)))
	}
	if r := validate.Validate(body); !r.Valid || !r.HolesContained {
		t.Fatalf("S7 intact-sphere setback shell invalid: Valid=%v HolesContained=%v (baseline leaves boss footprints piercing host planes)", r.Valid, r.HolesContained)
	}
	if got := countSurfaceFacesNear[geom.Sphere](body, 1061.86, 12); got != 1 {
		t.Fatalf("wired S7: want ONE intact sphere cap near 1061.86 (NOT ~0 pole-collapse or 2123.7 full sphere), got %d", got)
	}
}

// assertSingleBossWatertight is the shared single-boss (#2007) closure gate: the wired FilletEdges result
// must be a watertight, hole-contained solid whose EVERY face is WIRE:1 (the crossing boss footprint is
// absorbed into the fillet cut, no surviving inner loop) — the malformed-B-rep poison pill the do-no-harm
// baseline left (HolesContained=false) is exactly what this proves is fixed.
func assertSingleBossWatertight(t *testing.T, body *topo.Body, name string) {
	t.Helper()
	if !body.IsSolid() {
		t.Fatalf("%s single-boss setback shell is not a solid: %d open edges", name, len(openEdges(body)))
	}
	if r := validate.Validate(body); !r.Valid || !r.HolesContained {
		t.Fatalf("%s single-boss setback shell invalid: Valid=%v HolesContained=%v (baseline leaves the boss "+
			"footprint piercing the host plane)", name, r.Valid, r.HolesContained)
	}
	for _, f := range body.Faces() {
		if n := len(f.Loops()); n != 1 {
			t.Fatalf("%s face %T has %d loops, want WIRE:1 (footprint opened into the cut)", name, f.Geometry(), n)
		}
	}
}

// TestFilletEdges_S6SphereSingleBoss is the SINGLE-BOSS setback closure proof for a SPHERE boss (S6, #2007):
// a box + ONE crossing sphere boss (r13) whose footprint used to protrude past the shrunken host outer loop
// (HolesContained=false). The single-boss tiling (2 plain wings + one central run-out patch that absorbs the
// footprint, both edge faces re-clipped single-loop) must weld to a watertight, hole-contained SOLID whose
// R=13 hemisphere stays ONE intact geom.Sphere face near 2πR²=1061.86. The sphere host footprint arc is
// densified (densifyHostArc, one-boss only) so the whole-footprint host notch matches the analytic disc.
func TestFilletEdges_S6SphereSingleBoss(t *testing.T) {
	t.Parallel()
	body := filletedCorpusEdge(t, "simple/S6", math.P3(0, -15, 0), 3)
	assertSingleBossWatertight(t, body, "S6")
	if got := countSurfaceFacesNear[geom.Sphere](body, 1061.86, 12); got != 1 {
		t.Fatalf("wired S6: want ONE intact sphere cap near 1061.86 (single-boss setback), got %d", got)
	}
}

// TestFilletEdges_S9TorusSingleBoss is TestFilletEdges_S6SphereSingleBoss for a TORUS boss (S9, #2007): a box
// + ONE crossing Torus(20,5) boss. The single-boss tiling must weld watertight and keep the torus wall as
// ONE intact chorded band near its area ≈1144 (NOT the full donut), footprint absorbed, every face WIRE:1.
func TestFilletEdges_S9TorusSingleBoss(t *testing.T) {
	t.Parallel()
	body := filletedCorpusEdge(t, "simple/S9", math.P3(0, -30, 0), 10)
	assertSingleBossWatertight(t, body, "S9")
	if got := countSurfaceFacesNear[geom.Torus](body, 1144.04, 12); got != 1 {
		t.Fatalf("wired S9: want ONE intact torus band near 1144.04 (single-boss setback), got %d", got)
	}
}

// TestFilletEdges_T3TorusObliqueSingleBoss is TestFilletEdges_S9TorusSingleBoss for an OBLIQUE torus boss
// (T3, #2007): a box + ONE crossing Torus(35,10) boss whose axis is NOT normal to the host plane (the
// "oblique" corpus family, occtparity corpus.json's T3 pick: r=8, midpoint (-8.056074683, -25.47882455,
// 13.63558433)). The single-boss tiling must weld watertight and keep the torus wall as ONE intact
// chorded band, footprint absorbed, every face WIRE:1 — same shape as S6/S9, proving the closure gate
// holds for a non-normal boss axis too. Torus band area (measured post-fillet via TessellateFace, the
// same probe S6/S9's 1061.86/1144.04 constants came from) is ~2827.23.
func TestFilletEdges_T3TorusObliqueSingleBoss(t *testing.T) {
	t.Parallel()
	body := filletedCorpusEdge(t, "simple/T3", math.P3(-8.056074683, -25.47882455, 13.63558433), 8)
	assertSingleBossWatertight(t, body, "T3")
	if got := countSurfaceFacesNear[geom.Torus](body, 2827.23, 28); got != 1 {
		t.Fatalf("wired T3: want ONE intact torus band near 2827.23 (single-boss setback), got %d", got)
	}
}

// TestResolveSetbackTiling_TwoBossUnchanged pins the do-no-harm gate for the single-boss addition: the
// 2-boss S1 shape still resolves to the S1 tiling (outer+inner bosses, both hosts distinct).
func TestResolveSetbackTiling_TwoBossUnchanged(t *testing.T) {
	t.Parallel()
	ef, res := runoutFixtureCrossingBoss(t)
	b, ok := detectSetbackBands(ef, res)
	if !ok || len(b.bosses) != 2 {
		t.Fatalf("S1 fixture: want 2-boss bands, got ok=%v bosses=%d", ok, len(b.bosses))
	}
	if _, ok := resolveSetbackTiling(b, ef, res); !ok {
		t.Fatalf("resolveSetbackTiling rejected the 2-boss S1 shape")
	}
}

// synthTorusSetbackBoss builds a crossingBoss mimicking T1's intact torus wall: the host-plane footprint
// is a circle of radius r_f=25 centered at the origin in the z=0 plane, with its seam vertex at world
// (25,0,0) (azimuth 0°). The fillet R=8 band runs along the box edge at y=-22 (contact line σ=0), so the
// fillet interference crosses the footprint circle at cross1 (−11.874,−22,0) [az −118.4°] and cross2
// (11.874,−22,0) [az −61.6°]. Only footEdge (the conic) and host (the plane) are read by bossRimSubArcs,
// plus cyl/seam/cross1/cross2, so a fully-fused body is not needed. Exact frame: m4-spike.md §(a).
func synthTorusSetbackBoss(t *testing.T) (crossingBoss, geom.Cylinder, math.Point3, math.Point3, math.Point3) {
	t.Helper()
	seam := math.P3(25, 0, 0)
	c1, c2 := math.P3(-11.874, -22, 0), math.P3(11.874, -22, 0)
	circle, err := geom.NewCircle(math.P3(0, 0, 0), math.V3(0, 0, 1), 25)
	if err != nil {
		t.Fatalf("synthTorusSetbackBoss: NewCircle: %v", err)
	}
	plane, err := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	if err != nil {
		t.Fatalf("synthTorusSetbackBoss: NewPlane: %v", err)
	}
	bld := topo.NewBuilder(true, topo.NewLineage(topo.Tok("synthTorus", "body", 0)))
	seamV := bld.AddVertex(seam, topo.NewLineage(topo.Tok("synthTorus", "v", 0)))
	footEdge := bld.AddEdge(circle, seamV, seamV, topo.NewLineage(topo.Tok("synthTorus", "e", 0)))
	host := bld.AddFace(plane, topo.NewLineage(topo.Tok("synthTorus", "face", 0)))
	// Fillet R=8 cylinder tangent to z=0 along the line y=-22: axis at height R above that line, running
	// along +x (the box edge). Its contact foot at the footprint centre station is (0,-22,0).
	cyl, err := geom.NewCylinder(math.P3(0, -22, 8), math.V3(1, 0, 0), 8)
	if err != nil {
		t.Fatalf("synthTorusSetbackBoss: NewCylinder: %v", err)
	}
	return crossingBoss{footEdge: footEdge, host: host}, cyl, seam, c1, c2
}

// TestBossRimSubArcs_TorusFootprintSpansFullCircle pins the scale-invariant σ-partition (M4 Task 1): an
// intact torus wall's footprint rim must be rebuilt as the FULL 360° conic (host major 241.6° + band
// notch 57° + host 61.6°), not the 118° out-and-back slit the old local minor/major midpoint test emitted
// on a large footprint (m4-spike.md §CRITICAL — 242° dropped).
func TestBossRimSubArcs_TorusFootprintSpansFullCircle(t *testing.T) {
	t.Parallel()
	boss, fillet, seam, c1, c2 := synthTorusSetbackBoss(t)
	subs, ok := bossRimSubArcs(boss, fillet, seam, c1, c2, nil)
	if !ok {
		t.Fatalf("bossRimSubArcs rejected a valid torus footprint boss")
	}
	span := totalDirectedSpan(subs)
	if stdmath.Abs(span-2*stdmath.Pi) > 1e-3 {
		t.Fatalf("footprint rim spans %.1f°, want 360° — a partial span means the minor arc was chosen (the 118° slit bug); subs=%d",
			span*180/stdmath.Pi, len(subs))
	}
	if a := directedSpan(subs[0]); a < stdmath.Pi {
		t.Fatalf("hostA span %.1f° < 180° — chose the minor arc for the large torus footprint", a*180/stdmath.Pi)
	}
}

// directedSpan is the swept native-parameter angle (magnitude) of one emitted footprint sub-arc — its
// own SweepAngle, which for a circle/ellipse arc is the exact angular extent it covers. It is the ruler
// the closure invariant Δ_hostA+Δ_band+Δ_hostB=2π is checked with (m4-rim-partition-derivation.md §D3).
func directedSpan(c geom.Curve3) float64 {
	switch a := c.(type) {
	case geom.Arc3d:
		return stdmath.Abs(a.SweepAngle)
	case geom.EllipticalArc:
		return stdmath.Abs(a.SweepAngle)
	default:
		return 0
	}
}

// totalDirectedSpan sums the directed native spans of every emitted sub-arc — 2π exactly for a valid
// full-conic partition (a partial sum betrays a dropped span, the minor-arc slit bug).
func totalDirectedSpan(subs []geom.Curve3) float64 {
	total := 0.0
	for _, c := range subs {
		total += directedSpan(c)
	}
	return total
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
