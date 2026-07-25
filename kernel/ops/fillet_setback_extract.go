// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// extractSetbackPatches turns Task 2's setbackBands (the D2 partition of a runout edge's interfered
// span) into one valence-4 RailLoop per band — the setback-patch topology OCCT ships here, keeping
// each boss wall INTACT (setback-patch-derivation.md D3, "Candidate method ii").
//
// Each loop carries a RunoutCanal payload: the EXACT rolling-ball stations of its band, so
// runoutCanalProvider skins the true envelope instead of a Coons fill through the same four rails
// (coons4-audit.md measured that fill 9–19% of r off OCCT's own surface with the rails already right
// to 1e-14). The bands are the PARTITION the rolling ball actually traces: the two flanks (and the
// one-boss central) are SURF-RST — the ball is tangent to the plain host plane and passes THROUGH the
// blocking boss's footprint conic — while the two-boss central is RST-RST, passing through both
// footprints and touching no plane at all.
//
// It tiles the 2-boss S1 shape (left flank / central / right flank → 3 loops) and the 1-boss shape
// (single central run-out, #2007). N>2 bosses (2N−1 bands) is a later task — this milestone
// honest-rejects (ok=false) anything else, and any loop that will not close or whose stations will not
// solve. WIRED: runoutFacesFor (fillet_runout_faces.go) calls this on every constant-radius edge with a
// detected setback band; it greens the corpus's S1/S4/T1/T4/T7/S7 (two-boss) and S6/S9/T3 (one-boss).
func extractSetbackPatches(b setbackBands, ef edgeFillet, res Resolution) ([]RailLoop, bool) {
	t, ok := resolveSetbackTiling(b, ef, res)
	if !ok {
		return nil, false
	}
	return buildSetbackLoops(setbackLoopBuilders(b, t), res)
}

// setbackLoopBuilders selects the band-loop builders for the tiling: the SINGLE central run-out for a
// one-boss body (S6/S9/T3, #2007) or the S1 three-loop flank/central/flank tiling for the two-boss body.
// The one-boss central is the degenerate S1 shape — one footprint rail (the intact boss wall) closed off
// by an arm arc at EACH cut station and the tangency contact locus, instead of two flanks joined by
// internal seams.
func setbackLoopBuilders(b setbackBands, t setbackTiling) []func() (RailLoop, bool) {
	if len(b.bosses) == 1 {
		return []func() (RailLoop, bool){t.singleCentral}
	}
	return []func() (RailLoop, bool){t.leftFlank, t.centralBand, t.rightFlank}
}

// buildSetbackLoops runs each band builder and certifies it is a closed valence-4 RailLoop (the shape
// resolveBlend accepts), honest-rejecting the WHOLE edge if any loop fails — never a partial tiling.
func buildSetbackLoops(builders []func() (RailLoop, bool), res Resolution) ([]RailLoop, bool) {
	loops := make([]RailLoop, 0, len(builders))
	for _, build := range builders {
		loop, ok := build()
		if !ok || loop.Valence() != 4 || !loop.Closed(res.Weld()) {
			return nil, false
		}
		loops = append(loops, loop)
	}
	return loops, true
}

// setbackTiling is extractSetbackPatches' resolved geometry: the fillet cylinder and its run-out
// envelope frame, the two host planes (pOuter carries the outer boss the flanks reach, pInner carries
// the inner boss + the flanks' tangency contact), the two bosses, the four ABSOLUTE spine stations,
// the eight shared band corners, and the resolved bands. Every corner is computed ONCE so a corner
// shared between the central patch and a flank is bit-identical, and the two seam cross-sections are
// built ONCE and handed to both loops, so the tiling is watertight by identity.
type setbackTiling struct {
	cyl            geom.Cylinder
	env            runoutEnvelope
	weld           float64
	pOuter, pInner geom.Plane
	outer, inner   crossingBoss
	cutLo, cutHi   float64 // outer boss's setback stations (the plain-fillet ends)
	seamLo, seamHi float64 // flank/central boundaries, ascending (the φ(s)=0 roots, see seamStation)
	// aX = fillet∩pOuter or the outer footprint point; bX = the run-out tangency contact on pInner.
	aCutLo, bCutLo, aSeamLo, bSeamLo math.Point3
	aCutHi, bCutHi, aSeamHi, bSeamHi math.Point3
	// left/mid/right are the resolved bands (left/right nil in the one-boss tiling, where `mid` is the
	// single surf-rst run-out spanning the whole freed span).
	left, mid, right *runoutBand
	// seamLoArc/seamHiArc are the exact cross-section arcs at the seam stations — the rail a flank and
	// the central patch SHARE (one traced each way from the same object).
	seamLoArc, seamHiArc geom.Arc3d
}

