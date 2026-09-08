// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	stdmath "math"
	"sort"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// spiricBandMesh meshes a v-wrapping torus BAND: the strip swept around the tube between two edges that
// each go the whole way round it. A plane parallel to the axis cutting through the central hole leaves
// one, bounded by the two spiric ovals (Oblikovati/Oblikovati#1375); so does a ball swallowing a stretch
// of the tube, bounded by two torus∩quadric sections (ADR-0061 stage 5). The band wraps the tube seam,
// so toUVLoops cannot chart it; instead it lofts in u between the two edges — each boundary row is the
// EXACT discretization of its edge, so it welds to whatever meets it there; interior rows fill u between
// the branches at the finer edge's tube stations; and consecutive rows are stitched watertight by the
// shared band zipper.
//
// What it accepts is the SHAPE, not the curve kind: a torus face with exactly two edges that each WRAP
// the tube. That test is also the guard the spiric-only version needed for its own reason — an arc
// fillet run out on a side plane at each end carries one QUARTER-tube spiric section per end, cut by two
// different planes, and lofting between those sweeps the whole tube (measured on simple/W2, whose 0.418
// band read 4.9146, 52% of the entire torus). Neither of those wraps, so neither reaches here.
func spiricBandMesh(f *topo.Face, b spiricTubeTrim, q Quality) (*Mesh, bool) {
	t, first, second := b.torus, b.first, b.second
	m := &Mesh{}
	lo := spiricRow(m, t, dropClosingDup(DiscretizeEdge(first, q)))
	hi := spiricRow(m, t, dropClosingDup(DiscretizeEdge(second, q)))
	if len(lo.idx) < 3 || len(hi.idx) < 3 {
		return nil, false
	}
	loU, hiU := branchAzimuthAt(m, t, first, lo), branchAzimuthAt(m, t, second, hi)
	vs := finerRowVs(lo, hi)
	dir := bandDirection(f.Chart(), loU, hiU, vs)
	rows := []bandRow{lo}
	rows = append(rows, tubeInteriorRows(m, t, loU, hiU, vs, dir, q)...)
	rows = append(rows, hi)
	for i := 0; i+1 < len(rows); i++ {
		stitchBandRows(m, rows[i], rows[i+1])
	}
	return m, true
}

// tubeWrappingEdges returns the face's two edges that each go the whole way round the TUBE, which is
// what makes the face a band the loft can sweep. ok=false for any other count: one such edge bounds a
// cap, none bounds an ordinary patch, and three or more is not a band.
func tubeWrappingEdges(f *topo.Face, s geom.Surface, q Quality) (t geom.Torus, first, second *topo.Edge, ok bool) {
	t, isTorus := s.(geom.Torus)
	if !isTorus {
		return t, nil, nil, false
	}
	var wrapping []*topo.Edge
	for _, e := range f.Edges() {
		if edgeWrapsTheTube(t, DiscretizeEdge(e, q)) {
			wrapping = append(wrapping, e)
		}
	}
	if len(wrapping) != 2 {
		return t, nil, nil, false
	}
	return t, wrapping[0], wrapping[1], true
}

// edgeWrapsTheTube reports whether a discretised edge's NET turn around the tube is a whole period. It
// is the net, so a chain that runs part way round and back — a quarter-tube fillet section, a cap's rim
// — turns by less and is not a band boundary.
func edgeWrapsTheTube(t geom.Torus, pts []math.Point3) bool {
	if len(pts) < 3 {
		return false
	}
	turn, prev := 0.0, vParam(t, pts[0])
	for _, p := range pts[1:] {
		v := unwrapNearAngle(prev, vParam(t, p))
		turn, prev = turn+v-prev, v
	}
	return stdmath.Abs(turn) > stdmath.Pi // a net turn past a half period can only be the whole one
}

// unwrapNearAngle carries an angle onto the branch nearest a reference, so a walk of a periodic
// coordinate accumulates its true turn instead of a saw-tooth.
func unwrapNearAngle(ref, a float64) float64 {
	return a - 2*stdmath.Pi*stdmath.Round((a-ref)/(2*stdmath.Pi))
}

// spiricRow adds a band row from exact 3D boundary points, keyed by each point's tube parameter v (so the
// zipper merges rows of differing counts in tube order).
func spiricRow(m *Mesh, t geom.Torus, pts []math.Point3) bandRow {
	ang := make([]float64, len(pts))
	for i, p := range pts {
		ang[i] = vParam(t, p)
	}
	return addRow(m, t, pts, ang)
}

