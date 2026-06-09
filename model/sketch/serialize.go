// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"fmt"

	"oblikovati.org/math"
)

// This file defines the git-friendly YAML projection of a sketch (ADR-0020) and the
// marshal half of the round trip; serialize_restore.go rebuilds a live sketch from it.
// A sketch is captured as its host plane, its constrainable points (the solver's
// variables), its curve entities (referencing points by id), and its geometric and
// dimensional constraints (referencing points/curves by id). Shared points — a corner
// where two lines meet *is* a coincidence — are preserved because every point is
// recorded once and referenced by id.

// SketchData is the serializable form of one sketch. Hidden inverts Visible so the
// common visible=true case omits the field (M21-F01 PBI-201 persists name/display props,
// closing the previous name-not-persisted gap).
type SketchData struct {
	Name         string           `yaml:"name,omitempty"`
	Hidden       bool             `yaml:"hidden,omitempty"`
	Color        string           `yaml:"color,omitempty"`
	LineType     string           `yaml:"lineType,omitempty"`
	LineWeight   float64          `yaml:"lineWeight,omitempty"`
	DeferUpdates bool             `yaml:"deferUpdates,omitempty"`
	Plane        PlaneData        `yaml:"plane"`
	Points       []PointData      `yaml:"points,omitempty"`
	Entities     []EntityData     `yaml:"entities,omitempty"`
	Constraints  []ConstraintData `yaml:"constraints,omitempty"`
	Dimensions   []DimensionData  `yaml:"dimensions,omitempty"`
}

// PlaneData is a sketch plane as an origin and two in-plane axes (model space).
type PlaneData struct {
	Origin [3]float64 `yaml:"origin"`
	XAxis  [3]float64 `yaml:"xAxis"`
	YAxis  [3]float64 `yaml:"yAxis"`
}

// PointData is one constrainable point. Standalone marks a SketchPoint (a point that
// is itself an entity), distinct from a curve's owned endpoint/center.
type PointData struct {
	ID         int     `yaml:"id"`
	X          float64 `yaml:"x"`
	Y          float64 `yaml:"y"`
	Standalone bool    `yaml:"standalone,omitempty"`
}

// EntityData is one curve entity. Points lists the curve's defining point ids in a
// kind-specific order (line: A,B; circle: center; arc: center,start,end; ellipse:
// center; spline: all). Unused fields stay zero/omitted per kind.
type EntityData struct {
	ID           int       `yaml:"id"`
	Kind         string    `yaml:"kind"`
	Points       []int     `yaml:"points,omitempty"`
	Radius       float64   `yaml:"radius,omitempty"`
	CCW          bool      `yaml:"ccw,omitempty"`
	MajorAxis    []float64 `yaml:"majorAxis,omitempty"` // ellipse/ellipticalArc: [x, y]
	MajorRadius  float64   `yaml:"majorRadius,omitempty"`
	MinorRadius  float64   `yaml:"minorRadius,omitempty"`
	StartAngle   float64   `yaml:"startAngle,omitempty"` // ellipticalArc only (radians)
	EndAngle     float64   `yaml:"endAngle,omitempty"`   // ellipticalArc only (radians)
	Closed       bool      `yaml:"closed,omitempty"`
	Fit          bool      `yaml:"fit,omitempty"`
	Construction bool      `yaml:"construction,omitempty"`
	Centerline   bool      `yaml:"centerline,omitempty"`   // line only: an axis
	ImageRef     string    `yaml:"imageRef,omitempty"`     // image only
	Anchor       []float64 `yaml:"anchor,omitempty"`       // image/text: [x, y]
	Size         []float64 `yaml:"size,omitempty"`         // image only: [w, h]
	Rotation     float64   `yaml:"rotation,omitempty"`     // image/text (radians)
	Opacity      float64   `yaml:"opacity,omitempty"`      // image only
	Text         string    `yaml:"text,omitempty"`         // text only
	TextHeight   float64   `yaml:"textHeight,omitempty"`   // text only
	Justify      int       `yaml:"justify,omitempty"`      // text only: horizontal
	VJustify     int       `yaml:"vJustify,omitempty"`     // text only: vertical
	FontFamily   string    `yaml:"fontFamily,omitempty"`   // text only
	FontResource string    `yaml:"fontResource,omitempty"` // text only: document font resource UUID (ADR-0031)
	FontSize     float64   `yaml:"fontSize,omitempty"`     // text only (cm; 0 ⇒ track height)
	Seed         []float64 `yaml:"seed,omitempty"`         // fillRegion only: [x, y]
	Style        string    `yaml:"style,omitempty"`        // fillRegion only
	XExpr        string    `yaml:"xExpr,omitempty"`        // equationCurve only
	YExpr        string    `yaml:"yExpr,omitempty"`        // equationCurve only
	T0           float64   `yaml:"t0,omitempty"`           // equationCurve only
	T1           float64   `yaml:"t1,omitempty"`           // equationCurve only
	Coords       []float64 `yaml:"coords,omitempty"`       // fixedSpline only: flattened [x,y,…]
	ParentID     int       `yaml:"parentId,omitempty"`     // offsetSpline only
	OffsetDist   float64   `yaml:"offsetDist,omitempty"`   // offsetSpline only
}

