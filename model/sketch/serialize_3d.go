// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/math"
	"oblikovati.org/model/seq"
)

// This file defines the git-friendly YAML projection of a 3D sketch (ADR-0020) and its
// round trip. A 3D sketch is captured as its display/solve properties, its constrainable
// 3D points (the solver's variables), and its curve entities (referencing points by id).
// Curve-entity and constraint/dimension codecs grow alongside their features (M22 F02+),
// each adding a case here; the spine (M22-F01) round-trips points and properties.

// SketchData3D is the serializable form of one 3D sketch. Hidden inverts Visible so the
// common visible=true case omits the field; DimsHidden inverts DimensionsVisible likewise.
type SketchData3D struct {
	// ID is the 3D sketch's document-local id, persisted (#153) so it and its entities'
	// derived reference keys survive the round trip. Zero in legacy recipes; restore then
	// keeps the freshly-minted id.
	ID           uint64            `yaml:"id,omitempty"`
	Name         string            `yaml:"name,omitempty"`
	Hidden       bool              `yaml:"hidden,omitempty"`
	DimsHidden   bool              `yaml:"dimsHidden,omitempty"`
	Shared       bool              `yaml:"shared,omitempty"` // Inventor Sketch3D.Shared — always false today (3D sketches are not in the browser timeline yet); round-tripped for forward compatibility (issue #132)
	Seq          uint64            `yaml:"seq,omitempty"`    // global creation stamp; see model/seq
	Color        string            `yaml:"color,omitempty"`
	DeferUpdates bool              `yaml:"deferUpdates,omitempty"`
	Points       []Point3DData     `yaml:"points,omitempty"`
	Entities     []Entity3DData    `yaml:"entities,omitempty"`
	Constraints  []Constraint3DRow `yaml:"constraints,omitempty"`
	Dimensions   []Dimension3DRow  `yaml:"dimensions,omitempty"`
}

// Dimension3DRow is one 3D dimensional constraint: its kind, point/curve operand ids, the
// value expression, the driving/driven flag, and (for a point-plane distance) the origin
// plane label.
type Dimension3DRow struct {
	Kind       string `yaml:"kind"`
	Points     []int  `yaml:"points,omitempty"`
	Curves     []int  `yaml:"curves,omitempty"`
	Expression string `yaml:"expression"`
	Driven     bool   `yaml:"driven,omitempty"`
	Plane      string `yaml:"plane,omitempty"`
}

// Point3DData is one constrainable 3D point. Standalone marks a SketchPoint3D (a point
// that is itself an entity), distinct from a curve's owned endpoint/center.
type Point3DData struct {
	ID         int     `yaml:"id"`
	X          float64 `yaml:"x"`
	Y          float64 `yaml:"y"`
	Z          float64 `yaml:"z"`
	Standalone bool    `yaml:"standalone,omitempty"`
}

// Entity3DData is one 3D curve entity. Points lists the curve's defining point ids in a
// kind-specific order (line: A,B; circle: center; arc: center,start,end). Axis is the
// circle's plane normal. Standalone points are captured in Points, not here. Unused
// fields stay zero/omitted per kind.
type Entity3DData struct {
	ID           int        `yaml:"id"`
	Kind         string     `yaml:"kind"`
	Points       []int      `yaml:"points,omitempty"`
	Radius       float64    `yaml:"radius,omitempty"`
	Axis         [3]float64 `yaml:"axis,omitempty"` // circle/helix axis (plane normal / winding axis)
	CCW          bool       `yaml:"ccw,omitempty"`  // arc sweep direction
	Construction bool       `yaml:"construction,omitempty"`

	// Helix-only fields (kind "helical").
	Pitch         float64 `yaml:"pitch,omitempty"`         // axial rise per revolution
	Turns         float64 `yaml:"turns,omitempty"`         // revolution count
	RadialPerTurn float64 `yaml:"radialPerTurn,omitempty"` // per-revolution radial growth
	Clockwise     bool    `yaml:"clockwise,omitempty"`     // helix handedness

	// Conic fields (kind "ellipse"/"ellipticalArc"). MajorAxis is the in-plane major
	// direction; Radius is the major radius and MinorRadius the minor; Start/Sweep bound
	// an elliptical arc (radians).
	MajorAxis   [3]float64 `yaml:"majorAxis,omitempty"`
	MinorRadius float64    `yaml:"minorRadius,omitempty"`
	StartAngle  float64    `yaml:"startAngle,omitempty"`
	SweepAngle  float64    `yaml:"sweepAngle,omitempty"`

	// Spline fields. Closed marks a closed loop; Coords is a fixed spline's flattened
	// [x,y,z,…] (Points carries the solver-point ids of an interpolation/control spline).
	// X/Y/ZExpr over [T0,T1] define an equation curve.
	Closed bool      `yaml:"closed,omitempty"`
	Coords []float64 `yaml:"coords,omitempty"`
	XExpr  string    `yaml:"xExpr,omitempty"`
	YExpr  string    `yaml:"yExpr,omitempty"`
	ZExpr  string    `yaml:"zExpr,omitempty"`
	T0     float64   `yaml:"t0,omitempty"`
	T1     float64   `yaml:"t1,omitempty"`
	// Spline only (M06-F11, #626): the active tangency handles.
	Handles []SplineHandle3DData `yaml:"handles,omitempty"`
	// Helical only (M06-F09, #624): the shape definition — kind spelling,
	// variable rows and end conditions; absent reads as a constant
	// pitch-and-revolution shape with natural ends.
	HelixShapeKind string         `yaml:"helixShapeKind,omitempty"`
	HelixRows      []HelixRowData `yaml:"helixRows,omitempty"`
	HelixStart     *HelixEndData  `yaml:"helixStart,omitempty"`
	HelixEnd       *HelixEndData  `yaml:"helixEnd,omitempty"`
}

