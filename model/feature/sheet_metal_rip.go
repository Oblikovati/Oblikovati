// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// Sheet-metal Rip feature (M13-F03). A rip cuts a narrow slit of a given gap width through the
// full thickness — opening a seam so a closed or folded sheet can be developed flat (a rip across
// a corner or roll lets the part unfold). The geometry is a thin rectangular cutter subtracted
// through-all, so a rip is the boolean complement of a sliver prism — the same machinery the Cut
// feature uses, narrowed to a line.
//
// Inventor's RipTypeEnum (#1965) draws that line three ways, all on a picked RipFace: point to
// point (the long-standing sketch-line form), single point (the face's full extent through one
// point), or the whole face's extents. GapSide places the removed material to one side of the
// line or straddling it (PartFeatureExtentDirectionEnum).

// ripOvershoot extends the cutter a hair past the line's endpoints so a rip that reaches a
// boundary cuts cleanly through it rather than leaving a coincident-face sliver.
const ripOvershoot = 1e-4

// RipType aliases the API's RipTypeEnum so the vocabulary is defined once (ADR-0018); the
// long-standing sketch-line rip is the PointToPointRip default.
type RipType = types.RipType

const (
	// PointToPointRip rips between two points (a sketch line, or two face vertices) — the default.
	PointToPointRip = types.PointToPointRip
	// SinglePointRip rips a face's full extent through one point; FaceExtentsRip rips the whole face.
	SinglePointRip = types.SinglePointRip
	FaceExtentsRip = types.FaceExtentsRip
)

// ParseRipType resolves a wire spelling to a rip type, delegating to the API enum.
func ParseRipType(s string) (RipType, bool) { return types.ParseRipType(s) }

// SheetMetalRipDefinition is the rip recipe. The sketch line (Sketch/LineIndex) drives the
// long-standing point-to-point rip; the face-based forms (#1965) name a RipFace by reference key
// and, for single-point / two-vertex point-to-point, the vertices the rip runs through. GapSide
// places the slit; it defaults to SymmetricDir at the wire boundary (a bare zero value is
// PositiveDir, per ExtentDirection, so every real construction sets it).
type SheetMetalRipDefinition struct {
	Sketch    *sketch.Sketch
	LineIndex int
	Gap       func() float64
	Type      RipType
	FaceKey   []byte
	// PointKey / PointTwoKey are vertex reference keys on FaceKey: PointKey for a single-point rip,
	// both for a two-vertex point-to-point rip.
	PointKey    []byte
	PointTwoKey []byte
	GapSide     ExtentDirection
}

// SheetMetalRipFeature slits the running sheet along the rip line each recompute.
type SheetMetalRipFeature struct {
	def      *SheetMetalRipDefinition
	featName string
}

// Definition returns the rip recipe.
func (f *SheetMetalRipFeature) Definition() *SheetMetalRipDefinition { return f.def }

// Kind identifies the feature for serialization and the model tree.
func (f *SheetMetalRipFeature) Kind() string { return "sheet-metal-rip" }

// Recompute builds the slit cutter and subtracts it through the sheet. A face-based rip (a
// RipFace is named) derives its line from the face; otherwise the sketch line drives it.
func (f *SheetMetalRipFeature) Recompute(in Input) (Output, error) {
	gap := evalFloat(f.def.Gap)
	if gap <= 0 {
		return Output{}, fmt.Errorf("sheet-metal rip: gap must be positive, got %g", gap)
	}
	if len(f.def.FaceKey) > 0 {
		return f.ripFace(in, gap)
	}
	return f.ripSketchLine(in, gap)
}

// ripSketchLine cuts the slit along the recipe's sketch line — the point-to-point rip's default,
// pre-face form.
func (f *SheetMetalRipFeature) ripSketchLine(in Input, gap float64) (Output, error) {
	a, b, err := sketchLineEnds(f.def.Sketch, f.def.LineIndex, "sheet-metal rip")
	if err != nil {
		return Output{}, err
	}
	plane := f.def.Sketch.Plane()
	sp, err := throughAllSpan(Extent{Type: ThroughAllExtent}, in.Bodies, plane)
	if err != nil {
		return Output{}, err
	}
	tool := buildPrism(ripPolygon(a, b, gap, f.def.GapSide), plane, sp, 0, f.featName)
	bodies, err := combine(in, tool, ops.Cut)
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: bodies}, nil
}

