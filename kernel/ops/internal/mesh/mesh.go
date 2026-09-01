// SPDX-License-Identifier: GPL-2.0-only

// Package mesh holds the tessellation VALUE TYPES — the triangle mesh a tessellator
// produces and the tolerance pair that controls its density.
//
// They live below kernel/ops because every operation family needs them: the blend
// engine, the boolean, the validator and the query layer all speak Mesh and Quality,
// so leaving them in kernel/ops made each of those an ops-internal symbol and any
// split of the package impossible. kernel/ops re-exports both as type aliases, so no
// call site outside the kernel changes (ops.Mesh IS mesh.Mesh).
//
//	m := &mesh.Mesh{}
//	i := m.AddVertex(p, n)
package mesh

import (
	stdmath "math"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/ops/internal/probe"
	"oblikovati.org/math"
)

// Mesh is a triangle mesh for display/export: positions, per-vertex normals, and
// triangle indices (triples). float64 here; the renderer narrows to float32 at the
// GPU boundary (core/03).
type Mesh struct {
	Positions []math.Point3
	Normals   []math.Vector3
	Indices   []int
	// Diagnostics carries any degradation the tessellator recorded while building this mesh — a cap it
	// saturated, an exact path it declined — the diag channel's tessellation carrier, so a consumer sees
	// the mesh may not meet tolerance instead of discovering it downstream (Oblikovati#1412). Empty means
	// a clean tessellation.
	Diagnostics []diag.Diagnostic
}

// Diagnose records a diagnostic onto the mesh — the emission site for a tessellator that degraded while
// building it (e.g. saturated the interior-cell cap below the chord tolerance, #1412).
func (m *Mesh) Diagnose(d diag.Diagnostic) { m.Diagnostics = append(m.Diagnostics, d) }

// CarryDiagnostics lifts a sub-mesh's diagnostics into m when the tessellator composes meshes, so a
// degradation recorded deep in a component surfaces on the final face mesh.
func (m *Mesh) CarryDiagnostics(child *Mesh) {
	if child != nil && len(child.Diagnostics) > 0 {
		m.Diagnostics = append(m.Diagnostics, child.Diagnostics...)
	}
}

// TriangleCount returns the number of triangles.
func (m *Mesh) TriangleCount() int { return len(m.Indices) / 3 }

// VertexCount returns the number of vertices.
func (m *Mesh) VertexCount() int { return len(m.Positions) }

func (m *Mesh) AddVertex(p math.Point3, n math.Vector3) int {
	m.Positions = append(m.Positions, p)
	m.Normals = append(m.Normals, n)
	return len(m.Positions) - 1
}

func (m *Mesh) AddTriangle(i, j, k int) { m.Indices = append(m.Indices, i, j, k) }

// Quality controls faceting density via two tolerances: a chordal deviation (max gap
// between a facet and the true geometry) and an angular deflection (max turn per facet).
// The angle bound guarantees small-radius curves still get enough facets to read as round
// — a 4 mm bore would pass the chord test at 8 facets (an octagon) on chord tolerance
// alone, so the angle bound forces the minimum segment count a circle needs.
type Quality struct {
	ChordTolerance float64
	AngleTolerance float64 // max turning angle per facet, radians (0 → default)
}

// DefaultQuality returns a reasonable display tolerance. The angle bound is a max
// chord-to-chord deflection; with recursive bisection it gives a full circle 32 facets
// regardless of radius, so even tiny holes (a 4 mm bore) render round, not polygonal.
func DefaultQuality() Quality {
	return Quality{ChordTolerance: 0.05, AngleTolerance: 10 * stdmath.Pi / 180}
}

// PropertyQuality returns the tessellation tolerance for MASS/GEOMETRY PROPERTY readouts (volume,
// area, centroid, inertia), which are integrated over the mesh (see BodyGeometryProperties) and so
// need far finer faceting than the display default. An inscribed N-gon approximation of a curved
// face under-reports its area/volume by ~π²/(3N²): the display default's ~36 facets/circle biases
// a curved solid's volume by ~−0.64% (the recurring delta the exporter saw against Inventor's
// analytic oracle), whereas this ~1°/360-facets tolerance holds it under ~0.01% — engineering
// parity. Use it wherever a property value is reported, never for display (it is ~10× the facets).
func PropertyQuality() Quality {
	return Quality{ChordTolerance: 1e-3, AngleTolerance: 1 * stdmath.Pi / 180}
}

func (q Quality) Tol() float64 {
	if q.ChordTolerance <= 0 {
		return DefaultQuality().ChordTolerance
	}
	return q.ChordTolerance
}

func (q Quality) AngleTol() float64 {
	if q.AngleTolerance <= 0 {
		return DefaultQuality().AngleTolerance
	}
	return q.AngleTolerance
}

// CellNormal returns the (unnormalized) normal of triangle abc by position.
func (m *Mesh) CellNormal(a, b, c int) math.Vector3 {
	pa, pb, pc := m.Positions[a], m.Positions[b], m.Positions[c]
	return pa.VectorTo(pb).Cross(pa.VectorTo(pc))
}

// Tri is a triangle with its supporting plane (unit normal n, offset w: n·p = w).
type Tri struct {
	A, B, C math.Point3
	N       math.Vector3
	W       float64
}

// Flipped returns the triangle with the opposite winding and outward sense.
func (t Tri) Flipped() Tri {
	return Tri{A: t.A, B: t.C, C: t.B, N: t.N.Scale(-1), W: -t.W}
}

// Points returns the triangle's three corners in winding order.
func (t Tri) Points() [3]math.Point3 { return [3]math.Point3{t.A, t.B, t.C} }

