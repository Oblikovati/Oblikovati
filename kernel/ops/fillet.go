// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// FilletRadiusPoint is an intermediate radius along a variable fillet edge: R at parameter T in
// (0,1) measured from the start vertex (#695). Points let the radius follow a piecewise profile —
// e.g. a bulge in the middle — instead of only the linear R0→R1 taper.
type FilletRadiusPoint struct {
	T, R float64
}

// FilletCrossSection is the shape of a fillet's cross-section profile (M36-F08). It ALIASES the
// canonical api/types definition (ADR-0018): a string enum whose empty/"arc" value is the circular
// rolling-ball blend (G1 — tangent to both walls), with G2 (curvature-continuous) and Conic
// (rho-controlled) flowing more smoothly for Class-A styling. The kernel selects its blend builder on
// this value (see crossSectionChords); IsArc() reports the default so a never-set Cross reads as arc.
type FilletCrossSection = types.FilletCrossSection

const (
	FilletArc   = types.FilletSectionArc   // circular rolling-ball cross-section (G1, the default)
	FilletG2    = types.FilletSectionG2    // curvature-continuous cross-section (quintic, no curvature jump)
	FilletConic = types.FilletSectionConic // conic (rho-controlled) cross-section; pair with Rho
)

// EdgeFilletRadii is one picked edge with its blend radius at each end: R0 at the edge's
// start vertex, R1 at its end vertex. Equal radii give a constant fillet; differing radii a
// variable fillet whose radius runs linearly along the edge (#323). Mids adds intermediate radius
// points the blend interpolates through (#695), sorted by T; empty = the plain R0→R1 taper.
// Cross selects the cross-section shape (default arc); Rho sets a conic's fullness (#1284).
type EdgeFilletRadii struct {
	Key    []byte
	R0, R1 float64
	Mids   []FilletRadiusPoint
	Cross  FilletCrossSection
	Rho    float64
}

// FilletEdges rounds the selected convex straight edges of a planar solid with a constant-
// radius rolling-ball blend: each edge between two planar faces is replaced by a cylinder
// face of radius r tangent to both, the two faces are retrimmed back to the tangent lines,
// and the end faces gain a quarter-arc at the rounded corner. All edges are resolved and
// solved on the original body, then applied in a single rebuild, so independent edges that
// share a face (e.g. the four verticals of a box) all retrim that face correctly. Convex,
// straight edges with one extra face at each end (box/prism edges); chains, corners where
// fillets meet, and concave edges are follow-ups.
func FilletEdges(body *topo.Body, edgeKeys [][]byte, r float64) (*topo.Body, error) {
	picks := make([]EdgeFilletRadii, len(edgeKeys))
	for i, k := range edgeKeys {
		picks[i] = EdgeFilletRadii{Key: k, R0: r, R1: r}
	}
	return FilletEdgesVarying(body, picks)
}

// FilletEdgesVarying is FilletEdges with a per-end radius for each edge. A variable edge's
// blend is a generalized cone (the rolling ball grows linearly along the edge), built as
// planar trapezoids between successive rulings: adjacent rulings meet at the cone's apex on
// the edge line, so each strip face is EXACTLY planar — the only approximation is the end
// arcs as chords, the same density convention as a hole's faceted cylinder.
func FilletEdgesVarying(body *topo.Body, picks []EdgeFilletRadii) (*topo.Body, error) {
	return FilletEdgesCorner(body, picks, CornerMiter, FillConcaveOutward)
}

// CornerStrategy selects how a corner where two filleted edges meet a vertex whose third edge stays
// sharp is treated (mirrors api types.FilletCornerType). Round and setback both augment the selection
// with the sharp third edge — round at constant radius, setback as a taper that runs out to 0 — so
// the corner resolves as a 3-edge sphere blend; miter keeps the two edges' cylinders meeting at a seam.
type CornerStrategy int

const (
	// CornerMiter mutually trims the two cylinders along their intersection seam (a crease).
	CornerMiter CornerStrategy = iota
	// CornerSetback rounds the corner into a sphere and tapers the third edge to a run-out (set-back).
	CornerSetback
	// CornerRound rounds the corner fully by also filleting the third edge at constant radius (sphere).
	CornerRound
)

// ConcaveFill selects how a fillet treats a CONCAVE (internal) edge — one whose faces fold over the
// material (dihedral > π). A convex edge always rounds its corner away (material removed) and ignores
// this. Mirrors api types.FilletConcaveStrategy.
type ConcaveFill int

const (
	// FillConcaveOutward fills the inside corner with material: the rolling ball sits in the void and
	// the cylinder face bridges the two faces (the default — also the zero value).
	FillConcaveOutward ConcaveFill = iota
	// FillConcaveInward rounds a recess into the corner instead (the ball sits in the material).
	FillConcaveInward
)

// FilletEdgesCorner is FilletEdgesVarying with an explicit 2-edge corner strategy. Lone curved
// (rim/arc) picks ignore it (they have no shared corner). CornerRound augments the selection with the
// sharp third edge of each 2-edge corner so the corner resolves as a watertight 3-edge sphere blend.
// CodeFilletFacetedBlend marks a fillet whose blend shipped as the C0 polyhedral strip fallback
// — a G2-quintic cross-section, or a variable blend terminated by a miter/blend corner — so
// "advertised smooth, shipped faceted" is a counted degradation, never silent (#1606, audit A10).
const CodeFilletFacetedBlend diag.Code = "fillet.faceted-blend"

// FilletEdgesCornerDiag is FilletEdgesCorner with a diagnostic recorder (nil to discard): any
// pick whose blend falls back to the faceted strip records CodeFilletFacetedBlend.
func FilletEdgesCornerDiag(body *topo.Body, picks []EdgeFilletRadii, corner CornerStrategy, concave ConcaveFill, rec *diag.Recorder) (*topo.Body, error) {
	return filletEdgesCornerRec(body, picks, corner, concave, rec)
}

func FilletEdgesCorner(body *topo.Body, picks []EdgeFilletRadii, corner CornerStrategy, concave ConcaveFill) (*topo.Body, error) {
	return filletEdgesCornerRec(body, picks, corner, concave, nil)
}

func filletEdgesCornerRec(body *topo.Body, picks []EdgeFilletRadii, corner CornerStrategy, concave ConcaveFill, rec *diag.Recorder) (*topo.Body, error) {
	if rim := loneRimPick(body, picks); rim != nil {
		return FilletCylinderRim(body, rim.Key, rim.R0) // a circular cylinder/cap rim → toroidal band
	}
	if arc := loneArcPick(body, picks); arc != nil {
		return FilletCylinderArc(body, arc.Key, arc.R0) // a cylinder/cap arc → torus + setback end-caps
	}
	if chain, r, closed, ok := curvedTangentChain(body, picks); ok {
		// A closed mixed tangent loop (#1797) rounds as one continuous stripe; a contiguous open run
		// (ADR-0050 P6) rounds the same way but terminates in a flat setback cap at each end.
		return filletTangentStripe(body, chain, closed, r)
	}
	edges, err := resolveFilletPicks(body, picks)
	if err != nil {
		return nil, err
	}
	switch corner {
	case CornerRound:
		edges = roundThirdEdges(edges) // fillet the third edge at constant radius → 3-edge sphere
	case CornerSetback:
		edges = setbackThirdEdges(edges) // taper the third edge (r→0 run-out) → smooth set-back sphere
	}
	return filletResolvedEdges(body, edges, concave, rec)
}

