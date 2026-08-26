// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"slices"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// canalRimBandMesh meshes the CLOSED elliptic-rim canal band (fillet_elliptic_rim_canal.go): a
// geom.BSplineSurface face bounded by TWO closed contact rails (its u=0 and u=1 isocurves, shared with
// the receded wall and cap) plus one seam cross-section arc used twice. It is the canal counterpart of
// closedBandLoftMesh, which serves the analytic TORUS rim band — that one keys on the surface's
// periodic (u = around the axis, v = tube) frame, which a canal band does not have: here the ring
// direction is v (around the rim) and the cross direction is u (across the arc), and NEITHER is
// periodic. Rather than contort the torus path (and risk its greens), this is a separate entry.
//
// It exists because the general trimmed-B-spline meshers cannot see this face: a loop whose edges are
// two CLOSED curves has no usable planar (u,v) boundary, and the surface's own v=0/v=1 isocurves
// coincide (the band closes on itself), so a pcurve/CDT trim degenerates. Lofting instead makes each
// boundary ring the EXACT tessellation of its own rail edge, so the band conforms to both neighbours
// by construction, and the interior rows are sampled on the surface.
func canalRimBandMesh(f *topo.Face, s geom.Surface, q Quality) (*Mesh, bool) {
	// The O(4) topological loop check runs FIRST: closesAlongV costs 10 surface evaluations plus a full
	// control-net sweep, and splineFaceMesh reaches here for EVERY B-spline face — on an imported model
	// with thousands of large NURBS faces that scan would be paid per face per tessellation.
	loE, hiE, seamE, ok := canalBandBoundary(f)
	if !ok {
		return nil, false
	}
	bs, isSpline := s.(geom.BSplineSurface)
	if !isSpline || !closesAlongV(bs) {
		return nil, false
	}
	loE, hiE, ok = orderRailsByU(s, loE, hiE)
	if !ok {
		return nil, false
	}
	uLo, uHi := s.UDomain()
	lo, loV, ok1 := canalRailRow(s, loE, uLo, q)
	hi, hiV, ok2 := canalRailRow(s, hiE, uHi, q)
	seamN := len(TessellateEdge(seamE, q))
	if !ok1 || !ok2 || seamN < 2 {
		return nil, false
	}
	return canalLoftRows(s, lo, loV, hi, hiV, seamN), true
}

// canalBandBoundary reads the band face's single loop and splits it into the u=0 rail, the u=1 rail
// and the seam arc. ok=false unless the loop is EXACTLY the canal band's four uses — two distinct
// CLOSED edges plus one open edge used twice — so no other B-spline face in the kernel (every canal
// arm and corner patch is an open four-rail patch) can be diverted here.
func canalBandBoundary(f *topo.Face) (loE, hiE, seamE *topo.Edge, ok bool) {
	loops := f.Loops()
	if len(loops) != 1 || len(loops[0].EdgeUses()) != 4 {
		return nil, nil, nil, false
	}
	var closed []*topo.Edge
	seamUses := 0
	for _, u := range loops[0].EdgeUses() {
		e := u.Edge()
		if e.StartVertex() == e.EndVertex() {
			closed = appendDistinctEdge(closed, e)
			continue
		}
		if seamE != nil && seamE != e {
			return nil, nil, nil, false
		}
		seamE, seamUses = e, seamUses+1
	}
	if len(closed) != 2 || seamUses != 2 {
		return nil, nil, nil, false
	}
	return closed[0], closed[1], seamE, true
}

// canalBandSeamTol is the RELATIVE closure test for "the surface joins itself at v": the canal band's
// last station repeats its first EXACTLY (the same control column), so its two v-ends coincide to
// round-off. Scaling by the control-net extent keeps it model-relative (ADR-0042).
const canalBandSeamTol = 1e-9

// closesAlongV reports that the surface rejoins itself along v and NOT along u — the canal band's
// signature, and what separates it from an imported tube patch, whose closed boundary edges look the
// same to a loop-shape test but whose closure runs the OTHER way (its two closed edges are v-isos, so
// lofting it rail-to-rail across u would mesh the wrong direction and crack it, #TestImportedNurbsDuct).
func closesAlongV(s geom.BSplineSurface) bool {
	uLo, uHi := s.UDomain()
	vLo, vHi := s.VDomain()
	tol := canalBandSeamTol * ctrlNetExtent(s)
	vGap, uGap := 0.0, 0.0
	for k := 0; k <= 4; k++ {
		t := float64(k) / 4
		u := uLo + (uHi-uLo)*t
		v := vLo + (vHi-vLo)*t
		vGap = stdmath.Max(vGap, float64(s.PointAt(u, vLo).DistanceTo(s.PointAt(u, vHi))))
		uGap = stdmath.Max(uGap, float64(s.PointAt(uLo, v).DistanceTo(s.PointAt(uHi, v))))
	}
	return vGap <= tol && uGap > tol
}

