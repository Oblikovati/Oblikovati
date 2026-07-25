// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
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

// carryDiagnostics lifts a sub-mesh's diagnostics into m when the tessellator composes meshes, so a
// degradation recorded deep in a component surfaces on the final face mesh.
func (m *Mesh) carryDiagnostics(child *Mesh) {
	if child != nil && len(child.Diagnostics) > 0 {
		m.Diagnostics = append(m.Diagnostics, child.Diagnostics...)
	}
}

// TriangleCount returns the number of triangles.
func (m *Mesh) TriangleCount() int { return len(m.Indices) / 3 }

// VertexCount returns the number of vertices.
func (m *Mesh) VertexCount() int { return len(m.Positions) }

func (m *Mesh) addVertex(p math.Point3, n math.Vector3) int {
	m.Positions = append(m.Positions, p)
	m.Normals = append(m.Normals, n)
	return len(m.Positions) - 1
}

func (m *Mesh) addTriangle(i, j, k int) { m.Indices = append(m.Indices, i, j, k) }

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

func (q Quality) tol() float64 {
	if q.ChordTolerance <= 0 {
		return DefaultQuality().ChordTolerance
	}
	return q.ChordTolerance
}

func (q Quality) angleTol() float64 {
	if q.AngleTolerance <= 0 {
		return DefaultQuality().AngleTolerance
	}
	return q.AngleTolerance
}

// TessellateEdge facets an edge into a polyline honoring the chordal and angular tolerances.
func TessellateEdge(e *topo.Edge, q Quality) []math.Point3 {
	pts, _ := tessellateEdgeWithParams(e, q)
	return pts
}

// tessellateEdgeWithParams is [TessellateEdge] plus the CURVE PARAMETER each sample was taken at.
// A mesher that has to label a tessellated edge with its position on the adjoining SURFACE needs that
// parameter, not just the point: re-deriving it from chord length along the polyline is only correct
// when the curve is parametrised by arc length, and a rolling-ball canal rail is not (it drifted the
// canal band's v by 1.2e-2 and sheared every boundary strip — see canalRailRow).
func tessellateEdgeWithParams(e *topo.Edge, q Quality) ([]math.Point3, []float64) {
	c := e.Geometry()
	lo, hi := c.Domain()
	params := adaptiveParams(c.PointAt, lo, hi, q.tol(), q.angleTol())
	pts := make([]math.Point3, len(params))
	for i, t := range params {
		pts[i] = c.PointAt(t)
	}
	return pts, params
}

// TessellateBody facets every face into one mesh and every edge into a polyline. After the per-face
// meshing it runs a cross-face conformance repair (conformCylConeFaces) that re-meshes only the
// cyl/cone faces touching a crack so a trimmed wall conforms to its planar neighbour; a watertight body
// has no cracks and is untouched.
func TessellateBody(b *topo.Body, q Quality) (*Mesh, [][]math.Point3) {
	_, fm := tessellateBodyFaces(b, q)
	mesh := &Mesh{}
	for _, m := range fm {
		mergeMesh(mesh, m)
	}
	var edges [][]math.Point3
	for _, e := range b.Edges() {
		edges = append(edges, TessellateEdge(e, q))
	}
	return mesh, edges
}

// tessellateBodyFaces runs the per-face meshing pipeline (facet, cross-face
// conformance repair, outward orientation) and returns the faces with their
// repaired meshes — shared by [TessellateBody] and [CalculateBodyFacets].
func tessellateBodyFaces(b *topo.Body, q Quality) ([]*topo.Face, []*Mesh) {
	faces := b.Faces()
	fm := make([]*Mesh, len(faces))
	idx := make(map[*topo.Face]int, len(faces))
	for i, f := range faces {
		fm[i] = TessellateFace(f, q)
		idx[f] = i
	}
	conformCylConeFaces(faces, idx, fm, q)
	orientFacesOutward(fm) // re-orient imported faces whose B-rep sense came in inverted (Normal-Debug red)
	return faces, fm
}

// TessellateFace facets a face. Planar faces are triangulated exactly from their
// outer boundary (watertight); curved faces are sampled on an adaptive UV grid.
func TessellateFace(f *topo.Face, q Quality) *Mesh {
	mesh := tessellateFaceSurface(f, q)
	if f.Reversed() {
		reverseMesh(mesh) // cut wall: surface normal points into the removed material
	}
	return mesh
}

// tessellateFaceSurface meshes a face by its surface kind, ignoring its sense.
func tessellateFaceSurface(f *topo.Face, q Quality) *Mesh {
	switch s := f.Geometry().(type) {
	case geom.Plane:
		return tessellatePlanarFace(f, q)
	case geom.ThreadedCylinder:
		return tessellateThreadedFace(f, s, q)
	}
	return tessellateCurvedFace(f, q)
}

