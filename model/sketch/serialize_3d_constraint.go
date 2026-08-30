// SPDX-License-Identifier: GPL-2.0-only

package sketch

import "fmt"

// 3D-sketch serialization — CONSTRAINT family (M48 #2243 split of serialize_3d.go). The serialize/restore
// of the 3D dimensional and geometric constraints (dimension, bend, helical) — resolving each row back to
// its points/entities. The DTOs, dispatch and shared codec helpers live in serialize_3d.go; the entities
// in serialize_3d_entity.go.

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

// serializeConstraint3D encodes through the paired 3D codec registry, keyed on
// the constraint's self-reported kind (#1625) — no per-type switch, so an
// encode half can no longer ship without its decode half (Equal3D did exactly
// that: creatable, enumerable, and failing every save at runtime).
func serializeConstraint3D(c Constraint) (Constraint3DRow, error) {
	kc, ok := c.(KindedConstraint)
	if !ok {
		return Constraint3DRow{}, fmt.Errorf("cannot serialize 3D constraint of type %T (no ConstraintKind capability)", c)
	}
	codec, ok := constraintCodecs3D[kc.ConstraintKind()]
	if !ok {
		return Constraint3DRow{}, fmt.Errorf("cannot serialize 3D constraint kind %q of type %T (no codec)", kc.ConstraintKind(), c)
	}
	row, err := codec.encode(c)
	if err != nil {
		return Constraint3DRow{}, err
	}
	row.Kind = string(kc.ConstraintKind())
	return row, nil
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

// restoreConstraint3D decodes one geometric 3D constraint through the paired
// codec registry (#1625), binding its point operands through idmap and its
// curve operands through entmap. An unknown kind is a corrupt-recipe error.
func restoreConstraint3D(s *Sketch3D, cd Constraint3DRow, idmap map[int]*Point3D, entmap map[int]Entity) error {
	codec, ok := constraintCodecs3D[ConstraintKind(cd.Kind)]
	if !ok {
		return fmt.Errorf("unknown constraint kind %q", cd.Kind)
	}
	pts, err := lookupPoints3D(cd.Points, idmap)
	if err != nil {
		return fmt.Errorf(errConstraintWrap, cd.Kind, err)
	}
	if err := codec.decode(s, cd, pts, entmap); err != nil {
		return fmt.Errorf(errConstraintWrap, cd.Kind, err)
	}
	return nil
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
