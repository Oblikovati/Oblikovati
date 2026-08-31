// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"sort"

	stdmath "math"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The band∩obstacle IMPRINT — OCCT's blend-with-obstacle, as a marching walk in the band's own chart.
//
// WHAT IS MISSING WITHOUT IT. chainRetrimLoop (fillet_retrim_chain.go) can re-trim a host face's
// boundary across a CHAIN of faces, and each wrong face on simple/Y2 and simple/Y4 is exactly one call
// of it. Nothing COMPUTES those chains. They are the fillet band's imprint on the faces the band runs
// into: the band's ideal rectangle in the chart, cut by every obstacle the body puts in its way, and
// the surviving region's boundary walked back out into 3D.
//
// THE WALK, in three steps:
//  1. CUT. Every body face whose plane meets the band contributes a chart-aligned cut: a plane
//     PERPENDICULAR to the axis cuts a section circle (u = const), one PARALLEL to it cuts rulings
//     (v = const). A cut is kept only where the intersection really lies inside that face's own
//     trimmed boundary, so the far side of the body does not trim the band.
//  2. CLASSIFY. The cuts arrange the ideal box into a grid of cells. A cell survives iff its interior
//     is INSIDE the original solid — the analytic point-in-solid classifier (brep.InsideQuery), the
//     same ground truth the concave-arm void gate already uses. The band is the part of the cylinder
//     the material still backs; a cell in a slot is not.
//  3. VERIFY, then WALK. The arrangement is a HYPOTHESIS: it is right only if no cut was missed. Every
//     cell is re-probed on an interior grid and must agree with its own classification throughout —
//     one disagreement means an obstacle the cut step could not express, and the whole rebuild is
//     DECLINED rather than shipped half-right. Only then is the surviving region's boundary traced.
//
// ATOMICITY. The walk yields the WHOLE boundary, not one face's share of it: Y2's five wrong faces and
// Y4's seven share edges, and re-trimming any subset opens the shell (routing Y2's host plane alone
// takes 8475 → 8450 while the band still claims x ∈ [0,100] at z = 85). The router therefore takes all
// the runs or none of them.

// bandProbeGrid is the per-cell interior probe used both to classify a cell and to FALSIFY the
// arrangement. The stations sit at k/(bandProbeGrid+1) of the cell in each direction, so they stay
// clear of the cell's own walls — where "inside the solid" is a boundary question, not an interior one
// — while resolving any undrawn cut that splits the cell by more than 1/(bandProbeGrid+1) of it.
//
// 15 is sized from the corpus slot family, whose smallest genuine sub-cut is the wall at x = 90 across
// the 100-long band: 10 % of the span, which a 3×3 probe steps straight over (its stations stop at
// 75 %). A finite probe can never PROVE a cell homogeneous — it can only falsify it — so this is the
// second line of defence behind the cut step's own exhaustive face sweep, not the first.
const bandProbeGrid = 15

// bandCutOnFaceSamples is the number of stations a candidate cut is tested at for actually lying
// inside the cutting face's own trimmed boundary. A cut no station lands on is dropped; if that drop
// was wrong, the cell verification sees it and the rebuild is declined.
const bandCutOnFaceSamples = 33

// bandImprint is one filleted edge's solved imprint: the chart it was solved in, the cut grid, and the
// surviving region's boundary as an ordered, closed list of runs (the band's own rebuilt loop, and the
// per-face contact chains, are read off it by fillet_band_imprint_faces.go).
type bandImprint struct {
	chart bandChart
	us    []float64 // the u grid lines, 0 … uMax
	vs    []float64 // the v grid lines, 0 … vMax
	runs  []bandRun
}

// bandRun is one straight run of the surviving region's boundary in the chart: either a section arc at
// constant u or a ruling at constant v, spanning from→to of the other coordinate in traversal order.
type bandRun struct {
	constU   bool
	at       float64
	from, to float64
}

