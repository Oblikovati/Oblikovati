// SPDX-License-Identifier: GPL-2.0-only

package translate

import (
	"bytes"
	"encoding/binary"
	"math"

	"oblikovati.org/kernel/exchange/meshio"
)

// cmToMM converts the B-rep's centimetre coordinates to the millimetres that STL
// files conventionally carry (matching Inventor's own STL export).
const cmToMM = 10.0

// encodeBinarySTL writes a triangle soup as binary STL, scaling cm -> mm. Per-facet
// normals are left zero; downstream welding recomputes orientation.
func encodeBinarySTL(raw meshio.RawMesh) []byte {
	var buf bytes.Buffer
	buf.Write(make([]byte, 80)) // 80-byte header
	binary.Write(&buf, binary.LittleEndian, uint32(len(raw.Tris)))
	var zero [3]float32
	for _, tri := range raw.Tris {
		writeVec3(&buf, zero) // normal (recomputed on import)
		for _, vi := range tri {
			p := raw.Verts[vi]
			writeVec3(&buf, [3]float32{
				float32(p.X * cmToMM), float32(p.Y * cmToMM), float32(p.Z * cmToMM),
			})
		}
		buf.Write([]byte{0, 0}) // attribute byte count
	}
	return buf.Bytes()
}

func writeVec3(buf *bytes.Buffer, v [3]float32) {
	for _, f := range v {
		if math.IsNaN(float64(f)) {
			f = 0
		}
		binary.Write(buf, binary.LittleEndian, f)
	}
}
