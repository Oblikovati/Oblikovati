// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/ops/internal/tol"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// concaveInwardRealizable reports whether an INWARD recess fillet is geometrically realizable at
// edge e. The rolling ball sits in the material (cen = station + offDir·r) and is tangent to the
// two walls at cen+nA·r and cen+nB·r (the frame the corner solver actually uses — see
// cornerTangents; NOT the forensic's c−nA·r). The recess is realizable only where BOTH tangent
// points land on the bounded faces — material behind, void in front. At a reflex prism corner
// (TestFilletConcaveInwardDegenerate's L) both tangents fall into the bulk (material on both
// sides), so trimming the walls to those phantom tangent lines self-intersects; reject honestly.
//
// This gate is the explicit realizability check the inward branch never had: pre-B2
// (fee0da5c, orientFilletShell) the impossible inward shell was rejected only ACCIDENTALLY by
// inconsistent loop winding, which the combinatorial winding-normalizer then laundered into
// Validate-passing but self-intersecting topology.
//
// It returns the first offending tangent point (for the sick-feature message) and false when
// unrealizable; math.Point3{}/true when every station is material-backed.
func concaveInwardRealizable(body *topo.Body, e *topo.Edge, nA, nB, offDir math.Vector3, r float64) (math.Point3, bool) {
	inside := brep.NewInsideQuery(body)
	eps := tol.ForBody(body).Plane() // model-relative on-surface nudge (ADR-0042), not a bare 1e-6
	for _, station := range edgeStations(e) {
		cen := station.TranslateBy(offDir.Scale(r))
		if ta := cen.TranslateBy(nA.Scale(r)); !tangentBackedByMaterial(inside, ta, nA, eps) {
			return ta, false
		}
		if tb := cen.TranslateBy(nB.Scale(r)); !tangentBackedByMaterial(inside, tb, nB, eps) {
			return tb, false
		}
	}
	return math.Point3{}, true
}

// edgeStations samples an edge at both end vertices and its midpoint. Realizability varies along
// the edge (an end tangent can fall off a face the midpoint tangent still covers), so a mid-only
// check can miss it — test every station.
func edgeStations(e *topo.Edge) []math.Point3 {
	a, b := e.StartVertex().Point(), e.EndVertex().Point()
	return []math.Point3{a, b, a.Midpoint(b)}
}

// tangentBackedByMaterial reports whether tangent point t sits on the solid's boundary with the
// material on the −n side: nudged inward (−n·eps) it is inside, nudged outward (+n·eps) it is
// outside. n is the wall's material-outward normal. A BURIED tangent (inside on both sides) or a
// FLOATING one (outside on both sides) is not backed — the recess wall would be phantom there.
func tangentBackedByMaterial(inside *brep.InsideQuery, t math.Point3, n math.Vector3, eps float64) bool {
	behind := inside.Inside(t.TranslateBy(n.Scale(-eps)))
	front := inside.Inside(t.TranslateBy(n.Scale(eps)))
	return behind && !front
}
