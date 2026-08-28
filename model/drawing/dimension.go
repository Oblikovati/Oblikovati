// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"fmt"
	"strconv"

	"oblikovati.org/api/contract"
	"oblikovati.org/api/types"
	gmath "oblikovati.org/math"
	"oblikovati.org/model/collview"
)

// Drawing dimensions (M14-F03 PBI-141, #388): a linear dimension annotates the distance between
// two points on a drawing view. It attaches to two model vertices (by reference key), so it is
// associative — on recompute it re-resolves the vertices against the current model, re-projects
// them through the view and re-measures, and the value updates when the model size changes.
//
// This file holds the core: the DrawingDimension value type, its text decoration (overrides,
// tolerance, inspection, dual-unit), the DrawingDimensions collection and its lookup/edit. The
// per-family geometry (linear/angular/radial/ordinate) and the shared attachment/recompute path
// live in the sibling dimension_*.go files (M48 #2225).

// DrawingDimension is one linear dimension on a sheet: its measured value (true model size), the
// view it annotates, the two attached model vertices, and the drawing curves of its glyph
// (extension lines, dimension line, arrowheads).
type DrawingDimension struct {
	name     string
	dimType  types.DrawingDimensionType
	viewName string
	keyA     []byte // linear: attached model vertices (reference keys) — the associativity anchors
	keyB     []byte
	edgeKey  []byte  // radial: the attached circular edge; angular: the first straight edge
	edgeKeyB []byte  // angular: the second straight edge
	offset   float64 // dimension-line standoff from the measured points (signed, sheet mm)
	// ordinate: measure the view-X offset from the datum (keyA) when true, else the view-Y offset.
	axisHorizontal bool
	valueMM        float64 // measured model distance (mm), scale-independent (0 for angular)
	valueDeg       float64 // measured angle (degrees) for an angular dimension
	// Text overrides (#1992/#1993). overrideText replaces the whole label with free text; prefix and
	// suffix wrap the value; hideValue drops the value (leaving only prefix/suffix); dualUnit appends
	// the value converted to inches in brackets.
	prefix       string
	suffix       string
	overrideText string
	hideValue    bool
	dualUnit     bool
	tolerance    types.DimensionTolerance  // engineering tolerance shown after the value (#1990)
	inspection   types.InspectionDimension // inspection border + label + rate (#1996)
	text         string                    // displayed text (the decorated value)
	anchorX      float64                   // text anchor (sheet mm), lifted off the dimension line by textGapMM
	anchorY      float64
	textDX       float64 // user text nudge from the anchor (sheet mm) — drag the text to set it
	textDY       float64
	nx           float64 // unit perpendicular of the dimension line — text-lift + line-drag direction
	ny           float64
	// Retrieved model dimension (#1991): retrievedFrom is the source model-dimension (parameter) name
	// ("" for an ordinary picked dimension); worldA/worldB are the model-space 3D endpoints it spans,
	// re-fetched from the referenced model on recompute so the value tracks the parameter.
	retrievedFrom  string
	worldA, worldB gmath.Point3
	curves         []DrawingCurve
}

// textGapMM lifts the value text off the dimension line so it stays readable by default.
const textGapMM = 5.0

// mmPerInch converts the primary millimetre value to the dual-unit inch value.
const mmPerInch = 25.4

// decorate applies the dimension's text overrides to a formatted value (#1992/#1993): free-text
// override replaces everything; otherwise the value (unless hidden) is wrapped by prefix/suffix and,
// for a dual-unit dimension, followed by the value in inches. It is the single point every
// per-type formatter routes its value through, so the overrides apply uniformly.
func (d *DrawingDimension) decorate(value string) string {
	return d.inspectionWrap(d.decorateCore(value))
}

// decorateCore applies the text overrides (override / prefix-suffix / tolerance / dual-unit) to a
// formatted value, before any inspection wrapping (#1992/#1993).
func (d *DrawingDimension) decorateCore(value string) string {
	if d.overrideText != "" {
		return d.overrideText
	}
	core := value
	if d.hideValue {
		core = ""
	}
	out := d.prefix + core + d.toleranceNote() + d.suffix
	if d.dualUnit {
		out += " [" + strconv.FormatFloat(d.valueMM/mmPerInch, 'f', 3, 64) + " in]"
	}
	return out
}