// HelixRowData is one persisted variable-shape station (cm / turns).
type HelixRowData struct {
	Diameter   float64 `yaml:"diameter,omitempty"`
	Pitch      float64 `yaml:"pitch,omitempty"`
	Height     float64 `yaml:"height,omitempty"`
	Revolution float64 `yaml:"revolution,omitempty"`
}

// HelixEndData is one persisted end condition (angles in radians).
type HelixEndData struct {
	Kind            string  `yaml:"kind"`
	TransitionAngle float64 `yaml:"transitionAngle,omitempty"`
	FlatAngle       float64 `yaml:"flatAngle,omitempty"`
}

// SplineHandle3DData is one active 3D spline tangency handle.
type SplineHandle3DData struct {
	FitIndex int        `yaml:"fitIndex"`
	End      [3]float64 `yaml:"end"`
}

// Constraint3DRow is one geometric 3D constraint: its kind plus operand ids split into
// Points (point operands) and Curves (line/curve entity operands), in the order the
// constraint's factory expects. Index is the splineFitPoints fit-point index (the
// constraint binds a specific fit point, not a re-derived nearest one); Radius is the
// bend's held radius.
type Constraint3DRow struct {
	Kind   string  `yaml:"kind"`
	Points []int   `yaml:"points,omitempty"`
	Curves []int   `yaml:"curves,omitempty"`
	Index  int     `yaml:"index,omitempty"`
	Radius float64 `yaml:"radius,omitempty"`
}

// MarshalRecipe3D projects every 3D sketch into its serializable form, in order.
func (c *Sketches3D) MarshalRecipe3D() ([]SketchData3D, error) {
	out := make([]SketchData3D, 0, c.Count())
	for i := 0; i < c.Count(); i++ {
		sd, err := serializeSketch3D(c.Item(i))
		if err != nil {
			return nil, fmt.Errorf("3D sketch %d: %w", i, err)
		}
		out = append(out, sd)
	}
	return out, nil
}

func serializeSketch3D(s *Sketch3D) (SketchData3D, error) {
	sd := SketchData3D{
		ID:           uint64(s.id),
		Name:         s.name,
		Hidden:       !s.visible,
		DimsHidden:   !s.dimensionsVisible,
		Shared:       s.shared,
		Seq:          s.seq,
		Color:        s.color,
		DeferUpdates: s.deferUpdates,
	}
	for _, p := range s.pts {
		_, standalone := s.byID[p.id]
		sd.Points = append(sd.Points, Point3DData{
			ID: int(p.id), X: float64(p.X), Y: float64(p.Y), Z: float64(p.Z), Standalone: standalone,
		})
	}
	for _, e := range s.ents {
		if _, isPoint := e.(*Point3D); isPoint {
			continue // standalone points are captured in Points, not Entities
		}
		if isDerivedCurve3D(e) {
			continue // surface-derived curves rebind from references on recompute (M22-F11)
		}
		ed, err := serializeEntity3D(e)
		if err != nil {
			return SketchData3D{}, err
		}
		sd.Entities = append(sd.Entities, ed)
	}
	for _, con := range s.geomCons.All() {
		cd, err := serializeConstraint3D(con)
		if err != nil {
			return SketchData3D{}, err
		}
		sd.Constraints = append(sd.Constraints, cd)
	}
	for _, d := range s.dimCons.items {
		dd, err := serializeDimension3D(d)
		if err != nil {
			return SketchData3D{}, err
		}
		sd.Dimensions = append(sd.Dimensions, dd)
	}
	return sd, nil
}