// filletResolvedEdges solves the corners and edge fillets of an already-resolved pick list and
// assembles the validated result body. Round/setback corners have already been reduced to 3-edge
// sphere blends by augmenting the third edge, so the corner solver only ever sees miters and blends.
func filletResolvedEdges(body *topo.Body, edges []filletPick, concave ConcaveFill, rec *diag.Recorder) (*topo.Body, error) {
	if err := validateFilletRadii(edges, concave); err != nil {
		return nil, err // #1800: reject an over-large radius before it self-intersects
	}
	blends, miters, err := computeCorners(body, edges)
	if err != nil {
		return nil, err
	}
	fils, err := computeFillets(body, edges, blends, miters, concave, rec)
	if err != nil {
		return nil, err
	}
	if curvedArmFils(fils) {
		return weldCurvedArmOrFloor(body, fils, blends, miters) // M5 Slice A weld or the do-no-harm floor
	}
	return assemblePlanarFilletBody(body, edges, fils, blends, miters, concave)
}

// assemblePlanarFilletBody runs the planar fillet's runout guards, assembles the do-no-harm body, and
// certifies it — naming the #1797 corner-into-round cause when the build-then-certify result still fails.
func assemblePlanarFilletBody(body *topo.Body, edges []filletPick, fils []edgeFillet, blends map[uint64]*cornerBlend, miters map[uint64]*cornerMiter, concave ConcaveFill) (*topo.Body, error) {
	if err := applyRunoutSetback(fils); err != nil {
		return nil, err // a runout flank rail is parallel to its far plane — no pierce (n-valent degeneracy)
	}
	if err := validateRunoutFans(fils); err != nil {
		return nil, err // n-valent analogue of #1800: reject a self-intersecting/over-radius runout before it silently drops to an open shell
	}
	var res *topo.Body
	if !blendsCarryRadiusTorus(blends) {
		// A mixed-radius torus corner's transient band ends are not weldable, so its baseline body is
		// never built: the setback pass either certifies the torus composition or the nil baseline
		// falls through to the honest decline below (never a garbage fallback).
		res = assembleFilletBody(body, fils, blends)
	}
	res = adoptCornerSetback(body, edges, fils, blends, miters, concave, res) // corner setback (dihedral+trihedral), do-no-harm floor
	if res == nil {
		return nil, fmt.Errorf("fillet: mixed-radius torus corner did not compose into a certified solid")
	}
	rep := Validate(res)
	if rep.Valid && res.IsSolid() {
		return res, nil
	}
	// build-then-certify (#1797): the corner-into-round was BUILT, not rejected up front. Most such
	// junctions close into a valid solid (asymmetric round); only the symmetric equal-radius corner
	// still fails. Name that actionable cause instead of the generic invalid-solid message.
	if e, round := firstCornerIntoRound(edges); round != nil {
		return nil, cornerIntoRoundError(e, round)
	}
	return nil, fmt.Errorf("fillet: result is not a valid solid %v", rep.Issues)
}

// rebuildChoice names one do-no-harm candidate composition of the two independent local
// rebuilds (mid-span obstacle, ADR-4; double-interference runout, ADR-5).
type rebuildChoice int

const (
	chooseBoth     rebuildChoice = iota // obstacle + runout composed into one watertight solid
	chooseObstacle                      // only the obstacle rebuild improves; runout dropped
	chooseRunout                        // only the runout rebuild improves; obstacle dropped
	chooseBaseline                      // neither improves — the pre-rebuild fillet (do-no-harm)
)

// chooseRebuild picks the highest-priority rebuild composition whose assembled body clears the
// do-no-harm bar. {both} wins when the two rebuilds compose watertight; else the best single path
// (obstacle preferred — the older, more-proven path); else baseline. Splitting the ADR-4 verdict so
// a failing runout can never veto a passing obstacle rebuild (M2 whole-branch review, systemic minor).
func chooseRebuild(improved func(rebuildChoice) bool) rebuildChoice {
	for _, c := range []rebuildChoice{chooseBoth, chooseObstacle, chooseRunout} {
		if improved(c) {
			return c
		}
	}
	return chooseBaseline
}

// assembleFilletBody builds the do-no-harm candidate bodies (ADR-4/ADR-5, Option 1, 2026-07-14; split
// into independent obstacle/runout verdicts, M3 whole-branch review) and picks the highest-priority one
// that clears the bar: a local rebuild may FIRE on a body it cannot fully resolve (e.g. a second obstacle
// column it does not model, or a runout that opens the shell), producing a degraded shell. Gating the two
// verdicts independently means a failing runout can never veto a passing obstacle rebuild (and vice
// versa) — only a strict improvement over the baseline (no-rebuild) fillet is ever kept. That baseline
// fallback is the same green body as before ADR-4 (HolesContained is a tripwire, not folded into Valid).
func assembleFilletBody(body *topo.Body, fils []edgeFillet, blends map[uint64]*cornerBlend) *topo.Body {
	cands := rebuildCandidates(body, fils, blends) // lazily assembled; chooseBaseline always present
	choice := chooseRebuild(func(c rebuildChoice) bool {
		b, ok := cands[c]
		return ok && obstacleImprovedSolid(b)
	})
	return cands[choice]
}

// rebuildCandidates lazily assembles the do-no-harm candidate bodies: the baseline (no local rebuild) is
// always built; {both}/{obstacle-only}/{runout-only} are built only when that composition's collectors
// actually handle an edge — so the overwhelmingly common body (no obstacle, no runout anywhere) costs a
// SINGLE assembleBody call, the same cost as before this split.
func rebuildCandidates(body *topo.Body, fils []edgeFillet, blends map[uint64]*cornerBlend) map[rebuildChoice]*topo.Body {
	cands := map[rebuildChoice]*topo.Body{chooseBaseline: assembleFilletFaces(body, fils, blends, false, false)}
	if _, bothFired := filletResultFaces(body, fils, blends, true, true); !bothFired {
		return cands // neither local rebuild handled any edge: baseline is the only useful candidate
	}
	addRebuildCandidate(cands, chooseBoth, body, fils, blends, true, true)
	addRebuildCandidate(cands, chooseObstacle, body, fils, blends, true, false)
	addRebuildCandidate(cands, chooseRunout, body, fils, blends, false, true)
	return cands
}

// addRebuildCandidate assembles one composition (both, obstacle-only, or runout-only) and records
// it under choice only when that composition's collectors handled an edge on their own.
func addRebuildCandidate(cands map[rebuildChoice]*topo.Body, choice rebuildChoice, body *topo.Body,
	fils []edgeFillet, blends map[uint64]*cornerBlend, enableObstacles, enableRunout bool) {
	if _, fired := filletResultFaces(body, fils, blends, enableObstacles, enableRunout); fired {
		cands[choice] = assembleFilletFaces(body, fils, blends, enableObstacles, enableRunout)
	}
}

