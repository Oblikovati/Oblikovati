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
// each boss wall INTACT (setback-patch-derivation.md D3, "Candidate method ii"). Unlike the removed
// split-boss tiler, the footprint rails carry the intact boss wall as a G1 Adjacent, so the coons4
// ribbon fairs the fill out to the true cylinder/cone/torus instead of to a split sub-arc.
//
// It tiles the 2-boss S1 shape (left flank / central / right flank → 3 loops): both flanks run out to
// the OUTER boss only (plain on the other tangent side, a G1 host-plane contact seam), and the central
// band runs out to BOTH boss walls (D2). It also tiles the 1-boss shape (single central run-out, #2007)
// via setbackLoopBuilders. N>2 bosses (2N−1 bands) is a later task — this milestone honest-rejects
// (ok=false) anything but the one-boss and two-boss dual-host cases, and any loop that will not close or
// is mis-tiled (a station outside a footprint, both bosses on one plane). WIRED: runoutFacesFor
// (fillet_runout_faces.go) calls this on every constant-radius edge with a detected setback band; it is
// what greens the corpus's S1/S4/T1/T4/T7/S7 (two-boss) and S6/S9/T3 (one-boss) cases.
func extractSetbackPatches(b setbackBands, ef edgeFillet, res Resolution) ([]RailLoop, bool) {
	t, ok := resolveSetbackTiling(b, ef)
	if !ok {
		return nil, false
	}
	return buildSetbackLoops(setbackLoopBuilders(b, t), res)
}

