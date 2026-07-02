// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"fmt"

	"oblikovati.org/math"
)

// Codec registrations for the 2D geometric curves (line/circle/arc/ellipse/
// ellipticalArc/spline). Bodies are the former serializeEntity/restoreEntity
// switch cases, paired so encode and decode can never drift (#1624; the
// pattern of model/feature's #1416 fix). serializeEntity stamps ed.Kind
// centrally, so encode closures fill only the kind-specific payload.

func init() {
	registerEntityCodec(LineKind, entityCodec{
		encode: func(e Entity) (EntityData, error) {
			v := e.(*Line)
			return EntityData{ID: int(v.id), Points: []int{int(v.A.id), int(v.B.id)}, Construction: v.construction, Centerline: v.centerline}, nil
		},
		decode: func(r *sketchRestorer, ed EntityData) (Entity, error) {
			p, err := r.points(ed.Points, 2)
			if err != nil {
				return nil, err
			}
			return r.s.lines.Add(p[0], p[1]), nil
		},
	})
	registerEntityCodec(CircleKind, entityCodec{
		encode: func(e Entity) (EntityData, error) {
			v := e.(*Circle)
			return EntityData{ID: int(v.id), Points: []int{int(v.Center.id)}, Radius: float64(v.Radius), Construction: v.construction}, nil
		},
		decode: func(r *sketchRestorer, ed EntityData) (Entity, error) {
			p, err := r.points(ed.Points, 1)
			if err != nil {
				return nil, err
			}
			return r.s.circles.Add(p[0], math.Scalar(ed.Radius)), nil
		},
	})
	registerEntityCodec(ArcKind, entityCodec{
		encode: func(e Entity) (EntityData, error) {
			v := e.(*Arc)
			return EntityData{ID: int(v.id), Points: []int{int(v.Center.id), int(v.Start.id), int(v.End.id)}, CCW: v.CounterClockwise, Construction: v.construction}, nil
		},
		decode: func(r *sketchRestorer, ed EntityData) (Entity, error) {
			p, err := r.points(ed.Points, 3)
			if err != nil {
				return nil, err
			}
			return r.s.arcs.Add(p[0], p[1], p[2], ed.CCW), nil
		},
	})
	registerEntityCodec(EllipseKind, entityCodec{
		encode: func(e Entity) (EntityData, error) {
			v := e.(*Ellipse)
			return EntityData{
				ID: int(v.id), Points: []int{int(v.Center.id)},
				MajorAxis:   []float64{float64(v.MajorAxis.X), float64(v.MajorAxis.Y)},
				MajorRadius: float64(v.MajorRadius), MinorRadius: float64(v.MinorRadius),
				Construction: v.construction,
			}, nil
		},
		decode: func(r *sketchRestorer, ed EntityData) (Entity, error) {
			p, axis, err := r.conicOperands(ed)
			if err != nil {
				return nil, err
			}
			return r.s.ellipses.AddWithCenter(p, axis, math.Scalar(ed.MajorRadius), math.Scalar(ed.MinorRadius)), nil
		},
	})
	registerEntityCodec(EllipticalArcKind, entityCodec{
		encode: func(e Entity) (EntityData, error) {
			v := e.(*EllipticalArc)
			return EntityData{
				ID: int(v.id), Points: []int{int(v.Center.id)},
				MajorAxis:   []float64{float64(v.MajorAxis.X), float64(v.MajorAxis.Y)},
				MajorRadius: float64(v.MajorRadius), MinorRadius: float64(v.MinorRadius),
				StartAngle: float64(v.StartAngle), EndAngle: float64(v.EndAngle),
				Construction: v.construction,
			}, nil
		},
		decode: func(r *sketchRestorer, ed EntityData) (Entity, error) {
			p, axis, err := r.conicOperands(ed)
			if err != nil {
				return nil, err
			}
			return r.s.ellArcs.AddWithCenter(p, axis, math.Scalar(ed.MajorRadius), math.Scalar(ed.MinorRadius),
				math.Scalar(ed.StartAngle), math.Scalar(ed.EndAngle)), nil
		},
	})
	registerEntityCodec(SplineKind, entityCodec{
		encode: func(e Entity) (EntityData, error) {
			v := e.(*Spline)
			return EntityData{
				ID: int(v.id), Points: pointIDsOf(v.Points), Closed: v.Closed, Fit: v.fit,
				FitMethod: fitMethodSpelling(v.FitMethod), Handles: serializeSplineHandles(v),
				Construction: v.construction,
			}, nil
		},
		decode: func(r *sketchRestorer, ed EntityData) (Entity, error) {
			p, err := r.points(ed.Points, len(ed.Points))
			if err != nil {
				return nil, err
			}
			sp := r.s.splines.AddWithPoints(p, ed.Closed, ed.Fit)
			if err := restoreSplineExtras(r.s, sp, ed); err != nil {
				return nil, err
			}
			return sp, nil
		},
	})
}

// conicOperands resolves an ellipse/ellipticalArc's center point and major-axis
// direction, shared by both conic decoders.
func (r *sketchRestorer) conicOperands(ed EntityData) (*Point, math.Vector2, error) {
	p, err := r.points(ed.Points, 1)
	if err != nil {
		return nil, math.Vector2{}, err
	}
	if len(ed.MajorAxis) != 2 {
		return nil, math.Vector2{}, fmt.Errorf("%s needs a 2-component majorAxis, got %d", ed.Kind, len(ed.MajorAxis))
	}
	return p[0], math.V2(ed.MajorAxis[0], ed.MajorAxis[1]), nil
}