// resolveSetbackTiling classifies the two-boss dual-host shape, solves the seam stations, and computes
// the shared corners + bands. ok=false for any other configuration (this milestone tiles the S1 shape
// only) or any band whose exact stations will not solve.
func resolveSetbackTiling(b setbackBands, ef edgeFillet, res Resolution) (setbackTiling, bool) {
	if len(b.bosses) == 1 && len(b.seams) == 0 {
		return resolveSingleBossTiling(b, ef, res) // one boss → 2 wings + one central run-out (#2007)
	}
	if len(b.bosses) != 2 || len(b.seams) != 2 {
		return setbackTiling{}, false // 2-boss S1 shape only (2N−1 general bands: a later task)
	}
	pOuter, pInner, ok := setbackHostPlanes(b.bosses[0], b.bosses[1])
	if !ok {
		return setbackTiling{}, false
	}
	t := setbackTiling{
		cyl: ef.cyl, env: newRunoutEnvelope(ef.cyl), weld: res.Weld(),
		pOuter: pOuter, pInner: pInner, outer: b.bosses[0], inner: b.bosses[1],
		cutLo: b.cutLo, cutHi: b.cutHi,
	}
	if !t.resolveSeamStations() || !t.resolveCorners() || !t.resolveTwoBossBands() {
		return setbackTiling{}, false
	}
	return t, true
}

// resolveSeamStations replaces Task 2's mirror-ordered `seams` (where the inner footprint crosses the
// PLAIN fillet contact line) with the true band boundary φ(s)=0 — where the SURF-RST contact locus on
// pInner reaches the inner footprint (seamStation). Measured against DRAWEXE the two differ by 32% on
// S1 (±4.4721 vs OCCT's ±3.38093) and 56% on S4 (±7.141 vs ±4.56723), so this is a defect in its own
// right, independent of the fill. ok=false when either bracket does not straddle.
func (t *setbackTiling) resolveSeamStations() bool {
	mid := 0.5 * (t.cutLo + t.cutHi)
	hi, okHi := seamStation(t.env, t.pOuter, t.pInner, t.outer, t.inner, mid, t.cutHi, t.weld)
	lo, okLo := seamStation(t.env, t.pOuter, t.pInner, t.outer, t.inner, mid, t.cutLo, t.weld)
	t.seamLo, t.seamHi = stdmath.Min(lo, hi), stdmath.Max(lo, hi)
	return okHi && okLo
}

// setbackHostPlanes reads the two host support planes: pOuter carries the outer boss (bosses[0], which
// both flanks and the central patch run out to on the A side), pInner carries the inner boss (central
// B side) and the flanks' tangency contact. ok=false when the two bosses share one host (not the S1
// dual-host shape) or a host is not planar.
func setbackHostPlanes(outer, inner crossingBoss) (geom.Plane, geom.Plane, bool) {
	if outer.host == inner.host {
		return geom.Plane{}, geom.Plane{}, false // both bosses on one plane: not this milestone's shape
	}
	po, ok0 := outer.host.Geometry().(geom.Plane)
	pi, ok1 := inner.host.Geometry().(geom.Plane)
	return po, pi, ok0 && ok1
}

