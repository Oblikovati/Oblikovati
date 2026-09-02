// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	"fmt"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The fillet's inputs: resolving a caller's edge keys into picks, and the per-edge and per-corner
// records the solver fills in (split out of fillet.go for #2217).
//
// A pick is one selected edge plus the radius law along it; an edgeFillet is what the solver
// produces for that pick. Keeping the two record types beside the resolution that mints them is
// what makes the solver files below read as pure transformations.

// computeFillets solves every picked edge's edgeFillet against the already-solved corners,
// recording a faceted-blend diagnostic for any that fell back to the C0 strip.
func computeFillets(body *topo.Body, edges []filletPick, blends map[uint64]*cornerBlend, miters map[uint64]*cornerMiter, concave ConcaveFill, rec *diag.Recorder) ([]edgeFillet, error) {
	fils := make([]edgeFillet, 0, len(edges))
	for _, p := range edges {
		ef, err := computeEdgeFillet(body, p, blends, miters, concave)
		if err != nil {
			return nil, err
		}
		if ef.varying && !ef.exact {
			rec.Recordf(CodeFilletFacetedBlend, diag.Defect,
				"fillet blend on edge %d shipped as the C0 faceted strip (G2-quintic section or miter/blend-terminated variable end)", p.edge.ID())
		}
		fils = append(fils, ef)
	}
	return fils, nil
}

// filletPick is one resolved fillet input: the edge, its per-end radii, and cross-section.
type filletPick struct {
	edge   *topo.Edge
	r0, r1 float64
	mids   []FilletRadiusPoint
	cross  FilletCrossSection
	rho    float64
}

// varying reports whether the pick's radius changes along the edge (differing ends, or any
// intermediate radius point that bulges/pinches the profile).
func (p filletPick) varying() bool { return p.r0 != p.r1 || len(p.mids) > 0 }

// chordPath reports whether the fillet builds via the chord-sampled ruling band rather than the
// analytic cylinder: a varying radius OR any non-arc (G2/conic) cross-section, since those are swept
// NURBS profiles, not a cylinder.
func (p filletPick) chordPath() bool { return p.varying() || !p.cross.IsArc() }

// resolveFilletPicks resolves the edge reference keys against the body, erroring on a lost
// key or a non-positive radius.
func resolveFilletPicks(body *topo.Body, picks []EdgeFilletRadii) ([]filletPick, error) {
	out := make([]filletPick, 0, len(picks))
	for _, p := range picks {
		if p.R0 < 0 || p.R1 < 0 || p.R0+p.R1 <= 0 {
			return nil, fmt.Errorf("fillet: radii %g/%g must be >= 0 with at least one > 0 (a run-out tapers to 0 at one end)", p.R0, p.R1)
		}
		if err := validateRadiusPoints(p.Mids); err != nil {
			return nil, err
		}
		e, ok := body.FindEdgeByKey(p.Key)
		if !ok {
			return nil, fmt.Errorf("fillet: edge reference lost: %x", p.Key)
		}
		out = append(out, filletPick{edge: e, r0: p.R0, r1: p.R1, mids: p.Mids, cross: p.Cross, rho: p.Rho})
	}
	return out, nil
}

// corner is one rounded end of a filleted edge: the cylinder centre at that end, the tangent
// points on faces a/b, and the arc midpoint (the cylinder point nearest the sharp corner).
// At a blend corner the centre is the corner sphere's centre and the tangent points are the
// sphere's tangents (the cylinder ends there and its arc joins the sphere patch). A variable
// fillet's corner additionally carries the arc sampled as chords (ta…tb), shared between the
// blend's ruling strips and the end face so they stay watertight.
type corner struct {
	a, b    *topo.Face
	cen     math.Point3 // cylinder centre at this end (sphere centre when blended)
	ta, tb  math.Point3
	mid     math.Point3
	sh      math.Point3   // exact blends (#1606): the shoulder (tangent-intersection) control point
	crossW  float64       // exact CONIC cross-section only: the shoulder weight the end trim must carry
	chords  []math.Point3 // variable fillet only: the end arc as chord samples ta…tb
	endFace *topo.Face    // the flat end cap to arc (nil at a blend or miter corner)
	// endCurve is the EXACT band∩wall trim of this terminal section when the stop face is not a plane
	// perpendicular to the edge axis (fillet_farend_trim.go). Nil on every corner whose flat section cap
	// already lies on its stop face, which keeps the whole planar corpus byte-identical; when set it
	// replaces the section ARC on both the band's own far end and the wall's loop.
	endCurve geom.Curve3
	// endPieces is the same terminal trim resolved across the CHAIN of faces it actually crosses, ta → tb
	// (fillet_farend_split.go). It is set only when the section leaves the stop face, and it is a
	// PROPOSAL: nothing reads it until commitFarEndSplits accepts the whole multi-face rebuild atomically
	// and sets edgeFillet.splitEnds. On a decline the corner keeps endCurve and is byte-identical.
	endPieces []endPiece
	vertex    *topo.Vertex
	blend     bool
	miter     bool          // two-fillet corner: the end is bounded by seam (no end face, no sphere)
	seam      []math.Point3 // miter only: the seam chords from ta to tb, shared with the other cylinder
	runout    bool          // variable fillet only: r=0 here, the blend collapses to an apex on the edge
}

