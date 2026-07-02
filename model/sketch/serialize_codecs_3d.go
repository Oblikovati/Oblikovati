// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"fmt"

	"oblikovati.org/math"
)

// Codec registrations for the persisted 3D entities. Bodies are the former
// serializeEntity3D/restoreEntity3D switch cases, paired per kind (#1624).
// The surface-derived curves (intersection/silhouette/on-face/…) have no codec
// on purpose — they rebind from their references on recompute (M22-F11), so
// serializeSketch3D skips them before dispatching here. A Spline3D registers
// under two kinds (spline / controlPointSpline) sharing one encode; its Kind()
// picks the spelling by the fit flag.

func init() {
	registerEntityCodec3D(LineKind, entityCodec3D{
		encode: func(e Entity) (Entity3DData, error) {
			v := e.(*Line3D)
			return Entity3DData{ID: int(v.id), Points: []int{int(v.A.id), int(v.B.id)}, Construction: v.construction}, nil
		},
		decode: func(s *Sketch3D, ed Entity3DData, pts []*Point3D) (Entity, error) {
			l := s.addLine3DPts(pts[0], pts[1])
			l.SetConstruction(ed.Construction)
			return l, nil
		},
	})
	registerEntityCodec3D(CircleKind, entityCodec3D{
		encode: func(e Entity) (Entity3DData, error) {
			v := e.(*Circle3D)
			return Entity3DData{
				ID: int(v.id), Points: []int{int(v.Center.id)},
				Radius: float64(v.Radius), Axis: axisTriple(v.Axis), Construction: v.construction,
			}, nil
		},
		decode: func(s *Sketch3D, ed Entity3DData, pts []*Point3D) (Entity, error) {
			axis, err := unitFromTriple(ed.Axis)
			if err != nil {
				return nil, fmt.Errorf("circle entity: axis %v: %w", ed.Axis, err)
			}
			c := s.addCircle3DPts(pts[0], axis, ed.Radius)
			c.SetConstruction(ed.Construction)
			return c, nil
		},
	})
	registerEntityCodec3D(ArcKind, entityCodec3D{
		encode: func(e Entity) (Entity3DData, error) {
			v := e.(*Arc3D)
			return Entity3DData{
				ID: int(v.id), Points: []int{int(v.Center.id), int(v.Start.id), int(v.End.id)},
				CCW: v.CounterClockwise, Construction: v.construction,
			}, nil
		},
		decode: func(s *Sketch3D, ed Entity3DData, pts []*Point3D) (Entity, error) {
			a := s.addArc3DPts(pts[0], pts[1], pts[2], ed.CCW)
			a.SetConstruction(ed.Construction)
			return a, nil
		},
	})
	registerEntityCodec3D(HelicalKind, entityCodec3D{
		encode: encodeHelical3D,
		decode: decodeHelical3D,
	})
	registerConic3DCodecs()
	registerSpline3DCodecs()
}

// encodeHelical3D captures a helix's placement plus its M06-F09 shape definition.
func encodeHelical3D(e Entity) (Entity3DData, error) {
	v := e.(*HelicalCurve3D)
	ed := Entity3DData{
		ID: int(v.id), Points: []int{int(v.Origin.id)},
		Radius: float64(v.StartRadius), Axis: axisTriple(v.Axis),
		Pitch: v.AxialPerTurn, Turns: v.Turns, RadialPerTurn: v.RadialPerTurn,
		Clockwise: v.Clockwise, Construction: v.construction,
	}
	serializeHelixDefinition(&ed, v)
	return ed, nil
}

// decodeHelical3D rebuilds a helix and its persisted shape definition.
func decodeHelical3D(s *Sketch3D, ed Entity3DData, pts []*Point3D) (Entity, error) {
	axis, err := unitFromTriple(ed.Axis)
	if err != nil {
		return nil, fmt.Errorf("helical entity: axis %v: %w", ed.Axis, err)
	}
	h := s.addHelix3DPt(pts[0], axis, ed.Radius, ed.Pitch, ed.RadialPerTurn, ed.Turns, ed.Clockwise)
	h.SetConstruction(ed.Construction)
	if err := restoreHelixDefinition(h, ed); err != nil {
		return nil, err
	}
	return h, nil
}