// resolveCorners fills the eight band corners: the fillet cross-section contacts on each plane at the
// two CUT stations (filletContact — at a cut the run-out ball IS the plain fillet ball, t=0, so these
// are exact and the wings weld to them unchanged), the outer-footprint points at the seam stations, and
// the two bSeam corners as the SURF-RST tangency contacts on pInner. The bSeam corners lie ON the inner
// footprint by construction — that IS the seam condition φ(s)=0 — so they still double as the inner
// footprint endpoints the central patch and the inner wall rim share.
func (t *setbackTiling) resolveCorners() bool {
	t.aCutHi, t.bCutHi = filletContact(t.cyl, t.pOuter, t.cutHi), filletContact(t.cyl, t.pInner, t.cutHi)
	t.aCutLo, t.bCutLo = filletContact(t.cyl, t.pOuter, t.cutLo), filletContact(t.cyl, t.pInner, t.cutLo)
	var okHi, okLo bool
	t.aSeamHi, okHi = footprintPointAtStation(t.outer, t.cyl, t.seamHi)
	t.aSeamLo, okLo = footprintPointAtStation(t.outer, t.cyl, t.seamLo)
	if !okHi || !okLo {
		return false
	}
	t.bSeamHi, okHi = t.tangencyContact(t.seamHi, t.aSeamHi)
	t.bSeamLo, okLo = t.tangencyContact(t.seamLo, t.aSeamLo)
	return okHi && okLo
}

// tangencyContact is the SURF-RST band's contact point on pInner at station s (the ball tangent to
// pInner, through the outer footprint point q) — the corner the flank, the central patch and the inner
// host notch all meet at.
func (t setbackTiling) tangencyContact(s float64, q math.Point3) (math.Point3, bool) {
	c, ok := t.env.surfRstCentre(t.pInner, t.pOuter, s, q, t.weld)
	if !ok {
		return math.Point3{}, false
	}
	return projectOntoPlane(c, t.pInner), true
}

// resolveTwoBossBands solves the three bands' exact stations and the two shared seam cross-sections.
// The flanks are SURF-RST (their B rail is the synthesised tangency locus); the central is RST-RST
// between the two footprint sub-arcs. The seam arcs come from the FLANKS' end stations, so the flank
// and the central are handed the identical rail object at each seam.
func (t *setbackTiling) resolveTwoBossBands() bool {
	right, ok0 := t.surfRstBand(t.aSeamHi, t.aCutHi)
	left, ok1 := t.surfRstBand(t.aCutLo, t.aSeamLo)
	mid, ok2 := t.rstRstBand()
	if !ok0 || !ok1 || !ok2 {
		return false
	}
	t.right, t.left, t.mid = right, left, mid
	var okHi, okLo bool
	t.seamHiArc, okHi = right.endStation(false).sectionArcFrom(t.cyl.Radius, t.bSeamHi, t.weld)
	t.seamLoArc, okLo = left.endStation(true).sectionArcFrom(t.cyl.Radius, t.aSeamLo, t.weld)
	return okHi && okLo
}

// surfRstBand resolves one flank: its A rail is the outer footprint sub-arc from→to, its B rail the
// tangency locus on pInner synthesised from the closed-form solve.
func (t setbackTiling) surfRstBand(from, to math.Point3) (*runoutBand, bool) {
	rail, ok := footprintSubArc(t.outer.footEdge, from, to)
	if !ok {
		return nil, false
	}
	band, ok := buildRunoutBand(t.env, rail, nil, t.outer, surfRstFeet(t.env, t.pOuter, t.pInner, t.weld))
	if !ok {
		return nil, false
	}
	return &band, true
}

// rstRstBand resolves the two-boss CENTRAL band: both rails are footprint sub-arcs (outer on pOuter,
// inner on pInner) and the ball touches neither plane.
func (t setbackTiling) rstRstBand() (*runoutBand, bool) {
	outerRail, ok0 := footprintSubArc(t.outer.footEdge, t.aSeamLo, t.aSeamHi)
	innerRail, ok1 := footprintSubArc(t.inner.footEdge, t.bSeamLo, t.bSeamHi)
	if !ok0 || !ok1 {
		return nil, false
	}
	band, ok := buildRunoutBand(t.env, outerRail, innerRail, t.outer, rstRstFeet(t.env, t.inner, t.weld))
	if !ok {
		return nil, false
	}
	return &band, true
}