// branchAzimuthAt returns the azimuth one boundary reaches at any tube station. A spiric oval carries a
// closed form for it and keeps it: the interior rows are then exact where they were before, which is
// what the figure-eight pinch needs — its band closes to zero width at the tangency, and a boundary
// read from samples rounds that corner. Any other section curve has no such form, and the row IS the
// edge's agreed discretisation — the same points whatever meets the edge meshes — so interpolating it
// keeps the loft's interior consistent with the boundary the mesh actually carries.
func branchAzimuthAt(m *Mesh, t geom.Torus, e *topo.Edge, row bandRow) func(float64) float64 {
	if arc, isSpiric := e.Geometry().(geom.SpiricArc); isSpiric {
		return arc.UAt
	}
	return sampledAzimuthAt(m, t, row)
}

// sampledAzimuthAt reads a boundary's azimuth from its own row samples.
func sampledAzimuthAt(m *Mesh, t geom.Torus, row bandRow) func(float64) float64 {
	us := make([]float64, len(row.idx))
	for i, ix := range row.idx {
		us[i], _ = t.ParamAt(m.Positions[ix])
	}
	return azimuthInterpolator(append([]float64(nil), row.ang...), us)
}

// azimuthInterpolator interpolates u over the tube period from a boundary's (v, u) samples. Both axes
// are periodic, so the stations are read as a closed ring and each u is carried onto the branch nearest
// its predecessor before interpolating — a boundary crossing the azimuth seam is one curve, not a jump.
func azimuthInterpolator(vs, us []float64) func(float64) float64 {
	if len(vs) == 0 {
		return func(float64) float64 { return 0 }
	}
	order := ringOrderByAngle(vs)
	sv, su := make([]float64, len(order)), make([]float64, len(order))
	for i, k := range order {
		sv[i] = vs[k]
		su[i] = us[k]
		if i > 0 {
			su[i] = unwrapNearAngle(su[i-1], su[i])
		}
	}
	return func(v float64) float64 { return interpolateOnRing(sv, su, wrapToPeriod(v)) }
}

// interpolateOnRing reads the ring's value at station v, the stations taken as a closed cycle so the
// segment spanning the seam is one segment like any other. Each segment's far end is carried onto the
// branch nearest its near end, so the interpolation follows the curve instead of averaging across a
// whole turn.
func interpolateOnRing(sv, su []float64, v float64) float64 {
	for i := range sv {
		lo, hi := sv[i], sv[(i+1)%len(sv)]
		if hi < lo {
			hi += 2 * stdmath.Pi
		}
		w := v
		if w < lo {
			w += 2 * stdmath.Pi
		}
		if w > hi {
			continue
		}
		if hi == lo {
			return su[i]
		}
		far := unwrapNearAngle(su[i], su[(i+1)%len(su)])
		return su[i] + (far-su[i])*(w-lo)/(hi-lo)
	}
	return su[0]
}

// ringOrderByAngle returns the indices of vs in ascending angle, each folded onto one period.
func ringOrderByAngle(vs []float64) []int {
	order := make([]int, len(vs))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(i, j int) bool {
		return wrapToPeriod(vs[order[i]]) < wrapToPeriod(vs[order[j]])
	})
	return order
}

// wrapToPeriod folds an angle onto [0, 2π).
func wrapToPeriod(a float64) float64 {
	a = stdmath.Mod(a, 2*stdmath.Pi)
	if a < 0 {
		a += 2 * stdmath.Pi
	}
	return a
}

// bandDirection is which way round the tube the loft travels from the first boundary to the second:
// +1 forward in u, −1 backward. Those are the only two candidates, and they are the two bands the pair
// of boundaries bounds — the strip directly between them and the band the other way round. The FACE
// says which, through its chart (ADR-0063), and it says it by AREA: the chart's own (u,v) area over the
// tube period is the mean azimuth width of the region the face covers, and the forward band's mean
// width plus the backward band's is exactly one period.
//
// Area rather than a containment test, because the complement band WRAPS the azimuth and the chart
// records it as two contours split at the chart's own seam. An even-odd test then has to fold the query
// onto each contour's branch separately, and a point can land inside both — which reads as outside, and
// sent every cut and union round the short way (ADR-0061 stage 5).
//
// A face without a chart travels forward, which is the direction the spiric-only loft always took.
func bandDirection(chart [][]math.Point2, loU, hiU func(float64) float64, vs []float64) float64 {
	if len(chart) == 0 || len(vs) == 0 {
		return 1
	}
	forward := meanBandWidth(loU, hiU, vs, 1)
	want := chartMeanWidth(chart)
	if stdmath.Abs(want-forward) <= stdmath.Abs(want-(2*stdmath.Pi-forward)) {
		return 1
	}
	return -1
}

