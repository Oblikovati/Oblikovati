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