// resolveSingleBossTiling classifies the ONE-boss shape (S6 sphere / S9 torus / T3 oblique torus, #2007):
// the boss sits on ONE fillet-edge host plane (pOuter here), the OTHER edge face (pInner) carries only the
// tangency contact. The two cut stations cutLo/cutHi are where the footprint conic crosses the fillet
// band boundary; there are NO interior seams. ok=false when the boss host is neither fillet face, a face
// is not planar, or the single band's stations will not solve.
func resolveSingleBossTiling(b setbackBands, ef edgeFillet, res Resolution) (setbackTiling, bool) {
	boss := b.bosses[0]
	pHost, pOther, ok := singleBossHostPlanes(boss, ef)
	if !ok {
		return setbackTiling{}, false
	}
	t := setbackTiling{
		cyl: ef.cyl, env: newRunoutEnvelope(ef.cyl), weld: res.Weld(),
		pOuter: pHost, pInner: pOther, outer: boss,
		cutLo: b.cutLo, cutHi: b.cutHi,
	}
	t.aCutHi = filletContact(t.cyl, t.pOuter, t.cutHi)
	t.aCutLo = filletContact(t.cyl, t.pOuter, t.cutLo)
	band, ok := t.surfRstBand(t.aCutLo, t.aCutHi)
	if !ok {
		return setbackTiling{}, false
	}
	t.mid = band
	t.bCutLo, t.bCutHi = band.endStation(false).footB, band.endStation(true).footB
	return t, true
}

// singleBossHostPlanes reads the boss's host plane (pHost, carrying the footprint) and the OTHER fillet-
// edge face (pOther, tangency contact only). The boss host MUST be one of ef.a/ef.b — a boss whose
// footprint lands on some third face is not a fillet-edge host and honest-rejects. Both faces must be
// planar.
func singleBossHostPlanes(boss crossingBoss, ef edgeFillet) (geom.Plane, geom.Plane, bool) {
	var other *topo.Face
	switch boss.host {
	case ef.a:
		other = ef.b
	case ef.b:
		other = ef.a
	default:
		return geom.Plane{}, geom.Plane{}, false // boss host is neither fillet face
	}
	ph, ok0 := boss.host.Geometry().(geom.Plane)
	po, ok1 := other.Geometry().(geom.Plane)
	return ph, po, ok0 && ok1
}

// singleCentral is the ONE-boss central run-out band [cutLo, cutHi] (#2007): the footprint band-side arc
// on the host wall (aCutLo→aCutHi, G0 — the ball passes THROUGH that edge, it is not tangent to the boss
// wall there), the arm cross-section arc at cutHi (aCutHi→bCutHi, G1→fillet cyl), the tangency contact
// locus on pInner (bCutHi→bCutLo, G1→pInner), and the arm arc at cutLo (bCutLo→aCutLo, G1→fillet cyl).
// The footprint band-side arc IS the σ-partition band the wall rim tiles, so the patch welds to the
// subdivided wall rim. Its pInner rail is the RECEDED run-out contact, not the plain fillet's contact
// line — the separable host-plane defect coons4-audit.md §C.4 isolated.
func (t setbackTiling) singleCentral() (RailLoop, bool) {
	if t.mid == nil {
		return RailLoop{}, false
	}
	armHi, ok1 := armSectionArc(t.cyl, t.pOuter, t.pInner, t.cutHi)
	armLo, ok2 := armSectionArc(t.cyl, t.pInner, t.pOuter, t.cutLo)
	if !ok1 || !ok2 {
		return RailLoop{}, false
	}
	sides := []Side{
		{Curve: t.mid.railA, Adjacent: t.outer.wall, Cont: G0},
		{Curve: armHi, Adjacent: t.cyl, Cont: G1},
		{Curve: orientedLocus(t.mid.railB, t.bCutHi, t.weld), Adjacent: t.pInner, Cont: G1},
		{Curve: armLo, Adjacent: t.cyl, Cont: G1},
	}
	return t.loop(sides, t.mid, t.surfRstEnvelope()), true
}

// rightFlank is the +x flank band [seamHi, cutHi], running out to the outer boss only. Its four sides:
// the fillet ¼-cross-section arc at cutHi (G1→fillet cyl); the tangency contact locus on pInner
// (G1→pInner); the shared seam cross-section arc to the central patch (G0); and the outer footprint
// sub-arc back to the cut (G0 — the ball passes THROUGH the intact outer boss's footprint edge).
func (t setbackTiling) rightFlank() (RailLoop, bool) {
	if t.right == nil {
		return RailLoop{}, false
	}
	arc, ok := armSectionArc(t.cyl, t.pOuter, t.pInner, t.cutHi)
	if !ok {
		return RailLoop{}, false
	}
	sides := []Side{
		{Curve: arc, Adjacent: t.cyl, Cont: G1},
		{Curve: orientedLocus(t.right.railB, t.bCutHi, t.weld), Adjacent: t.pInner, Cont: G1},
		{Curve: t.seamHiArc, Cont: G0},
		{Curve: t.right.railA, Adjacent: t.outer.wall, Cont: G0},
	}
	return t.loop(sides, t.right, t.surfRstEnvelope()), true
}

