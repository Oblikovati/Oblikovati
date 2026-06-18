// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"fmt"

	"oblikovati.org/api/types"
)

// GD&T datum feature symbols (M14-F03 PBI-142, #389): the datum letter in a square box with a
// filled datum triangle, marking a datum that feature control frames reference. The box and
// triangle are drawing curves; the letter is a text label.

// datumBoxMM is the lettered box side; datumTriMM is the datum triangle's base width.
const (
	datumBoxMM  = 8.0
	datumStemMM = 4.0
	datumTriMM  = 5.0
)

// AddDatumFeature adds a GD&T datum feature symbol at (x, y) on the sheet (the box's top-left
// corner), labelled with the datum letter.
func (as *DrawingAnnotations) AddDatumFeature(name string, x, y float64, letter string) (*DrawingAnnotation, error) {
	if letter == "" {
		return nil, fmt.Errorf("drawing: a datum feature symbol needs a datum letter")
	}
	a := &DrawingAnnotation{name: as.uniqueName(name), kind: types.DatumFeatureAnnotation, x: x, y: y, tag: letter}
	a.curves, a.labels = datumFeatureGeometry(x, y, letter)
	as.items = append(as.items, a)
	return a, nil
}

// datumFeatureGeometry builds the datum symbol (sheet mm): a lettered box at (x, y), a short stem
// down from its bottom centre, and a datum triangle at the stem's end; plus the letter label.
func datumFeatureGeometry(x, y float64, letter string) ([]DrawingCurve, []AnnotationLabel) {
	curves := frameSquare(x, y, datumBoxMM)
	cx := x + datumBoxMM/2
	stemEnd := y + datumBoxMM + datumStemMM
	curves = append(curves, dimSegment(cx, y+datumBoxMM, cx, stemEnd))
	curves = append(curves, datumTriangle(cx, stemEnd)...)
	labels := []AnnotationLabel{{Text: letter, X: cx, Y: y + datumBoxMM/2}}
	return curves, labels
}

// frameSquare is a square outline of side s with its top-left corner at (x, y).
func frameSquare(x, y, s float64) []DrawingCurve {
	return []DrawingCurve{
		dimSegment(x, y, x+s, y), dimSegment(x+s, y, x+s, y+s),
		dimSegment(x+s, y+s, x, y+s), dimSegment(x, y+s, x, y),
	}
}

// datumTriangle draws the datum triangle (an upright filled triangle, drawn as its outline plus a
// couple of fill strokes) whose apex sits at (apexX, apexY), pointing up toward the stem.
func datumTriangle(apexX, apexY float64) []DrawingCurve {
	half := datumTriMM / 2
	baseY := apexY + datumTriMM
	left, right := apexX-half, apexX+half
	out := []DrawingCurve{
		dimSegment(apexX, apexY, left, baseY),
		dimSegment(apexX, apexY, right, baseY),
		dimSegment(left, baseY, right, baseY),
	}
	// A pair of interior strokes stands in for the solid fill (the head only strokes lines).
	return append(out, dimSegment(apexX, apexY, apexX, baseY), dimSegment((apexX+left)/2, (apexY+baseY)/2, (apexX+right)/2, (apexY+baseY)/2))
}
