// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"fmt"

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

	// Per-entity format overrides of a STANDALONE point (#2039); a defining point of a curve
	// carries none. Same three fields as EntityData, spelled the same way in the file.
	FormatLine   string  `yaml:"formatLineType,omitempty"`
	FormatColor  string  `yaml:"formatColor,omitempty"` // "#RRGGBB"; absent ⇒ inherit
	FormatWeight float64 `yaml:"formatLineWeight,omitempty"`
}

// Entity3DData is one 3D curve entity. Points lists the curve's defining point ids in a
// kind-specific order (line: A,B; circle: center; arc: center,start,end). Axis is the
// circle's plane normal. Standalone points are captured in Points, not here. Unused
// fields stay zero/omitted per kind.
type Entity3DData struct {
	ID   int    `yaml:"id"`
	Kind string `yaml:"kind"`
	// Per-entity format overrides (#2039), the same three fields the planar EntityData carries.
	FormatLine   string     `yaml:"formatLineType,omitempty"`
	FormatColor  string     `yaml:"formatColor,omitempty"` // "#RRGGBB"; absent ⇒ inherit
	FormatWeight float64    `yaml:"formatLineWeight,omitempty"`
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
	// CoordinateSystem is an equation curve's coordinate-system id (0 ⇒ cartesian; #1846).
	CoordinateSystem int32 `yaml:"coordinateSystem,omitempty"`
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
	// FaceRef is the face reference key an onFace constraint holds its point on (#1839).
	FaceRef string `yaml:"faceRef,omitempty"`
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
		pd := Point3DData{
			ID: int(p.id), X: float64(p.X), Y: float64(p.Y), Z: float64(p.Z), Standalone: standalone,
		}
		pd.FormatLine, pd.FormatColor, pd.FormatWeight = s.encodeEntityFormat(p.id)
		sd.Points = append(sd.Points, pd)
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
		ed.FormatLine, ed.FormatColor, ed.FormatWeight = s.encodeEntityFormat(e.EntityID())
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

// entity3DIDPair returns two curve entities' ids in order.
func entity3DIDPair(a, b Entity) []int { return []int{int(a.EntityID()), int(b.EntityID())} }

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
		s.decodeEntityFormat(p.EntityID(), pd.FormatLine, pd.FormatColor, pd.FormatWeight)
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