// serializeDimension3D captures one 3D dimension: its kind, operands (points vs curves),
// value expression, driving state, and the reference plane for a point-plane distance.
func serializeDimension3D(d *DimensionConstraint3D) (Dimension3DRow, error) {
	dr := Dimension3DRow{Kind: d.KindName(), Expression: d.param.Expression(), Driven: d.driven}
	for _, ref := range d.refs {
		if _, isPoint := ref.(*Point3D); isPoint {
			dr.Points = append(dr.Points, int(ref.EntityID()))
		} else {
			dr.Curves = append(dr.Curves, int(ref.EntityID()))
		}
	}
	if d.kind == PointPlaneDimKind3D {
		dr.Plane = planeNameFromNormal(d.planeNormal)
	}
	if dr.Kind == "unknown" {
		return Dimension3DRow{}, fmt.Errorf("cannot serialize 3D dimension of kind %d (no codec)", d.kind)
	}
	return dr, nil
}

// planeNameFromNormal maps an origin-plane normal to its label.
func planeNameFromNormal(n math.Vector3) string {
	switch {
	case n.Z != 0:
		return "XY"
	case n.Y != 0:
		return "XZ"
	default:
		return "YZ"
	}
}

// serializeEntity3D dispatches a 3D entity to its registered codec by its Kind
// and stamps the kind onto the row, mirroring serializeEntity (#1624).
func serializeEntity3D(e Entity) (Entity3DData, error) {
	ke, ok := e.(kindedEntity)
	if !ok {
		return Entity3DData{}, fmt.Errorf("cannot serialize 3D entity of type %T: it has no Kind (register it in serialize_codecs_3d.go)", e)
	}
	c, ok := entityCodecs3D[ke.Kind()]
	if !ok {
		return Entity3DData{}, fmt.Errorf("cannot serialize 3D entity of type %T: kind %q has no 3D codec", e, ke.Kind())
	}
	ed, err := c.encode(e)
	if err != nil {
		return Entity3DData{}, err
	}
	ed.Kind = string(ke.Kind())
	return ed, nil
}

// axisTriple flattens a unit axis to a serializable triple.
func axisTriple(u math.UnitVector3) [3]float64 {
	return [3]float64{float64(u.X()), float64(u.Y()), float64(u.Z())}
}

// point3DIDs returns the ids of a list of 3D points.
func point3DIDs(pts []*Point3D) []int {
	out := make([]int, len(pts))
	for i, p := range pts {
		out[i] = int(p.id)
	}
	return out
}

// flattenPoint3s flattens points to an [x,y,z,…] slice.
func flattenPoint3s(pts []math.Point3) []float64 {
	out := make([]float64, 0, len(pts)*3)
	for _, p := range pts {
		out = append(out, float64(p.X), float64(p.Y), float64(p.Z))
	}
	return out
}

// unflattenPoint3s rebuilds points from an [x,y,z,…] slice.
func unflattenPoint3s(coords []float64) []math.Point3 {
	out := make([]math.Point3, 0, len(coords)/3)
	for i := 0; i+2 < len(coords); i += 3 {
		out = append(out, math.P3(math.Scalar(coords[i]), math.Scalar(coords[i+1]), math.Scalar(coords[i+2])))
	}
	return out
}