func NewTri(a, b, c math.Point3) (Tri, bool) {
	n, err := math.UnitVector3FromVector(a.VectorTo(b).Cross(a.VectorTo(c)))
	if err != nil {
		return Tri{}, false // degenerate (zero-area) triangle: drop it
	}
	nv := n.AsVector()
	return Tri{A: a, B: b, C: c, N: nv, W: nv.Dot(a.AsVector())}, true
}

// Quantize snaps a coordinate to a weld grid (database units), so points within a
// grid cell collapse to one vertex. The grid is the model-relative resolution the
// caller derives (ADR-0042), not a fixed constant.
func Quantize(v, grid float64) int64 { return int64(stdmath.Round(v / grid)) }

// PointWelder merges coincident 3D points onto a shared index list, snapping to a
// model-relative weld grid (ADR-0042) the caller derives from the points' size.
// byID carries each point's SOURCE topological vertex identity through the weld: two points
// with the same non-zero id are the SAME vertex, while two DISTINCT non-zero ids are kept apart
// even when they quantize to one cell — so a boolean's pinch-split coincident vertices (a
// kissing tangency) survive a re-weld instead of collapsing back into a non-manifold pinch
// (#1600). Identity-less points (id 0, e.g. op-generated tangent points) weld by coordinate.
type PointWelder struct {
	index  map[[3]int64]int
	cellID map[[3]int64]uint64 // owning source-vertex id of each claimed cell (0 = claimed only anonymously)
	byID   map[uint64]int
	Points []math.Point3
	grid   float64
}

func NewPointWelder(grid float64) *PointWelder {
	return &PointWelder{index: map[[3]int64]int{}, cellID: map[[3]int64]uint64{}, byID: map[uint64]int{}, grid: grid}
}

// PointCell quantizes p to the weld grid cell used to detect coordinate coincidence.
func PointCell(p math.Point3, grid float64) [3]int64 {
	return [3]int64{Quantize(p.X, grid), Quantize(p.Y, grid), Quantize(p.Z, grid)}
}

func (w *PointWelder) Add(p math.Point3) int {
	k := PointCell(p, w.grid)
	if i, ok := w.index[k]; ok {
		return i
	}
	return w.AppendPoint(p, k, 0)
}

// AppendPoint stores p as a fresh vertex, claiming its cell (with owner id) for coordinate welds
// only if empty (so a later id-0 point at a pinch coordinate welds to the FIRST fan there, not the
// newest). owner is the point's source-vertex id (0 = anonymous), recorded so addID can tell an
// anonymously-claimed cell (adoptable) from one owned by a DISTINCT id (a #1600 pinch — kept apart).
func (w *PointWelder) AppendPoint(p math.Point3, k [3]int64, owner uint64) int {
	i := len(w.Points)
	w.Points = append(w.Points, p)
	if _, ok := w.index[k]; !ok {
		w.index[k] = i
		w.cellID[k] = owner
	}
	return i
}

// AddID welds p under its carried source-vertex identity: a non-zero id resolves to the one
// vertex that id was first seen at (distinct ids never merge, preserving a pinch split); id 0
// falls back to coordinate welding. When the id is new but its cell is already claimed ANONYMOUSLY
// (id 0 — an op-generated point of a rebuilt face at the same real vertex, e.g. a far-runout host's
// unchanged corner welding to a pass-through neighbour when the fillet spreads onto the base solid's
// OWN faces, corner-blend-weld Piece 2 / N1), the ided point ADOPTS that vertex rather than forking a
// coincident duplicate that would leave the shared edge 1-incident. A cell owned by a DIFFERENT
// non-zero id is a genuine pinch and is never adopted (#1600). See the type comment.
func (w *PointWelder) AddID(p math.Point3, id uint64) int {
	if id == 0 {
		return w.Add(p)
	}
	if i, ok := w.byID[id]; ok {
		return i
	}
	k := PointCell(p, w.grid)
	if i, ok := w.index[k]; ok && w.cellID[k] == 0 {
		w.byID[id], w.cellID[k] = i, id // adopt the anonymously-claimed vertex at this coordinate
		return i
	}
	i := w.AppendPoint(p, k, id)
	w.byID[id] = i
	return i
}

func (w *PointWelder) WeldRing(r []math.Point3) []int {
	out := make([]int, len(r))
	for i, p := range r {
		out[i] = w.Add(p)
	}
	return out
}

// WeldRingID welds a ring carrying a parallel source-vertex id per point (ids may be shorter than
// pts, in which case the missing tail is treated as op-generated, id 0).
func (w *PointWelder) WeldRingID(pts []math.Point3, ids []uint64) []int {
	out := make([]int, len(pts))
	for i, p := range pts {
		out[i] = w.AddID(p, probe.SrcIDAt(ids, i))
	}
	return out
}

// Area is the summed area of the mesh's triangles. It is a measurement OF THE MESH, not of the
// B-rep it came from: an approximation whose error falls with the chord tolerance. A test that
// wants the true area of a face must ask the analytic integrator (kernel/ops/query), not this.
//
// Example: if a := m.Area(); math.Abs(a-want)/want > 0.01 { /* the tessellation lost a facet */ }
func (m *Mesh) Area() float64 {
	area := 0.0
	for i := 0; i+2 < len(m.Indices); i += 3 {
		a, b, c := m.Positions[m.Indices[i]], m.Positions[m.Indices[i+1]], m.Positions[m.Indices[i+2]]
		area += 0.5 * float64(a.VectorTo(b).Cross(a.VectorTo(c)).Length())
	}
	return area
}
