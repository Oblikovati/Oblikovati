// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati/kernel/geom"
	"oblikovati/kernel/topo"
)

// ReconstructFacePcurves computes and attaches the PCURVE to every edge-use of f: each edge is
// discretized into the SAME chord polyline the mesher uses (discretizeEdge, so the pcurve aligns
// with the mesh boundary exactly), oriented to the use, then projected onto f's surface
// (geom.ProjectCurveToSurface — a seeded march that stays on one branch). This is the parameter-space
// trim boundary SolidWorks STEP omits; with it the face's trim region is known exactly in (u,v),
// which the tolerant NURBS mesher requires (M25 F01, ADR-0031). Idempotent: re-running recomputes.
func ReconstructFacePcurves(f *topo.Face, q Quality) {
	s := f.Geometry()
	for _, l := range f.Loops() {
		for _, u := range l.EdgeUses() {
			pts := discretizeEdge(u.Edge(), q)
			if u.Reversed() {
				pts = reverse3(pts)
			}
			u.SetPcurve(geom.ProjectCurveToSurface(s, pts))
		}
	}
}

// ReconstructPcurves attaches pcurves to every face of a body (the import-healing entry point).
func ReconstructPcurves(b *topo.Body, q Quality) {
	for _, f := range b.Faces() {
		ReconstructFacePcurves(f, q)
	}
}