// bandWidthAt is how far to travel in u from the first boundary to the second in the given direction —
// the gap folded onto ONE period and signed by dir, so every station crosses the SAME band.
//
// Folding is what makes it safe: an azimuth read from a boundary's own samples carries an arbitrary
// whole turn (the samples were unwrapped along the curve), so a raw difference can be a period out at
// one station and not at the next. The loft then varies its width by 2π across the band and covers the
// tube more than once — measured as 636 mm³ on a torus of 395 (ADR-0061 stage 5).
func bandWidthAt(loU, hiU func(float64) float64, v, dir float64) float64 {
	forward := wrapToPeriod(hiU(v) - loU(v))
	if dir > 0 {
		return forward
	}
	// The backward travel is the REST of the period, which is a period when the two boundaries meet.
	// Folding the reversed difference instead would answer zero there, and the band would collapse at
	// exactly the station where it is widest: the figure-eight's two ovals touch at their tangency, and
	// the complement band goes the whole way round the tube precisely there (ADR-0061 stage 5).
	return forward - 2*stdmath.Pi
}

// meanBandWidth is the mean magnitude of that travel over the boundary's own stations.
func meanBandWidth(loU, hiU func(float64) float64, vs []float64, dir float64) float64 {
	sum := 0.0
	for _, v := range vs {
		sum += stdmath.Abs(bandWidthAt(loU, hiU, v, dir))
	}
	return sum / float64(len(vs))
}

// chartMeanWidth is the chart's own (u,v) area spread over the tube period — the mean azimuth width of
// the region the face actually covers, which is what the two candidate bands are compared against.
func chartMeanWidth(chart [][]math.Point2) float64 {
	area := 0.0
	for _, contour := range chart {
		if len(contour) < 3 {
			continue
		}
		sum := 0.0
		for i := range contour {
			a, b := contour[i], contour[(i+1)%len(contour)]
			sum += float64(a.X*b.Y - b.X*a.Y)
		}
		area += stdmath.Abs(sum / 2)
	}
	return area / (2 * stdmath.Pi)
}

// tubeInteriorRows builds the interior loft rows: at each fraction of the way across the band, fill u
// from the first boundary toward the second in the chosen direction, at the given tube stations. The
// rows always run the same way from the first row, so the zipper winds them alike.
func tubeInteriorRows(m *Mesh, t geom.Torus, loU, hiU func(float64) float64, vs []float64, dir float64, q Quality) []bandRow {
	nCols := tubeBandColumns(loU, hiU, vs, dir, q)
	rows := make([]bandRow, 0, nCols-1)
	for k := 1; k < nCols; k++ {
		frac := float64(k) / float64(nCols)
		pts := make([]math.Point3, len(vs))
		for i, v := range vs {
			pts[i] = t.PointAt(loU(v)+frac*bandWidthAt(loU, hiU, v, dir), v)
		}
		rows = append(rows, addRow(m, t, pts, vs))
	}
	return rows
}

// tubeBandColumns picks the loft column count from the band's widest travel and the angular tolerance,
// so even a wide band is faceted to the chord deflection.
func tubeBandColumns(loU, hiU func(float64) float64, vs []float64, dir float64, q Quality) int {
	var maxSpan float64
	for _, v := range vs {
		if w := stdmath.Abs(bandWidthAt(loU, hiU, v, dir)); w > maxSpan {
			maxSpan = w
		}
	}
	if n := int(stdmath.Ceil(maxSpan / q.AngleTol())); n > 2 {
		return n
	}
	return 2
}

// finerRowVs returns the tube parameters of whichever boundary row has more samples — the interior loft
// rows reuse them so a clean quad strip forms against the finer boundary (the other zips).
func finerRowVs(a, b bandRow) []float64 {
	if len(a.ang) >= len(b.ang) {
		return a.ang
	}
	return b.ang
}