// tOf returns the tangent point on face f (a or b).
func (c corner) tOf(f *topo.Face) math.Point3 {
	if f == c.a {
		return c.ta
	}
	return c.tb
}

// edgeFillet is a fully solved fillet of one edge: its two faces, the cylinder (constant
// radius only), the two rounded corners, and whether the radius varies along the edge.
type edgeFillet struct {
	a, b    *topo.Face
	cyl     geom.Cylinder
	c0, c1  corner
	mids    []corner // variable fillet only: intermediate profiles ruled between c0 and c1 (#695)
	edge    *topo.Edge
	varying bool
	flip    bool // concave fillet: the cylinder face's outward sense is inverted (surface faces the centre)
	// exact marks a varying/conic blend emitted as the EXACT rational ruled surface (#1606,
	// audit A10) instead of the C0 polyhedral strip; secW is the sections' shoulder weight.
	exact bool
	secW  float64
	// splitEnds records that commitFarEndSplits ACCEPTED both terminal sections' multi-face split and
	// rebuilt every host the chain touches. It is the one switch the band's own cap reads, so the band and
	// the hosts can never disagree about where the trim runs (chain-retrim-report.md §5.2: a partial
	// application is an unclosed shell).
	splitEnds bool
	// armSurface is the exact analytic rolling-ball arm on a CONVEX axis-aligned Plane∧Cylinder edge
	// (M5 Slice A): a geom.Torus (axis ⊥ plane) or a geom.Cylinder (axis ∥ plane). Nil on the ordinary
	// planar straight-edge fillet, whose surface is `cyl`. The corner engine (Task 4) reads it for the
	// section rail; it is byte-invisible to the planar/straight paths, which never set it.
	armSurface geom.Surface
	// armCanalSpine is the exact hyperbola ball-centre spine of a Cone∧Plane RULING-edge canal arm (CN2),
	// carried alongside armSurface (a geom.BSplineSurface — the tessellator keys on the concrete type, so
	// the analytic spine cannot ride inside it). Nil on every non-canal arm. The cone-host corner weld
	// (CN4) reads it for the closed-form arm station; byte-invisible to all other paths.
	armCanalSpine *coneCanalSpine
	// armConcave marks the exact analytic arm as the CONCAVE Cylinder∧Plane cylinder arm (N3/M4/N9): the
	// ball rolls in the reentrant VOID and the fillet ADDS the fill wedge (fillet_concave_arm.go). Its
	// material-outward normal is negated vs the convex arm ((centre−P)/r), so the single-arm runout weld
	// winds the arm band the other way (singleRunoutFaces). FALSE on every convex arm, keeping the convex
	// single-arm runout greens (B6/C9/C1/M7/…) byte-identical.
	armConcave bool
	// armEllipticRim is the CLOSED elliptic-rim canal band payload (J6/J8, fillet_elliptic_rim_canal.go):
	// the lofted canal surface plus its two closed contact rails and seam. It rides alongside armSurface
	// (a geom.BSplineSurface, whose concrete type carries no rails) and is the SOLE dispatch key for the
	// elliptic closed-rim weld — nothing else sets it, so no existing weld can be diverted there. Nil on
	// every other arm, hence byte-invisible to all of them.
	armEllipticRim *ellipticRimCanal
}
