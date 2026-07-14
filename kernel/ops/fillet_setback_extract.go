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
// each boss wall INTACT (setback-patch-derivation.md D3, "Candidate method ii"). Unlike the split-boss
// extractRunout, the footprint rails carry the intact boss wall as a G1 Adjacent, so the coons4 ribbon
// fairs the fill out to the true cylinder/cone/torus instead of to a split sub-arc.
//
// It tiles the 2-boss S1 shape (left flank / central / right flank → 3 loops): both flanks run out to
// the OUTER boss only (plain on the other tangent side, a G1 host-plane contact seam), and the central
// band runs out to BOTH boss walls (D2). N>2 bosses (2N−1 bands) is a later task — this milestone
// honest-rejects (ok=false) anything but the two-boss dual-host case, and any loop that will not close
// or is mis-tiled (a station outside a footprint, both bosses on one plane). UNWIRED: no caller in
// runoutFacesFor yet, so the corpus stays byte-identical.
func extractSetbackPatches(b setbackBands, ef edgeFillet, res Resolution) ([]RailLoop, bool) {
	t, ok := resolveSetbackTiling(b, ef)
	if !ok {
		return nil, false
	}
	loops := make([]RailLoop, 0, 3)
	for _, build := range []func() (RailLoop, bool){t.leftFlank, t.central, t.rightFlank} {
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
// pInner→pOuter) so its internal seam is traversed opposite to the central patch's — the same mirror
// convention extractRunout's left/right loops use, which keeps the shared seams weld-consistent.
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
// the edgeward side (toward the fillet band) — the setback sibling of seamPointOnFeature, reading the
// boss from a crossingBoss (footEdge conic + host plane) instead of a runoutImprint. The edgeward
// in-plane direction is center→(the fillet contact at the footprint's OWN station): perpendicular to the
// spine (the host plane contains the spine-parallel edge) and pointing at the band. ok=false when the
// station falls outside the footprint circle (|s−center-station| ≥ radius), so the caller honest-rejects.
func footprintPointAtStation(boss crossingBoss, cyl geom.Cylinder, s float64) (math.Point3, bool) {
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

// footprintSubArc is the minor sub-arc of a footprint conic (a full geom.Circle/geom.Arc3d read from
// footEdge via footprintConic) between from and to, built through the conic point on their angular
// bisector — the exact intact-footprint rail (no fitting) the setback patch is G1 to along the boss
// wall. The split-boss featureSubArc now delegates here, so both read the footprint conic identically.
func footprintSubArc(footEdge *topo.Edge, from, to math.Point3) (geom.Arc3d, bool) {
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