// assembleFilletFaces builds and assembles one rebuild composition's faces in a single call. It goes
// through assembleCornerBlendBody (not bare assembleBody) because the planar trihedral path emits the
// same absolute-winding-sensitive corner sphere patch the curved path does: orientFilletShell only
// unifies RELATIVE windings, so a VOID-side corner ball (K6/L4's concave pocket corner) landed wound
// so the sphere-patch mesher filled the 7/8 COMPLEMENT (Ω = 7π/2, area 274.35 vs OCCT's octant
// 39.2699 = 25π/2) at every quality — a +235 area / +522 (= 4πr³/3·mesh) volume mis-measure the 1%
// corpus deps absorbed silently (patchgridcap-report.md §region).
func assembleFilletFaces(body *topo.Body, fils []edgeFillet, blends map[uint64]*cornerBlend, enableObstacles, enableRunout bool) *topo.Body {
	faces, _ := filletResultFaces(body, fils, blends, enableObstacles, enableRunout)
	return assembleCornerBlendBody(body, faces)
}

// obstacleImprovedSolid reports whether an obstacle-rebuilt body is a watertight, hole-contained solid —
// the bar the rebuild must clear to be kept over the baseline fillet.
func obstacleImprovedSolid(res *topo.Body) bool {
	r := Validate(res)
	return r.Valid && res.IsSolid() && r.HolesContained
}

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

// computeEdgeFillet solves the rolling-ball geometry for one convex straight edge, using a
// corner blend at either endpoint that is a shared corner. A varying pick gets its end arcs
// sampled as chords (shared by the ruling strips and the end faces).
func computeEdgeFillet(body *topo.Body, p filletPick, blends map[uint64]*cornerBlend, miters map[uint64]*cornerMiter, concave ConcaveFill) (edgeFillet, error) {
	e := p.edge
	if ef, handled, err := curvedHostArmEdge(body, e, p, concave); handled {
		return ef, err // an exact cylinder/sphere/cone-host arm on a convex curved rim (or concave cyl arm), or its honest reject
	}
	if err := curvedAdjacentError(e); err != nil {
		return edgeFillet{}, err // any other curved neighbour (cyl∩cyl miter seam, torus, sphere)
	}
	// A planar edge whose END runs into a PRIOR round (#1797) is NO LONGER rejected here: the corner
	// is built and certified by Validate downstream (build-then-certify). filletResolvedEdges names the
	// #1797 cause only if that certificate fails (the still-uncloseable symmetric equal-radius corner).
	a, b, nA, nB, err := edgePlanarFaces(e)
	if err != nil {
		return edgeFillet{}, err
	}
	axis, err := math.UnitVector3FromVector(e.StartVertex().Point().VectorTo(e.EndVertex().Point()))
	if err != nil {
		return edgeFillet{}, fmt.Errorf("fillet: degenerate edge")
	}
	in, err := filletFrame(body, e, nA, nB, (p.r0+p.r1)/2, concave)
	if err != nil {
		return edgeFillet{}, err
	}
	in.a, in.b, in.axis = a, b, axis.AsVector()
	in.weld = ResolutionForBody(body).Weld() // F3a: the spine-concurrence tolerance cornerAt gates the override on
	return solvedEdgeFillet(e, p, in, blends, miters)
}

// curvedHostArmEdge dispatches an edge that borders a CURVED host face (cylinder, sphere, cone, or torus)
// to the matching exact-arm builder, in the do-no-harm order cylinder → sphere → cone → torus (each fires
// only for its own host pair, so a Plane∧Plane edge and every other host mix falls through unchanged). handled=true
// means one builder OWNED the edge and computeEdgeFillet must return its result — the built arm or the
// cause-specific honest reject; handled=false leaves the edge to curvedAdjacentError / the planar path.
func curvedHostArmEdge(body *topo.Body, e *topo.Edge, p filletPick, concave ConcaveFill) (edgeFillet, bool, error) {
	if ef, handled, err := concaveCurvedRimArmEdge(body, e, p, concave); handled {
		return ef, handled, err // S2/S5: concave CLOSED sphere/cone cap rim → external-tangency cove arm (or spill reject)
	}
	if ef, handled, err := cylinderArmEdge(body, e, p, concave); handled {
		return ef, handled, err // M5 Slice A: exact cylinder/torus arm on a convex (or concave N3/M4/N9) axis-aligned rim
	}
	if ef, handled, err := sphereArmEdge(body, e, p); handled {
		return ef, handled, err // SP1: exact torus arm on a convex Sphere∧Plane rim
	}
	if ef, handled, err := coneArmEdge(body, e, p); handled {
		return ef, handled, err // CN1: exact torus arm on a convex Cone∧Plane cap (circle) edge
	}
	if ef, handled := ellipticalCylinderArmEdge(body, e, p); handled {
		return ef, true, nil // F4: exact circular-cylinder arm on a convex EllipticalCylinder∧Plane ruling edge
	}
	if ef, handled := ellipticClosedRimArmEdge(body, e, p); handled {
		return ef, true, nil // J6/J8: canal band on a CLOSED EllipticalCylinder∧Plane rim (spine = a closed non-analytic curve)
	}
	if ef, handled := cylCylMiterArmEdge(body, e, p); handled {
		return ef, true, nil // family B: exact cylinder arm on an equal-parallel Cylinder∧Cylinder miter edge (P5)
	}
	return torusArmEdge(body, e, p) // E7: exact torus arm on a convex latitude-cut Torus∧Plane rim
}

// filletFrame resolves the rolling-ball centre offset and the tangent-point normals for an edge,
// choosing the side from the edge's convexity and (for concave edges) the fill strategy:
//   - convex: the ball centre sits INSIDE the solid (offDir = −(nA+nB)/(1+nA·nB)); the corner is
//     rounded away. A centre that is not inside means the edge is not actually convex.
//   - concave + outward (default): the ball sits in the VOID so the cylinder bridges the faces and
//     FILLS the inside corner — the same offDir/normals negated to put the centre on the void side.
//   - concave + inward: the ball sits in the MATERIAL (the convex formula's side), rounding a recess
//     into the corner.
func filletFrame(body *topo.Body, e *topo.Edge, nA, nB math.Vector3, rMid float64, concave ConcaveFill) (cornerInputs, error) {
	offDir := nA.Add(nB).Scale(-1 / (1 + nA.Dot(nB))) // per-unit-radius centre offset into the solid
	mid := e.StartVertex().Point().Midpoint(e.EndVertex().Point())
	if ClassifyEdgeConvexity(e) == EdgeConcave {
		if concave == FillConcaveOutward {
			return cornerInputs{nA: nA.Scale(-1), nB: nB.Scale(-1), offDir: offDir.Scale(-1), flip: true}, nil
		}
		// Inward recess: the ball rolls in the MATERIAL (the convex-formula side). Its tangent points
		// land off the bounded faces unless they extend that way, so it is only valid on geometry that
		// permits it (e.g. a pocket). The explicit realizability gate rejects the impossible case
		// honestly — before it existed the rejection was an ACCIDENT of inconsistent loop winding, which
		// B2's orientFilletShell (fee0da5c) laundered into a Validate-passing self-intersecting solid.
		// A concave edge's natural fillet is the outward fill above.
		if p, ok := concaveInwardRealizable(body, e, nA, nB, offDir, rMid); !ok {
			return cornerInputs{}, fmt.Errorf("fillet: inward recess unrealizable at concave edge — tangent point %v is not material-backed (must lie on a bounded face with material behind and void in front)", p)
		}
		return cornerInputs{nA: nA, nB: nB, offDir: offDir}, nil
	}
	if !PointInsideBody(body, mid.TranslateBy(offDir.Scale(rMid))) {
		return cornerInputs{}, fmt.Errorf("fillet: edge is not convex (only convex edges are supported)")
	}
	return cornerInputs{nA: nA, nB: nB, offDir: offDir}, nil
}

