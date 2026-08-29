// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
)

// Analytic-face reconstruction assembly (ADR-0056 Layer 2b). The exact mesh-arrangement
// boolean (kernel/meshbool) produces a watertight but FACETED result; the ops adapter
// groups those facets by provenance tag and hands this package, per output face, either
// an original face to copy wholesale (an untouched survivor keeps its exact surface AND
// its original analytic boundary edges — the only way a periodic wall's seam loop
// survives correctly) or a rebuilt spec (surface + analytic boundary curves) for a face
// the boolean split. Both feed the curved-boolean stitch, which welds by geometry so a
// shared edge two faces both carry as the same curve fuses into one — watertight.

// ReconEdge is one boundary edge of a rebuilt face: an analytic curve and the oriented
// parameter interval this face's loop walks it (t0→t1, so PointAt(t0) begins the edge).
type ReconEdge struct {
	Curve  geom.Curve3
	T0, T1 float64
}

// ReconLoop is one boundary loop of a rebuilt face, its edges ordered head-to-tail.
// Outer marks the outer boundary (there is at most one; the rest are holes).
type ReconLoop struct {
	Outer bool
	Edges []ReconEdge
}

// ReconInput is one output face of the reconstruction. Exactly one form is used:
//   - PassThrough != nil copies that original face (its surface, sense and original
//     boundary edges), with the sense flipped when ForceReversed (a Difference cavity
//     wall keeps b's surface but bounds the removed material).
//   - otherwise it is a rebuilt face: Surface + Loops (analytic boundary curves), with
//     Reversed the face sense and Lineage its provenance.
type ReconInput struct {
	PassThrough   *topo.Face
	ForceReversed bool

	Surface  geom.Surface
	Reversed bool
	Loops    []ReconLoop
	Lineage  topo.Lineage
	// AliasKeys are the reference keys of coplanar operand faces the boolean merged into this rebuilt
	// face; ReconstructBooleanBody registers the built face under each so a pick on any merged parent
	// survives (ADR-0057). Empty for an unmerged face and always empty for a PassThrough.
	AliasKeys [][]byte
}

// ReconstructBooleanBody welds reconstructed analytic faces into a watertight solid,
// reusing curvedStitch (weld vertices by position, edges by endpoints+midpoint). A
// pass-through face contributes its original analytic surface and boundary; a rebuilt
// face its given surface and curves. Because both a shared edge's incident faces carry
// the identical curve, the stitch fuses them — the result is closed by construction.
func ReconstructBooleanBody(inputs []ReconInput) *topo.Body {
	faces := make([]curvedFace, 0, len(inputs))
	for _, in := range inputs {
		faces = append(faces, in.toCurvedFace())
	}
	// The mesh arrangement can split one straight edge across runs, leaving a spurious 2-valent
	// collinear vertex the exact boolean never makes; dissolve those so the reconstructed topology
	// matches brep's and a downstream near-tangent boolean is not broken by the extra edge (#2247).
	return dissolveCollinearVertices(curvedStitch(faces))
}

// toCurvedFace converts one ReconInput to the internal curvedFace model.
func (in ReconInput) toCurvedFace() curvedFace {
	if in.PassThrough != nil {
		f := in.PassThrough
		return canonicalPlanarOutward(curvedFace{
			surface:  f.Geometry(),
			reversed: f.Reversed() != in.ForceReversed, // XOR: flip an untouched b wall for Difference
			loops:    loopsOf(f),
			lineage:  f.Lineage(),
		})
	}
	return canonicalPlanarOutward(curvedFace{
		surface:   in.Surface,
		reversed:  in.Reversed,
		loops:     reconLoopsToCurved(in.Loops),
		lineage:   in.Lineage,
		aliasKeys: in.AliasKeys,
	})
}

// canonicalPlanarOutward enforces the kernel convention that a PLANAR face stores its surface with
// the normal pointing OUTWARD (reversed=false). Many consumers read f.Geometry().NormalAt as the
// face's outward normal — boss, hole, chamfer-corner, thicken, flat-pattern, pick — and brep upholds
// the convention, so a reconstructed body must too or those consumers silently invert. A Difference
// cut plane otherwise arrives reversed=true (surface normal inward). Swapping the plane's U/V axes
// negates its normal over the IDENTICAL point-set, so clearing reversed keeps the exact same face —
// same outward normal, same 3D loops — in the canonical representation. Curved faces keep reversed:
// an inward-facing cylinder/sphere has no negated-normal surface, so reversed IS its legitimate form.
func canonicalPlanarOutward(cf curvedFace) curvedFace {
	if !cf.reversed {
		return cf
	}
	pl, ok := cf.surface.(geom.Plane)
	if !ok {
		return cf
	}
	cf.surface = geom.Plane{Origin: pl.Origin, UAxis: pl.VAxis, VAxis: pl.UAxis} // U×V negates the normal
	cf.reversed = false
	return cf
}

// reconLoopsToCurved converts rebuilt loops to curvedLoops, outer loop first (curvedStitch
// treats loops[0] as the outer boundary).
func reconLoopsToCurved(loops []ReconLoop) []curvedLoop {
	var outer, holes []curvedLoop
	for _, l := range loops {
		cl := curvedLoop{edges: reconEdgesToLoop(l.Edges)}
		if l.Outer {
			outer = append(outer, cl)
		} else {
			holes = append(holes, cl)
		}
	}
	return append(outer, holes...)
}

// reconEdgesToLoop converts rebuilt edges to loopEdges.
func reconEdgesToLoop(edges []ReconEdge) []loopEdge {
	out := make([]loopEdge, len(edges))
	for i, e := range edges {
		out[i] = loopEdge{curve: e.Curve, t0: e.T0, t1: e.T1}
	}
	return out
}