// setbackLoopBuilders selects the band-loop builders for the tiling: the SINGLE central run-out for a
// one-boss body (S6/S9/T3, #2007) or the S1 three-loop flank/central/flank tiling for the two-boss body.
// The one-boss central is the degenerate S1 shape — one footprint rail (the intact boss wall) closed off
// by an arm arc at EACH cut station and the plain host-b contact seam, instead of two flanks joined by
// internal seams.
func setbackLoopBuilders(b setbackBands, t setbackTiling) []func() (RailLoop, bool) {
	if len(b.bosses) == 1 {
		return []func() (RailLoop, bool){t.singleCentral}
	}
	return []func() (RailLoop, bool){t.leftFlank, t.central, t.rightFlank}
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

// setbackTiling is extractSetbackPatches' resolved geometry: the fillet cylinder, the two host planes
// (pOuter carries the outer boss the flanks reach, pInner carries the inner boss + the flanks' plain
// contact seam), the two bosses, the four ABSOLUTE spine stations, and the eight shared band corners.
// Every corner is computed ONCE (resolveCorners) so a corner shared between the central patch and a
// flank is bit-identical, and the internal seams built from them weld watertight when the path is wired.
type setbackTiling struct {
	cyl            geom.Cylinder
	pOuter, pInner geom.Plane
	outer, inner   crossingBoss
	cutLo, cutHi   float64 // outer boss's setback stations (the plain-fillet ends)
	seamLo, seamHi float64 // inner boss's setback stations (flank/central boundaries), sorted ascending
	// aX = fillet∩pOuter or the outer footprint point; bX = fillet∩pInner. Hi=+x end, Lo=−x end.
	aCutLo, bCutLo, aSeamLo, bSeamLo math.Point3
	aCutHi, bCutHi, aSeamHi, bSeamHi math.Point3
}

// resolveSetbackTiling classifies the two-boss dual-host shape, sorts the (mirror-ordered) seam
// stations ascending, and computes the shared corners. ok=false for any other configuration (this
// milestone tiles the S1 shape only). The seams arrive mirror-ordered ([mid+x, mid−x]) from Task 2's
// bandsFromBosses, NOT monotone, so they MUST be sorted before use as band boundaries.
func resolveSetbackTiling(b setbackBands, ef edgeFillet) (setbackTiling, bool) {
	if len(b.bosses) == 1 && len(b.seams) == 0 {
		return resolveSingleBossTiling(b, ef) // one boss → 2 flanks + one central run-out (#2007)
	}
	if len(b.bosses) != 2 || len(b.seams) != 2 {
		return setbackTiling{}, false // 2-boss S1 shape only (2N−1 general bands: a later task)
	}
	pOuter, pInner, ok := setbackHostPlanes(b.bosses[0], b.bosses[1])
	if !ok {
		return setbackTiling{}, false
	}
	seamLo, seamHi := stdmath.Min(b.seams[0], b.seams[1]), stdmath.Max(b.seams[0], b.seams[1])
	t := setbackTiling{
		cyl: ef.cyl, pOuter: pOuter, pInner: pInner, outer: b.bosses[0], inner: b.bosses[1],
		cutLo: b.cutLo, cutHi: b.cutHi, seamLo: seamLo, seamHi: seamHi,
	}
	if !t.resolveCorners() {
		return setbackTiling{}, false
	}
	return t, true
}

// setbackHostPlanes reads the two host support planes: pOuter carries the outer boss (bosses[0], which
// both flanks and the central patch run out to on the A side), pInner carries the inner boss (central
// B side) and the flanks' plain contact seam. ok=false when the two bosses share one host (not the S1
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
// cut/seam stations (filletContact), plus the two outer-footprint points at the seam stations
// (footprintPointAtStation — where the outer boss footprint bounds the central/flank A side). The
// inner-footprint seam corners coincide with the fillet∩pInner contacts by construction (the seam IS
// the inner boss's setback station), so bSeamHi/bSeamLo double as the inner footprint endpoints.
func (t *setbackTiling) resolveCorners() bool {
	t.aCutHi, t.bCutHi = filletContact(t.cyl, t.pOuter, t.cutHi), filletContact(t.cyl, t.pInner, t.cutHi)
	t.aCutLo, t.bCutLo = filletContact(t.cyl, t.pOuter, t.cutLo), filletContact(t.cyl, t.pInner, t.cutLo)
	t.bSeamHi, t.bSeamLo = filletContact(t.cyl, t.pInner, t.seamHi), filletContact(t.cyl, t.pInner, t.seamLo)
	var okHi, okLo bool
	t.aSeamHi, okHi = footprintPointAtStation(t.outer, t.cyl, t.seamHi)
	t.aSeamLo, okLo = footprintPointAtStation(t.outer, t.cyl, t.seamLo)
	return okHi && okLo
}

// resolveSingleBossTiling classifies the ONE-boss shape (S6 sphere / S9 torus / T3 oblique torus, #2007):
// the boss sits on ONE fillet-edge host plane (pOuter here), the OTHER edge face (pInner) carries only the
// plain fillet contact. The two cut stations cutLo/cutHi are where the footprint conic crosses the fillet
// band boundary; there are NO interior seams. ok=false when the boss host is neither fillet face or a face
// is not planar.
func resolveSingleBossTiling(b setbackBands, ef edgeFillet) (setbackTiling, bool) {
	boss := b.bosses[0]
	pHost, pOther, ok := singleBossHostPlanes(boss, ef)
	if !ok {
		return setbackTiling{}, false
	}
	boss.densifyHostArc = true // one-boss sphere host arc: densify vs the coarse-chorded 2-boss default (S7)
	t := setbackTiling{
		cyl: ef.cyl, pOuter: pHost, pInner: pOther, outer: boss,
		cutLo: b.cutLo, cutHi: b.cutHi,
	}
	t.resolveSingleCorners()
	return t, true
}

// singleBossHostPlanes reads the boss's host plane (pHost, carrying the footprint) and the OTHER fillet-
// edge face (pOther, plain contact only). The boss host MUST be one of ef.a/ef.b — a boss whose footprint
// lands on some third face is not a fillet-edge host and honest-rejects. Both faces must be planar.
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

// resolveSingleCorners fills the four one-boss corners: at each cut station the fillet cross-section
// contacts pOuter (aCut) and pInner (bCut). The aCut contacts lie EXACTLY on the footprint conic by
// construction — the cut stations ARE where the footprint crosses the fillet contact line L, so
// filletContact on the host plane at that station returns the footprint crossing point itself (verified
// on S6/S9/T3: distance-from-rim = 0). No footprintPointAtStation is needed as it is for the 2-boss seams.
func (t *setbackTiling) resolveSingleCorners() {
	t.aCutHi, t.bCutHi = filletContact(t.cyl, t.pOuter, t.cutHi), filletContact(t.cyl, t.pInner, t.cutHi)
	t.aCutLo, t.bCutLo = filletContact(t.cyl, t.pOuter, t.cutLo), filletContact(t.cyl, t.pInner, t.cutLo)
}

// singleCentral is the ONE-boss central run-out band [cutLo, cutHi] (#2007): the footprint band-side arc on
// the host wall (aCutLo→aCutHi, G1→the INTACT boss wall), the arm cross-section arc at cutHi (aCutHi→bCutHi,
// G1→fillet cyl), the plain host-b contact seam (bCutHi→bCutLo, G1→pInner), and the arm arc at cutLo
// (bCutLo→aCutLo, G1→fillet cyl). It is the S1 flank shape closed off by a SECOND arm arc instead of an
// internal seam — the footprint band-side arc IS the σ-partition band the wall rim tiles (footprintSubArc,
// verified equal to the partition band on S6/S9/T3), so the patch welds to the subdivided wall rim.
func (t setbackTiling) singleCentral() (RailLoop, bool) {
	foot, ok0 := footprintSubArc(t.outer.footEdge, t.aCutLo, t.aCutHi)
	armHi, ok1 := armSectionArc(t.cyl, t.pOuter, t.pInner, t.cutHi)
	armLo, ok2 := armSectionArc(t.cyl, t.pInner, t.pOuter, t.cutLo)
	if !ok0 || !ok1 || !ok2 {
		return RailLoop{}, false
	}
	sides := []Side{
		{Curve: foot, Adjacent: t.outer.wall, Cont: G1},
		{Curve: armHi, Adjacent: t.cyl, Cont: G1},
		{Curve: geom.NewLineSegment(t.bCutHi, t.bCutLo), Adjacent: t.pInner, Cont: G1},
		{Curve: armLo, Adjacent: t.cyl, Cont: G1},
	}
	return RailLoop{Sides: sides, Provenance: topo.Lineage{}}, true
}

// rightFlank is the +x flank band [seamHi, cutHi], running out to the outer boss only. Its four sides:
// the fillet ¼-cross-section arc at cutHi (G1→fillet cyl); the plain contact seam on pInner (G1→pInner,
// the fillet is still tangent to that plane here); the internal seam to the central patch (G0); and the
// outer footprint sub-arc back to the cut (G1→the INTACT outer boss wall — the D3 correction).
func (t setbackTiling) rightFlank() (RailLoop, bool) {
	arc, ok0 := armSectionArc(t.cyl, t.pOuter, t.pInner, t.cutHi)
	foot, ok1 := footprintSubArc(t.outer.footEdge, t.aSeamHi, t.aCutHi)
	if !ok0 || !ok1 {
		return RailLoop{}, false
	}
	sides := []Side{
		{Curve: arc, Adjacent: t.cyl, Cont: G1},
		{Curve: geom.NewLineSegment(t.bCutHi, t.bSeamHi), Adjacent: t.pInner, Cont: G1},
		{Curve: internalSeam(t.bSeamHi, t.aSeamHi), Cont: G0},
		{Curve: foot, Adjacent: t.outer.wall, Cont: G1},
	}
	return RailLoop{Sides: sides, Provenance: topo.Lineage{}}, true
}

// leftFlank is rightFlank mirrored to the −x band [cutLo, seamLo], wound the OPPOSITE way (arm arc
// pInner→pOuter) so its internal seam is traversed opposite to the central patch's — the mirror
// convention that keeps the shared seams weld-consistent between the flank and central patches.
func (t setbackTiling) leftFlank() (RailLoop, bool) {
	arc, ok0 := armSectionArc(t.cyl, t.pInner, t.pOuter, t.cutLo)
	foot, ok1 := footprintSubArc(t.outer.footEdge, t.aCutLo, t.aSeamLo)
	if !ok0 || !ok1 {
		return RailLoop{}, false
	}
	sides := []Side{
		{Curve: arc, Adjacent: t.cyl, Cont: G1},
		{Curve: foot, Adjacent: t.outer.wall, Cont: G1},
		{Curve: internalSeam(t.aSeamLo, t.bSeamLo), Cont: G0},
		{Curve: geom.NewLineSegment(t.bSeamLo, t.bCutLo), Adjacent: t.pInner, Cont: G1},
	}
	return RailLoop{Sides: sides, Provenance: topo.Lineage{}}, true
}

// central is the [seamLo, seamHi] band running out to BOTH boss walls (D2): the outer footprint arc on
// the A side (G1→outer wall) and the inner footprint arc on the B side (G1→inner wall), joined by the
// two internal G0 seams it shares (reversed) with the flanks. Its winding is opposite to both flanks'
// on their shared seams, so the tiling is orientation-consistent for the watertight weld (Task 5).
func (t setbackTiling) central() (RailLoop, bool) {
	footOuter, ok0 := footprintSubArc(t.outer.footEdge, t.aSeamLo, t.aSeamHi)
	footInner, ok1 := footprintSubArc(t.inner.footEdge, t.bSeamHi, t.bSeamLo)
	if !ok0 || !ok1 {
		return RailLoop{}, false
	}
	sides := []Side{
		{Curve: footOuter, Adjacent: t.outer.wall, Cont: G1},
		{Curve: internalSeam(t.aSeamHi, t.bSeamHi), Cont: G0},
		{Curve: footInner, Adjacent: t.inner.wall, Cont: G1},
		{Curve: internalSeam(t.bSeamLo, t.aSeamLo), Cont: G0},
	}
	return RailLoop{Sides: sides, Provenance: topo.Lineage{}}, true
}

// footprintPointAtStation is the point on boss's INTACT footprint conic at absolute spine station s, on
// the edgeward side (toward the fillet band), reading the boss from a crossingBoss (footEdge conic +
// host plane). The edgeward
// in-plane direction is center→(the fillet contact at the footprint's OWN station): perpendicular to the
// spine (the host plane contains the spine-parallel edge) and pointing at the band. ok=false when the
// station falls outside the footprint circle (|s−center-station| ≥ radius), so the caller honest-rejects.
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
// footprint rail (no fitting) the setback patch is G1 to along the boss wall. It dispatches on the
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