// solvedEdgeFillet assembles the edgeFillet once the edge's frame is known: corners per end
// radius, then either the chord-sampled varying blend or the constant cylinder.
func solvedEdgeFillet(e *topo.Edge, p filletPick, in cornerInputs, blends map[uint64]*cornerBlend, miters map[uint64]*cornerMiter) (edgeFillet, error) {
	c0, c1, err := edgeCorners(e, p, in, blends, miters)
	if err != nil {
		return edgeFillet{}, err
	}
	if p.chordPath() {
		mids := midProfiles(e, in, p.mids, cornerChordCount(in), p.cross, p.rho)
		ef := edgeFillet{a: in.a, b: in.b, c0: c0, c1: c1, mids: mids, edge: e, varying: true, flip: in.flip}
		if w, ok := exactSectionWeight(p, in); ok && plainEnds(c0, c1) {
			// The blend is exactly a rational ruled surface between conic sections (#1606):
			// no chord sampling — corners keep their true arcs (or carry the conic weight for
			// the end trims) and the faces are emitted analytic. G2-quintic cross-sections and
			// miter/blend-terminated ends keep the strip fallback, now diagnostic-flagged.
			ef.exact, ef.secW = true, w
			setBlendShoulders(&ef, in, p)
			return ef, nil
		}
		sampleCornerChords(&ef.c0, &ef.c1, in, p.cross, p.rho)
		return ef, nil
	}
	cyl, err := geom.NewCylinder(c0.cen, in.axis, p.r0)
	if err != nil {
		return edgeFillet{}, err
	}
	// The band must END where the solid does: trim each terminal section against the wall it stops on
	// instead of squaring it off in the section plane at the edge's end vertex (fillet_farend_trim.go).
	trimBandEndsToWalls(&c0, &c1, in)
	return edgeFillet{a: in.a, b: in.b, cyl: cyl, c0: c0, c1: c1, edge: e, flip: in.flip}, nil
}

// edgeCorners solves the rounded corners at both endpoints of an edge (each blended when its
// vertex is a shared corner), with the pick's per-end radius.
func edgeCorners(e *topo.Edge, p filletPick, in cornerInputs, blends map[uint64]*cornerBlend, miters map[uint64]*cornerMiter) (c0, c1 corner, err error) {
	if c0, err = cornerAt(e.StartVertex(), in, p.r0, blends[e.StartVertex().ID()], miters[e.StartVertex().ID()], p.chordPath()); err != nil {
		return corner{}, corner{}, err
	}
	c1, err = cornerAt(e.EndVertex(), in, p.r1, blends[e.EndVertex().ID()], miters[e.EndVertex().ID()], p.chordPath())
	return c0, c1, err
}

// filletChordsPerTurn matches holeFacets' density: chords are sized as if the full circle
// had this many sides, so a 90° wedge gets 8.
const filletChordsPerTurn = 32

// runoutTol is the radius at or below which a variable fillet is treated as a run-out: the blend
// collapses to a single apex on the edge (no end face), so the fillet fades smoothly into the corner.
const runoutTol = 1e-9

// cornerChordCount is the number of chord segments spanning the corner's rolling-ball wedge — sized
// as if the full circle had filletChordsPerTurn sides (a 90° wedge gets 8), with a floor of 4.
func cornerChordCount(in cornerInputs) int {
	wedge := stdmath.Acos(float64(in.nA.Dot(in.nB)))
	k := int(stdmath.Ceil(wedge / (2 * stdmath.Pi / filletChordsPerTurn)))
	if k < 4 {
		k = 4
	}
	return k
}

// validateRadiusPoints checks intermediate fillet radius points are strictly inside the edge
// (0 < T < 1), positive, and strictly increasing in T (so the ruled profiles stay in order).
func validateRadiusPoints(mids []FilletRadiusPoint) error {
	prev := 0.0
	for _, m := range mids {
		if m.T <= 0 || m.T >= 1 {
			return fmt.Errorf("fillet: intermediate radius point T=%g must be strictly between 0 and 1", m.T)
		}
		if m.R <= 0 {
			return fmt.Errorf("fillet: intermediate radius %g must be > 0", m.R)
		}
		if m.T <= prev {
			return fmt.Errorf("fillet: intermediate radius points must be strictly increasing in T (got %g after %g)", m.T, prev)
		}
		prev = m.T
	}
	return nil
}

// midProfiles builds one corner cross-section per intermediate radius point: the rolling-ball circle
// at the interpolated edge point and radius, sampled as chords with the same frame as the end corners
// (#695). They have no end face/blend — they are pure ruling profiles between c0 and c1.
func midProfiles(e *topo.Edge, in cornerInputs, mids []FilletRadiusPoint, k int, cross FilletCrossSection, rho float64) []corner {
	if len(mids) == 0 {
		return nil
	}
	p0, p1 := e.StartVertex().Point(), e.EndVertex().Point()
	span := p0.VectorTo(p1)
	out := make([]corner, 0, len(mids))
	for _, m := range mids {
		p := p0.TranslateBy(span.Scale(m.T))
		cen := p.TranslateBy(in.offDir.Scale(m.R))
		c := corner{a: in.a, b: in.b, cen: cen, ta: cen.TranslateBy(in.nA.Scale(m.R)), tb: cen.TranslateBy(in.nB.Scale(m.R))}
		c.mid = cen.TranslateBy(slerpVec(in.nA, in.nB, 0.5).Scale(m.R))
		c.chords = crossSectionChords(c, in, k, cross, rho)
		out = append(out, c)
	}
	return out
}

// sampleCornerChords samples both corners' cross-section profiles at the same stations, so chord j of
// one corner pairs with chord j of the other as a straight ruling of the blend band.
func sampleCornerChords(c0, c1 *corner, in cornerInputs, cross FilletCrossSection, rho float64) {
	k := cornerChordCount(in)
	c0.chords = crossSectionChords(*c0, in, k, cross, rho)
	c1.chords = crossSectionChords(*c1, in, k, cross, rho)
}

// crossSectionChords samples a corner's cross-section ta…tb into k+1 points for the requested shape
// (M36-F08): the circular arc (G1), a curvature-continuous G2 quintic, or a rho-controlled conic.
func crossSectionChords(c corner, in cornerInputs, k int, cross FilletCrossSection, rho float64) []math.Point3 {
	switch cross {
	case FilletG2:
		return g2Chords(c, in, k)
	case FilletConic:
		return conicChords(c, in, k, rho)
	default:
		return arcChords(c, in, k)
	}
}

