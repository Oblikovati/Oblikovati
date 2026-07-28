// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
)

// TestEveryShippedMeshIsWatertight is the corpus-wide RATCHET on the one defect class every other gate
// in this harness is structurally blind to: a body whose TESSELLATION leaks even though its TOPOLOGY is
// a perfect closed solid.
//
// WHY THE EXISTING GATES CANNOT SEE IT. isWatertightSolid (watertight.go) asks the B-rep — Valid &&
// Closed && Manifold && HolesContained && IsSolid — and every one of those held on complex/D8 while its
// welded mesh leaked 36 free edges at property quality and 8 at default (d8-multiface-report.md §3.3).
// TOPOLOGICAL watertightness is not MESH watertightness: two faces can share an edge in the B-rep and
// still tile that edge with different vertices, and only the welded triangle soup says so. D8 passed
// eleven slices of scrutiny in that state, and the sweep that opened this ratchet found it was never
// unique — TWELVE of the 124 measurable cases leaked, NINE of them scored fully green. Ten and seven
// today: the two largest (bfuseblend/A2, simple/J1) were closed by band_rim_stations.go.
//
// WHY THE MESH MATTERS AT ALL. The user only ever sees the mesh (CLAUDE.md's first priority). A leaking
// mesh renders with cracks, exports an unprintable STL, and hands the next boolean a non-closed operand
// — and every one of those failures is invisible to an area gate, because a torn rim removes no area.
//
// WHAT IT MEASURES. ops.FreeEdgeCount over ops.CalculateBodyFacets(body, q).Mesh — i.e. the SHIPPED
// tessellation of the SHIPPED body, welded at the MODEL's own resolution (ADR-0042), counting triangle
// edges not used by exactly two triangles. The weld ruler is deliberately NOT an absolute quantum: a
// fixed grid over-merges whenever the model's own feature separation drops below it and then reports the
// over-merge as a crack (FreeEdgeCount's own receipt on the #1818 near-pinch). Run at EVERY
// gateQualities() entry, because a leak can be quality-dependent in either direction: simple/B2 and
// simple/C2 are CLEAN at default and leak 3 at property (and the retired A2/J1 were clean at default and
// leaked 1536 at property — the whole reason a Default-only gate could not have found them), while
// simple/U6 leaks 13 at default and 12 at property.
//
// It also gates FOLD edges, the sibling half of the same invariant and the same measurement pass (a
// mesh must be free-edge-free to be a closed surface and fold-free to bound a well-defined volume —
// FreeEdgeCount's own docstring). Two cases carry folds, and only one of them also leaks.
//
// It is a RATCHET, not a tolerance: each offending case is listed with its MEASURED count per quality,
// so a listed case may improve freely while a new case — or a listed one growing — fails loud. The
// ceilings are the exact measured integers, not padded: a free-edge count is a COUNT, so there is no
// float noise for a margin to absorb, and any increase at all is a regression. Both tables must shrink,
// never widen.
func TestEveryShippedMeshIsWatertight(t *testing.T) {
	dir := CorpusFixtureDir()
	leaks, folds := meshDebtIndex(knownMeshLeaks()), meshDebtIndex(knownFoldedMeshes())
	for _, r := range Corpus() {
		body, ok := shippedCaseBody(r, dir)
		if !ok {
			continue // skipped / faulty: no single healthy body to measure
		}
		key := r.Grid + "/" + r.Case
		assertMeshIntegrityWithinDebt(t, r, body, leaks[key], folds[key])
	}
}

// assertMeshIntegrityWithinDebt fails when a case's welded mesh carries more free edges — or its faces
// more fold edges — at any gate quality than its recorded debt. A case with no entry may carry none at
// all, at any quality. Both metrics read the SAME BodyFacets, so the second costs no tessellation.
func assertMeshIntegrityWithinDebt(t *testing.T, r Record, body *topo.Body, leak, fold meshDebtEntry) {
	t.Helper()
	for _, gq := range gateQualities() {
		facets := ops.CalculateBodyFacets(body, gq.q)
		if free := ops.FreeEdgeCount(facets.Mesh); free > leak.ceilingAt(gq.name) {
			t.Errorf("%s/%s: welded body mesh leaks %d free edge(s) at %s quality, recorded debt %d — the "+
				"tessellation is not a closed surface even though the B-rep validates as a solid",
				r.Grid, r.Case, free, gq.name, leak.ceilingAt(gq.name))
		}
		if n := meshFoldEdges(facets); n > fold.ceilingAt(gq.name) {
			t.Errorf("%s/%s: body faces carry %d fold edge(s) at %s quality, recorded debt %d — a folded "+
				"face does not bound a well-defined volume", r.Grid, r.Case, n, gq.name, fold.ceilingAt(gq.name))
		}
	}
}