// serializeConstraint3D captures one geometric 3D constraint by its point operands.
func serializeConstraint3D(c Constraint) (Constraint3DRow, error) {
	switch v := c.(type) {
	case *Coincident3D:
		return Constraint3DRow{Kind: "coincident", Points: []int{int(v.A.id), int(v.B.id)}}, nil
	case *Collinear3D:
		return Constraint3DRow{Kind: "collinear", Points: []int{int(v.A.id), int(v.B.id), int(v.C.id)}}, nil
	case *Concentric3D:
		return Constraint3DRow{Kind: "concentric", Points: []int{int(v.Center1.id), int(v.Center2.id)}}, nil
	case *Parallel3D:
		return Constraint3DRow{Kind: "parallel", Curves: []int{int(v.L1.id), int(v.L2.id)}}, nil
	case *Perpendicular3D:
		return Constraint3DRow{Kind: "perpendicular", Curves: []int{int(v.L1.id), int(v.L2.id)}}, nil
	case *Midpoint3D:
		return Constraint3DRow{Kind: "midpoint", Points: []int{int(v.P.id)}, Curves: []int{int(v.L.id)}}, nil
	case *Ground3D:
		return Constraint3DRow{Kind: "ground", Points: []int{int(v.P.id)}}, nil
	case *ParallelToAxis3D:
		return Constraint3DRow{Kind: axisRowKind(v.Axis), Curves: []int{int(v.L.id)}}, nil
	case *ParallelToPlane3D:
		return Constraint3DRow{Kind: planeRowKind(v.Normal), Curves: []int{int(v.L.id)}}, nil
	case *Tangent3D:
		return Constraint3DRow{Kind: "tangent", Curves: entity3DIDPair(v.C1, v.C2), Points: []int{int(v.P1.id), int(v.P2.id)}}, nil
	case *Smooth3D:
		return Constraint3DRow{Kind: "smooth", Curves: entity3DIDPair(v.C1, v.C2), Points: []int{int(v.P1.id), int(v.P2.id)}}, nil
	case *SplineFitPoints3D:
		return Constraint3DRow{Kind: "splineFitPoints", Curves: []int{int(v.Spline.id)}, Points: []int{int(v.P.id)}, Index: v.FitIndex}, nil
	case *Helical3D:
		return Constraint3DRow{Kind: "helical", Curves: []int{int(v.H.id), int(v.C.id)}}, nil
	case *Bend3D:
		return Constraint3DRow{
			Kind: "bend", Curves: []int{int(v.Arc.id), int(v.L1.id), int(v.L2.id)},
			Points: []int{int(v.P1.id), int(v.P2.id)}, Radius: v.Radius,
		}, nil
	default:
		return Constraint3DRow{}, fmt.Errorf("cannot serialize 3D constraint of type %T (no codec yet)", c)
	}
}

// entity3DIDPair returns two curve entities' ids in order.
func entity3DIDPair(a, b Entity) []int { return []int{int(a.EntityID()), int(b.EntityID())} }

// axisRowKind names a parallel-to-axis constraint for serialization by its axis vector.
func axisRowKind(axis math.Vector3) string {
	switch {
	case axis.X != 0:
		return "parallelToXAxis"
	case axis.Y != 0:
		return "parallelToYAxis"
	default:
		return "parallelToZAxis"
	}
}

// planeRowKind names a parallel-to-plane constraint for serialization by its normal.
func planeRowKind(normal math.Vector3) string {
	switch {
	case normal.Z != 0:
		return "parallelToXYPlane"
	case normal.Y != 0:
		return "parallelToXZPlane"
	default:
		return "parallelToYZPlane"
	}
}

// ApplyRecipe3D rebuilds the collection's 3D sketches from their serialized forms.
func (c *Sketches3D) ApplyRecipe3D(data []SketchData3D) error {
	for i, sd := range data {
		if err := c.restoreSketch3D(sd); err != nil {
			return fmt.Errorf("3D sketch %d (%q): %w", i, sd.Name, err)
		}
	}
	return nil
}

func (c *Sketches3D) restoreSketch3D(sd SketchData3D) error {
	s := c.AddNamed(sd.Name)
	c.restoreSketch3DID(s, sd.ID)
	restoreSketch3DProps(s, sd)
	idmap, maxPoint := restorePoints3D(s, sd.Points)
	entmap, maxEntity, err := restoreEntities3D(s, sd.Entities, idmap)
	if err != nil {
		return err
	}
	raiseIDSeq(maxPoint) // ids minted after load must not collide with the restored ones
	raiseIDSeq(maxEntity)
	for _, cd := range sd.Constraints {
		if err := restoreConstraint3D(s, cd, idmap, entmap); err != nil {
			return err
		}
	}
	for _, dr := range sd.Dimensions {
		if err := restoreDimension3D(s, dr, idmap, entmap); err != nil {
			return err
		}
	}
	return nil
}

// restoreSketch3DProps reapplies a 3D sketch's persisted display/solve properties.
func restoreSketch3DProps(s *Sketch3D, sd SketchData3D) {
	s.visible = !sd.Hidden
	s.dimensionsVisible = !sd.DimsHidden
	s.shared = sd.Shared
	s.color = sd.Color
	s.deferUpdates = sd.DeferUpdates
	seq.Restore(&s.seq, sd.Seq)
}

