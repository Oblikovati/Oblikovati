// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	"oblikovati.org/api/types"
	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/topo"
)

// The fillet's PUBLIC surface: the option types a caller fills in and the four entry points.
// Everything below the entry points was 1,272 lines in this file; #2217 split it by
// responsibility, in pipeline order:
//
//	fillet_pipeline.go       picks -> corners -> per-edge blends -> assembled body
//	fillet_pick.go           resolving edge keys into picks; the pick and edgeFillet records
//	fillet_corner_solve.go   which vertices become blends and which become miters
//	fillet_corner_frame.go   the frame one corner is solved in, and its blend/miter records
//	fillet_edge_solve.go     one pick -> one edgeFillet
//	fillet_section_chords.go the cross-section profile, sampled into chords
//	fillet_assemble.go       welding solved blend faces back onto the host body
//
// The engine these call into is kernel/blend (ADR-0050). New blend work belongs THERE, not in a
// new fillet_*.go here — see ADR-0050 "Strangler status" for what deletes this catalog.

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
