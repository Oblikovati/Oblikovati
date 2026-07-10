// SPDX-License-Identifier: GPL-2.0-only

package sketch

import "fmt"

// Codec registrations for the 3D geometric constraints. Bodies are the former
// switch cases of serializeConstraint3D and constraint3DFromRow/
// orientationFromRow/curveConstraint3DFromRow, paired per kind (#1625, audit
// I2). The "equal" codec is NEW: Equal3D was creatable and enumerable over the
// wire but missing from BOTH serialize switches, so saving a document with one
// failed at runtime — the exact drift class this registry closes (#1416).

func init() {
	registerPoint3DConstraintCodecs()
	registerLine3DConstraintCodecs()
	registerOrientation3DConstraintCodecs()
	registerCurveJoin3DConstraintCodecs()
	registerOnFace3DConstraintCodec()
}

// registerOnFace3DConstraintCodec persists the point-on-face constraint (#1839): its point id plus
// the face reference key. Decode restores it FROZEN (no live surface); the host rebinds a source
// after load (compdef.rebindSketch3DConstraints), mirroring projected geometry.
func registerOnFace3DConstraintCodec() {
	registerConstraintCodec3D(OnFaceKind, constraintCodec3D{
		encode: func(c Constraint) (Constraint3DRow, error) {
			v := c.(*OnFace3D)
			return Constraint3DRow{Points: []int{int(v.P.id)}, FaceRef: v.ref}, nil
		},
		decode: func(s *Sketch3D, cd Constraint3DRow, pts []*Point3D, _ map[int]Entity) error {
			if len(pts) != 1 {
				return fmt.Errorf("onFace needs 1 point, got %d", len(pts))
			}
			s.geomCons.add(NewOnFace3D(pts[0], nil, cd.FaceRef))
			return nil
		},
	})
}

// point3DConstraintCodec pairs an all-points 3D constraint row with its factory.
func point3DConstraintCodec(n int, operands func(Constraint) []*Point3D, build func(pts []*Point3D) Constraint) constraintCodec3D {
	return constraintCodec3D{
		encode: func(c Constraint) (Constraint3DRow, error) {
			return Constraint3DRow{Points: point3DIDs(operands(c))}, nil
		},
		decode: func(s *Sketch3D, cd Constraint3DRow, pts []*Point3D, entmap map[int]Entity) error {
			if len(pts) != n {
				return fmt.Errorf("needs %d points, got %d", n, len(pts))
			}
			s.geomCons.add(build(pts))
			return nil
		},
	}
}

// line3DConstraintCodec pairs an all-lines 3D constraint row with its factory.
func line3DConstraintCodec(n int, operands func(Constraint) []*Line3D, build func(lines []*Line3D) Constraint) constraintCodec3D {
	return constraintCodec3D{
		encode: func(c Constraint) (Constraint3DRow, error) {
			ls := operands(c)
			ids := make([]int, len(ls))
			for i, l := range ls {
				ids[i] = int(l.id)
			}
			return Constraint3DRow{Curves: ids}, nil
		},
		decode: func(s *Sketch3D, cd Constraint3DRow, pts []*Point3D, entmap map[int]Entity) error {
			lines, err := lookupLines3D(cd.Curves, entmap)
			if err != nil {
				return err
			}
			if len(lines) != n {
				return fmt.Errorf("needs %d lines, got %d", n, len(lines))
			}
			s.geomCons.add(build(lines))
			return nil
		},
	}
}

func registerPoint3DConstraintCodecs() {
	registerConstraintCodec3D(CoincidentKind, point3DConstraintCodec(2,
		func(c Constraint) []*Point3D { v := c.(*Coincident3D); return []*Point3D{v.A, v.B} },
		func(pts []*Point3D) Constraint { return NewCoincident3D(pts[0], pts[1]) }))
	registerConstraintCodec3D(Collinear3PointKind, point3DConstraintCodec(3,
		func(c Constraint) []*Point3D { v := c.(*Collinear3D); return []*Point3D{v.A, v.B, v.C} },
		func(pts []*Point3D) Constraint { return NewCollinear3D(pts[0], pts[1], pts[2]) }))
	registerConstraintCodec3D(ConcentricKind, point3DConstraintCodec(2,
		func(c Constraint) []*Point3D { v := c.(*Concentric3D); return []*Point3D{v.Center1, v.Center2} },
		func(pts []*Point3D) Constraint { return NewConcentric3D(pts[0], pts[1]) }))
	registerConstraintCodec3D(GroundKind, point3DConstraintCodec(1,
		func(c Constraint) []*Point3D { return []*Point3D{c.(*Ground3D).P} },
		func(pts []*Point3D) Constraint { return NewGround3D(pts[0]) }))
	registerConstraintCodec3D(EqualKind, equal3DCodec())
}

