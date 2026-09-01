// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// THE SETBACK END-CAP IS NOT ALWAYS A FACE OF ITS OWN.
//
// rebuildWithArcFillet closes each end of the torus band with a flat SETBACK triangle in the RADIAL
// plane through that end (vc → vt → rimV, endCapNormal). That triangle is a genuine new face only when
// the radial plane differs from the SIDE face the arc end runs into. When the side face IS that radial
// plane — every quarter/sector solid, where the cut wall contains the axis — the triangle lies inside
// the side face's own plane and the two are ONE face of the solid. Emitting them separately does three
// things at once, all measured on simple/B2 (r=10 on the top rim of a 90° cylinder sector):
//
//  1. the side face keeps its un-receded corner: 5000 shipped against DRAWEXE's 4978.54 (per wall);
//  2. the setback triangle is shipped a SECOND time as its own 21.46 face — so the body over-encloses
//     by 2·(21.46 + 21.46) = 85.84, +0.401 % of OCCT's 21389.8, hidden inside the 1 % area gate;
//  3. the cap face's loop is routed rimV → vt through capLine, so it runs r out to the superseded rim
//     vertex and r straight back — a RETRACE of exactly the fillet radius, enclosing zero area and
//     therefore invisible to every area gate (retrace-detector-report.md §7.2, knownRetracingLoops).
//
// The merge below supersedes the rim vertex instead of drawing to it: the side face absorbs the band's
// terminal cross-section arc, the cap∩side edge is re-ended on the cap-tangent point, and no separate
// end-cap face (and no rim vertex) is built. It is gated on EXACT coplanarity, so an end whose radial
// plane genuinely differs from its side face keeps the setback triangle it needs.

// endCapCoplanarTol is the angular budget (radians) for "the setback cap's radial plane IS the side
// face's plane". It is deliberately at construction-noise level rather than a modelling tolerance: the
// two planes either coincide by construction (a radial cut wall — B2's x=0/y=0, N6's x=50 — where the
// measured deviation is 2.2e-16) or differ by a real angle (N6's second end at 0.6435 rad off, W2's two
// ends at 1e-4 from an imported cylinder whose ref direction is not exact). Merging a MERELY NEAR
// coplanar cap would put the absorbed cross-section arc off the side face's own surface, which
// TestEveryLoopSegmentLiesOnItsFace exists to forbid — so near-coplanar declines and keeps its triangle.
const endCapCoplanarTol = 1e-9

// resolveEndCapMerges decides, per end, whether the side face ABSORBS the band's terminal cross-section
// (so the rim vertex is superseded) and records the cap∩side edge that absorption re-ends. An end that
// declines keeps the pre-existing separate-triangle topology verbatim.
func (g *arcBuild) resolveEndCapMerges() {
	for i := range 2 {
		g.capSide[i], g.merged[i] = absorbableEndCap(g.af, i)
	}
}

// absorbed reports whether end i's terminal cross-section belongs to the side face rather than to a
// setback triangle of its own. TWO constructions land here and they share every topological consequence
// — the rim vertex is superseded, the cap∩side edge is re-ended on the cap-tangent point, the side face
// carries the terminal curve, and no end-cap face is emitted: the exact coplanar MERGE (a wall through
// the axis) and the RUN-OUT (the band terminated on the side plane, fillet_arc_runout.go).
func (g *arcBuild) absorbed(i int) bool { return g.merged[i] }

// absorbableEndCap reports whether end i is absorbed by its side face, returning the cap∩side edge that
// absorption shortens. It declines unless the rim vertex carries EXACTLY the three edges the rebuild
// knows how to retire — the filleted arc, the cyl∩side smooth line, and one straight cap∩side edge —
// because superseding a vertex is only safe when every edge reaching it is rebuilt.
func absorbableEndCap(af *arcFillet, i int) (*topo.Edge, bool) {
	end := af.ends[i]
	side, isPlane := end.sideF.Geometry().(geom.Plane)
	if !isPlane {
		return nil, false
	}
	if end.runout == nil && !endCapLiesInSide(af, i, side) {
		return nil, false
	}
	return soleCapSideEdge(end, af.arcEdge, af.capF)
}

// endCapLiesInSide reports whether end i's setback cap is exactly coplanar with that end's side face.
func endCapLiesInSide(af *arcFillet, i int, side geom.Plane) bool {
	capN, err := math.UnitVector3FromVector(endCapNormal(af, i))
	return err == nil && side.Normal().AsUnit().IsParallelTo(capN, endCapCoplanarTol)
}

