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
// unique — TWELVE of the 124 measurable cases leaked, NINE of them scored fully green. FIVE and THREE
// today: the two largest (bfuseblend/A2, simple/J1) were closed by band_rim_stations.go, two more
// (simple/B2, simple/N6) by fillet_arc_endcap.go, the largest that was left — simple/Q5's 937 — by
// admitting a ONE-ENDED far-end split (fillet_farend_chain.go's splitEndCount), and simple/U4's 44 by
// routing saddleBandLoftMesh's rim read through discretizeEdge (saddle_band_loft.go).
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
// gateQualities() entry, because a leak can be quality-dependent in either direction: simple/C2 is CLEAN
// at default and leaks 3 at property (and the retired A2/J1 were clean at default and leaked 1536 at
// property — the whole reason a Default-only gate could not have found them), while simple/U6 leaks 13 at
// default and 12 at property.
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
	t.Parallel()
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
// closed surface: 5 cases of the 125 the corpus can measure (it opened at 12), derived by an
// instrumented corpus-wide sweep at both gateQualities() entries, not from any report. The sweep reproduces byte-for-byte across
// runs, so these ceilings are exact rather than sampled.
//
// ★ EVERY ENTRY HERE IS DETECTED, NOT CAUSED. The slice that landed this ratchet added no geometry and
// touched no production file: the scoreboard is byte-identical (112 simple / 117 all-grid) and the other
// four ratchets are untouched. Every count below is what base 2f7115f9 already shipped — this table is
// simply the first instrument able to read it. Every subsequent change to it has been a RETIREMENT
// caused by a production fix, never a re-ceilinging: no entry has ever grown.
//
// Remaining corpus leakage, both qualities: **29** free edges at default and **81** at property, over
// these FIVE cases. It opened at 251 / 4146 over twelve.
//
// ★ FOUR ENTRIES HAVE BEEN RETIRED, not re-ceilinged. bfuseblend/A2 and simple/J1 (1536 free edges each
// at property quality, 3072 of the corpus's 4146 — 74 % of all its leakage) now measure ZERO at BOTH
// qualities; their root was the seam-bridged band grid imposing one station count on rims that discretize
// into two (band_rim_stations.go). simple/Q5 (937 property / 160 default — 88 % of what was left) now
// measures ZERO at both too: the far-end multi-face split, which used to require BOTH of a fillet's
// terminal sections to split and so declined Q5's one-ended one, now routes it
// (kernel/ops/fillet_farend_chain.go's splitEndCount). All three are gated at zero by this table's
// silence, and Q5 additionally by TestQ5FarEndSplitIsAtomicAndHitsItsClosedForms.
//
// complex/D8 is deliberately ABSENT: its leak (8 default / 36 property) was closed by the far-end split
// and is gated at ZERO both by this table's silence and by
// TestD8FarEndSplitIsAtomicAndHitsItsClosedForms. Recording it here would be recording a defect that no
// longer exists — and disabling that split makes this very test report D8's 8 and 36 again, which is the
// ratchet's own falsification.
//
// ★ THREE OF THE FIVE ARE SCORED GREEN (PASS): simple/C2 Y1 Z1. That is the D8 situation exactly, four
// times over — an area gate cannot see a torn rim, because tearing a rim removes no area. One more is
// FAIL(area) (U6) and one FAIL(faulty) (T9).
//
// The roots, largest first — each is a SEPARATE follow-up slice, none is fixed here:
//
//   - ★ simple/C2 (0 default / 3 property) — STATION-COUNT MISMATCH ON A SHARED STRAIGHT BOUNDARY, the
//     SAME class as U4 below. The inherited attribution ("canal patch rail vs host", i.e. two curves, and
//     possibly its knownOffSurfaceDebt 0.0192 seen downstream) is FALSIFIED by provenance, measured this
//     way: weld the shipped property-quality body mesh at the model's own grid (1.929301e-07), collect
//     every edge whose incidence != 2, then search each FaceMesh for that edge. All three are degree 1 and
//     all three lie between the SAME two points, (27.709873121, 0, 150) and (29.202254427, 0, 145.776225):
//     the geom.Plane face (id 76) tiles that segment with TWO chords (2.236965899 + 2.242708693 =
//     4.479674592) through the interior station (28.456063774, 0, 147.891157677), while the
//     geom.BSplineSurface face (id 52) tiles it with ONE chord of 4.479674132 between the same endpoints.
//     The two sides are the same straight line to 4.6e-07 — 2.4x the weld grid, which is exactly why the
//     interior station does not weld away. So this is one extra STATION on one side, not two different
//     curves, and no change to the patch rail's SURFACE can close it. Still a separate follow-up slice:
//     which of the two sides is wrong (the plane's discretizeEdge densification, or the B-spline face's
//     own boundary read skipping it) is not settled here.
//     ★ simple/U4's 44-at-BOTH-qualities entry IS RETIRED, and it is the receipt for how wrong this row's
//     reasoning was. The entry blamed "a fitted patch rail and its host tiling the same seam from
//     different curves", evidenced by chords differing in the 5th digit (0.0220470 vs 0.0220538). Swept by
//     provenance, those are not two sides of one seam at all — they are the two MIRROR sliver panels' own
//     chords, at opposite ends of the body. The real pairing was one geom.LineSegment edge (edge 440,
//     shared, same two vertices, exactly collinear polyline) tiled with 21 chords by the high-aspect
//     B-spline sliver and with ONE by the geom.EllipticalCylinder boss wall: a station COUNT mismatch, not
//     a curve mismatch, so no change of the panel's fill surface could ever have closed it. Root: the boss
//     wall is a notched two-rim band, and saddleBandLoftMesh's bandWrapRings read its rims through the raw
//     TessellateEdge sampler instead of discretizeEdge — silently opting out of the #2009 densification's
//     caller-INDEPENDENCE, the one property that makes it safe. Reading the ring through discretizeEdge
//     takes U4 to 0 / 0 with every per-face area unchanged to 2e-6. Gated at zero by this table's silence
//     and by TestU4DualHostMeshIsClosedAtEveryQuality.
//   - RETRACING FACE LOOP (simple/Y1 3/3, simple/W2 3/3). Both are in knownRetracingLoops, and the
//     correspondence is exact. A loop that runs out along a spike and straight back is not a simple
//     polygon, so that face's triangulation does not tile the boundary its neighbours tile: the free
//     edges are whole boundary segments, not slivers.
//     ★ RETIRED, and it is the receipt for that attribution: simple/B2 (0/3) and simple/N6 (6/6) both
//     went to 0/0 — fully watertight at BOTH gate qualities — when the arc fillet stopped emitting a
//     setback end-cap coplanar with its own side face (kernel/ops/fillet_arc_endcap.go). Nothing in that
//     change touches the mesher or the welder; the leaks were the retrace, exactly as predicted here.
//     N6 still carries one retracing loop (its non-radial end) and leaks nothing, which bounds the
//     claim honestly: a retrace is sufficient to leak, not necessary.
//     ★ simple/W2's 3/3 IS ALSO RETIRED — and the route there is the receipt for the correction this
//     table used to carry. Putting W2's band on the material side ALONE (the rolling-ball seat, i.e. the
//     Reversed cap normal plus the cylR+r groove tier) takes its retracing loops 2 → 0 but does NOT take
//     these three with them: they are replaced by 8 default / 29 property free edges of a DIFFERENT
//     origin, all on the plane y = 0 along a cap-tangent arc that spills 0.2 = r below the bottom plane.
//     Only the RUN-OUT termination (fillet_arc_runout.go — the band terminated on the side plane's own
//     spiric section, OCCT's own construction) closes both at once: W2 now measures 0 default / 0
//     property, 0 retraces, 0 self-crossings, and 11.766423 against DRAWEXE's 11.76665. So the retrace
//     and the leak were the SAME defect here after all, but neither half of the fix reaches it alone.
//   - simple/Z1 (3/3) is the only leaking case in NO other ratchet: a geom.Cylinder / geom.Plane rim at
//     z = 20 where three edges around one 0.245431 arc step fail to pair. Unattributed, and smallest.
//   - simple/T9 (60 / 10) and simple/U6 (12 / 13) are not green and are not tracked anywhere else. U6 is
//     the only case whose leak SHRINKS as the sampling refines, which rules out a fixed station count.
func knownMeshLeaks() []meshDebtEntry {
	return []meshDebtEntry{
		{"T9", "simple", 10, 60}, // FAIL(faulty) — one BSpline face; also the corpus's worst folds
		{"U6", "simple", 13, 12}, // FAIL(area) — cylinder/torus rim; note it SHRINKS with quality
		{"C2", "simple", 0, 3},   // PASS — 2-vs-1 stations on one shared straight segment (see above)
		{"Y1", "simple", 3, 3},   // PASS — knownRetracingLoops, 1 loop
		{"Z1", "simple", 3, 3},   // PASS — unattributed, in no other ratchet
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
	t.Parallel()
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