// equal3DCodec serializes an equal-radius constraint by its two radius-bearing
// curve operands (#1625) — the refs Equal3D now carries.
func equal3DCodec() constraintCodec3D {
	return constraintCodec3D{
		encode: func(c Constraint) (Constraint3DRow, error) {
			v := c.(*Equal3D)
			return Constraint3DRow{Curves: []int{int(v.E1.EntityID()), int(v.E2.EntityID())}}, nil
		},
		decode: func(s *Sketch3D, cd Constraint3DRow, pts []*Point3D, entmap map[int]Entity) error {
			if len(cd.Curves) != 2 {
				return fmt.Errorf("needs 2 radius-bearing curves, got %d", len(cd.Curves))
			}
			e1, ok1 := entmap[cd.Curves[0]]
			e2, ok2 := entmap[cd.Curves[1]]
			if !ok1 || !ok2 {
				return fmt.Errorf("unknown curve ref in %v", cd.Curves)
			}
			eq, err := NewEqual3D(e1, e2)
			if err != nil {
				return err
			}
			s.geomCons.add(eq)
			return nil
		},
	}
}

func registerLine3DConstraintCodecs() {
	registerConstraintCodec3D(ParallelKind, line3DConstraintCodec(2,
		func(c Constraint) []*Line3D { v := c.(*Parallel3D); return []*Line3D{v.L1, v.L2} },
		func(lines []*Line3D) Constraint { return NewParallel3D(lines[0], lines[1]) }))
	registerConstraintCodec3D(PerpendicularKind, line3DConstraintCodec(2,
		func(c Constraint) []*Line3D { v := c.(*Perpendicular3D); return []*Line3D{v.L1, v.L2} },
		func(lines []*Line3D) Constraint { return NewPerpendicular3D(lines[0], lines[1]) }))
	registerConstraintCodec3D(MidpointKind, midpoint3DCodec())
}

func midpoint3DCodec() constraintCodec3D {
	return constraintCodec3D{
		encode: func(c Constraint) (Constraint3DRow, error) {
			v := c.(*Midpoint3D)
			return Constraint3DRow{Points: []int{int(v.P.id)}, Curves: []int{int(v.L.id)}}, nil
		},
		decode: func(s *Sketch3D, cd Constraint3DRow, pts []*Point3D, entmap map[int]Entity) error {
			lines, err := lookupLines3D(cd.Curves, entmap)
			if err != nil {
				return err
			}
			if len(pts) != 1 || len(lines) != 1 {
				return fmt.Errorf("needs 1 point + 1 line, got %d/%d", len(pts), len(lines))
			}
			s.geomCons.add(NewMidpoint3D(pts[0], lines[0]))
			return nil
		},
	}
}

// registerOrientation3DConstraintCodecs covers the parallel-to-axis/plane
// family: ONE type serializes under three kinds each, derived from the
// constrained direction by ConstraintKind() (the pre-#1625 axisRowKind/
// planeRowKind spellings). All six share the one-line row shape; each kind's
// decode rebuilds the matching origin-frame variant.
func registerOrientation3DConstraintCodecs() {
	axisEncode := func(c Constraint) (Constraint3DRow, error) {
		return Constraint3DRow{Curves: []int{int(c.(*ParallelToAxis3D).L.id)}}, nil
	}
	planeEncode := func(c Constraint) (Constraint3DRow, error) {
		return Constraint3DRow{Curves: []int{int(c.(*ParallelToPlane3D).L.id)}}, nil
	}
	builders := map[ConstraintKind]struct {
		encode func(Constraint) (Constraint3DRow, error)
		build  func(*Line3D) Constraint
	}{
		ParallelToXAxisKind: {axisEncode, func(l *Line3D) Constraint { return NewParallelToXAxis3D(l) }},
		ParallelToYAxisKind: {axisEncode, func(l *Line3D) Constraint { return NewParallelToYAxis3D(l) }},
		ParallelToZAxisKind: {axisEncode, func(l *Line3D) Constraint { return NewParallelToZAxis3D(l) }},
		ParallelToXYKind:    {planeEncode, func(l *Line3D) Constraint { return NewParallelToXYPlane3D(l) }},
		ParallelToXZKind:    {planeEncode, func(l *Line3D) Constraint { return NewParallelToXZPlane3D(l) }},
		ParallelToYZKind:    {planeEncode, func(l *Line3D) Constraint { return NewParallelToYZPlane3D(l) }},
	}
	for kind, b := range builders {
		registerConstraintCodec3D(kind, orientation3DCodec(b.encode, b.build))
	}
}

func orientation3DCodec(encode func(Constraint) (Constraint3DRow, error), build func(*Line3D) Constraint) constraintCodec3D {
	return constraintCodec3D{
		encode: encode,
		decode: func(s *Sketch3D, cd Constraint3DRow, pts []*Point3D, entmap map[int]Entity) error {
			lines, err := lookupLines3D(cd.Curves, entmap)
			if err != nil {
				return err
			}
			if len(lines) != 1 {
				return fmt.Errorf("needs 1 line, got %d", len(lines))
			}
			s.geomCons.add(build(lines[0]))
			return nil
		},
	}
}

