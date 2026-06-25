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
	if !isDevelopableSide(s) || !hasFullCircleAndNotchedRim(f) {
		return nil, false
	}
	return saddleBandLoftMesh(f, s, q)
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

// hasFullCircleAndNotchedRim reports whether the face has BOTH a full-period closed-circle rim edge and
// at least one open (non-closed) rim edge — the notched-band signature. Seam edges (used twice within
// the face) are excluded, as they bridge the rims and belong to neither.
func hasFullCircleAndNotchedRim(f *topo.Face) bool {
	seam := seamEdgesOf(f)
	fullCircle, notched := false, false
	for _, e := range f.Edges() {
		if seam[e] {
			continue
		}
		if e.StartVertex() == e.EndVertex() {
			if _, isCircle := e.Geometry().(geom.Circle); isCircle {
				fullCircle = true
			}
		} else {
			notched = true
		}
	}
	return fullCircle && notched
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
	m := &Mesh{}
	stitchBandRows(m, addRow(m, s, lo, ringAngles(s, lo)), addRow(m, s, hi, ringAngles(s, hi)))
	return m, true
}

// bandWrapRings reads the band's rim rings by topology rather than curve type (a rim may be a circle or a
// saddle polyline). A seam edge (used twice within the face) bridges the two rims and belongs to neither, so
// it is dropped. Each remaining CLOSED edge is a rim on its own; the remaining OPEN edges are arcs that
// together form one saddle rim (e.g. an equal-radius Steinmetz band whose saddle rim is two ellipse arcs
// meeting at the pinch points), so their points are pooled into a single ring (orderedRing later sorts them
// by angle). Coincident points where arcs meet are de-duplicated.
func bandWrapRings(f *topo.Face, q Quality) [][]math.Point3 {
	seam := seamEdgesOf(f)
	var rings [][]math.Point3
	var openPts []math.Point3
	for _, e := range f.Edges() {
		if seam[e] {
			continue
		}
		pts := dropClosingDup(TessellateEdge(e, q))
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
