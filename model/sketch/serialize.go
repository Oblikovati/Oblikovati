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
	// ID is the sketch's document-local id, persisted (#153) so it survives the round trip
	// and the sketch's document-derived reference key stays stable. Zero in legacy recipes
	// that predate it; restore then keeps the freshly-minted id.
	ID           uint64  `yaml:"id,omitempty"`
	Name         string  `yaml:"name,omitempty"`
	Hidden       bool    `yaml:"hidden,omitempty"`
	Shared       bool    `yaml:"shared,omitempty"` // shared sketch stays at top level (issue #132)
	Seq          uint64  `yaml:"seq,omitempty"`    // global creation stamp; see model/seq
	Color        string  `yaml:"color,omitempty"`
	LineType     string  `yaml:"lineType,omitempty"`
	LineWeight   float64 `yaml:"lineWeight,omitempty"`
	DeferUpdates bool    `yaml:"deferUpdates,omitempty"`
	// Custom line type (issue #161): the pattern is stored so reopening never
	// needs the original .lin file.
	CustomLineName    string            `yaml:"customLineName,omitempty"`
	CustomLineFile    string            `yaml:"customLineFile,omitempty"`
	CustomLinePattern []float64         `yaml:"customLinePattern,omitempty,flow"`
	Plane             PlaneData         `yaml:"plane"`
	HostPlaneRef      string            `yaml:"hostPlaneRef,omitempty"` // datum ref the sketch is hosted on (#1849)
	Points            []PointData       `yaml:"points,omitempty"`
	Entities          []EntityData      `yaml:"entities,omitempty"`
	Constraints       []ConstraintData  `yaml:"constraints,omitempty"`
	Dimensions        []DimensionData   `yaml:"dimensions,omitempty"`
	CloudAnchors      []CloudAnchorData `yaml:"cloudAnchors,omitempty"` // scan-anchored points (provenance, #645)
}

// CloudAnchorData is the persisted provenance of one cloud-anchored sketch point: which standalone
// point it drives, the source cloud's id, and the cloud-local 3D anchor the point re-projects from.
type CloudAnchorData struct {
	PointID int        `yaml:"point"`
	CloudID string     `yaml:"cloud"`
	Local   [3]float64 `yaml:"local"`
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
	// CenterPoint marks a hole-centre marker rather than a plain point (#2015).
	CenterPoint bool `yaml:"centerPoint,omitempty"`
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
	Centerline   bool      `yaml:"centerline,omitempty"`     // line only: an axis
	CenterPoint  bool      `yaml:"centerPoint,omitempty"`    // point only: a hole-centre marker (#2015)
	FormatLine   string    `yaml:"formatLineType,omitempty"` // per-entity format overrides (#2015)
	FormatColor  string    `yaml:"formatColor,omitempty"`    // "#RRGGBB"; absent ⇒ inherit
	FormatWeight float64   `yaml:"formatLineWeight,omitempty"`
	Source       string    `yaml:"source,omitempty"`       // projected*: the source's SourceID
	SourceKind   string    `yaml:"sourceKind,omitempty"`   // projected*: vertex|edge|workPoint|workAxis|workPlane
	ProjShape    string    `yaml:"projShape,omitempty"`    // projectedCurve: analytic form line|circle|arc (ADR-0055)
	ProjParams   []float64 `yaml:"projParams,omitempty"`   // projectedCurve: line[ax,ay,bx,by]|circle[cx,cy,r]|arc[cx,cy,r,start,sweep]
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
	// Spline only (M06-F11, #626): the fit-method wire spelling (empty ⇒
	// smooth) and the active tangency handles.
	FitMethod string             `yaml:"fitMethod,omitempty"`
	Handles   []SplineHandleData `yaml:"handles,omitempty"`
	// Block instance only (M06-F07, #622): the placed definition's name and
	// the row-major 3×3 placement transform.
	Block     string    `yaml:"block,omitempty"`
	Transform []float64 `yaml:"transform,omitempty,flow"`
}

