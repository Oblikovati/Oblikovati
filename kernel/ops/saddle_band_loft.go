// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Saddle-band loft (M2 Phase 2, Oblikovati/Oblikovati#1335). A cylinder's wall inside a crossing
// cylinder is a full-period band whose two rims are NOT circles but the traced saddle curves where the
// surfaces meet — so each rim's axial parameter v varies with the angle u. periodicBandGrid meshes only
// bands with two constant-v (circular) rims, and closedDomainMesh grids a single v-resolution; neither
// fits a saddle rim. But a cylinder and a cone are RULED along v (the straight line between the two rims
// at a fixed angle lies exactly on the surface), so the band needs no interior rows at all: stitching the
// two rim rows — each the exact tessellation of its own saddle edge, so the band welds to the caps that
// share those edges — is itself an exact loft of the band.

// notchedRimBandMesh meshes a singly-periodic ruled side whose two rims are ONE full-period circle and
// ONE notched rim — a frustum flat that fades out before the small rim, leaving the kept cone side as a
// band with a complete circular rim plus a rim notched down to the hyperbola vertex
// (Oblikovati/Oblikovati#1374). toUVLoops "succeeds" on such a face (the notched outer loop unwraps) and
// would route it to metricPatchMesh, which treats the full circle as an interior hole and meshes a
// boundary that does not weld to the cap sharing that rim. But it is a genuine two-rim band, so the
// saddle-band loft meshes it exactly (ruled rows) AND tessellates each rim by its own shared edge, so it
// welds. ok=false unless the face is a developable side carrying both a full-circle rim and an open
// (notched) rim edge — so a plain two-circle band (which already bypasses toUVLoops) is untouched.
func notchedRimBandMesh(f *topo.Face, s geom.Surface, q Quality) (*Mesh, bool) {
	if !isDevelopableSide(s) || faceHasLensHole(f, s, q) || !hasFullCircleAndNotchedRim(f) {
		return nil, false // a band carrying a genuine (non-wrapping) lens hole is twoRimHoledBandMesh's job:
		// the saddle loft pools ALL open edges into one rim, so it would fold the lens into the base rim (#1591).
	}
	return saddleBandLoftMesh(f, s, q)
}

// faceHasLensHole reports whether the face carries a genuine LENS hole — an interior loop that does NOT wrap
// the full period. A non-outer loop that DOES wrap (a second full-wrap rim faceHoleBoundaries mis-demoted to a
// "hole") is still a valid two-rim-band rim the saddle loft meshes, so it must NOT disqualify the face; only a
// true lens (a drilled tunnel's entry, spanning a small arc) does, since the pure two-rim loft can't carry it.
func faceHasLensHole(f *topo.Face, s geom.Surface, q Quality) bool {
	for _, l := range f.Loops() {
		if l.IsOuter() {
			continue
		}
		if !holeWrapsPeriod(s, loopBoundary(l, q)) {
			return true
		}
	}
	return false
}

// twoClosedRimBandMesh meshes a developable side (cylinder/cone) bounded by exactly TWO closed full-wrap
// rim edges and no open rim edge — the shape produced by the (u,v) cone-side split, whose band has one
// circular rim and one oblique-cut ellipse rim, both closed loops with no seam (Oblikovati#1375). Such a
// face's two loops each "unwrap" through toUVLoops, which would mis-route it to metricPatchMesh — a flat
// best-fit-plane annulus that neither follows the cone nor welds to the caps sharing those rims. But it
// is a genuine two-rim ruled band, so the saddle-band loft meshes it exactly (ruled rim-to-rim rows) and
// tessellates each rim by its own shared edge, so it welds. ok=false unless the face is a developable
// side with exactly two closed rim edges and no open (non-seam) edge.
func twoClosedRimBandMesh(f *topo.Face, s geom.Surface, q Quality) (*Mesh, bool) {
	if !isDevelopableSide(s) || !hasTwoClosedRimsNoOpen(f) {
		return nil, false
	}
	return saddleBandLoftMesh(f, s, q)
}

