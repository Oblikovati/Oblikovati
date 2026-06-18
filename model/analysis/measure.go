// SPDX-License-Identifier: GPL-2.0-only

package analysis

import (
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Measurement (M18-F01 PBI-164, #428): geometric quantities of model entities, in user units (mm).

// EdgeLengthMm returns an edge's length in millimetres, summing its tessellated polyline (exact for
// lines/arcs/circles; converges with q for general curves).
func EdgeLengthMm(e *topo.Edge, q ops.Quality) float64 {
	pts := ops.TessellateEdge(e, q)
	var lengthCm float64
	for i := 1; i < len(pts); i++ {
		lengthCm += float64(pts[i-1].VectorTo(pts[i]).Length())
	}
	return lengthCm * cmToMM
}

// FaceAreaMm2 returns a face's area in square millimetres, summing its tessellated triangle areas
// (exact for planar faces; converges with q for curved ones).
func FaceAreaMm2(f *topo.Face, q ops.Quality) float64 {
	mesh := ops.TessellateFace(f, q)
	var areaCm2 float64
	for t := 0; t+2 < len(mesh.Indices); t += 3 {
		a := mesh.Positions[mesh.Indices[t]]
		b := mesh.Positions[mesh.Indices[t+1]]
		c := mesh.Positions[mesh.Indices[t+2]]
		areaCm2 += float64(a.VectorTo(b).Cross(a.VectorTo(c)).Length()) / 2
	}
	return areaCm2 * cmToMM * cmToMM
}

// VertexDistanceMm returns the straight-line distance between two vertices in millimetres.
func VertexDistanceMm(a, b *topo.Vertex) float64 {
	return distanceMm(a.Point(), b.Point())
}

// distanceMm is the millimetre distance between two database-unit points.
func distanceMm(a, b math.Point3) float64 {
	return float64(a.VectorTo(b).Length()) * cmToMM
}
