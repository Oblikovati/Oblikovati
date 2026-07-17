// SPDX-License-Identifier: GPL-2.0-only

package ipt

import (
	"encoding/binary"
	"math"
)

// Where a sketch LIVES. Its entities' coordinates are 2D in the sketch's own plane, so without the
// placement every sketch lands on XY at z=0 — and a feature authored on another plane is then built
// somewhere it never was. BigChunkyPlate has 9 sketches on its top face at z=3.00 whose bosses grow
// DOWN into the plate; placed at z=0 they hang below it instead, making the body 5.60 cm thick
// against a true 3.00 and costing the part every one of its features to the body gate (#29).
//
// Grounded in InventorLoader: Read_90874D11 "PlanarSketch" names a `transformation` cross-ref, and
// Read_90874D18 "Transformation" holds it as the sparse 4x4 of Transformation3D.read.

// transformNodeType is InventorLoader Read_90874D18 "Transformation".
const transformNodeType = 0x90874D18

// transformMatrixOffset is where the sparse matrix begins: the standard content header, exactly as
// on a Parameter or Boolean. Confirmed on every sketch of the corpus by the orthonormality check
// below — a wrong offset yields an all -1 matrix and is rejected outright.
const transformMatrixOffset = 34

// transformPrefix optionally precedes the masks.
const transformPrefix = 0x00000203

// Plane is a sketch's placement: where its 2D origin sits in 3D and which way its axes run.
type SketchPlacement struct {
	Origin [3]float64
	XAxis  [3]float64
	YAxis  [3]float64
}

// sketchPlane reads the placement the sketch node references, or ok=false when the file does not
// state one this layout can read — the caller then falls back to XY rather than inventing a plane.
func sketchPlane(nodes []dcNode, sk dcNode) (SketchPlacement, bool) {
	xn, ok := referencedTransform(nodes, sk)
	if !ok {
		return SketchPlacement{}, false
	}
	m, ok := transformMatrix(xn.payload, transformMatrixOffset)
	if !ok || !isRigidPlacement(m) {
		return SketchPlacement{}, false
	}
	return SketchPlacement{
		Origin: [3]float64{m[0][3], m[1][3], m[2][3]},
		XAxis:  [3]float64{m[0][0], m[1][0], m[2][0]},
		YAxis:  [3]float64{m[0][1], m[1][1], m[2][1]},
	}, true
}

// referencedTransform finds the Transformation the sketch names. The reference is NOT 4-aligned (a
// List8 of entities precedes it), so every byte offset is tried — a 4-stepped scan finds nothing.
func referencedTransform(nodes []dcNode, sk dcNode) (dcNode, bool) {
	for off := 0; off+4 <= len(sk.payload); off++ {
		ref := int(binary.LittleEndian.Uint32(sk.payload[off:]) & refIndexMask)
		if n, ok := nodeAt(nodes, ref); ok && n.typ == transformNodeType {
			return n, true
		}
	}
	return dcNode{}, false
}

// transformMatrix reads the sparse 4x4 (InventorLoader Transformation3D.read). Most cells of a
// rigid placement are 0 or ±1, so two 16-bit masks say which are stored as f64 and which are those
// constants — which is why the payload is only 38 bytes for an identity and 46 with a translation.
func transformMatrix(pay []byte, at int) ([4][4]float64, bool) {
	i := at
	if i+4 <= len(pay) && binary.LittleEndian.Uint32(pay[i:]) == transformPrefix {
		i += 4
	}
	var m [4][4]float64
	if i+4 > len(pay) {
		return m, false
	}
	d1 := uint32(binary.LittleEndian.Uint16(pay[i:]))
	d2 := uint32(binary.LittleEndian.Uint16(pay[i+2:]))
	i += 4
	for row := 0; row < 4; row++ {
		for col := 0; col < 4; col++ {
			v, next, ok := matrixCell(pay, i, d1, d2, uint32(1)<<uint(col+4*row))
			if !ok {
				return [4][4]float64{}, false
			}
			m[row][col], i = v, next
		}
	}
	return m, true
}

// matrixCell reads one cell: a stored f64, or the constant its mask bits name.
func matrixCell(pay []byte, i int, d1, d2, b uint32) (float64, int, bool) {
	if d2&b != 0 {
		if d1&b == 0 {
			return 0, i, true
		}
		return -1, i, true
	}
	if d1&b != 0 {
		return 1, i, true
	}
	if i+8 > len(pay) {
		return 0, i, false
	}
	v := math.Float64frombits(binary.LittleEndian.Uint64(pay[i:]))
	if math.IsNaN(v) || math.Abs(v) < 1e-6 {
		v = 0
	}
	return v, i + 8, true
}

// isRigidPlacement is the decode's own oracle: a placement's axes are orthonormal. A wrong offset
// produces an all -1 matrix, which this rejects — so the OFFSET needs no separate proof.
//
// It does NOT prove the ORIENTATION, and must not be read as doing so: a rotation's transpose is
// also a rotation, so reading the axes as rows instead of columns passes this check identically.
// What settles that is the translation, which is not symmetric — of the corpus's 517 sketch
// transforms, 346 carry it in the last COLUMN (m[0..2][3]) and 0 in the last row, i.e. the matrix is
// the column-vector convention p' = M·p and the basis vectors are its columns, as read below. That
// matters: 210 of those 517 placements have a normal pointing away from +Z, and a transposed read
// would have silently flipped them while still passing here (see translate.directionOf).
func isRigidPlacement(m [4][4]float64) bool {
	x := [3]float64{m[0][0], m[1][0], m[2][0]}
	y := [3]float64{m[0][1], m[1][1], m[2][1]}
	lx := math.Sqrt(x[0]*x[0] + x[1]*x[1] + x[2]*x[2])
	ly := math.Sqrt(y[0]*y[0] + y[1]*y[1] + y[2]*y[2])
	dot := x[0]*y[0] + x[1]*y[1] + x[2]*y[2]
	return math.Abs(lx-1) <= 1e-9 && math.Abs(ly-1) <= 1e-9 && math.Abs(dot) <= 1e-9 && m[3][3] == 1
}