// leftFlank is rightFlank mirrored to the −x band [cutLo, seamLo], wound the OPPOSITE way (arm arc
// pInner→pOuter) so its seam arc is traversed opposite to the central patch's — the mirror convention
// that keeps the shared seams weld-consistent between the flank and central patches.
func (t setbackTiling) leftFlank() (RailLoop, bool) {
	if t.left == nil {
		return RailLoop{}, false
	}
	arc, ok := armSectionArc(t.cyl, t.pInner, t.pOuter, t.cutLo)
	if !ok {
		return RailLoop{}, false
	}
	sides := []Side{
		{Curve: arc, Adjacent: t.cyl, Cont: G1},
		{Curve: t.left.railA, Adjacent: t.outer.wall, Cont: G0},
		{Curve: t.seamLoArc, Cont: G0},
		{Curve: orientedLocus(t.left.railB, t.bSeamLo, t.weld), Adjacent: t.pInner, Cont: G1},
	}
	return t.loop(sides, t.left, t.surfRstEnvelope()), true
}

// centralBand is the [seamLo, seamHi] band running out to BOTH boss walls (D2): the outer footprint arc
// on the A side and the inner footprint arc on the B side, joined by the two shared seam cross-section
// arcs. All four sides are G0 — an RST-RST ball touches neither host plane and is tangent to neither
// boss wall, it passes THROUGH both footprint edges. Its winding is opposite to both flanks' on their
// shared seams, so the tiling is orientation-consistent for the watertight weld.
func (t setbackTiling) centralBand() (RailLoop, bool) {
	if t.mid == nil {
		return RailLoop{}, false
	}
	seamHi, okHi := reorientedSeamArc(t.seamHiArc, t.aSeamHi, t.weld)
	seamLo, okLo := reorientedSeamArc(t.seamLoArc, t.bSeamLo, t.weld)
	innerRail, okIn := footprintSubArc(t.inner.footEdge, t.bSeamHi, t.bSeamLo)
	if !okHi || !okLo || !okIn {
		return RailLoop{}, false
	}
	sides := []Side{
		{Curve: t.mid.railA, Adjacent: t.outer.wall, Cont: G0},
		{Curve: seamHi, Cont: G0},
		{Curve: innerRail, Adjacent: t.inner.wall, Cont: G0},
		{Curve: seamLo, Cont: G0},
	}
	return t.loop(sides, t.mid, t.rstRstEnvelope()), true
}

// loop assembles one band's RailLoop with its exact-station payload and its envelope statement — the
// single place the two provider-scoped payloads are attached, so no band can ship one without the other.
func (t setbackTiling) loop(sides []Side, band *runoutBand, env BallEnvelope) RailLoop {
	return RailLoop{Sides: sides, Provenance: topo.Lineage{}, Runout: band.payload(env), Envelope: &env}
}

// surfRstEnvelope states what a flank / one-boss-central ball is the envelope of: TANGENT to pInner and
// THROUGH the outer boss's footprint conic. The certificate measures the patch interior against exactly
// this (never a guess — coons4-audit.md §B.4 measured a certify-time guess reading 5–19% even on OCCT's
// own correct patches).
func (t setbackTiling) surfRstEnvelope() BallEnvelope {
	return BallEnvelope{Radius: t.cyl.Radius, Tangents: []geom.Surface{t.pInner},
		Through: []geom.Curve3{footEdgeCurve(t.outer)}, Spine: t.env.spine}
}

// rstRstEnvelope states the two-boss CENTRAL ball's defining property: through BOTH footprint conics,
// tangent to nothing.
func (t setbackTiling) rstRstEnvelope() BallEnvelope {
	return BallEnvelope{Radius: t.cyl.Radius, Spine: t.env.spine,
		Through: []geom.Curve3{footEdgeCurve(t.outer), footEdgeCurve(t.inner)}}
}

