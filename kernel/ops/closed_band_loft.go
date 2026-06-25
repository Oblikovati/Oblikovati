// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"sort"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// closedBandLoftMesh meshes a closed curved band whose two edge circles can carry DIFFERENT vertex
// counts — a torus rim-fillet band, whose cap-tangent circle (smaller radius) tessellates coarser
// than its cyl-tangent circle. closedDomainMesh grids the wrap direction at ONE resolution (a
// `PointAt` grid), so it conforms only to neighbours sharing that resolution — fine for a cylinder
// wall (same-radius caps), but it cracks a torus band against its plane cap and cylinder wall at
// fine tolerance, and neither (a plane, a closed periodic band) can be re-meshed by the conformance
// pass. This lofts the tube instead: each boundary ring is the EXACT tessellation of its own circle
// edge (so it matches whatever neighbour shares that edge), the interior rings sample the finer
// ring's angular stations, and consecutive rings are stitched watertight even at differing counts.
//
// The two rings are read from the face's boundary EDGES — the two closed circles + the seam arc
// (its sample count sets the tube subdivision) — NOT from the boundary point soup, whose seam-arc
// points all share angle 0 and pollute the v-partition. ok=false unless the band is exactly two
// circle edges + one arc seam.
func closedBandLoftMesh(f *topo.Face, s geom.Surface, q Quality) (*Mesh, bool) {
	rings, seamN, seamMid, ok := bandRingsAndSeam(f, q)
	if !ok || len(rings) != 2 || seamN < 2 {
		return nil, false
	}
	lo, hi := orderedRing(s, rings[0]), orderedRing(s, rings[1])
	if len(lo) < 3 || len(hi) < 3 {
		return nil, false
	}
	// Loft from the lo ring to the hi ring ALONG THE SEAM — the seam's midpoint tube parameter says which
	// of the two arcs between the rings the band spans. A torus cut keeps the long (seam-wrapping) arc;
	// using min→max would loft the short complement instead (the dropped tube arc, Oblikovati#1375).
	vLo, vHiRaw, vMid := vParam(s, lo[0]), vParam(s, hi[0]), vParam(s, seamMid)
	vHi := vLo + wrapPi(vMid-vLo) + wrapPi(vHiRaw-vMid) // unwrapped so vLo→vMid→vHi is monotone
	mid := make([]float64, 0, seamN-2)
	for k := 1; k < seamN-1; k++ {
		mid = append(mid, vLo+(vHi-vLo)*float64(k)/float64(seamN-1))
	}
	return loftRows(s, lo, hi, mid), true
}

// wrapPi folds an angle into (−π, π] — the signed shortest step between two tube parameters.
func wrapPi(a float64) float64 {
	const twoPi = 2 * stdmath.Pi
	for a > stdmath.Pi {
		a -= twoPi
	}
	for a <= -stdmath.Pi {
		a += twoPi
	}
	return a
}

// bandRingsAndSeam reads the band face's boundary edges: each closed circle (with its closing
// duplicate dropped) is a ring; seamN is the largest arc-edge sample count (the tube subdivision) and
// seamMid that arc's midpoint (whose tube parameter picks which arc the band spans). ok=false with no seam.
func bandRingsAndSeam(f *topo.Face, q Quality) (rings [][]math.Point3, seamN int, seamMid math.Point3, ok bool) {
	for _, e := range f.Edges() {
		switch e.Geometry().(type) {
		case geom.Circle:
			rings = append(rings, dropClosingDup(TessellateEdge(e, q)))
		case geom.Arc3d:
			if pts := TessellateEdge(e, q); len(pts) > seamN {
				seamN, seamMid, ok = len(pts), pts[len(pts)/2], true
			}
		}
	}
	return rings, seamN, seamMid, ok
}

// dropClosingDup removes the trailing point a closed-edge tessellation repeats at its seam. The
// duplicate threshold is model-relative (ADR-0042, #1399), scaling with the ring's extent.
func dropClosingDup(pts []math.Point3) []math.Point3 {
	if len(pts) > 1 && pts[0].DistanceTo(pts[len(pts)-1]) < ResolutionForPoints(pts).Weld() {
		return pts[:len(pts)-1]
	}
	return pts
}

// vParam returns a point's tube parameter (the surface's v).
func vParam(s geom.Surface, p math.Point3) float64 { _, v := s.ParamAt(p); return v }

// orderedRing sorts a ring's points ascending by angle around the surface axis.
func orderedRing(s geom.Surface, pts []math.Point3) []math.Point3 {
	out := append([]math.Point3(nil), pts...)
	sort.SliceStable(out, func(i, j int) bool { return angleOf(s, out[i]) < angleOf(s, out[j]) })
	return out
}

