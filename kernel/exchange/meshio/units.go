// SPDX-License-Identifier: GPL-2.0-only

package meshio

import (
	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
)

// Mesh unit handling (Oblikovati/Oblikovati#146). STL and OBJ are unitless, so
// they are read/written assuming the millimetre convention; 3MF carries a unit
// attribute that is written from the document unit and read back on import. The
// kernel works in centimetres, so the model scales database↔file units via
// exchange.TranslationOptions.

// threeMFUnitNames maps an export file-unit name to the 3MF <model unit="…">
// spelling; unknown names fall back to millimetre.
var threeMFUnitNames = map[string]string{
	"mm": "millimeter", "cm": "centimeter", "m": "meter", "in": "inch", "ft": "foot",
}

// mmPer3MFUnit is the millimetre size of each 3MF unit spelling, for import scaling.
var mmPer3MFUnit = map[string]float64{
	"micron": 0.001, "millimeter": 1, "centimeter": 10, "meter": 1000,
	"inch": 25.4, "foot": 304.8,
}

// threeMFUnitName returns the 3MF unit spelling for a file-unit name (millimetre
// when unknown/empty).
func threeMFUnitName(fileUnit string) string {
	if n, ok := threeMFUnitNames[fileUnit]; ok {
		return n
	}
	return "millimeter"
}

// scaleMesh multiplies every position of an already-tessellated mesh by f (a
// no-op when f is 1, the common cm→mm-less case). Normals are unit vectors and
// are left untouched.
func scaleMesh(mesh *ops.Mesh, f float64) {
	if f == 1 {
		return
	}
	for i := range mesh.Positions {
		p := mesh.Positions[i]
		mesh.Positions[i] = math.P3(p.X*f, p.Y*f, p.Z*f)
	}
}

// scaleRaw multiplies every vertex of a decoded triangle soup by f.
func scaleRaw(raw *RawMesh, f float64) {
	if f == 1 {
		return
	}
	for i := range raw.Verts {
		raw.Verts[i] = math.P3(raw.Verts[i].X*f, raw.Verts[i].Y*f, raw.Verts[i].Z*f)
	}
}
