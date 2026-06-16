// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	stdmath "math"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/ops"
	gmath "oblikovati.org/math"
	"oblikovati.org/model/sheetmetal"
)

// Flat-pattern orientation reporting (M13-F05, #635). The active orientation frames the
// developed flat: its alignment rotation turns the flat in its plane and its alignment type
// decides which extent is the length. FlatLengthWidth develops the flat and measures it under
// a given orientation, so drawing views and export read a stable length/width.

// FlatLengthWidth returns the flat pattern's length and width under the orientation: the
// footprint is turned by the negative alignment rotation (bringing the alignment axis to the
// page horizontal) and bounded; a vertical alignment swaps the two extents.
func (d *PartComponentDefinition) FlatLengthWidth(or *sheetmetal.FlatPatternOrientation) (length, width float64, err error) {
	fp, err := d.Unfold()
	if err != nil {
		return 0, 0, err
	}
	plane := fp.Plane
	cos, sin := stdmath.Cos(-or.AlignmentRotation), stdmath.Sin(-or.AlignmentRotation)
	box := gmath.EmptyBox2d()
	for _, v := range fp.Body.Vertices() {
		p := plane.ToSketch(v.Point())
		box = box.ExtendPoint(gmath.P2(p.X*cos-p.Y*sin, p.X*sin+p.Y*cos))
	}
	diag := box.Diagonal()
	length, width = float64(diag.X), float64(diag.Y)
	if or.AlignmentType == types.VerticalAlignment {
		length, width = width, length
	}
	return length, width, nil
}

// FlatSettings returns the part's flat-pattern settings.
func (d *PartComponentDefinition) FlatSettings() sheetmetal.FlatPatternSettings {
	return d.flatSettings
}

// SetFlatDeferUpdate sets the deferred-flat-pattern-update flag.
func (d *PartComponentDefinition) SetFlatDeferUpdate(deferred bool) {
	d.flatSettings.DeferUpdate = deferred
}

// FlatPlate is one developed flat plate (a connected flat region) with its extents and area
// under the active orientation.
type FlatPlate struct {
	Length, Width, Area float64
}

// FlatPlates returns the developed flat's plates under the active orientation. A single-base
// sheet-metal part develops one connected plate; multi-body parts (each base its own region)
// are a follow-up, so this reports the one plate the model authors.
func (d *PartComponentDefinition) FlatPlates() ([]FlatPlate, error) {
	fp, err := d.Unfold()
	if err != nil {
		return nil, err
	}
	length, width, err := d.FlatLengthWidth(d.flatOrientations.Active())
	if err != nil {
		return nil, err
	}
	area := 0.0
	if fp.Thickness > 0 {
		area = ops.BodyGeometryProperties(fp.Body, ops.Quality{ChordTolerance: 1e-3}).Volume / fp.Thickness
	}
	return []FlatPlate{{Length: length, Width: width, Area: area}}, nil
}
