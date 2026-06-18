// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"fmt"
	"math"
	"unicode/utf8"

	"oblikovati.org/api/types"
)

// GD&T feature control frames (M14-F03 PBI-142, #389): a boxed geometric-tolerance callout placed
// on the sheet. Compartments left-to-right: the geometric characteristic symbol, the tolerance
// value, then one datum reference per compartment. The frame and symbol are drawing curves; the
// tolerance and datum letters are text labels (the head renders them at their anchors).

// fcfHeightMM is the compartment height; charWidthMM approximates the head's text glyph advance.
const (
	fcfHeightMM  = 8.0
	charWidthMM  = 2.4
	fcfPadMM     = 2.0
	datumWidthMM = 6.0
)

// AddFeatureControlFrame adds a GD&T feature control frame at (x, y) on the sheet (its top-left
// corner), stating characteristic with the given tolerance text and ordered datum references.
func (as *DrawingAnnotations) AddFeatureControlFrame(name string, x, y float64, characteristic types.GeometricCharacteristic, tolerance string, datums []string) (*DrawingAnnotation, error) {
	if tolerance == "" {
		return nil, fmt.Errorf("drawing: a feature control frame needs a tolerance value")
	}
	if characteristic.String() == "" {
		return nil, fmt.Errorf("drawing: unknown geometric characteristic %d", characteristic)
	}
	a := &DrawingAnnotation{
		name: as.uniqueName(name), kind: types.FeatureControlFrameAnnotation,
		x: x, y: y, characteristic: characteristic, tolerance: tolerance, datums: append([]string(nil), datums...),
	}
	a.curves, a.labels = featureControlFrameGeometry(x, y, characteristic, tolerance, datums)
	as.items = append(as.items, a)
	return a, nil
}

// featureControlFrameGeometry builds the frame's compartment boxes + characteristic symbol (drawing
// curves) and its tolerance/datum text (labels), with (x, y) the frame's top-left corner.
func featureControlFrameGeometry(x, y float64, characteristic types.GeometricCharacteristic, tolerance string, datums []string) ([]DrawingCurve, []AnnotationLabel) {
	widths := compartmentWidths(characteristic, tolerance, datums)
	var curves []DrawingCurve
	var labels []AnnotationLabel
	left := x
	for i, w := range widths {
		curves = append(curves, compartmentContent(i, left, y, w, characteristic, tolerance, datums, &labels)...)
		left += w
	}
	return append(curves, frameBox(x, y, left-x)...), labels
}

// compartmentWidths sizes each compartment: the symbol box, the tolerance box (text-fit) and one
// box per datum.
func compartmentWidths(characteristic types.GeometricCharacteristic, tolerance string, datums []string) []float64 {
	out := []float64{fcfHeightMM, textBoxWidth(tolerance)}
	_ = characteristic
	for range datums {
		out = append(out, datumWidthMM)
	}
	return out
}

// compartmentContent draws compartment i: the symbol (curves) for the first, otherwise a centred
// text label (the tolerance, then each datum); it appends the label and returns the divider line.
func compartmentContent(i int, left, top, w float64, characteristic types.GeometricCharacteristic, tolerance string, datums []string, labels *[]AnnotationLabel) []DrawingCurve {
	cx, cy := left+w/2, top+fcfHeightMM/2
	var curves []DrawingCurve
	switch i {
	case 0:
		if sym := characteristicSymbolCurves(characteristic, cx, cy, fcfHeightMM*0.5); sym != nil {
			curves = sym
		} else {
			*labels = append(*labels, AnnotationLabel{Text: characteristicAbbrev(characteristic), X: cx, Y: cy})
		}
	case 1:
		*labels = append(*labels, AnnotationLabel{Text: tolerance, X: cx, Y: cy})
	default:
		*labels = append(*labels, AnnotationLabel{Text: datums[i-2], X: cx, Y: cy})
	}
	if left > 0 && i > 0 { // a divider before every compartment except the first
		curves = append(curves, dimSegment(left, top, left, top+fcfHeightMM))
	}
	return curves
}

