// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"oblikovati.org/api/types"
)

// Surface texture symbols (M14-F03 PBI-142, #389): the ISO 1302 checkmark glyph with a roughness
// value, stating a surface's finish requirement. The checkmark (and the material-removal variant
// bar/circle) are drawing curves; the roughness value is a text label.

// surfTickMM sizes the surface-texture checkmark.
const surfTickMM = 4.0

// AddSurfaceTexture adds an ISO 1302 surface texture symbol at (x, y) on the sheet (the checkmark's
// vertex — the point that touches the surface), with the given roughness value and variant.
func (as *DrawingAnnotations) AddSurfaceTexture(name string, x, y float64, roughness string, variant types.MaterialRemoval) (*DrawingAnnotation, error) {
	a := &DrawingAnnotation{
		name: as.uniqueName(name), kind: types.SurfaceTextureAnnotation,
		x: x, y: y, tag: roughness, materialRemoval: variant,
	}
	a.curves, a.labels = surfaceTextureGeometry(x, y, roughness, variant)
	as.items = append(as.items, a)
	return a, nil
}

// surfaceTextureGeometry builds the surface-texture glyph (sheet mm): a checkmark whose vertex is at
// (x, y), a horizontal extension line off its tall right arm, the variant bar/circle, and the
// roughness value label above the extension.
func surfaceTextureGeometry(x, y float64, roughness string, variant types.MaterialRemoval) ([]DrawingCurve, []AnnotationLabel) {
	t := surfTickMM
	leftTop := [2]float64{x - t*0.75, y + t}      // short left arm (up-left)
	rightTop := [2]float64{x + t*1.25, y + t*2.2} // tall right arm (up-right)
	extEnd := [2]float64{rightTop[0] + t*3, rightTop[1]}
	curves := []DrawingCurve{
		dimSegment(x, y, leftTop[0], leftTop[1]),
		dimSegment(x, y, rightTop[0], rightTop[1]),
		dimSegment(rightTop[0], rightTop[1], extEnd[0], extEnd[1]),
	}
	curves = append(curves, materialRemovalGlyph(x, y, leftTop, rightTop, variant)...)
	var labels []AnnotationLabel
	if roughness != "" {
		labels = append(labels, AnnotationLabel{Text: roughness, X: (rightTop[0] + extEnd[0]) / 2, Y: rightTop[1] + t*0.6})
	}
	return curves, labels
}

// materialRemovalGlyph adds the variant marker: a bar across the checkmark (machining required) or a
// small circle in its vertex (machining prohibited); the basic "any" variant adds nothing.
func materialRemovalGlyph(vertexX, vertexY float64, leftTop, rightTop [2]float64, variant types.MaterialRemoval) []DrawingCurve {
	switch variant {
	case types.MaterialRemovalRequired:
		return []DrawingCurve{dimSegment(leftTop[0], leftTop[1], rightTop[0], rightTop[1])}
	case types.MaterialRemovalProhibited:
		return circlePolyline(vertexX, vertexY+surfTickMM*0.45, surfTickMM*0.45)
	default:
		return nil
	}
}
