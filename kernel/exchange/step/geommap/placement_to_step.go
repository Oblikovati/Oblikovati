// SPDX-License-Identifier: GPL-2.0-only

package geommap

import (
	"oblikovati.org/kernel/exchange/step/part21"
	"oblikovati.org/math"
)

// Emitter writes kernel geometry into a part21.Writer, sharing identical leaf
// entities (points, directions). It is the export counterpart of the from_step
// mappers and the only place /source emits STEP geometry, keeping the encoding in
// one module. divScale converts database mm back into the file's length unit.
type Emitter struct {
	w        *part21.Writer
	mmToUnit float64 // multiply a mm length by this to get the file unit
}

// NewEmitter builds an emitter writing into w, expressing lengths in file units
// where one file unit is unitMM millimeters (so mm→unit is 1/unitMM).
func NewEmitter(w *part21.Writer, unitMM float64) *Emitter {
	if unitMM == 0 {
		unitMM = 1
	}
	return &Emitter{w: w, mmToUnit: 1 / unitMM}
}

// Point emits a CARTESIAN_POINT (shared) for p, scaled mm→file-unit, returning its id.
func (e *Emitter) Point(p math.Point3) int {
	coords := part21.FormatList(
		part21.FormatReal(float64(p.X)*e.mmToUnit),
		part21.FormatReal(float64(p.Y)*e.mmToUnit),
		part21.FormatReal(float64(p.Z)*e.mmToUnit),
	)
	return e.w.AddShared("CARTESIAN_POINT", part21.QuoteString(""), coords)
}

// Direction emits a DIRECTION (shared) for the normalized v, returning its id.
func (e *Emitter) Direction(v math.Vector3) int {
	u := v.Scale(1 / v.Length())
	coords := part21.FormatList(
		part21.FormatReal(float64(u.X)),
		part21.FormatReal(float64(u.Y)),
		part21.FormatReal(float64(u.Z)),
	)
	return e.w.AddShared("DIRECTION", part21.QuoteString(""), coords)
}

// Placement emits an AXIS2_PLACEMENT_3D from an origin, Z axis and X axis.
func (e *Emitter) Placement(origin math.Point3, axisZ, axisX math.Vector3) int {
	loc := e.Point(origin)
	z := e.Direction(axisZ)
	x := e.Direction(axisX)
	return e.w.Add("AXIS2_PLACEMENT_3D", part21.QuoteString(""), part21.Ref(loc), part21.Ref(z), part21.Ref(x))
}

// Writer exposes the underlying part21 writer for the topology emitter.
func (e *Emitter) Writer() *part21.Writer { return e.w }

// LengthValue formats a mm length in file units as a Part 21 real.
func (e *Emitter) LengthValue(mm float64) string {
	return part21.FormatReal(mm * e.mmToUnit)
}