// inspectionWrap prepends the inspection label and appends the sampling rate to the displayed text
// when the dimension is an inspection dimension; the border shape is drawn separately by the head
// (#1996). A non-inspection dimension is returned unchanged.
func (d *DrawingDimension) inspectionWrap(body string) string {
	if d.inspection.Shape == types.NoInspectionBorder {
		return body
	}
	out := body
	if d.inspection.Label != "" {
		out = d.inspection.Label + " " + out
	}
	if d.inspection.Rate != "" {
		out += " " + d.inspection.Rate
	}
	return out
}

// toleranceNote formats the dimension's engineering tolerance to append after the value (#1990):
// a symmetric ±, an asymmetric +/− deviation, stacked max/min limits, or an ISO fit class.
func (d *DrawingDimension) toleranceNote() string {
	t := d.tolerance
	prec := max(t.Precision, 0)
	f := func(v float64) string { return formatDimValue(v, prec) }
	switch t.Type {
	case types.SymmetricTolerance:
		return " ±" + f(t.Plus)
	case types.DeviationTolerance:
		return " +" + f(t.Plus) + "/-" + f(t.Minus)
	case types.LimitsTolerance:
		return " " + f(d.valueMM+t.Plus) + "/" + f(d.valueMM-t.Minus)
	case types.FitsTolerance:
		return " " + t.Fit
	default:
		return ""
	}
}

// SetTolerance sets the named dimension's engineering tolerance and re-decorates its text (#1990).
func (ds *DrawingDimensions) SetTolerance(name string, tol types.DimensionTolerance) error {
	d, ok := ds.ByName(name)
	if !ok {
		return fmt.Errorf("drawing: no dimension named %q", name)
	}
	d.tolerance = tol
	ds.recompute(d)
	return nil
}

// Tolerance exposes the dimension's engineering tolerance (#1990).
func (d *DrawingDimension) Tolerance() types.DimensionTolerance { return d.tolerance }

// SetInspection flags the named dimension as an inspection dimension (or clears it with a
// NoInspectionBorder shape) and re-decorates its text (#1996).
func (ds *DrawingDimensions) SetInspection(name string, ins types.InspectionDimension) error {
	d, ok := ds.ByName(name)
	if !ok {
		return fmt.Errorf("drawing: no dimension named %q", name)
	}
	d.inspection = ins
	ds.recompute(d)
	return nil
}

// Inspection exposes the dimension's inspection annotation (#1996).
func (d *DrawingDimension) Inspection() types.InspectionDimension { return d.inspection }

var _ contract.DrawingDimension = (*DrawingDimension)(nil)

// Name, Type, ViewName, ValueMM, Text, CurveCount and Curves expose the dimension.
func (d *DrawingDimension) Name() string                     { return d.name }
func (d *DrawingDimension) Type() types.DrawingDimensionType { return d.dimType }
func (d *DrawingDimension) ViewName() string                 { return d.viewName }
func (d *DrawingDimension) ValueMM() float64                 { return d.valueMM }
func (d *DrawingDimension) ValueDeg() float64                { return d.valueDeg }
func (d *DrawingDimension) Text() string                     { return d.text }
func (d *DrawingDimension) CurveCount() int                  { return len(d.curves) }

// Prefix, Suffix, OverrideText, HideValue and DualUnit expose the text overrides (#1992/#1993).
func (d *DrawingDimension) Prefix() string         { return d.prefix }
func (d *DrawingDimension) Suffix() string         { return d.suffix }
func (d *DrawingDimension) OverrideText() string   { return d.overrideText }
func (d *DrawingDimension) HideValue() bool        { return d.hideValue }
func (d *DrawingDimension) DualUnit() bool         { return d.dualUnit }
func (d *DrawingDimension) Curves() []DrawingCurve { return d.curves }

// TextAnchorMM is the dimension line's midpoint (sheet mm) — where the value text is centred.
func (d *DrawingDimension) TextAnchorMM() (x, y float64) {
	return d.anchorX + d.textDX, d.anchorY + d.textDY
}

// DrawingDimensions is a sheet's dimension collection. It holds the view collection (to resolve a
// dimension's view and project through it) and the body-resolution hook (to re-bind the attached
// vertices on recompute).
type DrawingDimensions struct {
	items     []*DrawingDimension
	views     *DrawingViews
	body      bodyLookup
	modelDims modelDimLookup // referenced model's retrievable parametric dimensions (#1991); nil ⇒ none
	precision func() int     // active drafting standard's decimal places; set by the owning sheet
}