// soleCapSideEdge returns the one straight cap∩side edge at the end's rim vertex, or ok=false when the
// vertex carries anything other than exactly {arc, smooth line, that edge}.
func soleCapSideEdge(end arcEnd, arc *topo.Edge, capF *topo.Face) (*topo.Edge, bool) {
	edges := end.rimV.Edges()
	if len(edges) != 3 {
		return nil, false
	}
	for _, e := range edges {
		if e == arc || e == end.smoothLine {
			continue
		}
		if _, straight := e.Geometry().(geom.LineSegment); !straight {
			return nil, false
		}
		return e, bordersBoth(e, capF, end.sideF)
	}
	return nil, false
}

// bordersBoth reports whether edge e is shared by exactly the two named faces.
func bordersBoth(e *topo.Edge, a, b *topo.Face) bool {
	faces := e.Faces()
	if len(faces) != 2 {
		return false
	}
	return (faces[0] == a && faces[1] == b) || (faces[0] == b && faces[1] == a)
}

// addMergedCapSide builds end i's re-ended cap∩side edge: the same straight line, but stopping at the
// cap-tangent point instead of running on to the rim vertex it supersedes. Its natural direction is
// kept, so every use of the original edge maps across with its own Reversed flag untouched.
func (g *arcBuild) addMergedCapSide(i int, lin topo.Lineage) {
	e := g.capSide[i]
	from, to := g.mergedEnd(e.StartVertex(), i), g.mergedEnd(e.EndVertex(), i)
	g.capShort[i] = g.bld.AddEdge(geom.NewLineSegment(from.Point(), to.Point()), from, to, lin)
}

// mergedEnd maps one endpoint of the cap∩side edge: the superseded rim vertex becomes the cap-tangent
// vertex, any other endpoint is its plain copy.
func (g *arcBuild) mergedEnd(v *topo.Vertex, i int) *topo.Vertex {
	if v == g.af.ends[i].rimV {
		return g.vt[i]
	}
	return g.verts[v]
}

// supersededRimVertex reports whether v is a rim vertex the merge retires (so it is never copied into
// the rebuilt body — no edge of the result reaches it).
func (g *arcBuild) supersededRimVertex(v *topo.Vertex) bool {
	for i := range 2 {
		if g.absorbed(i) && v == g.af.ends[i].rimV {
			return true
		}
	}
	return false
}

// replacedEdge reports whether an original edge is rebuilt rather than copied: the filleted arc, each
// end's smooth line (split at the cyl-tangent point), and — where the end cap merged — that end's
// cap∩side edge (re-ended on the cap-tangent point).
func (g *arcBuild) replacedEdge(e *topo.Edge) bool {
	af := g.af
	if e == af.arcEdge || e == af.ends[0].smoothLine || e == af.ends[1].smoothLine {
		return true
	}
	return (g.absorbed(0) && e == g.capSide[0]) || (g.absorbed(1) && e == g.capSide[1])
}

// capSideIndex returns the end whose cap∩side edge e is (and whether the merge owns it).
func (g *arcBuild) capSideIndex(e *topo.Edge) (int, bool) {
	for i := range 2 {
		if g.absorbed(i) && e == g.capSide[i] {
			return i, true
		}
	}
	return 0, false
}

// arcChainOnCap is the substitution the CAP face makes for the filleted arc, walking end 0 → end 1. A
// merged end contributes nothing — the cap simply starts (or stops) on its cap-tangent point, because
// the cap∩side edge already reaches it — while an unmerged end still needs its cap line out to the rim
// vertex the setback triangle is drawn to.
func (g *arcBuild) arcChainOnCap() []chainEdge {
	af := g.af
	chain := make([]chainEdge, 0, 3)
	if !g.absorbed(0) {
		chain = append(chain, chainEdge{g.capLn[0], g.verts[af.ends[0].rimV], g.vt[0]})
	}
	chain = append(chain, chainEdge{g.capTan, g.vt[0], g.vt[1]})
	if !g.absorbed(1) {
		chain = append(chain, chainEdge{g.capLn[1], g.vt[1], g.verts[af.ends[1].rimV]})
	}
	return chain
}

// smoothChainOnSide is the substitution the SIDE face makes for end i's smooth line, walking away from
// the cap. Unmerged, that is the historical two-piece split of the line itself (rim vertex → cyl-tangent
// → bottom). Merged, the rim vertex is gone and the side face instead carries the band's terminal
// cross-section ARC from the cap-tangent point down to the cyl-tangent point, then the rest of the line.
// This is the whole geometric content of the merge: the side face is the one that gains (or loses) the
// setback triangle, so it is the one that must bound it.
func (g *arcBuild) smoothChainOnSide(i int) []chainEdge {
	af := g.af
	bottom := g.verts[af.ends[i].bottomV]
	if !g.absorbed(i) {
		return []chainEdge{{g.upper[i], g.verts[af.ends[i].rimV], g.vc[i]}, {g.lower[i], g.vc[i], bottom}}
	}
	return []chainEdge{{g.endArc[i], g.vt[i], g.vc[i]}, {g.lower[i], g.vc[i], bottom}}
}
