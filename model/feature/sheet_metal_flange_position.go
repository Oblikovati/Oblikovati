// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"
	stdmath "math"
)

// Where a flange's wall LANDS (#1957) — two controls that decide the part's overall dimensions
// without changing its height number, which is why two flanges with identical height and angle can
// still be different parts:
//
//   - BendPosition shifts the whole folded section along the sheet, so the bend can start at the
//     picked edge or sit back far enough for the wall to finish flush with it;
//   - HeightDatum says where the height is MEASURED from — the bend's tangent, or the corner the
//     outer/inner faces would make if the bend were sharp.
//
// Both come out as an offset applied to the section: a leading run for the position, a shortened
// wall run for the datum. See sheet_metal_band.go for the tracer they feed.

// BendPosition is where the bend sits relative to the picked edge (Inventor's BendPositionEnum).
type BendPosition int

const (
	// BendAtAdjacentFace starts the bend AT the picked edge — Inventor's kBendPositionAdjacentFace,
	// "starts the bend at the edge of the selected face". It is the zero value because it is what
	// this feature has always built, so an existing flange does not move.
	BendAtAdjacentFace BendPosition = iota
	// BendOutsideBaseFace sets the section back so the wall's OUTER face finishes flush with the
	// sheet's edge face — the wall does not overhang the part. Inventor's
	// kBendPositionOutsideBaseFace, "the outside face of the bend aligns to the selected edge".
	BendOutsideBaseFace
	// BendInsideBendFace sets it back so the wall's INNER face lands on the edge face instead —
	// Inventor's kBendPositionInsideBendFace.
	BendInsideBendFace
	// BendOuterEdgeOffset / BendInnerEdgeOffset are the two flush positions moved by an explicit
	// distance (Inventor's kBendPositionOuterEdgeOffset / kBendPositionInnerEdgeOffset).
	BendOuterEdgeOffset
	BendInnerEdgeOffset
)

// bendPositionNames is the stable wire/recipe vocabulary for the positions — one source shared by
// the op registry and the .obk codec so they cannot drift.
var bendPositionNames = map[BendPosition]string{
	BendAtAdjacentFace:  "adjacentFace",
	BendOutsideBaseFace: "outsideBaseFace",
	BendInsideBendFace:  "insideBendFace",
	BendOuterEdgeOffset: "outerEdgeOffset",
	BendInnerEdgeOffset: "innerEdgeOffset",
}

// BendPositionName renders a bend position as its stable name ("" for the default, so the common
// case serializes nothing).
func BendPositionName(p BendPosition) string {
	if p == BendAtAdjacentFace {
		return ""
	}
	return bendPositionNames[p]
}

// ParseBendPosition maps a name to its position (empty ⇒ the adjacent-face default). ok is false
// for an unknown name — including the positions Inventor has and this build does not, which must
// be refused rather than silently placed somewhere else. Those are the two reference-plane
// positions (they need the flange angle taken from a reference, which is not modelled) and the
// tangent-to-side-face and flipped variants, whose exact offsets need a live comparison to pin.
func ParseBendPosition(name string) (BendPosition, bool) {
	if name == "" {
		return BendAtAdjacentFace, true
	}
	for p, n := range bendPositionNames {
		if n == name {
			return p, true
		}
	}
	return BendAtAdjacentFace, false
}

// setbackFor returns how far the section moves back along the sheet for this position: far enough
// for the named face to reach the picked edge, plus any explicit offset. A positive result moves
// INTO the material, so the leading run is its negation.
func (p BendPosition) setbackFor(radius, thickness, offset float64) float64 {
	switch p {
	case BendOutsideBaseFace:
		return radius + thickness
	case BendInsideBendFace:
		return radius
	case BendOuterEdgeOffset:
		return radius + thickness + offset
	case BendInnerEdgeOffset:
		return radius + offset
	default:
		return 0
	}
}

// HeightDatum says where a flange's height is measured from (Inventor's HeightDatumTypeEnum).
type HeightDatum int

