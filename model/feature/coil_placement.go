// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// Where a coil's profile sits at each step of its rail, and which way it winds.
//
// One rail integral (coil_variable.go) serves both coil flavours; they differ only in which
// degree of freedom it is spent on. A HELIX spends it ALONG the axis — the accumulated pitch is
// the rise, and a taper then walks the section away from the axis in proportion to that rise. A
// flat SPIRAL (Inventor's kSpiralCoilExtent, #1883) spends the same integral radially: the pitch
// is a radial step per turn and there is no rise at all. So a variable-pitch table and the flat
// end conditions carry over to a spiral unchanged, meaning a per-turn radial step and a
// constant-radius (circular) end — which is how a real clock spring's ends are formed.

// CoilHandedness is the sense in which a coil winds about its axis.
type CoilHandedness int

const (
	// RightHandedCoil winds by the right-hand rule about the axis direction: the rotation and the
	// rise are both positive about/along it. The ordinary thread and spring sense, and the zero
	// value so an unset definition keeps the behaviour that predates handedness (#1883).
	RightHandedCoil CoilHandedness = iota
	// LeftHandedCoil reverses the rotation while keeping the rise, mirroring the helix.
	LeftHandedCoil
)

// rotationSense is the sign handedness applies to the sweep angle.
func (h CoilHandedness) rotationSense() float64 {
	if h == LeftHandedCoil {
		return -1
	}
	return 1
}

// coilPlacement is where one section sits on the rail: how far it has advanced along the axis
// (rise) and how far away from the axis (radial). Both are lengths in model units.
type coilPlacement struct{ rise, radial float64 }

// coilPlacer maps a sweep angle (radians from the start) to that section's placement. It is the
// single point at which the helical and spiral flavours differ.
type coilPlacer func(angle float64) coilPlacement

// helicalPlacer spends the rail integral axially: the accumulated pitch is the rise, and the
// taper walks the section radially outward by tan(taper)·rise — the tapered coil whose helix
// radius grows with height (M08 PBI-096, #316).
func helicalPlacer(rise func(float64) float64, taper float64) coilPlacer {
	return func(angle float64) coilPlacement {
		h := rise(angle)
		return coilPlacement{rise: h, radial: stdmath.Tan(taper) * h}
	}
}

// spiralPlacer spends the same rail integral radially, with no rise: a flat spiral whose radius
// grows by the pitch every turn (#1883).
func spiralPlacer(advance func(float64) float64) coilPlacer {
	return func(angle float64) coilPlacement {
		return coilPlacement{radial: advance(angle)}
	}
}

// coilPlacerFor picks the flavour's placer over the rail's accumulated advance, refusing the
// options a flavour cannot honour rather than accepting and quietly discarding them (#1883).
func coilPlacerFor(def *CoilDefinition, advance func(float64) float64) (coilPlacer, error) {
	if !def.Spiral {
		return helicalPlacer(advance, def.Taper), nil
	}
	if def.Taper != 0 {
		return nil, fmt.Errorf("coil: taper %g rad grows the radius in proportion to the AXIAL "+
			"RISE, and a spiral has none; drop the taper — a spiral's radial growth IS its pitch",
			def.Taper)
	}
	return spiralPlacer(advance), nil
}

// coilSections places the profile along the rail: at each step it is rotated about the axis by
// the running angle (negated for a left-handed coil) and displaced by that step's placement.
func coilSections(prof *sketch.Profile, plane sketch.Plane, axis *WorkAxis, place coilPlacer,
	revolutions float64, hand CoilHandedness) [][]math.Point3 {
	base := modelPolygon(prof, plane)
	sense := hand.rotationSense()
	k := int(stdmath.Max(3, stdmath.Round(revolveSegments*revolutions)))
	total := 2 * stdmath.Pi * revolutions
	sections := make([][]math.Point3, k+1)
	for s := 0; s <= k; s++ {
		angle := total * float64(s) / float64(k)
		sections[s] = coilSection(base, axis, sense*angle, place(angle))
	}
	return sections
}

// coilSection is one placed cross-section: the base polygon rotated about the axis and displaced
// axially then radially.
func coilSection(base []math.Point3, axis *WorkAxis, angle float64, at coilPlacement) []math.Point3 {
	rot := math.Rotation4(angle, axis.Direction(), axis.Origin())
	axisVec := axis.Direction().AsVector()
	out := make([]math.Point3, len(base))
	for i, p := range base {
		q := rot.TransformPoint(p).TranslateBy(axisVec.Scale(math.Scalar(at.rise)))
		out[i] = coilRadialOffset(q, axis, at.radial)
	}
	return out
}

// coilRadialOffset moves a placed section point directly away from the axis by d. The direction
// is taken from the ROTATED point, so it is that point's own radial direction — what both the
// taper and the spiral advance need.
func coilRadialOffset(p math.Point3, axis *WorkAxis, d float64) math.Point3 {
	if d == 0 {
		return p
	}
	a := axis.Direction().AsVector()
	v := axis.Origin().VectorTo(p)
	radial := v.Sub(a.Scale(v.Dot(a)))
	l := float64(radial.Length())
	if l == 0 {
		return p // a point ON the axis has no radial direction to move along
	}
	return p.TranslateBy(radial.Scale(math.Scalar(d / l)))
}