// ripPolygon is the slit cutter cross-section: the rectangle bounded by the line offset by the
// gap on the side GapSide selects (symmetric ±gap/2, or the whole gap to one side), extended by
// ripOvershoot past each end so a boundary-reaching rip cuts through.
func ripPolygon(a, b math.Point2, gap float64, side ExtentDirection) []math.Point2 {
	dir := a.VectorTo(b)
	if l := dir.Length(); l > 0 {
		dir = dir.Scale(1 / l)
	}
	perp := math.V2(-dir.Y, dir.X)
	lo, hi := ripGapOffsets(gap, side)
	a = a.TranslateBy(dir.Scale(-ripOvershoot))
	b = b.TranslateBy(dir.Scale(ripOvershoot))
	return []math.Point2{
		a.TranslateBy(perp.Scale(lo)), b.TranslateBy(perp.Scale(lo)),
		b.TranslateBy(perp.Scale(hi)), a.TranslateBy(perp.Scale(hi)),
	}
}

// ripGapOffsets is the slit's low/high perpendicular bounds for a gap side: symmetric straddles
// the line (±gap/2), positive takes the gap wholly to the +perp side, negative to the −perp side.
func ripGapOffsets(gap float64, side ExtentDirection) (lo, hi float64) {
	switch side {
	case PositiveDir:
		return 0, gap
	case NegativeDir:
		return -gap, 0
	default:
		return -gap / 2, gap / 2
	}
}

// ParseRipGapSide resolves a gap-side spelling. Both "" (an omitted side, and a legacy record that
// predates the field) and "symmetric" mean the symmetric straddle the rip has always used — the
// rip's default is symmetric even though ExtentDirection's zero value is PositiveDir.
func ParseRipGapSide(s string) (ExtentDirection, bool) {
	switch s {
	case "", "symmetric":
		return SymmetricDir, true
	case "positive":
		return PositiveDir, true
	case "negative":
		return NegativeDir, true
	}
	return SymmetricDir, false
}

// ripGapSideName is the wire spelling of a gap side, the inverse of [ParseRipGapSide].
func ripGapSideName(d ExtentDirection) string {
	switch d {
	case NegativeDir:
		return "negative"
	case PositiveDir:
		return "positive"
	default:
		return "symmetric"
	}
}

// sketchLineEnds returns the 2D endpoints of a sketch line, erroring (named by what) when the
// line index is out of range.
func sketchLineEnds(sk *sketch.Sketch, lineIndex int, what string) (a, b math.Point2, err error) {
	lines := sk.Lines()
	if lineIndex < 0 || lineIndex >= lines.Count() {
		return a, b, fmt.Errorf("%s: line index %d out of range (%d lines)", what, lineIndex, lines.Count())
	}
	l := lines.Item(lineIndex)
	return l.StartPoint().Position(), l.EndPoint().Position(), nil
}

// SheetMetalRipFeatures adds rip features into the engine.
type SheetMetalRipFeatures struct{ engine *PartFeatures }

// NewSheetMetalRipFeatures binds the collection to a feature engine.
func NewSheetMetalRipFeatures(engine *PartFeatures) *SheetMetalRipFeatures {
	return &SheetMetalRipFeatures{engine}
}

// Add appends a rip feature, naming it Rip1, Rip2, … .
func (c *SheetMetalRipFeatures) Add(def *SheetMetalRipDefinition) *PartFeature {
	f := &SheetMetalRipFeature{def: def}
	pf := c.engine.Add(f)
	pf.SetName(c.engine.UniqueName("Rip"))
	f.featName = pf.Name()
	return pf
}

var _ Feature = (*SheetMetalRipFeature)(nil)