// footEdgeCurve is a boss's whole intact footprint conic as a curve — the restriction the run-out ball
// passes through. The FULL conic is used (not the band sub-arc) so the residual is measured against the
// same curve OCCT's own blend runs out onto, with no sub-span bookkeeping.
func footEdgeCurve(boss crossingBoss) geom.Curve3 {
	return boss.footEdge.Geometry()
}

// reorientedSeamArc re-traces the shared seam arc from the requested endpoint. Reversing a circular
// arc's endpoints yields the mirrored parameter sweep, so both loops sample the IDENTICAL point set —
// the property the shared-seam weld rests on (the same one footprintSubArc already relies on).
func reorientedSeamArc(arc geom.Arc3d, from math.Point3, weld float64) (geom.Curve3, bool) {
	if float64(curveStart(arc).DistanceTo(from)) <= weld {
		return arc, true
	}
	rev, err := geom.Arc3dByThreePoints(curveEnd(arc), arc.PointAt(0.5), curveStart(arc))
	return rev, err == nil
}

// footprintPointAtStation is the point on boss's INTACT footprint conic at absolute spine station s, on
// the edgeward side (toward the fillet band), reading the boss from a crossingBoss (footEdge conic +
// host plane). The edgeward in-plane direction is center→(the fillet contact at the footprint's OWN
// station): perpendicular to the spine (the host plane contains the spine-parallel edge) and pointing at
// the band. ok=false when the station falls outside the footprint circle (|s−center-station| ≥ radius),
// so the caller honest-rejects.
func footprintPointAtStation(boss crossingBoss, cyl geom.Cylinder, s float64) (math.Point3, bool) {
	if e, ok := boss.footEdge.Geometry().(geom.EllipseFull); ok {
		return ellipseStationPoint(e, boss.host, cyl, s) // oblique elliptical-cylinder boss (T7)
	}
	center, r, ok := footprintConic(boss.footEdge)
	if !ok {
		return math.Point3{}, false
	}
	plane, ok := boss.host.Geometry().(geom.Plane)
	if !ok {
		return math.Point3{}, false
	}
	sc := spineParam(center, cyl)
	a := s - sc
	if a*a >= r*r {
		return math.Point3{}, false
	}
	edgeward, err := math.UnitVector3FromVector(center.VectorTo(filletContact(cyl, plane, sc)))
	if err != nil {
		return math.Point3{}, false
	}
	h := stdmath.Sqrt(r*r - a*a)
	return center.TranslateBy(cyl.AxisDir.AsVector().Scale(a)).TranslateBy(edgeward.AsVector().Scale(h)), true
}

// footprintSubArc is the minor sub-arc of a footprint conic between from and to — the exact intact-
// footprint rail (no fitting) the setback patch runs out onto along the boss wall. It dispatches on the
// footprint edge geometry: a circle/arc (geom.Circle/geom.Arc3d via footprintConic) built through the
// conic point on the endpoints' angular bisector as a geom.Arc3d, or an ELLIPSE (geom.EllipseFull, the
// oblique elliptical-cylinder boss of T7) as the exact geom.EllipticalArc via ellipseSubArc. The single
// source for every footprint sub-arc (wall rim, host detour, patch rails).
func footprintSubArc(footEdge *topo.Edge, from, to math.Point3) (geom.Curve3, bool) {
	if e, ok := footEdge.Geometry().(geom.EllipseFull); ok {
		return ellipseSubArc(e, from, to, false) // oblique elliptical-cylinder boss (T7): geom.EllipticalArc
	}
	c, r, ok := footprintConic(footEdge)
	if !ok {
		return geom.Arc3d{}, false
	}
	bis := c.VectorTo(from).Add(c.VectorTo(to))
	l := bis.Length()
	if l < arcBisectorTiny*r {
		return geom.Arc3d{}, false // endpoints near-antipodal on the footprint circle
	}
	mid := c.TranslateBy(bis.Scale(r / l))
	arc, err := geom.Arc3dByThreePoints(from, mid, to)
	return arc, err == nil
}
