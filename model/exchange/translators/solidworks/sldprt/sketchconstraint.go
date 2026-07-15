// SPDX-License-Identifier: GPL-2.0-only

package sldprt

import (
	"bytes"
	"encoding/binary"
)

// ConstraintKind is a SolidWorks sketch-relation type (swConstraintType_e). Only the kinds the
// translator acts on are named; others decode to their raw code.
type ConstraintKind uint32

const (
	Distance      ConstraintKind = 1 // a driven/driving dimension (handled as a dimension, not here)
	Angle         ConstraintKind = 2
	Radius        ConstraintKind = 3
	Horizontal    ConstraintKind = 4
	Vertical      ConstraintKind = 5
	Tangent       ConstraintKind = 6
	Parallel      ConstraintKind = 7
	Perpendicular ConstraintKind = 8
	Coincident    ConstraintKind = 9
	Concentric    ConstraintKind = 10
	Symmetric     ConstraintKind = 11
	Midpoint      ConstraintKind = 12
	EqualLength   ConstraintKind = 14
	Diameter      ConstraintKind = 15
	Fixed         ConstraintKind = 17
	Collinear     ConstraintKind = 27
)

// Constraint is one decoded sketch relation. Its entities are not yet resolved (that needs the MFC
// object-graph walk); the translator applies a geometric constraint to the matching emitted
// geometry, guarded by a degrees-of-freedom check — the Inventor-translator approach.
type Constraint struct {
	Kind ConstraintKind
}

// constraintMarker follows a relation's type code in the sketch serialization:
// [type:u32][02 00 00 00][00 00 fe ff][00 00][entity-ref…]. Reverse-engineered from generated parts
// and validated against the SolidWorks 2026 SketchRelationManager (a rectangle: 2 horizontal, 2
// vertical, coincident, equal-length; a rounded rectangle: + 3 tangent).
var constraintMarker = []byte{0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0xfe, 0xff}

// constraintsIn decodes the sketch relations in a sketch's byte region. It anchors on
// constraintMarker and reads the preceding type code, keeping only known relation kinds so the
// marker matching a coincidental byte run is filtered out.
func constraintsIn(region []byte) []Constraint {
	var out []Constraint
	for i := 0; i < len(region); {
		rel := bytes.Index(region[i:], constraintMarker)
		if rel < 0 {
			break
		}
		m := i + rel
		i = m + 1
		if m < 4 {
			continue
		}
		k := ConstraintKind(binary.LittleEndian.Uint32(region[m-4:]))
		if knownConstraint(k) {
			out = append(out, Constraint{Kind: k})
		}
	}
	return out
}

// knownConstraint reports whether k is a relation kind the decoder recognises (guards the marker
// scan against false positives).
func knownConstraint(k ConstraintKind) bool {
	switch k {
	case Distance, Angle, Radius, Horizontal, Vertical, Tangent, Parallel, Perpendicular,
		Coincident, Concentric, Symmetric, Midpoint, EqualLength, Diameter, Fixed, Collinear:
		return true
	}
	return false
}

// IsGeometric reports whether a constraint carries no value and can be applied to already-satisfied
// geometry (as opposed to a dimension, which sets a measured value).
func (k ConstraintKind) IsGeometric() bool {
	switch k {
	case Horizontal, Vertical, Tangent, Parallel, Perpendicular, Coincident, Concentric,
		Symmetric, Midpoint, EqualLength, Collinear:
		return true
	}
	return false
}