// arcChords samples a corner's arc ta…tb as k+1 points: cen + r·slerp(nA→nB), the exact
// rolling-ball contact directions at evenly spaced stations.
func arcChords(c corner, in cornerInputs, k int) []math.Point3 {
	r := c.cen.DistanceTo(c.ta)
	out := make([]math.Point3, k+1)
	for j := 0; j <= k; j++ {
		dir := slerpVec(in.nA, in.nB, float64(j)/float64(k))
		out[j] = c.cen.TranslateBy(dir.Scale(r))
	}
	return out
}

// shoulder is the sharp-corner point where the two walls' tangent lines (at ta along wall A, at tb
// along wall B) meet — cen + r·(nA+nB)/(1+nA·nB) — the apex a conic/G2 cross-section pulls toward.
func shoulder(c corner, in cornerInputs) math.Point3 {
	r := c.cen.DistanceTo(c.ta)
	cdot := in.nA.Dot(in.nB)
	return c.cen.TranslateBy(in.nA.Add(in.nB).Scale(r / (1 + cdot)))
}

// exactSectionWeight returns the shoulder weight of the pick's cross-section when it is a
// rational quadratic — an ARC (w = cos of the half wedge, since cos(wedge) = nA·nB) or a rho
// CONIC (w = rho/(1−rho)) — and ok=false for the G2 quintic, which is not (#1606).
func exactSectionWeight(p filletPick, in cornerInputs) (float64, bool) {
	switch {
	case p.cross.IsArc():
		return stdmath.Sqrt((1 + in.nA.Dot(in.nB)) / 2), true // tol-free: dihedral wedge < π keeps this > 0
	case p.cross == FilletConic:
		rho := p.rho
		if rho <= 0 || rho >= 1 {
			rho = 0.5
		}
		return rho / (1 - rho), true
	default:
		return 0, false
	}
}

// plainEnds reports whether both corners terminate against plain end faces or run-outs — the
// configurations the exact ruled blend covers; miter seams and corner-sphere blends keep the
// chord strips (their shared boundaries are chord polylines).
func plainEnds(c0, c1 corner) bool {
	return !c0.miter && !c0.blend && !c1.miter && !c1.blend
}

// setBlendShoulders stamps every profile's shoulder control point (and, for a conic
// cross-section, the weight its end trim must carry) on the exact blend's corners.
func setBlendShoulders(ef *edgeFillet, in cornerInputs, p filletPick) {
	conicW := 0.0
	if p.cross == FilletConic {
		conicW = ef.secW
	}
	stamp := func(c *corner) {
		c.sh = shoulder(*c, in)
		c.crossW = conicW
	}
	stamp(&ef.c0)
	stamp(&ef.c1)
	for i := range ef.mids {
		stamp(&ef.mids[i])
	}
}

// conicChords samples a rho-controlled conic cross-section (rational quadratic Bézier ta–S–tb) into
// k+1 points. The shoulder weight w follows the projective discriminant rho = w/(1+w): rho=0.5 ⇒ w=1
// (parabola), rho<0.5 flatter, rho>0.5 fuller. rho≤0 or ≥1 falls back to the parabola.
func conicChords(c corner, in cornerInputs, k int, rho float64) []math.Point3 {
	s := shoulder(c, in)
	if rho <= 0 || rho >= 1 {
		rho = 0.5
	}
	w := rho / (1 - rho) // shoulder weight
	out := make([]math.Point3, k+1)
	for j := 0; j <= k; j++ {
		out[j] = rationalQuad(c.ta, s, c.tb, w, float64(j)/float64(k))
	}
	return out
}

// rationalQuad evaluates the rational quadratic Bézier with end weights 1 and shoulder weight w at t.
func rationalQuad(p0, p1, p2 math.Point3, w, t float64) math.Point3 {
	b0 := (1 - t) * (1 - t)
	b1 := 2 * (1 - t) * t * w
	b2 := t * t
	den := b0 + b1 + b2
	x := (b0*float64(p0.X) + b1*float64(p1.X) + b2*float64(p2.X)) / den
	y := (b0*float64(p0.Y) + b1*float64(p1.Y) + b2*float64(p2.Y)) / den
	z := (b0*float64(p0.Z) + b1*float64(p1.Z) + b2*float64(p2.Z)) / den
	return math.P3(math.Scalar(x), math.Scalar(y), math.Scalar(z))
}

// g2Chords samples a curvature-continuous (G2) cross-section into k+1 points. It is a quintic Bézier
// whose first three control points are collinear along wall A's tangent (ta→shoulder) and last three
// along wall B's (shoulder→tb), so the profile's curvature is ZERO at both tangency lines — matching
// the flat walls' zero curvature, i.e. no curvature jump where the blend meets them.
func g2Chords(c corner, in cornerInputs, k int) []math.Point3 {
	s := shoulder(c, in)
	ctrl := [6]math.Point3{
		c.ta, c.ta.Lerp(s, 1.0/3), c.ta.Lerp(s, 2.0/3),
		s.Lerp(c.tb, 1.0/3), s.Lerp(c.tb, 2.0/3), c.tb,
	}
	out := make([]math.Point3, k+1)
	for j := 0; j <= k; j++ {
		out[j] = bezier5(ctrl, float64(j)/float64(k))
	}
	return out
}

// bezier5 evaluates a quintic Bézier via de Casteljau.
func bezier5(ctrl [6]math.Point3, t float64) math.Point3 {
	p := ctrl
	pts := p[:]
	for n := 5; n > 0; n-- {
		for i := 0; i < n; i++ {
			pts[i] = pts[i].Lerp(pts[i+1], t)
		}
	}
	return pts[0]
}

// edgePlanarFaces returns the edge's two faces and their outward normals, erroring unless
// the edge bounds exactly two planar faces.
func edgePlanarFaces(e *topo.Edge) (a, b *topo.Face, nA, nB math.Vector3, err error) {
	faces := e.Faces()
	if len(faces) != 2 {
		return nil, nil, nA, nB, fmt.Errorf("fillet: edge bounds %d faces, need 2", len(faces))
	}
	pa, oka := faces[0].Geometry().(geom.Plane)
	pb, okb := faces[1].Geometry().(geom.Plane)
	if !oka || !okb {
		return nil, nil, nA, nB, fmt.Errorf("fillet: both faces of the edge must be planar")
	}
	// Material-OUTWARD normals: a plane's geometric normal negated when its face is reversed.
	// Native construction leaves faces unreversed with outward plane normals, but STEP-imported
	// (and any oriented) faces carry a Reversed flag with an inward plane normal. Ignoring it
	// flips offDir outward, so the rolling-ball centre lands outside and a plainly convex edge
	// reads as non-convex — filleting every imported solid failed until this was applied.
	return faces[0], faces[1], outwardPlaneNormal(faces[0], pa), outwardPlaneNormal(faces[1], pb), nil
}