// ctrlNetExtent is the control net's bounding-box diagonal — the surface's own size, the scale the
// closure tolerance is relative to.
func ctrlNetExtent(s geom.BSplineSurface) float64 {
	box := math.EmptyBox()
	for _, row := range s.Ctrl {
		for _, p := range row {
			box = box.ExtendPoint(p)
		}
	}
	return float64(box.Max.DistanceTo(box.Min))
}

// orderRailsByU returns the two rails ordered (u = uLo rail, u = uHi rail), or ok=false when either
// is not one of those two isocurves. The loop traversal order carries no such promise, and lofting
// them the wrong way round makes every strip span the WHOLE cross-section arc instead of one slice of
// it (a ~3× area inflation, silently valid-looking). The side is read by inverting ONE point of each
// rail onto the surface — two inversions per band.
func orderRailsByU(s geom.Surface, a, b *topo.Edge) (loE, hiE *topo.Edge, ok bool) {
	uLo, uHi := s.UDomain()
	ua, _ := s.ParamAt(a.Geometry().PointAt(edgeStartParam(a)))
	ub, _ := s.ParamAt(b.Geometry().PointAt(edgeStartParam(b)))
	span := stdmath.Abs(uHi - uLo)
	near := func(x, target float64) bool { return stdmath.Abs(x-target) <= canalRailIsoTol*span }
	switch {
	case near(ua, uLo) && near(ub, uHi):
		return a, b, true
	case near(ub, uLo) && near(ua, uHi):
		return b, a, true
	}
	return nil, nil, false // neither rail is a u-boundary isocurve — not a canal band
}

// canalRailIsoTol is the fraction of the u span within which a rail's inverted parameter must sit at
// a u boundary to count as that boundary's isocurve. Generous (point inversion on a fine canal net is
// noisy near a boundary) yet far from the mid-domain, so a non-boundary curve cannot pass.
const canalRailIsoTol = 0.05

// edgeStartParam is an edge curve's domain start.
func edgeStartParam(e *topo.Edge) float64 { lo, _ := e.Geometry().Domain(); return lo }

// appendDistinctEdge appends e unless it is already in the list (a closed rail is used once, but the
// guard keeps the classification total).
func appendDistinctEdge(list []*topo.Edge, e *topo.Edge) []*topo.Edge {
	if slices.Contains(list, e) {
		return list
	}
	return append(list, e)
}

// canalRailRow tessellates one closed rail edge and returns its points with their TRUE ring parameters
// — each sample's own curve parameter, which IS the surface's v: the rail is the surface's u=uIso
// isocurve, extracted from the same v knot vector (geom.SurfaceIsoCurve → bsplineIso) and stored on the
// edge verbatim by the rim rebuild, so C(t) == S(uIso, t) to round-off (railIsSurfaceVIso asserts it).
//
// It must NOT be re-derived from chord length along the rail, as this did before: the surface's v is
// chord length on the SPINE CENTRES (spineChordParams → geom.LoftCanalStations), and on a canal of
// VARIABLE section the rail's own chord distribution is a DIFFERENT reparametrisation of the same ring
// — 1.2e-2 of the v span apart on J6's wall rail. Every rail vertex then carried a v that was not its
// own parameter, so its normal (canalAddRow) and the interior rows sampled at those v's (canalLoftRows)
// were evaluated at the wrong station, shearing every boundary strip ALONG the ring: the MESHED J6 band
// came out 7258.36 against the exact envelope area 7240.851 (+0.24%), while J8's near-circular rail —
// whose two distributions nearly agree — was 100x closer. Tessellation correctness outranks features
// (CLAUDE.md), and the meshed area is what mass properties, export and render all consume.
// ★ RECORDED HAZARD, measured not fixed (rimcrossings-report.md §sweep). This is the one remaining
// read of a SHARED face boundary that does not go through discretizeEdge: the two rails are closed
// edges with two uses each, so the neighbour on the other side tiles them through discretizeEdge while
// this loft tiles them from the raw sampler. It cannot simply be swapped, because the row NEEDS each
// sample's own curve PARAMETER (the surface's v, see above) and discretizeEdge returns points only — a
// healed edge's stored on-surface polyline (M25) has no curve parameters at all, and densifyStarvedRail
// inserts points with none. MEASURED on every canal band the corpus builds (simple/J4·J6·J8,
// bfuseblend/A3·A7·B1 — 16 rail reads): the two agree EXACTLY today, identical sample counts (109…1025)
// and worst point deviation 0.000e+00, because no canal rail is healed (SnappedCurve nil on all 16) and
// none is straight enough for the #2009 densification to fire. So this is a LATENT hazard, not a live
// crack: it becomes one the day a canal band's rail arrives healed from an import or gains a
// high-aspect B-spline neighbour. Closing it properly needs a parameter-carrying shared discretization.
func canalRailRow(s geom.Surface, e *topo.Edge, uIso float64, q Quality) ([]math.Point3, []float64, bool) {
	pts, vs := tessellateEdgeWithParams(e, q)
	pts, vs = dropClosingDupWithParams(pts, vs)
	if len(pts) < 3 || !railIsSurfaceVIso(s, e, uIso, pts) {
		return nil, nil, false
	}
	return pts, vs, true
}