const (
	// HeightFromTangent measures the wall from where the bend ends — what this feature has always
	// built, so it is the zero value and an existing flange keeps its size.
	HeightFromTangent HeightDatum = iota
	// HeightFromOuterFace measures from the sharp corner the OUTER faces would make, the ordinary
	// outside dimension on a drawing (Inventor's kHeightDatumOuter).
	HeightFromOuterFace
	// HeightFromInnerFace measures from the inner faces' corner (kHeightDatumInner).
	HeightFromInnerFace
	// HeightFromOuterFaceOrtho / HeightFromInnerFaceOrtho measure the same corners but PERPENDICULAR
	// to the base face rather than along the wall, so on anything but a right-angle bend they are a
	// different number (kHeightDatumOuterOrtho / kHeightDatumInnerOrtho).
	HeightFromOuterFaceOrtho
	HeightFromInnerFaceOrtho
)

// heightDatumNames is the stable wire/recipe vocabulary for the datums.
var heightDatumNames = map[HeightDatum]string{
	HeightFromTangent:        "tangent",
	HeightFromOuterFace:      "outer",
	HeightFromInnerFace:      "inner",
	HeightFromOuterFaceOrtho: "outerOrtho",
	HeightFromInnerFaceOrtho: "innerOrtho",
}

// HeightDatumName renders a height datum as its stable name ("" for the tangent default).
func HeightDatumName(d HeightDatum) string {
	if d == HeightFromTangent {
		return ""
	}
	return heightDatumNames[d]
}

// ParseHeightDatum maps a name to its datum (empty ⇒ tangent); ok is false for an unknown name,
// which must not fall back to the tangent and build a wall of a different length.
func ParseHeightDatum(name string) (HeightDatum, bool) {
	if name == "" {
		return HeightFromTangent, true
	}
	for d, n := range heightDatumNames {
		if n == name {
			return d, true
		}
	}
	return HeightFromTangent, false
}

// wallRun converts a height measured against this datum into the straight run after the bend.
//
// The sharp corner the outer faces would make sits (radius+thickness)·tan(angle/2) BEFORE the
// bend's outer tangent — the standard bend setback — and the inner corner radius·tan(angle/2)
// before the inner one; measuring from either therefore spends that much of the height inside the
// bend. An ortho datum measures the same corner perpendicular to the base face, so its height is
// the along-wall length times sin(angle).
func (d HeightDatum) wallRun(height, radius, thickness, angle float64) (float64, error) {
	along, err := d.alongWall(height, angle)
	if err != nil {
		return 0, err
	}
	run := along - d.setback(radius, thickness, angle)
	if run <= 0 {
		return 0, fmt.Errorf("sheet-metal flange: a height of %g measured from the %q datum is spent "+
			"inside the bend (its setback at radius %g, thickness %g, angle %g rad), leaving no wall; "+
			"raise the height or measure it from the tangent", height, heightDatumNameOr(d), radius,
			thickness, angle)
	}
	return run, nil
}

// alongWall converts an orthogonal height into a distance along the wall.
func (d HeightDatum) alongWall(height, angle float64) (float64, error) {
	if d != HeightFromOuterFaceOrtho && d != HeightFromInnerFaceOrtho {
		return height, nil
	}
	sin := stdmath.Sin(angle)
	if stdmath.Abs(sin) < 1e-9 {
		return 0, fmt.Errorf("sheet-metal flange: an orthogonal height cannot be measured on a %g rad "+
			"bend, whose wall runs back along the base face; measure it along the wall instead", angle)
	}
	return height / sin, nil
}

// setback is how much of the measured height the bend itself consumes.
func (d HeightDatum) setback(radius, thickness, angle float64) float64 {
	half := stdmath.Tan(angle / 2)
	switch d {
	case HeightFromOuterFace, HeightFromOuterFaceOrtho:
		return (radius + thickness) * half
	case HeightFromInnerFace, HeightFromInnerFaceOrtho:
		return radius * half
	default:
		return 0
	}
}

// heightDatumNameOr renders a datum for an error message, naming the tangent default explicitly.
func heightDatumNameOr(d HeightDatum) string {
	if n := HeightDatumName(d); n != "" {
		return n
	}
	return "tangent"
}