// registerConic3DCodecs pairs the 3D ellipse and elliptical arc; both decode
// through restoreConic3D, which dispatches on the kind spelling.
func registerConic3DCodecs() {
	decodeConic := func(s *Sketch3D, ed Entity3DData, pts []*Point3D) (Entity, error) {
		return restoreConic3D(s, ed, pts[0])
	}
	registerEntityCodec3D(EllipseKind, entityCodec3D{
		encode: func(e Entity) (Entity3DData, error) {
			v := e.(*Ellipse3D)
			return Entity3DData{
				ID: int(v.id), Points: []int{int(v.Center.id)},
				Axis: axisTriple(v.Normal), MajorAxis: axisTriple(v.MajorAxis),
				Radius: float64(v.MajorRadius), MinorRadius: float64(v.MinorRadius), Construction: v.construction,
			}, nil
		},
		decode: decodeConic,
	})
	registerEntityCodec3D(EllipticalArcKind, entityCodec3D{
		encode: func(e Entity) (Entity3DData, error) {
			v := e.(*EllipticalArc3D)
			return Entity3DData{
				ID: int(v.id), Points: []int{int(v.Center.id)},
				Axis: axisTriple(v.Normal), MajorAxis: axisTriple(v.MajorAxis),
				Radius: float64(v.MajorRadius), MinorRadius: float64(v.MinorRadius),
				StartAngle: v.StartAngle, SweepAngle: v.SweepAngle, Construction: v.construction,
			}, nil
		},
		decode: decodeConic,
	})
}

// registerSpline3DCodecs pairs the three 3D spline flavors. The interpolation
// (fit) and control-point spellings share both halves: encode reads the kind
// from Spline3D.Kind(), decode passes fit = (kind == "spline") to the factory.
func registerSpline3DCodecs() {
	splineCodec := entityCodec3D{
		encode: func(e Entity) (Entity3DData, error) {
			v := e.(*Spline3D)
			return Entity3DData{
				ID: int(v.id), Points: point3DIDs(v.Points),
				Closed: v.Closed, Handles: serializeSplineHandles3D(v), Construction: v.construction,
			}, nil
		},
		decode: decodeSpline3D,
	}
	registerEntityCodec3D(SplineKind, splineCodec)
	registerEntityCodec3D(ControlPointSplineKind, splineCodec)
	registerEntityCodec3D(FixedSplineKind, entityCodec3D{
		encode: func(e Entity) (Entity3DData, error) {
			v := e.(*FixedSpline3D)
			return Entity3DData{ID: int(v.id), Coords: flattenPoint3s(v.Pts), Closed: v.Closed, Construction: v.construction}, nil
		},
		decode: func(s *Sketch3D, ed Entity3DData, _ []*Point3D) (Entity, error) {
			sp := s.AddFixedSpline3D(unflattenPoint3s(ed.Coords), ed.Closed)
			sp.SetConstruction(ed.Construction)
			return sp, nil
		},
	})
	registerEntityCodec3D(EquationCurveKind, entityCodec3D{
		encode: func(e Entity) (Entity3DData, error) {
			v := e.(*EquationCurve3D)
			return Entity3DData{
				ID: int(v.id), XExpr: v.XExpr, YExpr: v.YExpr, ZExpr: v.ZExpr,
				T0: v.T0, T1: v.T1, Construction: v.construction,
			}, nil
		},
		decode: func(s *Sketch3D, ed Entity3DData, _ []*Point3D) (Entity, error) {
			e, err := s.AddEquationCurve3D(ed.XExpr, ed.YExpr, ed.ZExpr, ed.T0, ed.T1)
			if err != nil {
				return nil, fmt.Errorf("equationCurve entity: %w", err)
			}
			e.SetConstruction(ed.Construction)
			return e, nil
		},
	})
}

// decodeSpline3D rebuilds an interpolation or control-point spline over its
// restored solver points and re-activates its tangency handles.
func decodeSpline3D(s *Sketch3D, ed Entity3DData, pts []*Point3D) (Entity, error) {
	sp := s.addSpline3DPts(pts, ed.Closed, EntityKind(ed.Kind) == SplineKind)
	sp.SetConstruction(ed.Construction)
	for _, hd := range ed.Handles {
		h, err := s.ActivateSplineHandle3D(sp, hd.FitIndex)
		if err != nil {
			return nil, err
		}
		h.End.SetPosition(point3FromTriple(hd.End))
	}
	return sp, nil
}

// point3FromTriple rebuilds a point from its serialized triple.
func point3FromTriple(v [3]float64) math.Point3 {
	return math.P3(math.Scalar(v[0]), math.Scalar(v[1]), math.Scalar(v[2]))
}