// shippedMeshFreeEdges is this harness's welded-mesh watertightness measurement for one body at one
// quality, welded on the model's own resolution.
//
// It reads BodyFacets.Mesh — the merged mesh CalculateBodyFacets already builds — rather than re-welding
// the per-face meshes behind a hand-picked quantum, so the ruler is the production one
// (ops.FreeEdgeCount) and adjacent faces' shared boundary vertices weld exactly as the renderer and
// exporter weld them.
//
// Example:
//
//	if free := shippedMeshFreeEdges(body, ops.PropertyQuality()); free != 0 { /* the mesh leaks */ }
func shippedMeshFreeEdges(body *topo.Body, q ops.Quality) int {
	return ops.FreeEdgeCount(ops.CalculateBodyFacets(body, q).Mesh)
}

// meshFoldEdges is the body's total fold-edge count over its per-face meshes — folds are a per-face
// property, so they are summed rather than read off the merged mesh.
func meshFoldEdges(facets *ops.BodyFacets) int {
	n := 0
	for _, m := range facets.FaceMeshes {
		n += ops.FoldEdgeCount(m)
	}
	return n
}

// meshDebtEntry is one case's measured mesh-integrity count, recorded per gate quality because these
// defects are quality-dependent and a single worst-case ceiling would let the other quality grow
// silently up to it.
type meshDebtEntry struct {
	name, grid string
	def, prop  int
}

// ceilingAt returns the recorded ceiling for one gate quality's name. An unrecognised name reads 0, so
// adding a quality to gateQualities() gates every case at ZERO there — failing loud rather than going
// quietly blind. TestMeshDebtTablesAreWellFormed pins the two names these tables record.
func (e meshDebtEntry) ceilingAt(quality string) int {
	switch quality {
	case "default":
		return e.def
	case "property":
		return e.prop
	}
	return 0
}

// meshDebtIndex keys a mesh-integrity debt table by "grid/case" for O(1) lookup.
func meshDebtIndex(table []meshDebtEntry) map[string]meshDebtEntry {
	out := make(map[string]meshDebtEntry, len(table))
	for _, d := range table {
		out[d.grid+"/"+d.name] = d
	}
	return out
}

// knownMeshLeaks is the FULL measured population of shipped bodies whose WELDED tessellation is not a
// closed surface: 10 cases of the 124 the corpus can measure, derived by an instrumented corpus-wide
// sweep at both gateQualities() entries, not from any report. The sweep reproduces byte-for-byte across
// runs, so these ceilings are exact rather than sampled.
//
// ★ EVERY ENTRY HERE IS DETECTED, NOT CAUSED. The slice that landed this ratchet added no geometry and
// touched no production file: the scoreboard is byte-identical (112 simple / 117 all-grid) and the other
// four ratchets are untouched. Every count below is what base 2f7115f9 already shipped — this table is
// simply the first instrument able to read it.
//
// ★ TWO ENTRIES HAVE BEEN RETIRED, not re-ceilinged: bfuseblend/A2 and simple/J1 (1536 free edges each at
// property quality, 3072 of the corpus's 4146 — 74 % of all its leakage) now measure ZERO at BOTH
// qualities. Their root was the seam-bridged band grid imposing one station count on rims that discretize
// into two (band_rim_stations.go); they are gated at zero here by this table's silence.
//
// complex/D8 is deliberately ABSENT: its leak (8 default / 36 property) was closed by the far-end split
// and is gated at ZERO both by this table's silence and by
// TestD8FarEndSplitIsAtomicAndHitsItsClosedForms. Recording it here would be recording a defect that no
// longer exists — and disabling that split makes this very test report D8's 8 and 36 again, which is the
// ratchet's own falsification.
//
// ★ SEVEN OF THE TEN ARE SCORED GREEN (PASS): simple/B2 C2 N6 Q5 U4 Y1 Z1. That is the D8 situation
// exactly, seven times over — an area gate cannot see a torn rim, because tearing a rim removes no area.
// Two more are FAIL(area) (U6, W2) and one FAIL(faulty) (T9).
//
// The roots, largest first — each is a SEPARATE follow-up slice, none is fixed here:
//
//   - FAR-END TRIM RUNNING OFF ITS STOP FACE (simple/Q5, 937 property / 160 default). Q5 is one of the
//     two cases still in knownSelfCrossingLoops for exactly this root (84912.4 pinched off its r=2500
//     band's host wall), and the multi-face split that closed complex/D8 DECLINES it because Q5 carries
//     two fillets on one host (d8-multiface-report.md §8.1). The leaking faces are those same faces.
//   - CANAL PATCH RAIL vs HOST (simple/U4, 44 at BOTH qualities; simple/C2, 3 at property). U4's leak is
//     21 edges on each of two geom.BSplineSurface corner patches plus 2 on the geom.EllipticalCylinder
//     they meet, all in the x = 10 plane, with the two sides' chords differing in the 5th digit
//     (0.0220470 vs 0.0220538) — a fitted patch rail and its host tiling the same seam from different
//     curves. Both cases already carry a knownOffSurfaceDebt entry (U4 0.000246, C2 0.0192); this is that
//     same debt, seen downstream in the mesh.
//   - RETRACING FACE LOOP (simple/N6 6/6, simple/Y1 3/3, simple/W2 3/3, simple/B2 3 at property). All
//     four are the ENTIRE membership of knownRetracingLoops, and the correspondence is exact. A loop that
//     runs out along a spike and straight back is not a simple polygon, so that face's triangulation does
//     not tile the boundary its neighbours tile: the free edges are whole boundary segments (N6's are 30,
//     25, 5, 5, 75 and 70 long), not slivers. Closing the retracing root should retire these four.
//   - simple/Z1 (3/3) is the only leaking case in NO other ratchet: a geom.Cylinder / geom.Plane rim at
//     z = 20 where three edges around one 0.245431 arc step fail to pair. Unattributed, and smallest.
//   - simple/T9 (60 / 10) and simple/U6 (12 / 13) are not green and are not tracked anywhere else. U6 is
//     the only case whose leak SHRINKS as the sampling refines, which rules out a fixed station count.
func knownMeshLeaks() []meshDebtEntry {
	return []meshDebtEntry{
		{"Q5", "simple", 160, 937}, // PASS — the far-end-trim root the D8 split declines to take
		{"T9", "simple", 10, 60},   // FAIL(faulty) — one BSpline face; also the corpus's worst folds
		{"U4", "simple", 44, 44},   // PASS — canal patch rails vs their elliptical-cylinder host
		{"U6", "simple", 13, 12},   // FAIL(area) — cylinder/torus rim; note it SHRINKS with quality
		{"N6", "simple", 6, 6},     // PASS — knownRetracingLoops, 3 loops
		{"B2", "simple", 0, 3},     // PASS — knownRetracingLoops, 1 loop
		{"C2", "simple", 0, 3},     // PASS — knownOffSurfaceDebt 0.0192, BSpline/plane seam
		{"W2", "simple", 3, 3},     // FAIL(area) — knownRetracingLoops, 2 loops
		{"Y1", "simple", 3, 3},     // PASS — knownRetracingLoops, 1 loop
		{"Z1", "simple", 3, 3},     // PASS — unattributed, in no other ratchet
	}
}

