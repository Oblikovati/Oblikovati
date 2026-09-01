// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"sort"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// pinchedCanalBandMesh meshes the PINCHED canal band (fillet_elliptic_cone_canal.go): a
// geom.BSplineSurface whose cross-section collapses to a point at one or both v-ends (the
// host-tangency teardrop of the EllipticalCylinder∧Cone rims, tolblend B4..C3). The generic
// B-spline trim paths cannot see this face — its boundary is two u-boundary rails that MEET at
// the pinch (no planar UV trim), and near the degenerate column point inversion is ill-posed —
// so, exactly like the elliptic-rim band (canal_band_loft.go), it is lofted rail-to-rail: each
// boundary row is the rail edges' own tessellation (conforming to the retrimmed hosts by
// construction), interior rows sample the surface, and the pinch is ONE shared mesh vertex the
// end fans close onto — no degenerate slivers, no T-junction at the tangency point.
func pinchedCanalBandMesh(f *topo.Face, s geom.Surface, q Quality) (*Mesh, bool) {
	bs, isSpline := s.(geom.BSplineSurface)
	if !isSpline || len(f.Loops()) != 1 {
		return nil, false
	}
	pinchLo, pinchHi := pinchedVEnds(bs)
	if !pinchLo && !pinchHi {
		return nil, false
	}
	b, ok := pinchedBandBoundary(f, bs, q)
	if !ok {
		return nil, false
	}
	return loftPinchedBand(bs, b, pinchLo, pinchHi, q)
}

// pinchedVEndTol is the RELATIVE zero-width test for a collapsed v-end column, scaled by the
// control net extent (ADR-0042): the pinch column's controls are exactly coincident, so this is
// a round-off allowance, not a fit tolerance.
const pinchedVEndTol = 1e-9

// pinchedVEnds reports which v-boundary columns of the surface collapse to a point.
func pinchedVEnds(bs geom.BSplineSurface) (lo, hi bool) {
	uLo, uHi := bs.UDomain()
	vLo, vHi := bs.VDomain()
	tol := pinchedVEndTol * ctrlNetExtent(bs)
	width := func(v float64) float64 {
		w := 0.0
		for k := 0; k <= 4; k++ {
			u := uLo + (uHi-uLo)*float64(k)/4
			w = stdmath.Max(w, float64(bs.PointAt(u, v).DistanceTo(bs.PointAt(uLo, v))))
		}
		return w
	}
	return width(vLo) <= tol, width(vHi) <= tol
}

// pinchedBand carries the classified boundary: the two rail rows (their own edge tessellations,
// pinch samples stripped) and the optional open-end section arc's u-parameters.
type pinchedBand struct {
	loRow, hiRow bandRow // u=uLo / u=uHi rails: mesh-ready points keyed by v
	loPts, hiPts []math.Point3
	arcUs        []float64 // open-end cross-section sample parameters (nil for a closed teardrop)
	arcV         float64   // the arc's v iso-parameter
}

// pinchedBandBoundary classifies the loop's edges: every edge curve must be a u-boundary rail
// piece (v-parametrised, verified against the surface) or the single open-end v-iso section arc
// (u-parametrised). Anything else declines — no other face shape can be diverted here.
func pinchedBandBoundary(f *topo.Face, bs geom.BSplineSurface, q Quality) (*pinchedBand, bool) {
	uLo, uHi := bs.UDomain()
	var loPieces, hiPieces []*topo.Edge
	var arcE *topo.Edge
	for _, u := range f.Loops()[0].EdgeUses() {
		e := u.Edge()
		switch {
		case railPieceOnIso(bs, e, uLo):
			loPieces = appendDistinctEdge(loPieces, e)
		case railPieceOnIso(bs, e, uHi):
			hiPieces = appendDistinctEdge(hiPieces, e)
		case arcE == nil && sectionArcPiece(bs, e):
			arcE = e
		default:
			return nil, false
		}
	}
	if len(loPieces) == 0 || len(hiPieces) == 0 {
		return nil, false
	}
	return assemblePinchedBand(bs, loPieces, hiPieces, arcE, q)
}

// railPieceIsoProbes matches canalRailIsoProbes: a handful of probes decides an exact-by-
// construction premise.
const railPieceIsoProbes = 8

// railPieceOnIso verifies an edge's curve tracks the surface's u=uIso isocurve over the curve's
// OWN parameter range (a SplitCurve piece keeps the surface's v parametrisation verbatim).
func railPieceOnIso(bs geom.BSplineSurface, e *topo.Edge, uIso float64) bool {
	c := e.Geometry()
	lo, hi := c.Domain()
	tol := pinchedVEndTol * 1e3 * ctrlNetExtent(bs)
	for k := 0; k <= railPieceIsoProbes; k++ {
		t := lo + (hi-lo)*float64(k)/railPieceIsoProbes
		if float64(c.PointAt(t).DistanceTo(bs.PointAt(uIso, t))) > tol {
			return false
		}
	}
	return true
}