// restorePoints3D recreates the sketch's 3D points (standalone or curve-owned), pinning
// each saved id verbatim so derived reference keys stay stable, and returns the id map plus
// the largest id seen.
func restorePoints3D(s *Sketch3D, points []Point3DData) (map[int]*Point3D, uint64) {
	idmap := make(map[int]*Point3D, len(points))
	var maxID uint64
	for _, pd := range points {
		p := restorePoint3D(s, pd)
		s.pinEntityID3D(p, pd.ID)
		idmap[pd.ID] = p
		if uint64(pd.ID) > maxID {
			maxID = uint64(pd.ID)
		}
	}
	return idmap, maxID
}

// restorePoint3D creates one 3D point — standalone (an entity) or curve-owned — exactly one.
func restorePoint3D(s *Sketch3D, pd Point3DData) *Point3D {
	pos := math.P3(pd.X, pd.Y, pd.Z)
	if pd.Standalone {
		return s.AddPoint3D(pos)
	}
	return s.newPoint3D(pos)
}

// restoreEntities3D recreates the sketch's 3D curve entities over the restored points,
// pinning each saved id, and returns the entity map plus the largest id seen.
func restoreEntities3D(s *Sketch3D, entities []Entity3DData, idmap map[int]*Point3D) (map[int]Entity, uint64, error) {
	entmap := make(map[int]Entity, len(entities))
	var maxID uint64
	for _, ed := range entities {
		e, err := restoreEntity3D(s, ed, idmap)
		if err != nil {
			return nil, 0, err
		}
		s.pinEntityID3D(e, ed.ID)
		entmap[ed.ID] = e
		if uint64(ed.ID) > maxID {
			maxID = uint64(ed.ID)
		}
	}
	return entmap, maxID, nil
}

// restoreDimension3D re-adds one 3D dimension, binding operands through the id maps and
// re-applying its value expression + driving state.
func restoreDimension3D(s *Sketch3D, dr Dimension3DRow, idmap map[int]*Point3D, entmap map[int]Entity) error {
	pts, err := lookupPoints3D(dr.Points, idmap)
	if err != nil {
		return fmt.Errorf("%s dimension: %w", dr.Kind, err)
	}
	d, err := buildRestoredDimension3D(s, dr, pts, entmap)
	if err != nil {
		return fmt.Errorf("%s dimension: %w", dr.Kind, err)
	}
	d.SetDriven(dr.Driven)
	return nil
}

// buildRestoredDimension3D dispatches a serialized dimension kind to its factory.
func buildRestoredDimension3D(s *Sketch3D, dr Dimension3DRow, pts []*Point3D, entmap map[int]Entity) (*DimensionConstraint3D, error) {
	dc := s.DimensionConstraints3D()
	switch dr.Kind {
	case "distance":
		return dc.AddDistance(pts[0], pts[1], dr.Expression)
	case "lineLength":
		l, err := lookupLines3D(dr.Curves, entmap)
		if err != nil {
			return nil, err
		}
		return dc.AddLineLength(l[0], dr.Expression)
	case "radius":
		c, err := lookupCircle3D(dr.Curves, entmap)
		if err != nil {
			return nil, err
		}
		return dc.AddRadius(c, dr.Expression)
	case "pointPlaneDistance":
		return dc.AddPointPlaneDistance(pts[0], planeNormalFromLabel(dr.Plane), dr.Expression)
	case "twoLineAngle":
		l, err := lookupLines3D(dr.Curves, entmap)
		if err != nil {
			return nil, err
		}
		return dc.AddTwoLineAngle(l[0], l[1], dr.Expression)
	case "splineLength":
		sp, err := lookupSpline3D(dr.Curves, entmap)
		if err != nil {
			return nil, err
		}
		return dc.AddSplineLength(sp, dr.Expression)
	default:
		return nil, fmt.Errorf("unknown 3D dimension kind %q", dr.Kind)
	}
}

// planeNormalFromLabel maps an origin-plane label to its unit normal (default XY).
func planeNormalFromLabel(plane string) math.Vector3 {
	switch plane {
	case "XZ":
		return math.V3(0, 1, 0)
	case "YZ":
		return math.V3(1, 0, 0)
	default:
		return math.V3(0, 0, 1)
	}
}

// lookupCircle3D resolves a single saved entity id to a live 3D circle.
func lookupCircle3D(ids []int, entmap map[int]Entity) (*Circle3D, error) {
	if len(ids) != 1 {
		return nil, fmt.Errorf("radius needs 1 circle operand, got %d", len(ids))
	}
	e, ok := entmap[ids[0]]
	if !ok {
		return nil, fmt.Errorf(errUnknownEntityRef, ids[0])
	}
	c, ok := e.(*Circle3D)
	if !ok {
		return nil, fmt.Errorf("entity id %d is %T, want a 3D circle", ids[0], e)
	}
	return c, nil
}