// bandFullSide reports whether the run is one whole side of the IDEAL box — the boundary cylinderFace
// already emits. A run that is a full side changes nothing about the face it lies on, which is how the
// router keeps an untouched neighbour byte-identical.
func (r bandRun) bandFullSide(c bandChart) bool {
	lo, hi := stdmath.Min(r.from, r.to), stdmath.Max(r.from, r.to)
	span := c.vMax
	if r.constU {
		return lo == 0 && hi == span
	}
	return lo == 0 && hi == c.uMax
}

// solveBandImprint runs the whole walk for one filleted edge. ok=false is an HONEST DECLINE — no
// obstacle found (the overwhelmingly common case, and the reason the corpus cannot move), an obstacle
// the chart cannot express, a cell classification the probe falsifies, or a region that is not one
// simple loop. A decline leaves the edge on the existing path untouched.
func solveBandImprint(body *topo.Body, ef edgeFillet, tol float64) (bandImprint, bool) {
	chart, ok := newBandChart(ef)
	if !ok {
		return bandImprint{}, false
	}
	us, vs, ok := bandImprintCuts(body, ef, chart, tol)
	if !ok || (len(us) == 2 && len(vs) == 2) {
		return bandImprint{}, false // no interior cut: nothing obstructs this band
	}
	solid, ok := bandClassifyCells(body, chart, us, vs)
	if !ok {
		return bandImprint{}, false
	}
	runs, ok := bandTraceRegion(solid, us, vs)
	if !ok {
		return bandImprint{}, false
	}
	return bandImprint{chart: chart, us: us, vs: vs, runs: runs}, true
}

// bandImprintCuts collects the chart grid lines every body face draws on the band. ok=false when a
// face meets the band with a cut the chart cannot express (an OBLIQUE plane, or any non-planar
// surface): its imprint is a general curve, so the arrangement would be silently incomplete.
func bandImprintCuts(body *topo.Body, ef edgeFillet, c bandChart, tol float64) ([]float64, []float64, bool) {
	us, vs := []float64{0, c.uMax}, []float64{0, c.vMax}
	for _, f := range body.Faces() {
		if f == ef.a || f == ef.b {
			continue // the two hosts are the box's own v = 0 and v = vMax sides
		}
		fu, fv, ok := bandFaceCuts(f, c)
		if !ok {
			return nil, nil, false
		}
		us, vs = append(us, fu...), append(vs, fv...)
	}
	return bandGridLines(us, 0, c.uMax, tol), bandGridLines(vs, 0, c.vMax, tol*bandAngleScale(c)), true
}

// bandAngleScale converts a model-space tolerance into the chart's ANGULAR coordinate, so the v grid's
// duplicate merge is the same physical distance as the u grid's (ADR-0042: relative, not absolute).
func bandAngleScale(c bandChart) float64 { return 1 / c.radius }

// bandFaceCuts is one face's contribution to the chart grid. A plane perpendicular to the axis cuts a
// section circle (a u line); one parallel to it cuts rulings (v lines); anything else that actually
// reaches the band is refused (ok=false). A cut is kept only where the cut really lies inside the
// face's own boundary, so a parallel wall on the far side of the body draws nothing.
func bandFaceCuts(f *topo.Face, c bandChart) ([]float64, []float64, bool) {
	pl, planar := f.Geometry().(geom.Plane)
	if !planar {
		return nil, nil, !bandFaceReachesBand(f, c)
	}
	n, dot := pl.Normal(), 0.0
	dot = stdmath.Abs(float64(n.Dot(c.axis)))
	switch {
	case dot > 1-bandAxisAlignTol:
		return bandKeptCuts(f, c, []float64{c.bandAxialCut(pl.Origin)}, true), nil, true
	case dot < bandAxisAlignTol:
		return nil, bandKeptCuts(f, c, c.bandRulingCuts(pl.Origin, n), false), true
	}
	return nil, nil, !bandFaceReachesBand(f, c)
}

// bandAxisAlignTol is how square a plane must be to the band axis to be read as a section cut or a
// ruling cut. A plane between the two is oblique: its imprint is a general curve, not a grid line.
const bandAxisAlignTol = 1e-9

// bandKeptCuts drops every candidate cut that is outside the ideal box or does not actually land
// inside the cutting face's own trimmed boundary.
func bandKeptCuts(f *topo.Face, c bandChart, cand []float64, axial bool) []float64 {
	var out []float64
	for _, x := range cand {
		if !bandCutInsideBox(c, x, axial) || !bandCutOnFace(f, c, x, axial) {
			continue
		}
		out = append(out, x)
	}
	return out
}

