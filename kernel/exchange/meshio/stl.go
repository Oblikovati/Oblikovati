// SPDX-License-Identifier: GPL-2.0-only

package meshio

import (
	"bufio"
	"bytes"
	"encoding/binary"
	stdmath "math"
	"strconv"

	"oblikovati.org/math"
)

// DecodeSTL decodes an STL file (binary or ASCII, auto-detected) into a triangle soup.
// Binary STL is detected when the payload after the 80-byte header matches the declared
// triangle count's exact length; otherwise the bytes are parsed as ASCII. It errors on
// truncated/garbled input, naming the offending value.
//
// Example:
//
//	raw, err := meshio.DecodeSTL(data)
func DecodeSTL(data []byte) (RawMesh, error) {
	if isBinarySTL(data) {
		return decodeBinarySTL(data)
	}
	return decodeASCIISTL(data)
}

// isBinarySTL reports whether data is binary STL: an 80-byte header + uint32 count +
// exactly count×50 triangle bytes. (ASCII files start with "solid" too, so the length
// check — not the prefix — is the reliable discriminator.)
func isBinarySTL(data []byte) bool {
	const header = 84 // 80 header + 4 count
	if len(data) < header {
		return false
	}
	count := binary.LittleEndian.Uint32(data[80:84])
	return len(data) == header+int(count)*50
}

// decodeBinarySTL reads a little-endian binary STL: 80-byte header, uint32 triangle
// count, then per triangle a normal (3×float32, ignored) + 3 vertices + uint16 attr.
func decodeBinarySTL(data []byte) (RawMesh, error) {
	count := binary.LittleEndian.Uint32(data[80:84])
	var m RawMesh
	off := 84
	for i := uint32(0); i < count; i++ {
		off += 12 // skip the per-facet normal
		a := readVec32(data, off)
		b := readVec32(data, off+12)
		c := readVec32(data, off+24)
		m.AddTriangle(a, b, c)
		off += 36 + 2 // 3 vertices + uint16 attribute byte count
	}
	return m, nil
}

// readVec32 reads three little-endian float32 at off into a Point3.
func readVec32(data []byte, off int) math.Point3 {
	x := stdmath.Float32frombits(binary.LittleEndian.Uint32(data[off : off+4]))
	y := stdmath.Float32frombits(binary.LittleEndian.Uint32(data[off+4 : off+8]))
	z := stdmath.Float32frombits(binary.LittleEndian.Uint32(data[off+8 : off+12]))
	return math.P3(float64(x), float64(y), float64(z))
}

// decodeASCIISTL parses an ASCII STL (solid…facet normal…outer loop…vertex×3…endloop…
// endfacet…endsolid), collecting each facet's three vertices into the soup.
func decodeASCIISTL(data []byte) (RawMesh, error) {
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Split(bufio.ScanWords)
	var m RawMesh
	var facet []math.Point3
	for sc.Scan() {
		if sc.Text() != "vertex" {
			if sc.Text() == "endfacet" {
				appendFacet(&m, facet)
				facet = nil
			}
			continue
		}
		p, err := scanVertex(sc)
		if err != nil {
			return RawMesh{}, err
		}
		facet = append(facet, p)
	}
	return m, sc.Err()
}

// appendFacet adds a facet's first triangle (a triangulated fan for >3 verts) to m.
func appendFacet(m *RawMesh, facet []math.Point3) {
	for i := 2; i < len(facet); i++ {
		m.AddTriangle(facet[0], facet[i-1], facet[i])
	}
}

// scanVertex reads three whitespace-separated floats after a "vertex" token.
func scanVertex(sc *bufio.Scanner) (math.Point3, error) {
	var c [3]float64
	for i := 0; i < 3; i++ {
		if !sc.Scan() {
			return math.Point3{}, stlErr("vertex truncated at coordinate", strconv.Itoa(i))
		}
		v, err := strconv.ParseFloat(sc.Text(), 64)
		if err != nil {
			return math.Point3{}, stlErr("bad coordinate", sc.Text())
		}
		c[i] = v
	}
	return math.P3(c[0], c[1], c[2]), nil
}
