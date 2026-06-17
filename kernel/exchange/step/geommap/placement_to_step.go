// SPDX-License-Identifier: GPL-2.0-only

package geommap

import (
	"oblikovati.org/kernel/exchange/step/part21"
	"oblikovati.org/math"
)

// Emitter writes kernel geometry into a part21.Writer, sharing identical leaf
// entities (points, directions). It is the export counterpart of the from_step
// mappers and the only place /source emits STEP geometry, keeping the encoding in
// one module. dbToFile scales each database-unit (centimetre) length into the
// file's declared length unit.
type Emitter struct {
	w        *part21.Writer
	dbToFile float64 // multiply a database-unit length by this to get the file-unit value
}

// NewEmitter builds an emitter writing into w, scaling each database-unit length
// by dbToFile to express it in the file's declared length unit (the
// exchange.TranslationOptions.ExportScale). A zero scale is treated as 1.
func NewEmitter(w *part21.Writer, dbToFile float64) *Emitter {
	if dbToFile == 0 {
		dbToFile = 1
	}
	return &Emitter{w: w, dbToFile: dbToFile}
}

// Point emits a CARTESIAN_POINT (shared) for p, scaled database-unit→file-unit, returning its id.
func (e *Emitter) Point(p math.Point3) int {
	coords := part21.FormatList(
		part21.FormatReal(float64(p.X)*e.dbToFile),
		part21.FormatReal(float64(p.Y)*e.dbToFile),
		part21.FormatReal(float64(p.Z)*e.dbToFile),
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

// LengthValue formats a database-unit length in file units as a Part 21 real.
func (e *Emitter) LengthValue(dbLen float64) string {
	return part21.FormatReal(dbLen * e.dbToFile)
}