// restoreEntity3D re-creates one 3D curve entity over its already-restored points
// through its kind's registered codec — the pair its encode came from (#1624).
func restoreEntity3D(s *Sketch3D, ed Entity3DData, idmap map[int]*Point3D) (Entity, error) {
	pts, err := lookupPoints3D(ed.Points, idmap)
	if err != nil {
		return nil, fmt.Errorf("%s entity: %w", ed.Kind, err)
	}
	c, ok := entityCodecs3D[EntityKind(ed.Kind)]
	if !ok {
		return nil, fmt.Errorf("unknown 3D entity kind %q", ed.Kind)
	}
	return c.decode(s, ed, pts)
}

// restoreConstraint3D re-adds one geometric 3D constraint, binding its point operands
// through idmap and its line operands through entmap. The curve-join kinds (tangent/
// smooth/splineFitPoints/helical, issue #142) dispatch first — their Curves are not
// necessarily lines, so they resolve against the full entity map.
func restoreConstraint3D(s *Sketch3D, cd Constraint3DRow, idmap map[int]*Point3D, entmap map[int]Entity) error {
	pts, err := lookupPoints3D(cd.Points, idmap)
	if err != nil {
		return fmt.Errorf(errConstraintWrap, cd.Kind, err)
	}
	if c, handled, err := curveConstraint3DFromRow(cd, pts, entmap); handled {
		if err != nil {
			return fmt.Errorf(errConstraintWrap, cd.Kind, err)
		}
		s.geomCons.add(c)
		return nil
	}
	lines, err := lookupLines3D(cd.Curves, entmap)
	if err != nil {
		return fmt.Errorf(errConstraintWrap, cd.Kind, err)
	}
	c, err := constraint3DFromRow(cd.Kind, pts, lines)
	if err != nil {
		return err
	}
	s.geomCons.add(c)
	return nil
}

// curveConstraint3DFromRow rebuilds the curve-join constraint kinds; handled is false
// for every other kind (which restoreConstraint3D resolves over line operands).
func curveConstraint3DFromRow(cd Constraint3DRow, pts []*Point3D, entmap map[int]Entity) (Constraint, bool, error) {
	switch cd.Kind {
	case "tangent", "smooth":
		c, err := restoreSmoothJoin3D(cd, pts, entmap)
		return c, true, err
	case "splineFitPoints":
		sp, err := lookupSpline3D(cd.Curves, entmap)
		if err != nil {
			return nil, true, err
		}
		c, err := NewSplineFitPoints3DAt(sp, pts[0], cd.Index)
		return c, true, err
	case "helical":
		c, err := restoreHelical3D(cd, entmap)
		return c, true, err
	case "bend":
		c, err := restoreBend3D(cd, pts, entmap)
		return c, true, err
	default:
		return nil, false, nil
	}
}

// restoreSmoothJoin3D rebuilds a tangent or smooth join from its two curves and their
// serialized join endpoints.
func restoreSmoothJoin3D(cd Constraint3DRow, pts []*Point3D, entmap map[int]Entity) (Constraint, error) {
	if len(cd.Curves) != 2 || len(pts) != 2 {
		return nil, fmt.Errorf("needs 2 curves + 2 points, got %d/%d", len(cd.Curves), len(pts))
	}
	c1, err := lookupSmoothCurve3D(cd.Curves[0], entmap)
	if err != nil {
		return nil, err
	}
	c2, err := lookupSmoothCurve3D(cd.Curves[1], entmap)
	if err != nil {
		return nil, err
	}
	if cd.Kind == "tangent" {
		return NewTangent3D(c1, c2, pts[0], pts[1]), nil
	}
	return NewSmooth3D(c1, c2, pts[0], pts[1]), nil
}

// restoreBend3D rebuilds a bend from its arc + two lines, re-binding the exact saved
// join endpoints and held radius.
func restoreBend3D(cd Constraint3DRow, pts []*Point3D, entmap map[int]Entity) (Constraint, error) {
	if len(cd.Curves) != 3 || len(pts) != 2 {
		return nil, fmt.Errorf("needs an arc + 2 lines + 2 join points, got %d/%d", len(cd.Curves), len(pts))
	}
	arc, ok := entmap[cd.Curves[0]].(*Arc3D)
	if !ok {
		return nil, fmt.Errorf("entity id %d is %T, want a 3D arc", cd.Curves[0], entmap[cd.Curves[0]])
	}
	lines, err := lookupLines3D(cd.Curves[1:], entmap)
	if err != nil {
		return nil, err
	}
	return newBend3DBound(arc, lines[0], lines[1], pts[0], pts[1], cd.Radius), nil
}