// outwardPlaneNormal is a planar face's material-outward normal (its plane normal, negated
// when the face is reversed) — matching outwardFaceNormal's orientation handling.
func outwardPlaneNormal(f *topo.Face, p geom.Plane) math.Vector3 {
	if f.Reversed() {
		return p.Normal().Negate()
	}
	return p.Normal()
}

// cornerInputs bundles the per-edge data a corner needs. offDir is the centre offset from
// the edge into the solid PER UNIT RADIUS (a variable fillet's centre line follows offDir
// scaled by the local radius).
type cornerInputs struct {
	a, b   *topo.Face
	nA, nB math.Vector3
	offDir math.Vector3
	axis   math.Vector3
	flip   bool    // invert the cylinder face's outward sense (a concave fillet's surface faces the centre)
	weld   float64 // model-relative coincidence tolerance for the F3a spine-concurrence gate (armCornerCentre)
}

// cornerAt solves a fillet corner at vertex v with the local radius r. Without a blend it is
// a simple end: centre v+offDir·r, tangent points r along each face normal, an arc on the end
// face. With a blend (v is a shared corner) the centre is the blend sphere's centre and the
// tangent points are the sphere's tangents on the two faces; the corner-end arc joins the
// sphere patch (no end face), and the arc is registered on the blend.
func cornerAt(v *topo.Vertex, in cornerInputs, r float64, blend *cornerBlend, miter *cornerMiter, variable bool) (corner, error) {
	if r <= runoutTol { // a variable fillet tapered to 0: the blend collapses to an apex on the edge here
		p := v.Point()
		return corner{a: in.a, b: in.b, vertex: v, cen: p, ta: p, tb: p, mid: p, runout: true}, nil
	}
	cen, ta, tb, arcCen, end, seam, err := cornerTangents(v, in, r, blend, miter)
	if err != nil {
		return corner{}, err
	}
	// mid uses arcCen (the blend ball for a shared corner) so the sphere-patch arc registers ON the sphere
	// even when the arm SURFACE centre was kept frame-derived by the F3a spine-concurrence gate (armCornerCentre).
	mid := arcCen.TranslateBy(perpToward(arcCen, v.Point(), in.axis).Scale(r))
	c := corner{a: in.a, b: in.b, endFace: end, vertex: v, cen: cen, ta: ta, tb: tb, mid: mid, blend: blend != nil, miter: miter != nil, seam: seam}
	registerBlendArc(blend, c, in, variable)
	return c, nil
}

// cornerTangents resolves a corner's arm-surface centre (cen), the two face tangent points, the sphere-patch
// arc centre (arcCen), and the end face / miter seam, by corner kind: a miter seam, a blend ball (whose
// override is gated on spine concurrence via armCornerCentre, F3a), or a plain end-face round. Split out of
// cornerAt to keep it within funlen; it errors only on a plain corner with no end face to round.
func cornerTangents(v *topo.Vertex, in cornerInputs, r float64, blend *cornerBlend, miter *cornerMiter) (cen, ta, tb, arcCen math.Point3, end *topo.Face, seam []math.Point3, err error) {
	cen = v.Point().TranslateBy(in.offDir.Scale(r)) // the frame-derived rolling-ball centre, on the arm axis
	ta = cen.TranslateBy(in.nA.Scale(r))
	tb = cen.TranslateBy(in.nB.Scale(r))
	arcCen = cen // the sphere-patch arc's centre: the blend ball for a shared (concurrent OR canal) corner
	switch {
	case miter != nil:
		ta, tb, seam = miterTangents(in, miter) // the end is the seam, not an end-face arc
	case blend != nil:
		cen, ta, tb, arcCen = armCornerCentre(cen, in, blend), blend.tan[in.a.ID()], blend.tan[in.b.ID()], blend.center
	default:
		if end = endFaceAt(v, in.a, in.b); end == nil {
			return cen, ta, tb, arcCen, nil, nil, fmt.Errorf("fillet: edge endpoint has no end face to round")
		}
	}
	return cen, ta, tb, arcCen, end, seam, nil
}

// armCornerCentre returns the ARM SURFACE's rolling-ball centre at a shared (blend) corner, gating the
// blend-ball override on SPINE CONCURRENCE (F3a). The arm's offset spine is the LINE through the
// frame-derived centre along the edge axis; the blend ball sits ON that line — but OFFSET by the setback
// distance ALONG the axis, so the test is the ball's PERPENDICULAR distance to the spine line, not the raw
// point distance (which the setback would always overshoot). It adopts the blend ball only when that
// perpendicular gap ≤ in.weld. Where the blend ball lies on THIS arm's spine (perp ≈ 0 — every planar
// box/round/setback corner, and the arms of a partly-concurrent corner) this adopts the blend ball,
// byte-identical to the pre-F3a override. Where it does not — the s_10 canal arm, whose ball is 10 off its
// x=55 spine — the frame-derived centre is kept, so the arm cylinder is NOT built on the mirrored x=45
// side; the corner-blend arc still registers on the ball (blend.center/mid), decoupled from this centre.
// (Note: a corner may be concurrent for some arms and not others — L6 adopts on 1 of its 3 arms and keeps
// frame on the other 2; its byte-identity across F3a comes from that per-arm split + the curved-corner
// machinery rebuilding the kept-frame arms, NOT from a single ball lying on every arm's spine.)
func armCornerCentre(frame math.Point3, in cornerInputs, blend *cornerBlend) math.Point3 {
	d := frame.VectorTo(blend.center)
	perp := d.Sub(in.axis.Scale(d.Dot(in.axis))) // component of ball→spine offset ⊥ to the edge axis
	if perp.Length() <= in.weld {
		return blend.center
	}
	return frame
}

// registerBlendArc records the corner's boundary arc on the sphere patch when v is a blend corner. A
// variable edge stores the arc as the cone's chord polyline so the patch and cone meet edge-for-edge.
func registerBlendArc(blend *cornerBlend, c corner, in cornerInputs, variable bool) {
	if blend == nil {
		return
	}
	arc := blendArc{ta: c.ta, tb: c.tb, mid: c.mid}
	if variable {
		arc.chords = arcChords(c, in, cornerChordCount(in))
	}
	blend.arcs = append(blend.arcs, arc)
}

// miterTangents returns this edge's corner tangents and the seam oriented ta→tb: the shared
// face carries the seam's top (sTop, on the shared face), the outer face carries its bottom
// (sBot, on the now-shortened sharp edge). The seam is the SAME point list for both edges of
// the miter — reversed for the edge whose A face is the outer one — so the two cylinders weld
// along it watertight.
func miterTangents(in cornerInputs, m *cornerMiter) (ta, tb math.Point3, seam []math.Point3) {
	if m.shared == nil {
		return vertexOnlyMiterTangents(in, m) // D4: no shared face — orient by which sharp edge bounds in.a
	}
	if in.a == m.shared {
		return m.seam[0], m.sBot, m.seam
	}
	return m.sBot, m.seam[0], reversePoints(m.seam)
}

// reversePoints returns a reversed copy of pts.
func reversePoints(pts []math.Point3) []math.Point3 {
	out := make([]math.Point3, len(pts))
	for i, p := range pts {
		out[len(pts)-1-i] = p
	}
	return out
}