// ConstraintData is one geometric constraint: its kind plus operand ids split into
// Points (point operands) and Curves (line/circular/smooth entity operands), in the
// order the constraint's factory expects.
type ConstraintData struct {
	Kind   string  `yaml:"kind"`
	Points []int   `yaml:"points,omitempty"`
	Curves []int   `yaml:"curves,omitempty"`
	Value  float64 `yaml:"value,omitempty"` // offset constraint's signed distance
}

// DimensionData is one dimensional constraint: its kind, operand ids, the value
// expression, and driving/limit state.
type DimensionData struct {
	Kind       string      `yaml:"kind"`
	Points     []int       `yaml:"points,omitempty"`
	Curves     []int       `yaml:"curves,omitempty"`
	Expression string      `yaml:"expression"`
	Driven     bool        `yaml:"driven,omitempty"`
	Limits     *LimitsData `yaml:"limits,omitempty"`
}

// LimitsData carries a dimension's drive limits when enabled.
type LimitsData struct {
	Min float64 `yaml:"min"`
	Max float64 `yaml:"max"`
}

// MarshalRecipe projects every sketch into its serializable form, in order. It errors
// (rather than dropping data) on any geometry it cannot yet serialize, e.g. blocks.
func (sc *Sketches) MarshalRecipe() ([]SketchData, error) {
	out := make([]SketchData, 0, sc.Count())
	for i := 0; i < sc.Count(); i++ {
		sd, err := serializeSketch(sc.Item(i))
		if err != nil {
			return nil, fmt.Errorf("sketch %d: %w", i, err)
		}
		out = append(out, sd)
	}
	return out, nil
}

func serializeSketch(s *Sketch) (SketchData, error) {
	if len(s.blocks.defs) > 0 || len(s.blocks.instances) > 0 {
		return SketchData{}, fmt.Errorf("block serialization is not yet supported (%d defs, %d instances)", len(s.blocks.defs), len(s.blocks.instances))
	}
	sd := SketchData{
		Name:         s.name,
		Hidden:       !s.visible,
		Color:        s.color,
		LineType:     s.lineType,
		LineWeight:   s.lineWeight,
		DeferUpdates: s.deferUpdates,
		Plane:        serializePlane(s.plane),
	}
	standalone := standalonePointIDs(s)
	for _, p := range s.pts {
		sd.Points = append(sd.Points, PointData{ID: int(p.id), X: float64(p.X), Y: float64(p.Y), Standalone: standalone[p.id]})
	}
	for _, e := range s.ents {
		if _, isPoint := e.(*Point); isPoint {
			continue // standalone points are captured in Points, not Entities
		}
		ed, err := serializeEntity(e)
		if err != nil {
			return SketchData{}, err
		}
		sd.Entities = append(sd.Entities, ed)
	}
	for _, c := range s.geomCons.All() {
		cd, err := serializeConstraint(c)
		if err != nil {
			return SketchData{}, err
		}
		sd.Constraints = append(sd.Constraints, cd)
	}
	for _, d := range s.dimCons.items {
		dd, err := serializeDimension(d)
		if err != nil {
			return SketchData{}, err
		}
		sd.Dimensions = append(sd.Dimensions, dd)
	}
	return sd, nil
}

// standalonePointIDs is the set of point ids that are standalone SketchPoints.
func standalonePointIDs(s *Sketch) map[ID]bool {
	out := make(map[ID]bool, len(s.points.items))
	for _, p := range s.points.items {
		out[p.id] = true
	}
	return out
}

func serializePlane(p Plane) PlaneData {
	o, x, y := p.origin, p.xAxis, p.yAxis
	return PlaneData{
		Origin: [3]float64{float64(o.X), float64(o.Y), float64(o.Z)},
		XAxis:  [3]float64{float64(x.X()), float64(x.Y()), float64(x.Z())},
		YAxis:  [3]float64{float64(y.X()), float64(y.Y()), float64(y.Z())},
	}
}