// registerCurveJoin3DConstraintCodecs covers the curve-join kinds (tangent/
// smooth/splineFitPoints/helical/bend, issue #142) — their Curves are not
// necessarily lines, so they resolve against the full entity map.
func registerCurveJoin3DConstraintCodecs() {
	registerConstraintCodec3D(TangentKind, smoothJoin3DCodec(
		func(c Constraint) (SmoothCurve3D, SmoothCurve3D, *Point3D, *Point3D) {
			v := c.(*Tangent3D)
			return v.C1, v.C2, v.P1, v.P2
		},
		func(c1, c2 SmoothCurve3D, p1, p2 *Point3D) Constraint { return NewTangent3D(c1, c2, p1, p2) }))
	registerConstraintCodec3D(SmoothKind, smoothJoin3DCodec(
		func(c Constraint) (SmoothCurve3D, SmoothCurve3D, *Point3D, *Point3D) {
			v := c.(*Smooth3D)
			return v.C1, v.C2, v.P1, v.P2
		},
		func(c1, c2 SmoothCurve3D, p1, p2 *Point3D) Constraint { return NewSmooth3D(c1, c2, p1, p2) }))
	registerConstraintCodec3D(SplineFitPointsKind, splineFit3DCodec())
	registerConstraintCodec3D(HelicalJoinKind, helical3DCodec())
	registerConstraintCodec3D(BendKind, bend3DCodec())
}

// smoothJoin3DCodec pairs a tangent/smooth join row with its factory.
func smoothJoin3DCodec(operands func(Constraint) (SmoothCurve3D, SmoothCurve3D, *Point3D, *Point3D), build func(c1, c2 SmoothCurve3D, p1, p2 *Point3D) Constraint) constraintCodec3D {
	return constraintCodec3D{
		encode: func(c Constraint) (Constraint3DRow, error) {
			c1, c2, p1, p2 := operands(c)
			return Constraint3DRow{Curves: entity3DIDPair(c1, c2), Points: []int{int(p1.id), int(p2.id)}}, nil
		},
		decode: func(s *Sketch3D, cd Constraint3DRow, pts []*Point3D, entmap map[int]Entity) error {
			if len(cd.Curves) != 2 || len(pts) != 2 {
				return fmt.Errorf("needs 2 curves + 2 points, got %d/%d", len(cd.Curves), len(pts))
			}
			c1, err := lookupSmoothCurve3D(cd.Curves[0], entmap)
			if err != nil {
				return err
			}
			c2, err := lookupSmoothCurve3D(cd.Curves[1], entmap)
			if err != nil {
				return err
			}
			s.geomCons.add(build(c1, c2, pts[0], pts[1]))
			return nil
		},
	}
}

func splineFit3DCodec() constraintCodec3D {
	return constraintCodec3D{
		encode: func(c Constraint) (Constraint3DRow, error) {
			v := c.(*SplineFitPoints3D)
			return Constraint3DRow{Curves: []int{int(v.Spline.id)}, Points: []int{int(v.P.id)}, Index: v.FitIndex}, nil
		},
		decode: func(s *Sketch3D, cd Constraint3DRow, pts []*Point3D, entmap map[int]Entity) error {
			sp, err := lookupSpline3D(cd.Curves, entmap)
			if err != nil {
				return err
			}
			if len(pts) != 1 {
				return fmt.Errorf("needs 1 fit point, got %d", len(pts))
			}
			c, err := NewSplineFitPoints3DAt(sp, pts[0], cd.Index)
			if err != nil {
				return err
			}
			s.geomCons.add(c)
			return nil
		},
	}
}

func helical3DCodec() constraintCodec3D {
	return constraintCodec3D{
		encode: func(c Constraint) (Constraint3DRow, error) {
			v := c.(*Helical3D)
			return Constraint3DRow{Curves: []int{int(v.H.id), int(v.C.id)}}, nil
		},
		decode: func(s *Sketch3D, cd Constraint3DRow, pts []*Point3D, entmap map[int]Entity) error {
			c, err := restoreHelical3D(cd, entmap)
			if err != nil {
				return err
			}
			s.geomCons.add(c)
			return nil
		},
	}
}

func bend3DCodec() constraintCodec3D {
	return constraintCodec3D{
		encode: func(c Constraint) (Constraint3DRow, error) {
			v := c.(*Bend3D)
			return Constraint3DRow{
				Curves: []int{int(v.Arc.id), int(v.L1.id), int(v.L2.id)},
				Points: []int{int(v.P1.id), int(v.P2.id)}, Radius: v.Radius,
			}, nil
		},
		decode: func(s *Sketch3D, cd Constraint3DRow, pts []*Point3D, entmap map[int]Entity) error {
			c, err := restoreBend3D(cd, pts, entmap)
			if err != nil {
				return err
			}
			s.geomCons.add(c)
			return nil
		},
	}
}