// dropClosingDupWithParams is dropClosingDup keeping each point's parameter in lockstep with it — a
// closed rail's last sample repeats its first, and dropping the point without its v would shift every
// label by one station.
func dropClosingDupWithParams(pts []math.Point3, ts []float64) ([]math.Point3, []float64) {
	kept := len(dropClosingDup(pts))
	return pts[:kept], ts[:kept]
}

// canalRailIsoProbes is how many parameters railIsSurfaceVIso compares the rail against the isocurve
// at. The two agree to round-off wherever they agree at all (the same basis functions over the same knot
// vector), so a handful of probes spread over the span is decisive.
const canalRailIsoProbes = 8

// canalRailIsoParamTol is the dimensionless fraction of the v span within which the rail curve's own
// domain must match the surface's v domain to be usable as a v label. The band's rails come from
// geom.SurfaceIsoCurve, so they carry the surface's v knot vector EXACTLY; this is a guard against
// labelling with a parameter from some other range, not a fit tolerance.
const canalRailIsoParamTol = 1e-9 // tol:numeric (dimensionless fraction of the v span)

// railIsSurfaceVIso verifies the premise canalRailRow's v labels rest on: that the rail edge's curve is
// the surface's u=uIso isocurve WITH THE SAME PARAMETRISATION, so a curve parameter can be handed to the
// surface as a v verbatim. The elliptic-rim band's rails are built exactly that way
// (assembleEllipticRimCanal reads them off geom.SurfaceIsoCurve and rimBuild.addRimEdges stores them
// un-refitted), so it holds there to round-off; any other face is DECLINED rather than lofted on a false
// premise — the same do-no-harm direction as the rest of this file's gates.
func railIsSurfaceVIso(s geom.Surface, e *topo.Edge, uIso float64, pts []math.Point3) bool {
	c := e.Geometry()
	lo, hi := c.Domain()
	vLo, vHi := s.VDomain() // the ring direction: the band closes along v
	span := stdmath.Abs(vHi - vLo)
	if stdmath.Abs(lo-vLo) > canalRailIsoParamTol*span || stdmath.Abs(hi-vHi) > canalRailIsoParamTol*span {
		return false // the rail is parametrised over a different range than the surface's v
	}
	tol := ResolutionForPoints(pts).Weld()
	for k := 0; k <= canalRailIsoProbes; k++ {
		t := lo + (hi-lo)*float64(k)/canalRailIsoProbes
		if float64(c.PointAt(t).DistanceTo(s.PointAt(uIso, t))) > tol {
			return false
		}
	}
	return true
}

// canalLoftRows builds the band mesh: the two rail rings at their OWN edge tessellations (so each
// conforms to the neighbour sharing that rail) and seamN−2 interior rows across the arc, sampled at the
// FINER ring's stations, stitched with the shared closed-band zipper.
func canalLoftRows(s geom.Surface, lo []math.Point3, loV []float64, hi []math.Point3, hiV []float64, seamN int) *Mesh {
	m := &Mesh{}
	baseV := loV
	if len(hi) > len(lo) {
		baseV = hiV
	}
	uLo, uHi := s.UDomain()
	rows := []bandRow{canalAddRow(m, s, lo, loV, uLo)}
	for k := 1; k < seamN-1; k++ {
		u := uLo + (uHi-uLo)*float64(k)/float64(seamN-1)
		pts := make([]math.Point3, len(baseV))
		for i, v := range baseV {
			pts[i] = s.PointAt(u, v)
		}
		rows = append(rows, canalAddRow(m, s, pts, baseV, u))
	}
	rows = append(rows, canalAddRow(m, s, hi, hiV, uHi))
	for i := 0; i+1 < len(rows); i++ {
		stitchBandRows(m, rows[i], rows[i+1])
	}
	return m
}

// canalAddRow adds one cross-section row's vertices (with the surface normal at its own (u,v)) and
// returns it keyed by the ring parameter v — the ordering key stitchBandRows zips unequal rows on.
func canalAddRow(m *Mesh, s geom.Surface, pts []math.Point3, vs []float64, u float64) bandRow {
	idx := make([]int, len(pts))
	for i, p := range pts {
		idx[i] = m.addVertex(p, s.NormalAt(u, vs[i]))
	}
	return bandRow{idx: idx, ang: vs}
}