// perpToward returns the unit direction from cen toward p projected into the plane
// perpendicular to axis — the in-cross-section direction to the rounded corner.
func perpToward(cen, p math.Point3, axis math.Vector3) math.Vector3 {
	d := cen.VectorTo(p)
	perp := d.Sub(axis.Scale(d.Dot(axis)))
	u, err := math.UnitVector3FromVector(perp)
	if err != nil {
		return d
	}
	return u.AsVector()
}

// endFaceAt returns the face meeting at v that is neither a nor b (the end cap the fillet
// rounds), or nil if there is none.
func endFaceAt(v *topo.Vertex, a, b *topo.Face) *topo.Face {
	for _, e := range v.Edges() {
		for _, f := range e.Faces() {
			if f != a && f != b {
				return f
			}
		}
	}
	return nil
}

// blendArc is one boundary arc of a corner sphere patch (shared with a cylinder fillet). chords is
// nil for an analytic single arc (a constant cylinder), or the chord polyline ta…tb when the arc is
// shared with a VARIABLE cone, whose faceted end must match the patch edge-for-edge to stay watertight.
type blendArc struct {
	ta, tb, mid math.Point3
	chords      []math.Point3
}

// cornerBlend is a spherical corner patch where several filleted edges meet at one vertex:
// the rolling-ball sphere tangent to the corner's faces, its tangent point on each face
// (keyed by face id), and the arcs (filled in as the edges are solved) that bound the patch.
type cornerBlend struct {
	vertex *topo.Vertex
	center math.Point3
	sphere geom.Sphere
	tan    map[uint64]math.Point3
	arcs   []blendArc
	// radiusTorus marks a mixed-radius trihedral TORUS corner (A4: rB on the wall∧wall edge, equal
	// rS on the two top edges): the bands build against this transient blend and the unified setback
	// pass retracts them onto the solved torus (fillet_corner_radiustorus.go). Nil everywhere else,
	// so every equal-radius corner path is untouched.
	radiusTorus *radiusTorusCornerGeom
}

// computeCorners finds the shared corners of the filleted edge set and solves a corner
// treatment for each, keyed by corner vertex id:
//
//   - three filleted edges at a trihedral (3-face) vertex → a spherical corner patch (blend);
//   - two filleted edges that share a face, the third edge staying sharp → a miter seam where
//     the two rolling-ball cylinders mutually trim (miter).
//
// Edges meeting at a corner may carry DIFFERENT radii: an equal-radius corner solves the sphere
// blend / mirror-plane miter seam (solveCorner's uniform path), a mixed-radius corner (P9/V9 miter,
// A4 trihedral) routes to solveAsymmetricCorner. A variable edge's faceted end chords still cannot
// meet a corner watertight, so a varying pick at a mixed corner is rejected there.
func computeCorners(body *topo.Body, picks []filletPick) (map[uint64]*cornerBlend, map[uint64]*cornerMiter, error) {
	groups := map[uint64][]filletPick{}
	for _, p := range picks {
		groups[p.edge.StartVertex().ID()] = append(groups[p.edge.StartVertex().ID()], p)
		groups[p.edge.EndVertex().ID()] = append(groups[p.edge.EndVertex().ID()], p)
	}
	blends := map[uint64]*cornerBlend{}
	miters := map[uint64]*cornerMiter{}
	for vid, ps := range groups {
		if len(ps) < 2 {
			continue
		}
		cb, cm, err := solveCorner(body, vid, ps)
		if err != nil {
			return nil, nil, err
		}
		if cb != nil {
			blends[vid] = cb
		}
		if cm != nil {
			miters[vid] = cm
		}
	}
	return blends, miters, nil
}

// solveCorner solves the corner treatment at vertex vid where the picks ps meet: a sphere blend
// (3 edges, trihedral vertex) or a miter seam (2 edges sharing a face), at the corner's one shared
// radius. Exactly one of (blend, miter) is returned; any other configuration errors.
func solveCorner(body *topo.Body, vid uint64, ps []filletPick) (*cornerBlend, *cornerMiter, error) {
	if !uniformCornerRadii(vid, ps) {
		// Edges meeting at a shared corner with DIFFERENT radii (P9/V9: a box top corner, r1
		// on one arm, r0.5 on the other; A4: a trihedral corner, r10/r5/r5): the equal-radius
		// sphere/mirror-seam no longer applies — each arm keeps its own rolling-ball radius and
		// the corner reconciles them (the true cyl∩cyl seam for a 2-edge miter, a torus patch for
		// a trihedral). Routed off the byte-identical equal-radius path so that path is untouched.
		return solveAsymmetricCorner(body, vid, ps)
	}
	r, err := cornerRadius(vid, ps)
	if err != nil {
		return nil, nil, err
	}
	v := vertexByID(edgesOf(ps), vid)
	faces := facesAtVertex(v)
	switch {
	case len(ps) == 3 && len(faces) == 3:
		cb, err := solveBlend(body, v, faces, r)
		return cb, nil, err
	case len(ps) == 2:
		return solveTwoEdgeCorner(v, ps, r)
	case len(ps) >= 4 && len(faces) == len(ps):
		// Full-round K-arm corner (X8/A1): every edge at a K-valent planar vertex filleted at one
		// radius, closed by the exact common-tangent-sphere K-gon (fillet_corner_fullround.go).
		cb, err := solveFullRoundCorner(body, v, faces, ps, r)
		return cb, nil, err
	default:
		return nil, nil, fmt.Errorf("fillet: corner where %d filleted edges meet a %d-face vertex is not a supported blend (need 3 edges at a trihedral vertex, or 2 edges sharing a face)", len(ps), len(faces))
	}
}

// solveTwoEdgeCorner dispatches the 2-edge corner: a CLOSED rim takes no corner treatment, a
// variable-radius edge cannot miter, and the miter itself is either the ordinary shared-face seam
// or — when the two edges share NO face (D4: opposite pyramid arms) — the vertex-only seam whose
// ends lie on the corner's surviving sharp edges instead.
func solveTwoEdgeCorner(v *topo.Vertex, ps []filletPick, r float64) (*cornerBlend, *cornerMiter, error) {
	if closedRimPick(ps) {
		return nil, nil, nil // a CLOSED rim (one edge counted twice: StartVertex==EndVertex) is not a
		// 2-edge miter corner — it has no second edge and no shared face, so it takes no corner treatment
		// and reaches the closed-band arm assembly (fillet_curved_closed_rim.go) instead of solveMiter (J1).
	}
	if p := varyingPick(ps); p != nil {
		return nil, nil, fmt.Errorf("fillet: a variable-radius edge (radii %g→%g) cannot share a 2-edge miter corner (its cone has no seam with a cylinder); round the third edge for a setback instead", p.r0, p.r1)
	}
	if sharedFace(ps[0].edge, ps[1].edge) == nil {
		// D4: two filleted edges meeting ONLY at the vertex (opposite pyramid arms). The rolling-
		// ball cylinders still mutually trim — the seam is cylA∩cylB with both ends on the corner's
		// sharp edges (fillet_miter_vertexonly.go), matching DRAWEXE's D4 band boundary exactly.
		cm, err := solveVertexOnlyMiter(v, ps, r)
		return nil, cm, err
	}
	cm, err := solveMiter(v, ps, r)
	return nil, cm, err
}

