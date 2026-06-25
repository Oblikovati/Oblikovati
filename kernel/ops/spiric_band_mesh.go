// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// spiricBandMesh meshes the v-wrapping torus BAND a plane parallel to the axis cuts through the central
// hole (Oblikovati/Oblikovati#1375): the band swept around the tube between the two spiric ovals, the kept
// side {g ≤ 0} (through u = Phi+π). The band wraps the tube seam, so toUVLoops can't chart it; instead it
// lofts in u between the two oval edges — each boundary row is the EXACT discretization of its oval edge
// (so it welds to that oval's planar lid), interior rows fill u between the two branches at the finer oval's
// tube stations, and consecutive rows are stitched watertight by the shared band zipper. ok=false unless the
// face is a torus carrying exactly the two oval (spiric) edges.
func spiricBandMesh(f *topo.Face, s geom.Surface, q Quality) (*Mesh, bool) {
	t, isTorus := s.(geom.Torus)
	if !isTorus {
		return nil, false
	}
	plus, minus, ok := twoOvalEdges(f)
	if !ok {
		return nil, false
	}
	m := &Mesh{}
	lo := spiricRow(m, t, dropClosingDup(discretizeEdge(plus.edge, q)))
	hi := spiricRow(m, t, dropClosingDup(discretizeEdge(minus.edge, q)))
	if len(lo.idx) < 3 || len(hi.idx) < 3 {
		return nil, false
	}
	rows := []bandRow{lo}
	rows = append(rows, spiricInteriorRows(m, t, plus.arc, minus.arc, finerRowVs(lo, hi), q)...)
	rows = append(rows, hi)
	for i := 0; i+1 < len(rows); i++ {
		stitchBandRows(m, rows[i], rows[i+1])
	}
	return m, true
}

// spiricEdge pairs an oval edge with its analytic SpiricArc (the +1 / −1 branch the band lofts between).
type spiricEdge struct {
	edge *topo.Edge
	arc  geom.SpiricArc
}

// twoOvalEdges returns the face's two spiric oval edges, the +1 branch as plus and the −1 as minus. ok=false
// unless exactly two spiric edges are present (the two-oval band's boundary).
func twoOvalEdges(f *topo.Face) (plus, minus spiricEdge, ok bool) {
	var arcs []spiricEdge
	for _, e := range f.Edges() {
		if a, isSpiric := e.Geometry().(geom.SpiricArc); isSpiric {
			arcs = append(arcs, spiricEdge{edge: e, arc: a})
		}
	}
	if len(arcs) != 2 {
		return plus, minus, false
	}
	plus, minus = arcs[0], arcs[1]
	if plus.arc.Branch < 0 {
		plus, minus = minus, plus
	}
	return plus, minus, true
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

// spiricInteriorRows builds the interior loft rows: at each u-fraction between the branches, fill u from the
// +1 branch to the −1 branch (the long way, +2π, through u = Phi+π) at the given tube stations vs.
func spiricInteriorRows(m *Mesh, t geom.Torus, plus, minus geom.SpiricArc, vs []float64, q Quality) []bandRow {
	nCols := spiricBandColumns(plus, minus, q)
	rows := make([]bandRow, 0, nCols-1)
	for k := 1; k < nCols; k++ {
		frac := float64(k) / float64(nCols)
		pts := make([]math.Point3, len(vs))
		for i, v := range vs {
			uLo := plus.UAt(v)
			uHi := minus.UAt(v) + 2*stdmath.Pi
			pts[i] = t.PointAt(uLo+frac*(uHi-uLo), v)
		}
		rows = append(rows, addRow(m, t, pts, vs))
	}
	return rows
}

// spiricBandColumns picks the loft column count from the band's widest u-span and the angular tolerance, so
// even the wide ( |K|/M small ) band is faceted to the chord deflection.
func spiricBandColumns(plus, minus geom.SpiricArc, q Quality) int {
	var maxSpan float64
	for k := 0; k < 8; k++ {
		v := 2 * stdmath.Pi * float64(k) / 8
		if span := minus.UAt(v) + 2*stdmath.Pi - plus.UAt(v); span > maxSpan {
			maxSpan = span
		}
	}
	if n := int(stdmath.Ceil(maxSpan / q.angleTol())); n > 2 {
		return n
	}
	return 2
}

// finerRowVs returns the tube parameters of whichever boundary row has more samples — the interior loft
// rows reuse them so a clean quad strip forms against the finer oval (the other zips).
func finerRowVs(a, b bandRow) []float64 {
	if len(a.ang) >= len(b.ang) {
		return a.ang
	}
	return b.ang
}