// sectionArcPiece verifies an edge's curve tracks a v-iso of the surface (the open-end section
// arc, parametrised by u).
func sectionArcPiece(bs geom.BSplineSurface, e *topo.Edge) bool {
	c := e.Geometry()
	lo, hi := c.Domain()
	_, v := bs.ParamAt(c.PointAt(0.5 * (lo + hi)))
	tol := pinchedVEndTol * 1e3 * ctrlNetExtent(bs)
	for k := 0; k <= railPieceIsoProbes; k++ {
		t := lo + (hi-lo)*float64(k)/railPieceIsoProbes
		if float64(c.PointAt(t).DistanceTo(bs.PointAt(t, v))) > tol {
			return false
		}
	}
	return true
}

// assemblePinchedBand tessellates the rail pieces into the two boundary rows and reads the
// arc's u-samples.
func assemblePinchedBand(bs geom.BSplineSurface, loPieces, hiPieces []*topo.Edge, arcE *topo.Edge, q Quality) (*pinchedBand, bool) {
	loPts, loVs, ok1 := railPiecesRow(loPieces, q)
	hiPts, hiVs, ok2 := railPiecesRow(hiPieces, q)
	if !ok1 || !ok2 {
		return nil, false
	}
	b := &pinchedBand{loPts: loPts, hiPts: hiPts, loRow: bandRow{ang: loVs}, hiRow: bandRow{ang: hiVs}}
	if arcE != nil {
		pts, us := tessellateEdgeWithParams(arcE, q)
		if len(pts) < 2 {
			return nil, false
		}
		sort.Float64s(us)
		_, b.arcV = bs.ParamAt(pts[len(pts)/2])
		b.arcUs = us
	}
	return b, true
}

// railParamSample is one rail tessellation sample: the surface v it carries and its point.
type railParamSample struct {
	v float64
	p math.Point3
}

// railPiecesRow tessellates each rail piece and merges them in ascending v, dropping duplicate
// junction samples.
func railPiecesRow(pieces []*topo.Edge, q Quality) ([]math.Point3, []float64, bool) {
	var all []railParamSample
	for _, e := range pieces {
		pts, vs := tessellateEdgeWithParams(e, q)
		for i := range pts {
			all = append(all, railParamSample{vs[i], pts[i]})
		}
	}
	if len(all) < 3 {
		return nil, nil, false
	}
	sort.Slice(all, func(i, j int) bool { return all[i].v < all[j].v })
	pts, vs := dedupRailSamples(all)
	return pts, vs, len(pts) >= 3
}

// dedupRailSamples drops consecutive samples with (near-)equal parameters — the shared junction
// vertex of two split pieces appears once per piece.
func dedupRailSamples(all []railParamSample) ([]math.Point3, []float64) {
	span := all[len(all)-1].v - all[0].v
	var pts []math.Point3
	var vs []float64
	for _, s := range all {
		if len(vs) > 0 && s.v-vs[len(vs)-1] <= 1e-12*span {
			continue
		}
		pts = append(pts, s.p)
		vs = append(vs, s.v)
	}
	return pts, vs
}

// loftPinchedBand builds the mesh: boundary rows from the rails, interior rows sampled at the
// finer rail's stations, one shared vertex per pinched end, open-ladder zips between rows and
// end fans onto the pinch.
func loftPinchedBand(bs geom.BSplineSurface, b *pinchedBand, pinchLo, pinchHi bool, q Quality) (*Mesh, bool) {
	m := &Mesh{}
	us := pinchedRowParams(bs, b, q)
	if len(us) < 2 {
		return nil, false
	}
	rows := pinchedBandRows(m, bs, b, us, pinchLo, pinchHi)
	if rows == nil {
		return nil, false
	}
	for i := 0; i+1 < len(rows); i++ {
		zipOpenRowsByParam(m, rows[i], rows[i+1])
	}
	addPinchFans(m, bs, rows, pinchLo, pinchHi)
	return m, true
}

// pinchedRowParams is the cross-direction (u) sampling: the open-end arc's own tessellation
// parameters when present (the v=arcV boundary then conforms to the arc edge exactly), else an
// adaptive sampling of the widest cross-section.
func pinchedRowParams(bs geom.BSplineSurface, b *pinchedBand, q Quality) []float64 {
	if b.arcUs != nil {
		return b.arcUs
	}
	vMid := widestSectionV(bs, b)
	return adaptiveParams(func(u float64) math.Point3 { return bs.PointAt(u, vMid) },
		firstOf(bs.UDomain()), secondOf(bs.UDomain()), q.Tol(), q.AngleTol())
}

func firstOf(lo, _ float64) float64  { return lo }
func secondOf(_, hi float64) float64 { return hi }

// widestSectionV scans the rail stations for the v with the widest cross-section — the density
// driver for the interior sampling.
func widestSectionV(bs geom.BSplineSurface, b *pinchedBand) float64 {
	uLo, uHi := bs.UDomain()
	best, bestW := b.loRow.ang[0], -1.0
	for _, v := range b.loRow.ang {
		if w := float64(bs.PointAt(uLo, v).DistanceTo(bs.PointAt(uHi, v))); w > bestW {
			best, bestW = v, w
		}
	}
	return best
}