// restoreHelical3D rebuilds a helix-on-circle constraint from its two curve operands.
func restoreHelical3D(cd Constraint3DRow, entmap map[int]Entity) (Constraint, error) {
	if len(cd.Curves) != 2 {
		return nil, fmt.Errorf("needs a helix + circle, got %d curves", len(cd.Curves))
	}
	h, ok := entmap[cd.Curves[0]].(*HelicalCurve3D)
	if !ok {
		return nil, fmt.Errorf("entity id %d is %T, want a helical curve", cd.Curves[0], entmap[cd.Curves[0]])
	}
	circle, ok := entmap[cd.Curves[1]].(*Circle3D)
	if !ok {
		return nil, fmt.Errorf("entity id %d is %T, want a 3D circle", cd.Curves[1], entmap[cd.Curves[1]])
	}
	return NewHelical3D(h, circle)
}

// lookupSmoothCurve3D resolves a saved entity id to a tangent/smooth-capable curve.
func lookupSmoothCurve3D(id int, entmap map[int]Entity) (SmoothCurve3D, error) {
	e, ok := entmap[id]
	if !ok {
		return nil, fmt.Errorf(errUnknownEntityRef, id)
	}
	c, ok := e.(SmoothCurve3D)
	if !ok {
		return nil, fmt.Errorf("entity id %d is %T, want a line/arc/spline", id, e)
	}
	return c, nil
}

// lookupSpline3D resolves a single saved entity id to a live 3D spline.
func lookupSpline3D(ids []int, entmap map[int]Entity) (*Spline3D, error) {
	if len(ids) != 1 {
		return nil, fmt.Errorf("needs 1 spline operand, got %d", len(ids))
	}
	sp, ok := entmap[ids[0]].(*Spline3D)
	if !ok {
		return nil, fmt.Errorf("entity id %d is %T, want a 3D spline", ids[0], entmap[ids[0]])
	}
	return sp, nil
}

// constraint3DFromRow builds the constraint for a serialized kind from its resolved
// point and line operands.
func constraint3DFromRow(kind string, pts []*Point3D, lines []*Line3D) (Constraint, error) {
	switch kind {
	case "coincident":
		return NewCoincident3D(pts[0], pts[1]), nil
	case "collinear":
		return NewCollinear3D(pts[0], pts[1], pts[2]), nil
	case "concentric":
		return NewConcentric3D(pts[0], pts[1]), nil
	case "parallel":
		return NewParallel3D(lines[0], lines[1]), nil
	case "perpendicular":
		return NewPerpendicular3D(lines[0], lines[1]), nil
	case "midpoint":
		return NewMidpoint3D(pts[0], lines[0]), nil
	case "ground":
		return NewGround3D(pts[0]), nil
	default:
		return orientationFromRow(kind, lines)
	}
}

// orientationFromRow builds the parallel-to-axis/plane constraints from a serialized kind.
func orientationFromRow(kind string, lines []*Line3D) (Constraint, error) {
	switch kind {
	case "parallelToXAxis":
		return NewParallelToXAxis3D(lines[0]), nil
	case "parallelToYAxis":
		return NewParallelToYAxis3D(lines[0]), nil
	case "parallelToZAxis":
		return NewParallelToZAxis3D(lines[0]), nil
	case "parallelToXYPlane":
		return NewParallelToXYPlane3D(lines[0]), nil
	case "parallelToXZPlane":
		return NewParallelToXZPlane3D(lines[0]), nil
	case "parallelToYZPlane":
		return NewParallelToYZPlane3D(lines[0]), nil
	default:
		return nil, fmt.Errorf("unknown 3D constraint kind %q", kind)
	}
}

// restoreConic3D rebuilds a full or partial ellipse from its serialized form.
func restoreConic3D(s *Sketch3D, ed Entity3DData, center *Point3D) (Entity, error) {
	normal, err := unitFromTriple(ed.Axis)
	if err != nil {
		return nil, fmt.Errorf("%s entity: normal %v: %w", ed.Kind, ed.Axis, err)
	}
	major, err := unitFromTriple(ed.MajorAxis)
	if err != nil {
		return nil, fmt.Errorf("%s entity: major axis %v: %w", ed.Kind, ed.MajorAxis, err)
	}
	if ed.Kind == "ellipse" {
		e := s.addEllipse3DPt(center, normal, major, ed.Radius, ed.MinorRadius)
		e.SetConstruction(ed.Construction)
		return e, nil
	}
	e := s.addEllipticalArc3DPt(center, normal, major, ed.Radius, ed.MinorRadius, ed.StartAngle, ed.SweepAngle)
	e.SetConstruction(ed.Construction)
	return e, nil
}