func serializeEntity(e Entity) (EntityData, error) {
	switch v := e.(type) {
	case *Line:
		return EntityData{ID: int(v.id), Kind: "line", Points: []int{int(v.A.id), int(v.B.id)}, Construction: v.construction, Centerline: v.centerline}, nil
	case *Circle:
		return EntityData{ID: int(v.id), Kind: "circle", Points: []int{int(v.Center.id)}, Radius: float64(v.Radius), Construction: v.construction}, nil
	case *Arc:
		return EntityData{ID: int(v.id), Kind: "arc", Points: []int{int(v.Center.id), int(v.Start.id), int(v.End.id)}, CCW: v.CounterClockwise, Construction: v.construction}, nil
	case *Ellipse:
		return EntityData{ID: int(v.id), Kind: "ellipse", Points: []int{int(v.Center.id)}, MajorAxis: []float64{float64(v.MajorAxis.X), float64(v.MajorAxis.Y)}, MajorRadius: float64(v.MajorRadius), MinorRadius: float64(v.MinorRadius), Construction: v.construction}, nil
	case *EllipticalArc:
		return EntityData{ID: int(v.id), Kind: "ellipticalArc", Points: []int{int(v.Center.id)}, MajorAxis: []float64{float64(v.MajorAxis.X), float64(v.MajorAxis.Y)}, MajorRadius: float64(v.MajorRadius), MinorRadius: float64(v.MinorRadius), StartAngle: float64(v.StartAngle), EndAngle: float64(v.EndAngle), Construction: v.construction}, nil
	case *Spline:
		return EntityData{ID: int(v.id), Kind: "spline", Points: pointIDsOf(v.Points), Closed: v.Closed, Fit: v.fit, Construction: v.construction}, nil
	case *SketchImage:
		return EntityData{
			ID: int(v.id), Kind: "image", ImageRef: v.Ref,
			Anchor:   []float64{float64(v.Anchor.X), float64(v.Anchor.Y)},
			Size:     []float64{float64(v.Width), float64(v.Height)},
			Rotation: float64(v.Rotation), Opacity: v.Opacity,
		}, nil
	case *FillRegion:
		return EntityData{ID: int(v.id), Kind: "fillRegion", Seed: []float64{float64(v.Seed.X), float64(v.Seed.Y)}, Style: v.Style}, nil
	case *TextBox:
		return EntityData{
			ID: int(v.id), Kind: "text", Text: v.Text,
			Anchor:     []float64{float64(v.Anchor.X), float64(v.Anchor.Y)},
			TextHeight: float64(v.Height), Rotation: float64(v.Rotation),
			Justify: int(v.Justify), VJustify: int(v.VJustify),
			FontFamily: v.Family, FontResource: v.FontResource, FontSize: float64(v.FontSize),
		}, nil
	default:
		return serializeDerivedCurve(e)
	}
}

// serializeDerivedCurve handles the M21 derived curves (equation/fixed/offset spline);
// split out of serializeEntity to keep that switch small.
func serializeDerivedCurve(e Entity) (EntityData, error) {
	switch v := e.(type) {
	case *EquationCurve:
		return EntityData{ID: int(v.id), Kind: "equationCurve", XExpr: v.XExpr, YExpr: v.YExpr, T0: v.T0, T1: v.T1}, nil
	case *FixedSpline:
		return EntityData{ID: int(v.id), Kind: "fixedSpline", Coords: flattenPoints(v.Pts)}, nil
	case *OffsetSpline:
		return EntityData{ID: int(v.id), Kind: "offsetSpline", ParentID: int(v.Parent.id), OffsetDist: v.Dist}, nil
	default:
		return EntityData{}, fmt.Errorf("cannot serialize entity of type %T (no codec)", e)
	}
}

// flattenPoints flattens points to a [x,y,x,y,…] slice.
func flattenPoints(pts []math.Point2) []float64 {
	out := make([]float64, 0, len(pts)*2)
	for _, p := range pts {
		out = append(out, float64(p.X), float64(p.Y))
	}
	return out
}