// cornerRadius returns the radius every pick carries AT the shared corner vertex vid — the radius of
// the corner sphere. A variable edge is allowed (e.g. a setback's tapered third edge) as long as its
// radius at this corner matches the others; only the far ends may differ. Reached only on the
// equal-radius path (solveCorner routes a mixed-radius corner to solveAsymmetricCorner first), so the
// mismatch error is now a defensive backstop rather than the primary guard.
func cornerRadius(vid uint64, ps []filletPick) (float64, error) {
	r := radiusAtVertex(ps[0], vid)
	for _, p := range ps {
		if rv := radiusAtVertex(p, vid); rv != r {
			return 0, fmt.Errorf("fillet: edges meeting at a shared corner must use one radius there (got %g and %g)", r, rv)
		}
	}
	return r, nil
}

// uniformCornerRadii reports whether every pick carries the SAME radius at the shared corner vid. The
// equal-radius corner treatments (the sphere blend, the mirror-plane miter seam) require it; a corner
// whose arms differ (P9/V9, A4) takes the asymmetric path. This is exactly cornerRadius's old guard
// condition, split out so the equal-radius path stays byte-identical to before the asymmetric work.
func uniformCornerRadii(vid uint64, ps []filletPick) bool {
	r := radiusAtVertex(ps[0], vid)
	for _, p := range ps {
		if radiusAtVertex(p, vid) != r {
			return false
		}
	}
	return true
}

// closedRimPick reports whether the two picks grouped at a corner vertex are the SAME closed edge
// counted twice — a full-circle rim whose StartVertex==EndVertex lands its single pick in the
// seam-vertex group at both endpoints. That is NOT a 2-edge miter (no second edge, no shared face),
// so solveCorner returns no corner treatment and the rim reaches the closed-band arm assembly (J1).
func closedRimPick(ps []filletPick) bool {
	return len(ps) == 2 && ps[0].edge.ID() == ps[1].edge.ID() &&
		ps[0].edge.StartVertex().ID() == ps[0].edge.EndVertex().ID()
}

// varyingPick returns the first pick whose radius varies along the edge, or nil if all are constant.
func varyingPick(ps []filletPick) *filletPick {
	for i := range ps {
		if ps[i].varying() {
			return &ps[i]
		}
	}
	return nil
}

// radiusAtVertex returns the pick's radius at the endpoint vid (r0 at the start vertex, else r1).
func radiusAtVertex(p filletPick, vid uint64) float64 {
	if p.edge.StartVertex().ID() == vid {
		return p.r0
	}
	return p.r1
}

// edgesOf projects the picks' edges.
func edgesOf(ps []filletPick) []*topo.Edge {
	out := make([]*topo.Edge, len(ps))
	for i, p := range ps {
		out[i] = p.edge
	}
	return out
}

// solveBlend builds the corner sphere at the trihedral vertex v. An all-planar corner solves the
// sphere equidistant (r) from the three planes (the historical path, byte-identical below). A corner
// whose host set is ONE cylinder and two planes (M5 Slice A: a curved-rim boss corner) instead solves
// the ball tangent to the cylinder and the two planes — an analytic geom.Sphere of the same radius r,
// matching OCCT's equal-radius corner KPart (BREP surface code 4). A ONE sphere + two planes host set
// solves the same analytic corner via sphereHostCorner (SP2). Any other host mix (two curved faces, a
// cone/torus host, or ≥2 curved faces) still returns "corner face must be planar" (do-no-harm).
func solveBlend(body *topo.Body, v *topo.Vertex, faces []*topo.Face, r float64) (*cornerBlend, error) {
	if cyl, planes, ok := cylinderHostCorner(faces); ok {
		return solveCurvedBlend(body, v, faces, cyl, planes, r) // curved rim: analytic sphere corner
	}
	// SP2: a sphere host + two planes solves the analytic sphere corner too (sphere-host campaign).
	// Ordered AFTER the untouched cylinderHostCorner and before solvePlanarBlend, so the cylinder (M5)
	// and all-planar paths stay byte-identical (unreachable-by-construction for non-sphere hosts).
	if sph, sphereFace, planes, ok := sphereHostCorner(faces); ok {
		return solveSphereBlend(v, faces, sph, sphereFace, planes, r)
	}
	// CN3: a cone host + two planes solves the analytic sphere corner too (cone-host campaign). Ordered
	// AFTER sphereHostCorner and before solvePlanarBlend, so every non-cone corner stays byte-identical.
	if co, coneFace, planes, ok := coneHostCorner(faces); ok {
		return solveConeBlend(v, faces, co, coneFace, planes, r)
	}
	return solvePlanarBlend(v, faces, r)
}

// solvePlanarBlend builds the corner sphere from the three planar faces meeting at v (the point
// at distance r from all three, inside) and its tangent points on each.
func solvePlanarBlend(v *topo.Vertex, faces []*topo.Face, r float64) (*cornerBlend, error) {
	var a [3][3]float64
	var b [3]float64
	for i, f := range faces {
		pl, ok := f.Geometry().(geom.Plane)
		if !ok {
			return nil, fmt.Errorf("fillet: corner face must be planar")
		}
		// Material-OUTWARD normal (respects face.Reversed()): the centre sits r INSIDE each face,
		// n·s = n·origin − r, which only holds when n points outward. A raw plane normal on a
		// reversed (imported) face solves for a sphere on the wrong side (same defect as the miter).
		n := outwardPlaneNormal(f, pl)
		a[i] = [3]float64{n.X, n.Y, n.Z}
		b[i] = n.Dot(pl.Origin.AsVector()) - r // distance r on the inside of each face
	}
	x, ok := solve3(a, b)
	if !ok {
		return nil, fmt.Errorf("fillet: cannot solve corner blend sphere (degenerate faces)")
	}
	s := math.P3(x[0], x[1], x[2])
	sph, err := geom.NewSphere(s, r)
	if err != nil {
		return nil, err
	}
	tan := make(map[uint64]math.Point3, 3)
	for _, f := range faces {
		// Tangent point is the sphere centre pushed r along the OUTWARD normal to reach the face.
		tan[f.ID()] = s.TranslateBy(outwardPlaneNormal(f, f.Geometry().(geom.Plane)).Scale(r))
	}
	return &cornerBlend{vertex: v, center: s, sphere: sph, tan: tan}, nil
}

// vertexByID returns the vertex with id vid from the edge set.
func vertexByID(edges []*topo.Edge, vid uint64) *topo.Vertex {
	for _, e := range edges {
		if e.StartVertex().ID() == vid {
			return e.StartVertex()
		}
		if e.EndVertex().ID() == vid {
			return e.EndVertex()
		}
	}
	return nil
}

// facesAtVertex returns the distinct faces meeting at v.
func facesAtVertex(v *topo.Vertex) []*topo.Face {
	seen := map[uint64]bool{}
	var out []*topo.Face
	for _, e := range v.Edges() {
		for _, f := range e.Faces() {
			if !seen[f.ID()] {
				seen[f.ID()] = true
				out = append(out, f)
			}
		}
	}
	return out
}