// knownFoldedMeshes is the FULL measured population of shipped bodies with a FOLD edge — a triangle pair
// whose normals oppose across a shared edge, so the surface doubles back on itself. Two cases of 124,
// from the same sweep, and the pairing is instructive: the two populations are almost disjoint.
// complex/F2 folds without leaking a single free edge, so a free-edge gate alone would have missed it
// entirely; simple/T9 does both. Neither is green (both are already FAIL), which is why this is the
// smaller half of the gap — but a fold on a green case would be just as invisible, and nothing was
// watching for one corpus-wide.
//
// As with the leaks, these counts are DETECTED, not caused: they are what base 2f7115f9 ships.
func knownFoldedMeshes() []meshDebtEntry {
	return []meshDebtEntry{
		{"T9", "simple", 6, 15}, // FAIL(faulty) — all on one geom.BSplineSurface
		{"F2", "complex", 1, 2}, // FAIL(area) — folds but does NOT leak
	}
}

// TestMeshDebtTablesAreWellFormed keeps both tables honest: no duplicate key, no empty entry (one
// claiming zero at every quality would be dead weight hiding nothing), and — the load-bearing one — that
// ceilingAt still recognises every quality gateQualities() actually runs. If a quality is added and this
// is not updated, every ceiling there reads 0 and the ratchet gets STRICTER, which is the safe
// direction; this test makes that deliberate rather than accidental.
func TestMeshDebtTablesAreWellFormed(t *testing.T) {
	for _, table := range [][]meshDebtEntry{knownMeshLeaks(), knownFoldedMeshes()} {
		seen := map[string]bool{}
		for _, d := range table {
			key := d.grid + "/" + d.name
			if seen[key] {
				t.Errorf("duplicate mesh debt entry %s", key)
			}
			seen[key] = true
			if d.def < 0 || d.prop < 0 || (d.def == 0 && d.prop == 0) {
				t.Errorf("%s debt entry is empty (default %d, property %d) — delete it instead", key, d.def, d.prop)
			}
		}
	}
	probe := meshDebtEntry{def: 1, prop: 2}
	for _, gq := range gateQualities() {
		if probe.ceilingAt(gq.name) == 0 {
			t.Errorf("gate quality %q has no ceiling column in meshDebtEntry — every case is gated at zero "+
				"there; add the column or confirm zero is intended", gq.name)
		}
	}
}