func serializeConstraint(c Constraint) (ConstraintData, error) {
	switch v := c.(type) {
	case *CoincidentConstraint:
		return ConstraintData{Kind: "coincident", Points: []int{int(v.A.id), int(v.B.id)}}, nil
	case *HorizontalConstraint:
		return ConstraintData{Kind: "horizontal", Points: []int{int(v.A.id), int(v.B.id)}}, nil
	case *VerticalConstraint:
		return ConstraintData{Kind: "vertical", Points: []int{int(v.A.id), int(v.B.id)}}, nil
	case *PointOnLineConstraint:
		return ConstraintData{Kind: "pointOnLine", Points: []int{int(v.P.id)}, Curves: []int{int(v.L.id)}}, nil
	case *MidpointConstraint:
		return ConstraintData{Kind: "midpoint", Points: []int{int(v.P.id)}, Curves: []int{int(v.L.id)}}, nil
	case *PointOnCircleConstraint:
		return ConstraintData{Kind: "pointOnCircle", Points: []int{int(v.P.id)}, Curves: []int{int(v.C.EntityID())}}, nil
	case *ParallelConstraint:
		return ConstraintData{Kind: "parallel", Curves: []int{int(v.L1.id), int(v.L2.id)}}, nil
	case *PerpendicularConstraint:
		return ConstraintData{Kind: "perpendicular", Curves: []int{int(v.L1.id), int(v.L2.id)}}, nil
	case *CollinearConstraint:
		return ConstraintData{Kind: "collinear", Curves: []int{int(v.L1.id), int(v.L2.id)}}, nil
	case *EqualLengthConstraint:
		return ConstraintData{Kind: "equalLength", Curves: []int{int(v.L1.id), int(v.L2.id)}}, nil
	case *ConcentricConstraint:
		return ConstraintData{Kind: "concentric", Curves: []int{int(v.C1.EntityID()), int(v.C2.EntityID())}}, nil
	case *EqualRadiusConstraint:
		return ConstraintData{Kind: "equalRadius", Curves: []int{int(v.C1.EntityID()), int(v.C2.EntityID())}}, nil
	case *CircularTangentConstraint:
		return ConstraintData{Kind: "circularTangent", Curves: []int{int(v.C1.EntityID()), int(v.C2.EntityID())}}, nil
	case *TangentConstraint:
		return ConstraintData{Kind: "tangent", Curves: []int{int(v.L.id), int(v.C.EntityID())}}, nil
	case *SymmetryConstraint:
		return ConstraintData{Kind: "symmetry", Points: []int{int(v.A.id), int(v.B.id)}, Curves: []int{int(v.About.id)}}, nil
	case *FixConstraint:
		return ConstraintData{Kind: "fix", Points: []int{int(v.P.id)}}, nil
	case *SmoothConstraint:
		return ConstraintData{Kind: "smooth", Points: []int{int(v.P1.id), int(v.P2.id)}, Curves: []int{int(v.C1.EntityID()), int(v.C2.EntityID())}}, nil
	case *GroundConstraint:
		return ConstraintData{Kind: "ground", Points: pointIDsOf(v.pts)}, nil
	case *OffsetConstraint:
		return ConstraintData{Kind: "offset", Curves: []int{int(v.L1.id), int(v.L2.id)}, Value: v.Dist}, nil
	case *PatternConstraint:
		return ConstraintData{Kind: "patternLink", Points: []int{int(v.Seed.id), int(v.Member.id)}}, nil
	default:
		return ConstraintData{}, fmt.Errorf("cannot serialize constraint of type %T (no codec)", c)
	}
}

func serializeDimension(d *DimensionConstraint) (DimensionData, error) {
	dd := DimensionData{Expression: d.param.Expression(), Driven: d.driven}
	if d.limits.Enabled {
		dd.Limits = &LimitsData{Min: d.limits.Min, Max: d.limits.Max}
	}
	kind, err := dimKindName(d.kind)
	if err != nil {
		return DimensionData{}, err
	}
	dd.Kind = kind
	// Split the dimensioned geometry into point and curve operands by kind.
	switch d.kind {
	case DistanceDim, ThreePointAngleDim:
		dd.Points = entityIDsOf(d.refs)
	case OffsetDim:
		dd.Points = entityIDsOf(d.refs[:1]) // the point
		dd.Curves = entityIDsOf(d.refs[1:]) // the line
	default:
		dd.Curves = entityIDsOf(d.refs)
	}
	return dd, nil
}

func dimKindName(k DimKind) (string, error) {
	switch k {
	case DistanceDim:
		return "distance", nil
	case AngleDim:
		return "angle", nil
	case RadiusDim:
		return "radius", nil
	case DiameterDim:
		return "diameter", nil
	case ArcLengthDim:
		return "arcLength", nil
	case OffsetDim:
		return "offsetDim", nil
	case ThreePointAngleDim:
		return "threePointAngle", nil
	case EllipseRadiusDim:
		return "ellipseRadius", nil
	default:
		return "", fmt.Errorf("cannot serialize dimension of kind %d (no codec)", k)
	}
}

// pointIDsOf returns the ids of a list of points.
func pointIDsOf(pts []*Point) []int {
	out := make([]int, len(pts))
	for i, p := range pts {
		out[i] = int(p.id)
	}
	return out
}

// entityIDsOf returns the ids of a list of entities.
func entityIDsOf(ents []Entity) []int {
	out := make([]int, len(ents))
	for i, e := range ents {
		out[i] = int(e.EntityID())
	}
	return out
}

// vector3 builds a math.Vector3 from a serialized triple.
func vector3(v [3]float64) math.Vector3 { return math.V3(v[0], v[1], v[2]) }