// bandCutInsideBox reports whether a cut falls strictly between the ideal box's own two sides on that
// coordinate — a cut ON a side is the side, and changes nothing.
func bandCutInsideBox(c bandChart, x float64, axial bool) bool {
	hi := c.vMax
	if axial {
		hi = c.uMax
	}
	return x > 0 && x < hi
}

// bandCutOnFace reports whether the cut's line meets the face's OWN trimmed area at any station.
func bandCutOnFace(f *topo.Face, c bandChart, x float64, axial bool) bool {
	ev := topo.NewFaceEvaluator(f)
	for i := 0; i <= bandCutOnFaceSamples; i++ {
		t := float64(i) / bandCutOnFaceSamples
		if ev.Contains(bandCutStation(c, x, t, axial)) {
			return true
		}
	}
	return false
}

// bandCutStation is one station along a candidate cut, parameterised t ∈ [0,1] across the ideal box.
func bandCutStation(c bandChart, x, t float64, axial bool) math.Point3 {
	if axial {
		return c.bandPointAt(x, t*c.vMax)
	}
	return c.bandPointAt(t*c.uMax, x)
}

// bandFaceReachesBand reports whether a face the chart cannot express nonetheless touches the ideal
// band patch — the case that must be declined rather than ignored.
func bandFaceReachesBand(f *topo.Face, c bandChart) bool {
	ev := topo.NewFaceEvaluator(f)
	for i := 0; i <= bandCutOnFaceSamples; i++ {
		for j := 0; j <= bandCutOnFaceSamples; j++ {
			u, v := float64(i)/bandCutOnFaceSamples*c.uMax, float64(j)/bandCutOnFaceSamples*c.vMax
			if ev.Contains(c.bandPointAt(u, v)) {
				return true
			}
		}
	}
	return false
}

// bandGridLines sorts, clamps and de-duplicates a coordinate's grid lines, keeping the two box sides
// EXACT (lo and hi are returned verbatim) so bandBoxCorner's corner snap stays keyed on equality.
func bandGridLines(x []float64, lo, hi, tol float64) []float64 {
	sort.Float64s(x)
	out := []float64{lo}
	for _, v := range x {
		if v-out[len(out)-1] > tol && hi-v > tol {
			out = append(out, v)
		}
	}
	return append(out, hi)
}

// bandClassifyCells decides, for every cell of the arrangement, whether the band survives there — and
// FALSIFIES the arrangement while doing it: all bandProbeGrid² interior stations of a cell must agree,
// because a cell straddling an undrawn cut does not. ok=false on any disagreement.
func bandClassifyCells(body *topo.Body, c bandChart, us, vs []float64) ([][]bool, bool) {
	if len(body.Faces()) == 0 {
		return nil, false
	}
	inside := brep.NewInsideQuery(body)
	solid := make([][]bool, len(us)-1)
	for i := range solid {
		solid[i] = make([]bool, len(vs)-1)
		for j := range solid[i] {
			in, ok := bandCellInside(inside, c, us[i], us[i+1], vs[j], vs[j+1])
			if !ok {
				return nil, false
			}
			solid[i][j] = in
		}
	}
	return solid, true
}

// bandCellInside probes one cell's interior. ok=false when the stations disagree — the arrangement
// missed a cut through this cell, so nothing downstream may be trusted.
func bandCellInside(inside *brep.InsideQuery, c bandChart, u0, u1, v0, v1 float64) (bool, bool) {
	first, seen := false, false
	for i := 1; i <= bandProbeGrid; i++ {
		for j := 1; j <= bandProbeGrid; j++ {
			u := u0 + (u1-u0)*float64(i)/float64(bandProbeGrid+1)
			v := v0 + (v1-v0)*float64(j)/float64(bandProbeGrid+1)
			in := inside.Inside(c.bandPointAt(u, v))
			if seen && in != first {
				return false, false
			}
			first, seen = in, true
		}
	}
	return first, seen
}