// lookupLines3D resolves saved entity ids to live 3D lines through the entity map.
func lookupLines3D(ids []int, entmap map[int]Entity) ([]*Line3D, error) {
	out := make([]*Line3D, len(ids))
	for i, id := range ids {
		e, ok := entmap[id]
		if !ok {
			return nil, fmt.Errorf(errUnknownEntityRef, id)
		}
		l, ok := e.(*Line3D)
		if !ok {
			return nil, fmt.Errorf("entity id %d is %T, want a 3D line", id, e)
		}
		out[i] = l
	}
	return out, nil
}

// unitFromTriple builds a unit vector from a serialized triple.
func unitFromTriple(v [3]float64) (math.UnitVector3, error) {
	return math.NewUnitVector3(math.Scalar(v[0]), math.Scalar(v[1]), math.Scalar(v[2]))
}

// lookupPoints3D resolves saved point ids to live points through the id map.
func lookupPoints3D(ids []int, idmap map[int]*Point3D) ([]*Point3D, error) {
	out := make([]*Point3D, len(ids))
	for i, id := range ids {
		p, ok := idmap[id]
		if !ok {
			return nil, fmt.Errorf("references unknown point id %d", id)
		}
		out[i] = p
	}
	return out, nil
}

// serializeSplineHandles3D renders a 3D spline's active handles in fit order
// (M06-F11, #626).
func serializeSplineHandles3D(sp *Spline3D) []SplineHandle3DData {
	handles := sp.Handles()
	if len(handles) == 0 {
		return nil
	}
	out := make([]SplineHandle3DData, len(handles))
	for i, h := range handles {
		p := h.End.Position()
		out[i] = SplineHandle3DData{FitIndex: h.FitIndex, End: [3]float64{float64(p.X), float64(p.Y), float64(p.Z)}}
	}
	return out
}

// serializeHelixDefinition renders the M06-F09 shape definition onto the
// helix's row; a default constant/natural definition stays absent.
func serializeHelixDefinition(ed *Entity3DData, h *HelicalCurve3D) {
	def := h.Definition()
	if def.ShapeKind != types.HelixShapePitchRevolution || def.Variable() {
		ed.HelixShapeKind = def.ShapeKind.String()
	}
	for _, r := range def.Rows {
		ed.HelixRows = append(ed.HelixRows, HelixRowData(r))
	}
	ed.HelixStart = helixEndData(def.Start)
	ed.HelixEnd = helixEndData(def.End)
}

// helixEndData persists a non-natural end condition (nil for natural).
func helixEndData(c HelixEndCondition) *HelixEndData {
	if !c.flat() {
		return nil
	}
	return &HelixEndData{Kind: c.Kind.String(), TransitionAngle: c.TransitionAngle, FlatAngle: c.FlatAngle}
}

// restoreHelixDefinition rebuilds the persisted definition.
func restoreHelixDefinition(h *HelicalCurve3D, ed Entity3DData) error {
	def := h.Definition()
	if ed.HelixShapeKind != "" {
		kind, ok := types.ParseHelicalShapeDefinitionKind(ed.HelixShapeKind)
		if !ok {
			return fmt.Errorf("helical entity: unknown shape kind %q", ed.HelixShapeKind)
		}
		def.ShapeKind = kind
	}
	for _, r := range ed.HelixRows {
		def.Rows = append(def.Rows, HelixRow(r))
	}
	start, err := helixEndFromData(ed.HelixStart)
	if err != nil {
		return err
	}
	end, err := helixEndFromData(ed.HelixEnd)
	if err != nil {
		return err
	}
	h.SetEndConditions(start, end)
	return nil
}

// helixEndFromData parses a persisted end condition (nil stays natural).
func helixEndFromData(d *HelixEndData) (*HelixEndCondition, error) {
	if d == nil {
		return nil, nil
	}
	kind, ok := types.ParseHelixEndKind(d.Kind)
	if !ok {
		return nil, fmt.Errorf("helical entity: unknown end kind %q", d.Kind)
	}
	return &HelixEndCondition{Kind: kind, TransitionAngle: d.TransitionAngle, FlatAngle: d.FlatAngle}, nil
}