// frameBox is the outer rectangle of a frame of total width w (top-left at x, y).
func frameBox(x, y, w float64) []DrawingCurve {
	return []DrawingCurve{
		dimSegment(x, y, x+w, y), dimSegment(x+w, y, x+w, y+fcfHeightMM),
		dimSegment(x+w, y+fcfHeightMM, x, y+fcfHeightMM), dimSegment(x, y+fcfHeightMM, x, y),
	}
}

// textBoxWidth sizes a text compartment to its content plus padding.
func textBoxWidth(s string) float64 {
	return math.Max(fcfHeightMM, float64(utf8.RuneCountInString(s))*charWidthMM+2*fcfPadMM)
}

// characteristicSymbolCurves draws the geometric-characteristic symbol centred at (cx, cy) at the
// given half-size (sheet mm), for the characteristics with a line-drawable glyph; it returns nil
// for the rest (the caller falls back to a text abbreviation).
func characteristicSymbolCurves(c types.GeometricCharacteristic, cx, cy, r float64) []DrawingCurve {
	switch c {
	case types.CharacteristicPosition:
		out := circlePolyline(cx, cy, r)
		return append(out, dimSegment(cx-r*1.4, cy, cx+r*1.4, cy), dimSegment(cx, cy-r*1.4, cx, cy+r*1.4))
	case types.CharacteristicCircularity:
		return circlePolyline(cx, cy, r)
	case types.CharacteristicConcentricity:
		return append(circlePolyline(cx, cy, r), circlePolyline(cx, cy, r*0.55)...)
	case types.CharacteristicStraightness:
		return []DrawingCurve{dimSegment(cx-r*1.4, cy, cx+r*1.4, cy)}
	case types.CharacteristicFlatness:
		return parallelogram(cx, cy, r)
	case types.CharacteristicPerpendicularity:
		return []DrawingCurve{dimSegment(cx, cy-r, cx, cy+r), dimSegment(cx-r, cy+r, cx+r, cy+r)}
	case types.CharacteristicParallelism:
		return []DrawingCurve{dimSegment(cx-r*0.2, cy-r, cx-r, cy+r), dimSegment(cx+r, cy-r, cx+r*0.2, cy+r)}
	case types.CharacteristicAngularity:
		return []DrawingCurve{dimSegment(cx-r, cy+r, cx+r, cy-r), dimSegment(cx-r, cy+r, cx+r, cy+r)}
	default:
		return nil
	}
}

// parallelogram draws the flatness glyph: a leaning quadrilateral centred at (cx, cy).
func parallelogram(cx, cy, r float64) []DrawingCurve {
	tl, tr := [2]float64{cx - r*0.4, cy - r}, [2]float64{cx + r, cy - r}
	br, bl := [2]float64{cx + r*0.4, cy + r}, [2]float64{cx - r, cy + r}
	return []DrawingCurve{
		dimSegment(tl[0], tl[1], tr[0], tr[1]), dimSegment(tr[0], tr[1], br[0], br[1]),
		dimSegment(br[0], br[1], bl[0], bl[1]), dimSegment(bl[0], bl[1], tl[0], tl[1]),
	}
}

// characteristicAbbrev is the short text shown when a characteristic has no line-drawn symbol.
func characteristicAbbrev(c types.GeometricCharacteristic) string {
	switch c {
	case types.CharacteristicCylindricity:
		return "CYL"
	case types.CharacteristicProfileOfAnyLine:
		return "PROL"
	case types.CharacteristicProfileOfAnySurface:
		return "PROS"
	case types.CharacteristicSymmetry:
		return "SYM"
	case types.CharacteristicCircularRunout:
		return "RUN"
	case types.CharacteristicTotalRunout:
		return "TRUN"
	default:
		return "GD&T"
	}
}