// isPeriodicTwoRimBand reports whether the face is a no-seam two-rim ruled band — either two closed rims
// (a (u,v) cone split) or a full circle plus a notched rim (#1374) — which the saddle loft meshes exactly
// and whose full-wrap outer loop must NOT be re-meshed through the (u,v) CDT (it can spin / tear there).
func isPeriodicTwoRimBand(f *topo.Face) bool {
	return hasTwoClosedRimsNoOpen(f) || hasFullCircleAndNotchedRim(f)
}

// hasTwoClosedRimsNoOpen reports whether the face is the no-seam two-loop ruled band: exactly two closed
// rim edges, no open edge, and NO seam edge. The absence of a seam is what distinguishes it from an
// ordinary full periodic side (two closed circles bridged by a seam, used twice) — those are already
// meshed by periodicBandGrid/cylinderSide and must keep that path (e.g. the off-surface-rim snap repair).
func hasTwoClosedRimsNoOpen(f *topo.Face) bool {
	if len(seamEdgesOf(f)) != 0 {
		return false // a seam-bridged full side: handled by the grid path, not the two-loop band loft
	}
	closed := 0
	for _, e := range f.Edges() {
		if e.StartVertex() != e.EndVertex() {
			return false // an open rim: handled by notchedRimBandMesh, not here
		}
		closed++
	}
	return closed == 2
}

// hasFullCircleAndNotchedRim reports whether the face has BOTH a full-period closed rim edge and at least
// one open (non-closed) rim edge — the notched-band signature. Seam edges (used twice within the face) are
// excluded, as they bridge the rims and belong to neither. Closure (start==end vertex), not curve type,
// identifies the full-wrap rim — consistent with hasTwoClosedRimsNoOpen: on a developable side a closed rim
// necessarily wraps the whole period. The old geom.Circle-only gate mis-declined #1591's split-base boss
// wall, whose top rim is stored as a full-turn geom.Arc3d (anchored at the seam angle), not a geom.Circle,
// so the wall fell through to a full-periodic grid that ignores the shared arc discretization → a crack.
func hasFullCircleAndNotchedRim(f *topo.Face) bool {
	seam := seamEdgesOf(f)
	closedRim, notched := false, false
	for _, e := range f.Edges() {
		if seam[e] {
			continue
		}
		if e.StartVertex() == e.EndVertex() {
			closedRim = true
		} else {
			notched = true
		}
	}
	return closedRim && notched
}

// saddleBandLoftMesh meshes a singly-periodic ruled band (a trimmed cylinder/cone side) bounded by two
// full-wrap rim loops, stitching the rims directly. ok=false unless the surface is singly periodic (a
// cylinder/cone, not a torus/plane) and the face has exactly two closed rim edges.
func saddleBandLoftMesh(f *topo.Face, s geom.Surface, q Quality) (*Mesh, bool) {
	if isPeriodic(s.UDomain()) == isPeriodic(s.VDomain()) {
		return nil, false // need exactly one periodic direction (a cylinder/cone side, not a torus/plane)
	}
	rings := bandWrapRings(f, q)
	if len(rings) != 2 {
		return nil, false
	}
	lo, hi := orderedRing(s, rings[0]), orderedRing(s, rings[1])
	if len(lo) < 3 || len(hi) < 3 {
		return nil, false
	}
	if !ringSingleValuedInAngle(s, lo) || !ringSingleValuedInAngle(s, hi) {
		return nil, false // a rim with a vertical step (two axial stations at one angle) is no v(u) loft rim
	}
	m := &Mesh{}
	stitchBandRows(m, addRow(m, s, lo, ringAngles(s, lo)), addRow(m, s, hi, ringAngles(s, hi)))
	return m, true
}