func newDrawingDimensions(views *DrawingViews, body bodyLookup, modelDims modelDimLookup, precision func() int) *DrawingDimensions {
	return &DrawingDimensions{views: views, body: body, modelDims: modelDims, precision: precision}
}

// defaultDimDecimals is the fallback decimal places when no drafting-standard precision provider
// is wired (e.g. a bare DrawingDimensions in a unit test) — the ISO default.
const defaultDimDecimals = 2

// decimals returns the active drafting standard's dimension decimal places, or the ISO default
// when no precision provider is wired (clamped non-negative).
func (ds *DrawingDimensions) decimals() int {
	if ds.precision == nil {
		return defaultDimDecimals
	}
	if p := ds.precision(); p >= 0 {
		return p
	}
	return 0
}

// DimensionTextStyle carries the optional text overrides to apply to a dimension (#1992/#1993);
// a nil field leaves that override unchanged.
type DimensionTextStyle struct {
	Prefix       *string
	Suffix       *string
	OverrideText *string
	HideValue    *bool
	DualUnit     *bool
}

// SetTextStyle applies the given text overrides to the named dimension and re-decorates its text
// (#1992/#1993), erroring when no dimension carries that name.
func (ds *DrawingDimensions) SetTextStyle(name string, style DimensionTextStyle) error {
	d, ok := ds.ByName(name)
	if !ok {
		return fmt.Errorf("drawing: no dimension named %q", name)
	}
	if style.Prefix != nil {
		d.prefix = *style.Prefix
	}
	if style.Suffix != nil {
		d.suffix = *style.Suffix
	}
	if style.OverrideText != nil {
		d.overrideText = *style.OverrideText
	}
	if style.HideValue != nil {
		d.hideValue = *style.HideValue
	}
	if style.DualUnit != nil {
		d.dualUnit = *style.DualUnit
	}
	ds.recompute(d)
	return nil
}

// formatDimValue renders a dimension value at the active drafting standard's precision — fixed
// decimal places, so a value measured as 9.999999998 reads as the clean "10.00" and honors the
// configured precision instead of a bare 4-significant-figure float (Oblikovati/Oblikovati#146).
func formatDimValue(v float64, decimals int) string {
	return strconv.FormatFloat(v, 'f', decimals, 64)
}

// MoveText nudges the named dimension's value text by (dx, dy) sheet millimetres — the drag-the-
// text gesture, so overlapping dimensions can be separated for readability.
func (ds *DrawingDimensions) MoveText(name string, dx, dy float64) {
	if d, ok := ds.ByName(name); ok {
		d.textDX += dx
		d.textDY += dy
	}
}

// MoveLine shifts the named dimension's dimension line perpendicular to itself by the drag
// (dx, dy) sheet millimetres and re-derives the glyph — the drag-the-line gesture. Linear only in
// this increment (radial/angular line position is fixed; nudge their text instead).
func (ds *DrawingDimensions) MoveLine(name string, dx, dy float64) {
	d, ok := ds.ByName(name)
	if !ok || isRadial(d.dimType) || d.dimType == types.AngularDimension || d.dimType == types.OrdinateDimension {
		return
	}
	d.offset += dx*d.nx + dy*d.ny
	ds.recompute(d)
}

// Count, Item, ByName and Remove read/edit the collection.
func (ds *DrawingDimensions) Count() int { return len(ds.items) }

func (ds *DrawingDimensions) Item(i int) *DrawingDimension { return collview.At(ds.items, i) }

func (ds *DrawingDimensions) ByName(name string) (*DrawingDimension, bool) {
	for _, d := range ds.items {
		if d.name == name {
			return d, true
		}
	}
	return nil, false
}

func (ds *DrawingDimensions) Remove(name string) error {
	for i, d := range ds.items {
		if d.name == name {
			ds.items = append(ds.items[:i], ds.items[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("drawing: no dimension named %q", name)
}

func (ds *DrawingDimensions) uniqueName(requested string) string {
	if requested != "" {
		if _, exists := ds.ByName(requested); !exists {
			return requested
		}
	}
	for n := len(ds.items) + 1; ; n++ {
		name := fmt.Sprintf("DIM:%d", n)
		if _, exists := ds.ByName(name); !exists {
			return name
		}
	}
}