// bandRow is one loft cross-section: the vertex indices around the ring and their INTENDED angles
// (the tessellation/grid stations, NOT round-tripped through ParamAt — round-trip noise would make
// the zipper's per-station order inconsistent between rows and overlap triangles).
type bandRow struct {
	idx []int
	ang []float64
}

// loftRows builds the row vertices (the two edge rings at their own tessellations, the interior
// rings at the finer ring's angular stations) and stitches consecutive rows watertight.
func loftRows(s geom.Surface, lo, hi []math.Point3, midVs []float64) *Mesh {
	m := &Mesh{}
	loA, hiA := ringAngles(s, lo), ringAngles(s, hi)
	baseU := loA
	if len(hi) > len(lo) {
		baseU = hiA // loft the interior at the finer ring's stations
	}
	rows := []bandRow{addRow(m, s, lo, loA)}
	for _, v := range midVs {
		rows = append(rows, addRow(m, s, sampleRow(s, baseU, v), baseU))
	}
	rows = append(rows, addRow(m, s, hi, hiA))
	for i := 0; i+1 < len(rows); i++ {
		stitchBandRows(m, rows[i], rows[i+1])
	}
	return m
}

// ringAngles returns the angles of an ordered ring.
func ringAngles(s geom.Surface, ring []math.Point3) []float64 {
	out := make([]float64, len(ring))
	for i, p := range ring {
		out[i] = angleOf(s, p)
	}
	return out
}

// sampleRow evaluates the surface at angular stations us and tube parameter v.
func sampleRow(s geom.Surface, us []float64, v float64) []math.Point3 {
	out := make([]math.Point3, len(us))
	for i, u := range us {
		out[i] = s.PointAt(u, v)
	}
	return out
}

// addRow adds a row's vertices (with surface normals) and returns the row with its intended angles.
func addRow(m *Mesh, s geom.Surface, pts []math.Point3, ang []float64) bandRow {
	idx := make([]int, len(pts))
	for i, p := range pts {
		u, v := s.ParamAt(p)
		idx[i] = m.addVertex(p, s.NormalAt(u, v))
	}
	return bandRow{idx: idx, ang: ang}
}

// angleOf returns a point's angle around the surface axis (its u parameter).
func angleOf(s geom.Surface, p math.Point3) float64 {
	u, _ := s.ParamAt(p)
	return u
}

// stitchBandRows triangulates the closed strip between two rows. When they share a station count
// (an aligned pair, both at the same angular stations) it emits a clean quad strip by index; when
// the counts differ (the coarser edge circle against the finer interior) it zips them in angle order
// using the intended angles. emitClosedTri winds each triangle outward by its vertex normals.
func stitchBandRows(m *Mesh, a, b bandRow) {
	if len(a.idx) < 3 || len(b.idx) < 3 {
		return
	}
	if len(a.idx) == len(b.idx) {
		n := len(a.idx)
		for i := 0; i < n; i++ {
			ni := (i + 1) % n
			emitClosedTri(m, a.idx[i], a.idx[ni], b.idx[ni])
			emitClosedTri(m, a.idx[i], b.idx[ni], b.idx[i])
		}
		return
	}
	zipUnequalRows(m, a, b)
}

// zipUnequalRows stitches two rings of DIFFERING counts by merging their vertices in intended-angle
// order: each vertex forms a triangle with the running vertex of the opposite ring (a zipper),
// closing the seam. The intended angles (not ParamAt round-trips) keep the per-station order stable.
func zipUnequalRows(m *Mesh, a, b bandRow) {
	type event struct {
		ang float64
		v   int
		isA bool
	}
	evs := make([]event, 0, len(a.idx)+len(b.idx))
	for i, v := range a.idx {
		evs = append(evs, event{a.ang[i], v, true})
	}
	for i, v := range b.idx {
		evs = append(evs, event{b.ang[i], v, false})
	}
	sort.SliceStable(evs, func(i, j int) bool {
		if stdmath.Abs(evs[i].ang-evs[j].ang) > 1e-9 {
			return evs[i].ang < evs[j].ang
		}
		return evs[i].isA && !evs[j].isA // break ties A-before-B so an aligned pair makes a clean quad
	})
	curA, curB := a.idx[len(a.idx)-1], b.idx[len(b.idx)-1] // largest-angle verts: the first events bridge the seam
	for _, e := range evs {
		if e.isA {
			emitClosedTri(m, curA, e.v, curB)
			curA = e.v
		} else {
			emitClosedTri(m, curB, e.v, curA)
			curB = e.v
		}
	}
}
