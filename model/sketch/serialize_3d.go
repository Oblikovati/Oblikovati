// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"fmt"

	"github.com/Oblikovati/oblikovati/math"
)

// This file defines the git-friendly YAML projection of a 3D sketch (ADR-0020) and its
// round trip. A 3D sketch is captured as its display/solve properties, its constrainable
// 3D points (the solver's variables), and its curve entities (referencing points by id).
// Curve-entity and constraint/dimension codecs grow alongside their features (M22 F02+),
// each adding a case here; the spine (M22-F01) round-trips points and properties.

// SketchData3D is the serializable form of one 3D sketch. Hidden inverts Visible so the
// common visible=true case omits the field; DimsHidden inverts DimensionsVisible likewise.
type SketchData3D struct {
	Name         string            `yaml:"name,omitempty"`
	Hidden       bool              `yaml:"hidden,omitempty"`
	DimsHidden   bool              `yaml:"dimsHidden,omitempty"`
	Color        string            `yaml:"color,omitempty"`
	DeferUpdates bool              `yaml:"deferUpdates,omitempty"`
	Points       []Point3DData     `yaml:"points,omitempty"`
	Entities     []Entity3DData    `yaml:"entities,omitempty"`
	Constraints  []Constraint3DRow `yaml:"constraints,omitempty"`
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
	Axis         [3]float64 `yaml:"axis,omitempty"` // circle plane normal
	CCW          bool       `yaml:"ccw,omitempty"`  // arc sweep direction
	Construction bool       `yaml:"construction,omitempty"`
}

// Constraint3DRow is one geometric 3D constraint: its kind and the point operand ids in
// the order the constraint's factory expects.
type Constraint3DRow struct {
	Kind   string `yaml:"kind"`
	Points []int  `yaml:"points,omitempty"`
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
		Name:         s.name,
		Hidden:       !s.visible,
		DimsHidden:   !s.dimensionsVisible,
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
	return sd, nil
}

// serializeEntity3D captures one 3D curve entity by its kind, defining point ids, and
// kind-specific shape (radius/axis/sweep).
func serializeEntity3D(e Entity) (Entity3DData, error) {
	switch v := e.(type) {
	case *Line3D:
		return Entity3DData{ID: int(v.id), Kind: "line", Points: []int{int(v.A.id), int(v.B.id)}, Construction: v.construction}, nil
	case *Circle3D:
		return Entity3DData{
			ID: int(v.id), Kind: "circle", Points: []int{int(v.Center.id)},
			Radius: float64(v.Radius), Axis: axisTriple(v.Axis), Construction: v.construction,
		}, nil
	case *Arc3D:
		return Entity3DData{
			ID: int(v.id), Kind: "arc", Points: []int{int(v.Center.id), int(v.Start.id), int(v.End.id)},
			CCW: v.CounterClockwise, Construction: v.construction,
		}, nil
	default:
		return Entity3DData{}, fmt.Errorf("cannot serialize 3D entity of type %T (no codec)", e)
	}
}

// axisTriple flattens a unit axis to a serializable triple.
func axisTriple(u math.UnitVector3) [3]float64 {
	return [3]float64{float64(u.X()), float64(u.Y()), float64(u.Z())}
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
	default:
		return Constraint3DRow{}, fmt.Errorf("cannot serialize 3D constraint of type %T (no codec yet)", c)
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
	s.visible = !sd.Hidden
	s.dimensionsVisible = !sd.DimsHidden
	s.color = sd.Color
	s.deferUpdates = sd.DeferUpdates
	// Re-create points, mapping their saved ids onto the freshly minted ones so
	// constraints can re-bind by id.
	idmap := make(map[int]*Point3D, len(sd.Points))
	for _, pd := range sd.Points {
		var p *Point3D
		if pd.Standalone {
			p = s.AddPoint3D(math.P3(pd.X, pd.Y, pd.Z))
		} else {
			p = s.newPoint3D(math.P3(pd.X, pd.Y, pd.Z))
		}
		idmap[pd.ID] = p
	}
	for _, ed := range sd.Entities {
		if err := restoreEntity3D(s, ed, idmap); err != nil {
			return err
		}
	}
	for _, cd := range sd.Constraints {
		if err := restoreConstraint3D(s, cd, idmap); err != nil {
			return err
		}
	}
	return nil
}

// restoreEntity3D re-creates one 3D curve entity over its already-restored points.
func restoreEntity3D(s *Sketch3D, ed Entity3DData, idmap map[int]*Point3D) error {
	pts, err := lookupPoints3D(ed.Points, idmap)
	if err != nil {
		return fmt.Errorf("%s entity: %w", ed.Kind, err)
	}
	switch ed.Kind {
	case "line":
		s.addLine3DPts(pts[0], pts[1]).SetConstruction(ed.Construction)
	case "circle":
		axis, aerr := math.NewUnitVector3(math.Scalar(ed.Axis[0]), math.Scalar(ed.Axis[1]), math.Scalar(ed.Axis[2]))
		if aerr != nil {
			return fmt.Errorf("circle entity: axis %v: %w", ed.Axis, aerr)
		}
		s.addCircle3DPts(pts[0], axis, ed.Radius).SetConstruction(ed.Construction)
	case "arc":
		s.addArc3DPts(pts[0], pts[1], pts[2], ed.CCW).SetConstruction(ed.Construction)
	default:
		return fmt.Errorf("unknown 3D entity kind %q", ed.Kind)
	}
	return nil
}

// restoreConstraint3D re-adds one geometric 3D constraint, binding its point operands
// through the id map.
func restoreConstraint3D(s *Sketch3D, cd Constraint3DRow, idmap map[int]*Point3D) error {
	pts, err := lookupPoints3D(cd.Points, idmap)
	if err != nil {
		return fmt.Errorf("%s constraint: %w", cd.Kind, err)
	}
	switch cd.Kind {
	case "coincident":
		s.geomCons.add(NewCoincident3D(pts[0], pts[1]))
	case "collinear":
		s.geomCons.add(NewCollinear3D(pts[0], pts[1], pts[2]))
	case "concentric":
		s.geomCons.add(NewConcentric3D(pts[0], pts[1]))
	default:
		return fmt.Errorf("unknown 3D constraint kind %q", cd.Kind)
	}
	return nil
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