// pinchedBandRows assembles all rows: boundary rows carry the rail tessellations verbatim
// (pinch endpoint samples stripped), interior rows sample the surface at the finer rail's
// stations.
func pinchedBandRows(m *Mesh, bs geom.BSplineSurface, b *pinchedBand, us []float64, pinchLo, pinchHi bool) []bandRow {
	vLo, vHi := bs.VDomain()
	loPts, loVs := stripPinchSamples(b.loPts, b.loRow.ang, vLo, vHi, pinchLo, pinchHi)
	hiPts, hiVs := stripPinchSamples(b.hiPts, b.hiRow.ang, vLo, vHi, pinchLo, pinchHi)
	base := loVs
	if len(hiVs) > len(loVs) {
		base = hiVs
	}
	if len(loPts) < 2 || len(hiPts) < 2 {
		return nil
	}
	rows := []bandRow{addPinchedRow(m, bs, loPts, loVs, us[0])}
	for _, u := range us[1 : len(us)-1] {
		pts := make([]math.Point3, len(base))
		for i, v := range base {
			pts[i] = bs.PointAt(u, v)
		}
		rows = append(rows, addPinchedRow(m, bs, pts, base, u))
	}
	return append(rows, addPinchedRow(m, bs, hiPts, hiVs, us[len(us)-1]))
}

// stripPinchSamples removes a rail row's samples AT a pinched v-end (the shared pinch vertex
// replaces them; interior samples stay).
func stripPinchSamples(pts []math.Point3, vs []float64, vLo, vHi float64, pinchLo, pinchHi bool) ([]math.Point3, []float64) {
	span := vHi - vLo
	lo, hi := 0, len(pts)
	if pinchLo && lo < hi && vs[0]-vLo <= 1e-9*span {
		lo++
	}
	if pinchHi && hi > lo && vHi-vs[hi-1] <= 1e-9*span {
		hi--
	}
	return pts[lo:hi], vs[lo:hi]
}

// addPinchedRow adds one row's vertices with surface normals evaluated slightly inset from the
// degenerate columns (the normal is undefined exactly at the pinch).
func addPinchedRow(m *Mesh, bs geom.BSplineSurface, pts []math.Point3, vs []float64, u float64) bandRow {
	vLo, vHi := bs.VDomain()
	inset := 1e-9 * (vHi - vLo)
	idx := make([]int, len(pts))
	for i, p := range pts {
		v := stdmath.Max(vLo+inset, stdmath.Min(vHi-inset, vs[i]))
		idx[i] = m.AddVertex(p, bs.NormalAt(u, v))
	}
	return bandRow{idx: idx, ang: vs}
}

// zipOpenRowsByParam stitches two OPEN rows of possibly differing counts by merging their
// stations in v order — the non-cyclic sibling of zipUnequalRows.
func zipOpenRowsByParam(m *Mesh, a, b bandRow) {
	i, j := 0, 0
	for i+1 < len(a.idx) || j+1 < len(b.idx) {
		advanceA := j+1 >= len(b.idx) || (i+1 < len(a.idx) && a.ang[i+1] <= b.ang[j+1])
		if advanceA {
			emitClosedTri(m, a.idx[i], a.idx[i+1], b.idx[j])
			i++
			continue
		}
		emitClosedTri(m, a.idx[i], b.idx[j+1], b.idx[j])
		j++
	}
}

// addPinchFans closes each pinched end with a fan from ONE shared pinch vertex across every
// row's end station — the watertight closure of the teardrop tip.
func addPinchFans(m *Mesh, bs geom.BSplineSurface, rows []bandRow, pinchLo, pinchHi bool) {
	uLo, uHi := bs.UDomain()
	vLo, vHi := bs.VDomain()
	uMid := 0.5 * (uLo + uHi)
	if pinchLo {
		p := m.AddVertex(bs.PointAt(uMid, vLo), pinchNormal(bs, uMid, vLo, vHi, true))
		for i := 0; i+1 < len(rows); i++ {
			emitClosedTri(m, p, rows[i].idx[0], rows[i+1].idx[0])
		}
	}
	if pinchHi {
		p := m.AddVertex(bs.PointAt(uMid, vHi), pinchNormal(bs, uMid, vLo, vHi, false))
		for i := 0; i+1 < len(rows); i++ {
			emitClosedTri(m, rows[i].idx[len(rows[i].idx)-1], p, rows[i+1].idx[len(rows[i+1].idx)-1])
		}
	}
}

// pinchNormal evaluates the surface normal a hair inside the degenerate column.
func pinchNormal(bs geom.BSplineSurface, u, vLo, vHi float64, atLo bool) math.Vector3 {
	inset := 1e-6 * (vHi - vLo)
	if atLo {
		return bs.NormalAt(u, vLo+inset)
	}
	return bs.NormalAt(u, vHi-inset)
}
