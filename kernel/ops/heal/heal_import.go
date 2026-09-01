// SPDX-License-Identifier: GPL-2.0-only

package heal

import "oblikovati.org/kernel/topo"

// HealImportedBody repairs an imported B-rep (M25) so it tessellates watertight and its faces carry
// the parameter-space trim the NURBS mesher needs. Order matters: SnapEdgesToSurfaces runs FIRST so
// every edge lies on its adjacent surfaces, THEN ReconstructPcurves builds each edge-use's (u,v) curve
// from that on-surface polyline (a pcurve built from an off-surface edge would mis-trim the face).
//
// A natively-modelled or accurately-authored body passes through unchanged — snapping leaves edges
// already on their surfaces native (residual below snapResidualFloor). Idempotent.
//
// Example: bodies, _, _ := step.Reader{}.ImportSolids(data, opts); for _, b := range bodies { heal.HealImportedBody(b, ops.DefaultQuality()) }
func HealImportedBody(b *topo.Body, q Quality) {
	SnapEdgesToSurfaces(b, q)
	ReconstructPcurves(b, q)
}
