// SPDX-License-Identifier: GPL-2.0-only

package feature

import "fmt"

// How much of the picked edge a flange's wall covers (#1958). A flange need not span the whole
// edge: a bracket tab is a short wall centred on a long edge, and a chassis wall stops short of
// the corners so the neighbouring flanges have somewhere to go. Without this every flange is
// full-width, which is a large class of real parts that cannot be modelled at all.
//
// Every extent resolves to one sub-span of the edge, which the wall is extruded over AND which
// becomes the bend line the flat pattern develops — so a partial wall unfolds as the partial tab
// it is, with no separate bookkeeping.

// WidthExtent is how a flange's width is specified (Inventor's flange width-extent setters).
type WidthExtent int

const (
	// WidthFullEdge spans the whole picked edge — Inventor's SetEdgeWidthExtent, and the default,
	// so a flange authored before widths existed is unchanged.
	WidthFullEdge WidthExtent = iota
	// WidthCentered is a wall of Width centred on the edge (SetCenteredWidthExtent).
	WidthCentered
	// WidthOffsets is the edge less an offset at each end (SetOffsetWidthExtent).
	WidthOffsets
	// WidthOffsetAndWidth is a wall of Width starting Offset from the edge's start
	// (SetWidthOffsetWidthExtent).
	WidthOffsetAndWidth
)

// widthExtentNames is the stable wire/recipe vocabulary for the extents — one source shared by the
// op registry and the .obk codec so they cannot drift.
var widthExtentNames = map[WidthExtent]string{
	WidthFullEdge:       "edge",
	WidthCentered:       "centered",
	WidthOffsets:        "offsets",
	WidthOffsetAndWidth: "offsetWidth",
}

// WidthExtentName renders an extent as its stable name ("" for the full-edge default).
func WidthExtentName(w WidthExtent) string {
	if w == WidthFullEdge {
		return ""
	}
	return widthExtentNames[w]
}

// ParseWidthExtent maps a name to its extent (empty ⇒ the full edge); ok is false for an unknown
// name, which must not fall back to the full edge and build a wall the caller did not ask for.
// "fromTo" is Inventor's fifth extent and is deliberately absent: it is bounded by two REFERENCED
// entities rather than by distances, which needs vertex/plane reference binding this build does
// not have.
func ParseWidthExtent(name string) (WidthExtent, bool) {
	if name == "" {
		return WidthFullEdge, true
	}
	for w, n := range widthExtentNames {
		if n == name {
			return w, true
		}
	}
	return WidthFullEdge, false
}

// FlangeWidth is a flange's width extent and the distances it takes. Each is a closure so a
// parameter drives it, like every other flange dimension.
type FlangeWidth struct {
	Type WidthExtent
	// Width is the wall's length for the centred and offset-plus-width extents.
	Width func() float64
	// Offset is the distance from the edge's start for the offset extents; Offset2 is the distance
	// from its end, used only by WidthOffsets.
	Offset, Offset2 func() float64
}

// span resolves the extent against an edge of the given length, returning the sub-span the wall
// covers. A span that would fall outside the edge, or invert, is an error: clamping it would build
// a wall of a different width than the one asked for and say nothing.
func (w FlangeWidth) span(edgeLength float64) (from, to float64, err error) {
	from, to = w.rawSpan(edgeLength)
	if to-from <= 0 || from < -1e-9 || to > edgeLength+1e-9 {
		return 0, 0, fmt.Errorf("sheet-metal flange: the %q width extent puts the wall over [%g, %g] "+
			"of a %g-long edge; it must be a positive span inside the edge",
			widthExtentNameOr(w.Type), from, to, edgeLength)
	}
	return from, to, nil
}

// rawSpan applies the extent's arithmetic, before validation.
func (w FlangeWidth) rawSpan(edgeLength float64) (from, to float64) {
	width, offset := evalFloat(w.Width), evalFloat(w.Offset)
	switch w.Type {
	case WidthCentered:
		return (edgeLength - width) / 2, (edgeLength + width) / 2
	case WidthOffsets:
		return offset, edgeLength - evalFloat(w.Offset2)
	case WidthOffsetAndWidth:
		return offset, offset + width
	default:
		return 0, edgeLength
	}
}

// widthExtentNameOr renders an extent for an error message, naming the full-edge default.
func widthExtentNameOr(w WidthExtent) string {
	if n := WidthExtentName(w); n != "" {
		return n
	}
	return "edge"
}
