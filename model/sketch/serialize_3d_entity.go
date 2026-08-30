// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"fmt"

	"oblikovati.org/api/types"
)

// 3D-sketch serialization — CURVE / ENTITY family (M48 #2243 split of serialize_3d.go). The serialize/
// restore of the 3D-sketch entities (lines, circles, conics, splines, helical curves) including the helix
// definition and spline-handle encoding and the entity lookups. The DTOs, dispatch and shared codec
// helpers live in serialize_3d.go; the constraints in serialize_3d_constraint.go.

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
		s.decodeEntityFormat(e.EntityID(), ed.FormatLine, ed.FormatColor, ed.FormatWeight)
		entmap[ed.ID] = e
		if uint64(ed.ID) > maxID {
			maxID = uint64(ed.ID)
		}
	}
	return entmap, maxID, nil
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
