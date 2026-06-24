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

// bandWrapRings reads the band's rim rings by topology rather than curve type (the rims are saddle
// polylines, not circles): every closed boundary edge (its start and end vertex coincide) is a rim.
func bandWrapRings(f *topo.Face, q Quality) [][]math.Point3 {
	var rings [][]math.Point3
	for _, e := range f.Edges() {
		if e.StartVertex() == e.EndVertex() {
			rings = append(rings, dropClosingDup(TessellateEdge(e, q)))
		}
	}
	return rings
}
