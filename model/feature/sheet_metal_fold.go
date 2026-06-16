// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/math"
	"oblikovati.org/model/param"
	"oblikovati.org/model/sketch"
)

// Sheet-metal Fold feature (M13-F02). A fold bends a flat face along a sketch line drawn on
// it. Unlike the Bend (which treats the line as the bend's inside tangent), the Fold adds a
// BEND LOCATION: the line can be the start, centerline, or end of the bend, so the fold sits
// where the designer dimensioned it. The geometry reuses the shared fold (bendSolid); the
// location shifts the fold line along the across-direction by a fraction of the bend's flat
// footprint (r·sin(angle)) — a first-order placement refined by the flat-pattern unfold (F04).

// BendLocation places the sketch line relative to the bend it produces.
type BendLocation int

const (
	// CenterlineOfBend centres the bend on the line (the default).
	CenterlineOfBend BendLocation = iota
	// StartOfBend makes the line the inside tangent where the bend begins.
	StartOfBend
	// EndOfBend makes the line the tangent where the bend ends.
	EndOfBend
)

// locationFactor is how far into the bend's footprint the line sits, per location.
func (l BendLocation) locationFactor() float64 {
	switch l {
	case StartOfBend:
		return 0
	case EndOfBend:
		return 1
	default: // CenterlineOfBend
		return 0.5
	}
}

// ParseBendLocation resolves a wire spelling to a bend location.
func ParseBendLocation(s string) (BendLocation, bool) {
	switch s {
	case "", "centerline":
		return CenterlineOfBend, true
	case "start":
		return StartOfBend, true
	case "end":
		return EndOfBend, true
	}
	return 0, false
}

// SheetMetalFoldDefinition is the fold recipe: the sketch fold line (sketch + line index),
// the bend angle (parameter-backed; nil ⇒ 90°), an optional radius override (nil ⇒ the rule's
// bend radius), the bend location, and a flip that folds to the opposite side.
type SheetMetalFoldDefinition struct {
	Sketch    *sketch.Sketch
	LineIndex int
	Angle     func() float64
	Radius    func() float64
	Location  BendLocation
	Flip      bool
}

// SheetMetalFoldFeature folds the running sheet along the fold line each recompute.
type SheetMetalFoldFeature struct {
	def      *SheetMetalFoldDefinition
	featName string
}

// Definition returns the fold recipe.
func (f *SheetMetalFoldFeature) Definition() *SheetMetalFoldDefinition { return f.def }

// Kind identifies the feature for serialization and the model tree.
func (f *SheetMetalFoldFeature) Kind() string { return "sheet-metal-fold" }

// Recompute resolves the fold line, shifts it per the bend location, and folds the sheet.
func (f *SheetMetalFoldFeature) Recompute(in Input) (Output, error) {
	body, err := lastBody(in, "sheet-metal fold")
	if err != nil {
		return Output{}, err
	}
	point, dir, up, err := sketchBendLine(f.def.Sketch, f.def.LineIndex, f.def.Flip, "sheet-metal fold")
	if err != nil {
		return Output{}, err
	}
	radius := f.resolveRadius(in.Params)
	angle := f.resolveAngle()
	if radius <= 0 || angle <= 0 {
		return Output{}, fmt.Errorf("sheet-metal fold: radius/angle must be positive (r=%g a=%g)", radius, angle)
	}
	at := foldLinePoint(point, dir, up, radius, angle, f.def.Location)
	bent, err := bendSolid(body, at, dir, up, radius, angle, featOr(f.featName, "fold"))
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: replaceBody(in.Bodies, body, bent)}, nil
}

// foldLinePoint shifts the user's line toward the fixed side by the bend location's fraction
// of the bend footprint, so bendSolid (which puts the bend start at the point it is given)
// places the bend so the original line is at the requested start/center/end.
func foldLinePoint(point math.Point3, dir, up math.Vector3, radius, angle float64, loc BendLocation) math.Point3 {
	across, err := math.UnitVector3FromVector(dir.Cross(up))
	if err != nil {
		return point // degenerate; bendSolid will report it
	}
	footprint := radius * stdmath.Sin(angle) // flat across-extent the bend spans
	return point.TranslateBy(across.AsVector().Scale(-loc.locationFactor() * footprint))
}

// resolveRadius returns the bend radius: the override closure, else the rule's BendRadius
// parameter, else 0 (which Recompute rejects).
func (f *SheetMetalFoldFeature) resolveRadius(ps *param.Parameters) float64 {
	if f.def.Radius != nil {
		return f.def.Radius()
	}
	if ps != nil {
		if p, ok := ps.ByName(flangeBendParamName); ok {
			return p.ModelValue()
		}
	}
	return 0
}

// resolveAngle returns the bend angle, defaulting to a 90° fold.
func (f *SheetMetalFoldFeature) resolveAngle() float64 {
	if f.def.Angle == nil {
		return stdmath.Pi / 2
	}
	return f.def.Angle()
}

// BendSpecs reports the single bend this fold introduces, for the flat pattern. A nil
// radius override defers to the rule's default (signalled by a non-positive radius).
func (f *SheetMetalFoldFeature) BendSpecs(_ float64) []BendSpec {
	radius := 0.0
	if f.def.Radius != nil {
		radius = f.def.Radius()
	}
	return []BendSpec{{Angle: f.resolveAngle(), Radius: radius}}
}

// SheetMetalFoldFeatures adds fold features into the engine.
type SheetMetalFoldFeatures struct{ engine *PartFeatures }

// NewSheetMetalFoldFeatures binds the collection to a feature engine.
func NewSheetMetalFoldFeatures(engine *PartFeatures) *SheetMetalFoldFeatures {
	return &SheetMetalFoldFeatures{engine}
}

// Add appends a fold feature, naming it Fold1, Fold2, … .
func (c *SheetMetalFoldFeatures) Add(def *SheetMetalFoldDefinition) *PartFeature {
	f := &SheetMetalFoldFeature{def: def}
	pf := c.engine.Add(f)
	pf.SetName(c.engine.UniqueName("Fold"))
	f.featName = pf.Name()
	return pf
}

var _ Feature = (*SheetMetalFoldFeature)(nil)