// ringSingleValuedInAngle reports whether the rim ring is a genuine v(u) graph: no two ring points
// share the same periodic angle while sitting at distinct axial stations. The loft stitches rows by
// angle order, so a rim carrying a VERTICAL segment — a miter rail, a seam-line remnant, a bridge
// riser (blend/simple P5's retrimmed wall) — has an arbitrary order among its same-angle points and
// the loft would cross-stitch them into a bowtie crack. Such a face is not a two-rim band; declining
// here sends it to the general trimmed-CDT path. Genuine saddle/notched rims (Steinmetz, #1374's
// fading frustum flat, U4's oblique hole wall) are strictly monotone in angle and pass untouched.
func ringSingleValuedInAngle(s geom.Surface, ring []math.Point3) bool {
	angles := ringAngles(s, ring)
	weld := ResolutionForPoints(ring).Weld()
	for i := 1; i < len(angles); i++ {
		if angles[i]-angles[i-1] < seamAngularTol && ring[i].DistanceTo(ring[i-1]) > weld {
			return false // two distinct stations at one angle: a vertical step, not a v(u) rim
		}
	}
	return true
}

// bandWrapRings reads the band's rim rings by topology rather than curve type (a rim may be a circle or a
// saddle polyline). A seam edge (used twice within the face) bridges the two rims and belongs to neither, so
// it is dropped. Each remaining CLOSED edge is a rim on its own; the remaining OPEN edges are arcs that
// together form one saddle rim (e.g. an equal-radius Steinmetz band whose saddle rim is two ellipse arcs
// meeting at the pinch points), so their points are pooled into a single ring (orderedRing later sorts them
// by angle). Coincident points where arcs meet are de-duplicated.
//
// ★ Each rim edge is read through discretizeEdge — the ONE function every other face of that edge already
// shares (edge_discretize.go's package doc) — NOT through the raw TessellateEdge sampler. The two differ on
// exactly the case simple/U4 leaked on: TessellateEdge returns the bare chord-sagitta sampling of the curve,
// while discretizeEdge additionally (a) returns a healed edge's stored on-surface polyline and (b) applies
// the #2009 starved-rail densification (densifyStarvedRail), whose whole correctness argument is that it is
// caller-INDEPENDENT. Reading the ring raw silently opted this mesher out of that argument: U4's oblique-hole
// wall is a notched band, so it tiled the straight rim edge it shares with the high-aspect fillet sliver with
// ONE chord while the sliver's CDT tiled the SAME topological edge with 21 — 21 + 21 + 2 = 44 free edges at
// BOTH gate qualities, on a body every other gate scored green. The endpoints matched exactly; only the
// station counts differed, which is why no area, off-surface or loop gate could see it.
func bandWrapRings(f *topo.Face, q Quality) [][]math.Point3 {
	seam := seamEdgesOf(f)
	var rings [][]math.Point3
	var openPts []math.Point3
	for _, e := range f.Edges() {
		if seam[e] {
			continue
		}
		pts := dropClosingDup(discretizeEdge(e, q))
		if e.StartVertex() == e.EndVertex() {
			rings = append(rings, pts)
		} else {
			openPts = append(openPts, pts...)
		}
	}
	if len(openPts) > 0 {
		rings = append(rings, dedupRingPoints(openPts))
	}
	return rings
}

// seamEdgesOf returns the edges used more than once within the face — the seam(s) bridging its two rims,
// which belong to neither rim ring.
func seamEdgesOf(f *topo.Face) map[*topo.Edge]bool {
	count := map[*topo.Edge]int{}
	for _, l := range f.Loops() {
		for _, u := range l.EdgeUses() {
			count[u.Edge()]++
		}
	}
	seam := map[*topo.Edge]bool{}
	for e, n := range count {
		if n > 1 {
			seam[e] = true
		}
	}
	return seam
}

// dedupRingPoints drops points coincident with an earlier one (the pinch points where two saddle-rim arcs
// meet appear in both arcs). The coincidence tolerance is model-relative (ADR-0042, #1399): derived from
// the ring's own extent so a km-scale band still welds its pinch points instead of cracking.
func dedupRingPoints(pts []math.Point3) []math.Point3 {
	weld := ResolutionForPoints(pts).Weld()
	var out []math.Point3
	for _, p := range pts {
		dup := false
		for _, q := range out {
			if p.DistanceTo(q) < weld {
				dup = true
				break
			}
		}
		if !dup {
			out = append(out, p)
		}
	}
	return out
}
