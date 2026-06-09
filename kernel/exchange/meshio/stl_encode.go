// SPDX-License-Identifier: GPL-2.0-only

package meshio

import (
	"bytes"
	"encoding/binary"
	stdmath "math"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// stlErr builds an STL decode error naming the offending value (per CLAUDE.md).
func stlErr(what, value string) error {
	return &decodeError{format: "STL", what: what, value: value}
}

// decodeError is a format-tagged decode failure whose message names the offending value.
type decodeError struct{ format, what, value string }

func (e *decodeError) Error() string {
	return e.format + ": " + e.what + " \"" + e.value + "\""
}

// EncodeBinarySTL tessellates body at quality q and writes a little-endian binary STL.
// Per-facet normals are the triangle's geometric normal. This is the resolution knob: a
// finer q yields more triangles for curved bodies.
//
// Example:
//
//	data := meshio.EncodeBinarySTL(body, meshio.QualityFor(types.ResolutionHigh))
func EncodeBinarySTL(body *topo.Body, q ops.Quality) []byte {
	mesh, _ := ops.TessellateBody(body, q)
	return encodeBinarySTLMesh(mesh)
}

// encodeBinarySTLMesh writes an already-tessellated mesh as binary STL.
func encodeBinarySTLMesh(mesh *ops.Mesh) []byte {
	var buf bytes.Buffer
	buf.Write(make([]byte, 80)) // 80-byte header (zeroed)
	writeUint32(&buf, uint32(mesh.TriangleCount()))
	for t := 0; t+2 < len(mesh.Indices); t += 3 {
		writeFacet(&buf, mesh, t)
	}
	return buf.Bytes()
}

// writeFacet emits one 50-byte STL facet (normal + 3 vertices + attr count) for the
// triangle starting at index t in mesh.
func writeFacet(buf *bytes.Buffer, mesh *ops.Mesh, t int) {
	a := mesh.Positions[mesh.Indices[t]]
	b := mesh.Positions[mesh.Indices[t+1]]
	c := mesh.Positions[mesh.Indices[t+2]]
	writeVector(buf, triangleNormal(a, b, c))
	writePoint(buf, a)
	writePoint(buf, b)
	writePoint(buf, c)
	buf.Write([]byte{0, 0}) // attribute byte count
}

// triangleNormal returns the unit geometric normal of triangle a,b,c (zero for a
// degenerate triangle).
func triangleNormal(a, b, c math.Point3) math.Vector3 {
	n := a.VectorTo(b).Cross(a.VectorTo(c))
	if l := n.Length(); l > 0 {
		return n.Scale(1 / l)
	}
	return n
}

func writePoint(buf *bytes.Buffer, p math.Point3)   { writeXYZ(buf, p.X, p.Y, p.Z) }
func writeVector(buf *bytes.Buffer, v math.Vector3) { writeXYZ(buf, v.X, v.Y, v.Z) }

// writeXYZ writes three coordinates as little-endian float32.
func writeXYZ(buf *bytes.Buffer, x, y, z float64) {
	writeFloat32(buf, float32(x))
	writeFloat32(buf, float32(y))
	writeFloat32(buf, float32(z))
}

func writeUint32(buf *bytes.Buffer, v uint32) {
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], v)
	buf.Write(b[:])
}

func writeFloat32(buf *bytes.Buffer, v float32) { writeUint32(buf, stdmath.Float32bits(v)) }