// tessellateThreadedFace meshes a modeled thread as a height-field grid on the cylinder —
// fine enough in v to resolve the thread profile, periodic in u. Per-vertex normals come from
// the surface so shading and the divergence-theorem volume read the real threaded geometry.
func tessellateThreadedFace(f *topo.Face, s geom.ThreadedCylinder, q Quality) *Mesh {
	// The angular (u) samples come from the BASE cylinder's band path — they are the face's
	// boundary-edge angles, so the threaded mesh shares those vertices with the adjacent caps
	// (the runout makes v=ends the plain radius) → a watertight stitch. v is subdivided finely
	// to resolve the thread. Positions follow the threaded surface, but cell orientation and
	// shading normals come from the base cylinder (its analytic +radial normal is reliable;
	// the threaded surface's numerical normal misleads the outward-winding test).
	base := s.Cylinder
	us, vsEnds, ok := periodicBandGrid(base, faceOuterBoundary(f, q), faceHoleBoundaries(f, q))
	if !ok || len(vsEnds) < 2 {
		return tessellateCurvedFace(f, q)
	}
	vMin, vMax := vsEnds[0], vsEnds[len(vsEnds)-1]
	turns := (vMax - vMin) / s.Pitch
	nv := int(stdmath.Max(8, stdmath.Round(turns*10)))
	vs := make([]float64, nv+1)
	for j := range vs {
		vs[j] = vMin + (vMax-vMin)*float64(j)/float64(nv)
	}
	m := &Mesh{}
	idx := make([][]int, len(us))
	for i, u := range us {
		idx[i] = make([]int, len(vs))
		for j, v := range vs {
			idx[i][j] = m.addVertex(s.PointAt(u, v), base.NormalAt(u, v))
		}
	}
	for i := 0; i+1 < len(us); i++ {
		for j := 0; j+1 < len(vs); j++ {
			emitCellOutward(m, base, us[i], us[i+1], vs[j], vs[j+1], idx[i][j], idx[i+1][j], idx[i+1][j+1], idx[i][j+1])
		}
	}
	return m
}

// reverseMesh flips a reversed face's sense (see topo.Face.Reversed): every outward normal
// negates and every triangle's winding reverses, so the face presents its true material side
// to shading and to the divergence-theorem volume (mass-properties orient triangles by the
// resulting shared-edge winding — see consistentOutwardFlips/meshGeometryProperties).
func reverseMesh(m *Mesh) {
	for i := range m.Normals {
		m.Normals[i] = m.Normals[i].Scale(-1)
	}
	for t := 0; t+2 < len(m.Indices); t += 3 {
		m.Indices[t+1], m.Indices[t+2] = m.Indices[t+2], m.Indices[t+1]
	}
}

// mergeMesh appends src's geometry into dst, offsetting indices.
func mergeMesh(dst, src *Mesh) {
	base := dst.VertexCount()
	dst.Positions = append(dst.Positions, src.Positions...)
	dst.Normals = append(dst.Normals, src.Normals...)
	for _, idx := range src.Indices {
		dst.Indices = append(dst.Indices, base+idx)
	}
	dst.carryDiagnostics(src) // a face's tessellation diagnostics surface on the whole-body mesh
}

// adaptiveParams returns the parameter breakpoints (lo…hi inclusive) at which eval's chord
// both deviates from the true curve by no more than chordTol AND turns by no more than
// angleTol per facet, by recursive midpoint subdivision.
func adaptiveParams(eval func(float64) math.Point3, lo, hi, chordTol, angleTol float64) []float64 {
	out := []float64{lo}
	subdivide(eval, lo, hi, chordTol, angleTol, 0, &out)
	return out
}

func subdivide(eval func(float64) math.Point3, lo, hi, chordTol, angleTol float64, depth int, out *[]float64) {
	mid := (lo + hi) / 2
	pLo, pMid, pHi := eval(lo), eval(mid), eval(hi)
	if depth >= 22 || (pointToSegment(pMid, pLo, pHi) <= chordTol && turnAngle(pLo, pMid, pHi) <= angleTol) {
		*out = append(*out, hi)
		return
	}
	subdivide(eval, lo, mid, chordTol, angleTol, depth+1, out)
	subdivide(eval, mid, hi, chordTol, angleTol, depth+1, out)
}

// turnAngle returns the angle (radians) between the chords a→b and b→c — the curve's
// turning across this span. Zero for a straight run (no over-faceting of lines). Only an
// EXACTLY zero chord is directionless — the old `LengthSquared < DefaultTolerance` guard
// declared every sub-0.3 µm chord degenerate, silently disabling the angular-deflection
// criterion (the scale-free ≥32-facets-per-circle backstop) on tiny parts, which
// tessellated a 1 µm bore as a square (#1610).
func turnAngle(a, b, c math.Point3) float64 {
	d1, d2 := a.VectorTo(b), b.VectorTo(c)
	if d1.LengthSquared() == 0 || d2.LengthSquared() == 0 {
		return 0
	}
	cosA := math.Clamp(d1.Dot(d2)/(d1.Length()*d2.Length()), -1, 1)
	return stdmath.Acos(cosA)
}

// pointToSegment returns the distance from p to segment [a, b].
func pointToSegment(p, a, b math.Point3) float64 {
	ab := a.VectorTo(b)
	denom := ab.LengthSquared()
	if denom < math.DefaultTolerance {
		return p.DistanceTo(a)
	}
	t := math.Clamp01(a.VectorTo(p).Dot(ab) / denom)
	return p.DistanceTo(a.TranslateBy(ab.Scale(t)))
}

// clampSpan replaces infinite domain bounds with a finite window.
func clampSpan(lo, hi float64) (float64, float64) {
	const window = 10
	if stdmath.IsInf(lo, 0) {
		lo = -window
	}
	if stdmath.IsInf(hi, 0) {
		hi = window
	}
	return lo, hi
}