// SplineHandleData is one active spline tangency handle: which fit point it
// rides and where its draggable end sits.
type SplineHandleData struct {
	FitIndex int     `yaml:"fitIndex"`
	EndX     float64 `yaml:"endX"`
	EndY     float64 `yaml:"endY"`
}

// ConstraintData is one geometric constraint: its kind plus operand ids split into
// Points (point operands) and Curves (line/circular/smooth entity operands), in the
// order the constraint's factory expects.
type ConstraintData struct {
	Kind   string  `yaml:"kind"`
	Points []int   `yaml:"points,omitempty"`
	Curves []int   `yaml:"curves,omitempty"`
	Value  float64 `yaml:"value,omitempty"` // offset constraint's signed distance
	// AxisMajor is the per-operand major-axis flag of an ellipse-axis constraint (#1879), one
	// entry per Curves operand; absent for every other kind.
	AxisMajor []bool `yaml:"axisMajor,omitempty"`
	// Custom (add-in tag) constraints carry their owner and record name
	// (M06-F11, #626).
	ClientID string `yaml:"clientId,omitempty"`
	Name     string `yaml:"name,omitempty"`
}

// DimensionData is one dimensional constraint: its kind, operand ids, the value
// expression, and driving/limit state.
type DimensionData struct {
	Kind           string      `yaml:"kind"`
	Points         []int       `yaml:"points,omitempty"`
	Curves         []int       `yaml:"curves,omitempty"`
	Expression     string      `yaml:"expression"`
	Driven         bool        `yaml:"driven,omitempty"`
	FarSide        bool        `yaml:"farSide,omitempty"`        // tangentDistance only (#152)
	Orientation    string      `yaml:"orientation,omitempty"`    // distance only: horizontal/vertical (absent ⇒ aligned) (#1869)
	LinearDiameter bool        `yaml:"linearDiameter,omitempty"` // offset/tangentDistance only: value reads as a diameter (#1875)
	TextPoint      []float64   `yaml:"textPoint,omitempty"`      // [x,y] annotation placement, absent when unset (#1875)
	Limits         *LimitsData `yaml:"limits,omitempty"`
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
	sd := SketchData{
		ID:           uint64(s.id),
		Name:         s.name,
		Hidden:       !s.visible,
		Shared:       s.shared,
		Seq:          s.seq,
		Color:        s.color,
		LineType:     s.lineType,
		LineWeight:   s.lineWeight,
		DeferUpdates: s.deferUpdates,
		Plane:        serializePlane(s.plane),
		HostPlaneRef: s.hostWorkRef,
	}
	if d, file, ok := s.CustomLineType(); ok {
		sd.CustomLineName, sd.CustomLineFile, sd.CustomLinePattern = d.Name, file, d.Pattern
	}
	standalone := standalonePointIDs(s)
	handleEnds := splineHandleEndIDs(s)
	for _, p := range s.pts {
		if handleEnds[p.id] {
			continue // handle ends persist inside their spline's record and are re-minted on restore
		}
		sd.Points = append(sd.Points, PointData{
			ID: int(p.id), X: float64(p.X), Y: float64(p.Y),
			Standalone: standalone[p.id], CenterPoint: p.centerPoint,
		})
	}
	for _, e := range s.ents {
		if _, isPoint := e.(*Point); isPoint {
			continue // standalone points are captured in Points, not Entities
		}
		if _, isHandle := e.(*SplineHandle); isHandle {
			continue // handles persist inside their spline's record (M06-F11)
		}
		ed, err := serializeEntity(e)
		if err != nil {
			return SketchData{}, err
		}
		s.writeEntityFormat(&ed, e.EntityID()) // format lives on the sketch, not the entity (#2015)
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
	for _, a := range s.CloudAnchors() {
		sd.CloudAnchors = append(sd.CloudAnchors, CloudAnchorData{
			PointID: int(a.PointID), CloudID: a.CloudID,
			Local: [3]float64{float64(a.Local.X), float64(a.Local.Y), float64(a.Local.Z)},
		})
	}
	return sd, nil
}

// splineHandleEndIDs is the set of point ids owned by active spline handles.
func splineHandleEndIDs(s *Sketch) map[ID]bool {
	out := map[ID]bool{}
	for _, sp := range s.splines.items {
		for _, h := range sp.Handles() {
			out[h.End.id] = true
		}
	}
	return out
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

// serializeEntity dispatches an entity to its registered codec by its Kind and
// stamps the kind onto the row, so an encode closure fills only its payload.
// A kind that never registered a codec pair cannot have been created (its
// factory lives beside its registration), so the misses here guard programming
// errors, not user documents (#1624).
func serializeEntity(e Entity) (EntityData, error) {
	ke, ok := e.(kindedEntity)
	if !ok {
		return EntityData{}, fmt.Errorf("cannot serialize entity of type %T: it has no Kind (register it in a serialize_codecs_*.go)", e)
	}
	c, ok := entityCodecs2D[ke.Kind()]
	if !ok {
		return EntityData{}, fmt.Errorf("cannot serialize entity of type %T: kind %q has no 2D codec", e, ke.Kind())
	}
	ed, err := c.encode(e)
	if err != nil {
		return EntityData{}, err
	}
	ed.Kind = string(ke.Kind())
	return ed, nil
}

// flattenPoints flattens points to a [x,y,x,y,…] slice.
func flattenPoints(pts []math.Point2) []float64 {
	out := make([]float64, 0, len(pts)*2)
	for _, p := range pts {
		out = append(out, float64(p.X), float64(p.Y))
	}
	return out
}

// serializeConstraint encodes through the paired codec registry, keyed on the
// constraint's self-reported kind (#1625) — no per-type switch, so an encode
// half can no longer ship without its decode half.
func serializeConstraint(c Constraint) (ConstraintData, error) {
	kc, ok := c.(KindedConstraint)
	if !ok {
		return ConstraintData{}, fmt.Errorf("cannot serialize constraint of type %T (no ConstraintKind capability)", c)
	}
	codec, ok := constraintCodecs2D[kc.ConstraintKind()]
	if !ok {
		return ConstraintData{}, fmt.Errorf("cannot serialize constraint kind %q of type %T (no codec)", kc.ConstraintKind(), c)
	}
	cd, err := codec.encode(c)
	if err != nil {
		return ConstraintData{}, err
	}
	cd.Kind = string(kc.ConstraintKind())
	return cd, nil
}

// fitMethodSpelling renders a spline fit method for persistence; the smooth
// default stays empty so existing files do not churn.
func fitMethodSpelling(m SplineFitMethod) string {
	if m == 0 {
		return ""
	}
	return m.String()
}

// serializeSplineHandles renders a spline's active handles in fit order.
func serializeSplineHandles(sp *Spline) []SplineHandleData {
	handles := sp.Handles()
	if len(handles) == 0 {
		return nil
	}
	out := make([]SplineHandleData, len(handles))
	for i, h := range handles {
		out[i] = SplineHandleData{FitIndex: h.FitIndex, EndX: float64(h.End.X), EndY: float64(h.End.Y)}
	}
	return out
}

func serializeDimension(d *DimensionConstraint) (DimensionData, error) {
	dd := DimensionData{Expression: d.param.Expression(), Driven: d.driven, FarSide: d.farSide, Orientation: DistanceOrientationName(d.orientation), LinearDiameter: d.linearDiameter}
	if d.limits.Enabled {
		dd.Limits = &LimitsData{Min: d.limits.Min, Max: d.limits.Max}
	}
	if tp, ok := d.TextPoint(); ok {
		dd.TextPoint = []float64{float64(tp.X), float64(tp.Y)}
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
	case TangentDistanceDim:
		return "tangentDistance", nil
	case OffsetSplineDim:
		return "offsetSplineDim", nil
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
