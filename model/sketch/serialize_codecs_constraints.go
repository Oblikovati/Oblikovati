// SPDX-License-Identifier: GPL-2.0-only

package sketch

// Codec registrations for the 2D geometric constraints. Bodies are the former
// switch cases of serializeConstraint and restoreConstraint/
// restoreExtraConstraint, paired per kind (#1625, audit I2) — including the
// default: branch that used to fail a save at runtime for a kind missing from
// the encode switch.

func init() {
	registerTwoPointConstraintCodecs()
	registerSingleLineConstraintCodecs()
	registerTwoLineConstraintCodecs()
	registerTwoCurveConstraintCodecs()
	registerMixedConstraintCodecs()
	registerEntitySymmetryConstraintCodecs()
	registerEllipseAxisConstraintCodecs()
	registerTagConstraintCodecs()
}

// registerSingleLineConstraintCodecs covers the single-line horizontal/vertical forms (#1871):
// each stores the one line id in Curves and re-adds it through its factory.
func registerSingleLineConstraintCodecs() {
	registerConstraintCodec(SingleLineHorizontalKind, singleLineConstraintCodec(
		func(c Constraint) *Line { return c.(*SingleLineHorizontalConstraint).L },
		func(g *GeometricConstraints, l *Line) { g.AddLineHorizontal(l) }))
	registerConstraintCodec(SingleLineVerticalKind, singleLineConstraintCodec(
		func(c Constraint) *Line { return c.(*SingleLineVerticalConstraint).L },
		func(g *GeometricConstraints, l *Line) { g.AddLineVertical(l) }))
}

// singleLineConstraintCodec pairs a one-line constraint row with its factory.
func singleLineConstraintCodec(lineOf func(Constraint) *Line, add func(*GeometricConstraints, *Line)) constraintCodec {
	return constraintCodec{
		encode: func(c Constraint) (ConstraintData, error) {
			return ConstraintData{Curves: []int{int(lineOf(c).id)}}, nil
		},
		decode: func(r *sketchRestorer, cd ConstraintData) error {
			l, err := r.line(cd.Curves, 0)
			if err != nil {
				return err
			}
			add(r.s.geomCons, l)
			return nil
		},
	}
}

// twoPointConstraintCodec pairs a two-point constraint row with its factory.
func twoPointConstraintCodec(pts func(Constraint) (*Point, *Point), add func(*GeometricConstraints, *Point, *Point)) constraintCodec {
	return constraintCodec{
		encode: func(c Constraint) (ConstraintData, error) {
			a, b := pts(c)
			return ConstraintData{Points: []int{int(a.id), int(b.id)}}, nil
		},
		decode: func(r *sketchRestorer, cd ConstraintData) error {
			return r.twoPoints(cd, func(a, b *Point) { add(r.s.geomCons, a, b) })
		},
	}
}

// twoLineConstraintCodec pairs a two-line constraint row with its factory.
func twoLineConstraintCodec(lines func(Constraint) (*Line, *Line), add func(*GeometricConstraints, *Line, *Line)) constraintCodec {
	return constraintCodec{
		encode: func(c Constraint) (ConstraintData, error) {
			a, b := lines(c)
			return ConstraintData{Curves: []int{int(a.id), int(b.id)}}, nil
		},
		decode: func(r *sketchRestorer, cd ConstraintData) error {
			return r.twoLines(cd, func(a, b *Line) { add(r.s.geomCons, a, b) })
		},
	}
}

// twoCurveConstraintCodec pairs a two-circular-curve constraint row with its factory.
func twoCurveConstraintCodec(curves func(Constraint) (CircularCurve, CircularCurve), add func(*GeometricConstraints, CircularCurve, CircularCurve)) constraintCodec {
	return constraintCodec{
		encode: func(c Constraint) (ConstraintData, error) {
			a, b := curves(c)
			return ConstraintData{Curves: []int{int(a.EntityID()), int(b.EntityID())}}, nil
		},
		decode: func(r *sketchRestorer, cd ConstraintData) error {
			return r.twoCurves(cd, func(a, b CircularCurve) { add(r.s.geomCons, a, b) })
		},
	}
}

// pointLineConstraintCodec pairs a point-on-line-shaped row with its factory.
func pointLineConstraintCodec(ops func(Constraint) (*Point, *Line), add func(*GeometricConstraints, *Point, *Line)) constraintCodec {
	return constraintCodec{
		encode: func(c Constraint) (ConstraintData, error) {
			p, l := ops(c)
			return ConstraintData{Points: []int{int(p.id)}, Curves: []int{int(l.id)}}, nil
		},
		decode: func(r *sketchRestorer, cd ConstraintData) error {
			return r.pointAndLine(cd, func(p *Point, l *Line) { add(r.s.geomCons, p, l) })
		},
	}
}

func registerTwoPointConstraintCodecs() {
	registerConstraintCodec(CoincidentKind, twoPointConstraintCodec(
		func(c Constraint) (*Point, *Point) { v := c.(*CoincidentConstraint); return v.A, v.B },
		func(g *GeometricConstraints, a, b *Point) { g.AddCoincident(a, b) }))
	registerConstraintCodec(HorizontalKind, twoPointConstraintCodec(
		func(c Constraint) (*Point, *Point) { v := c.(*HorizontalConstraint); return v.A, v.B },
		func(g *GeometricConstraints, a, b *Point) { g.AddHorizontal(a, b) }))
	registerConstraintCodec(VerticalKind, twoPointConstraintCodec(
		func(c Constraint) (*Point, *Point) { v := c.(*VerticalConstraint); return v.A, v.B },
		func(g *GeometricConstraints, a, b *Point) { g.AddVertical(a, b) }))
	registerConstraintCodec(PatternLinkKind, twoPointConstraintCodec(
		func(c Constraint) (*Point, *Point) { v := c.(*PatternConstraint); return v.Seed, v.Member },
		func(g *GeometricConstraints, a, b *Point) { g.AddPatternLink(a, b) }))
}

func registerTwoLineConstraintCodecs() {
	registerConstraintCodec(ParallelKind, twoLineConstraintCodec(
		func(c Constraint) (*Line, *Line) { v := c.(*ParallelConstraint); return v.L1, v.L2 },
		func(g *GeometricConstraints, a, b *Line) { g.AddParallel(a, b) }))
	registerConstraintCodec(PerpendicularKind, twoLineConstraintCodec(
		func(c Constraint) (*Line, *Line) { v := c.(*PerpendicularConstraint); return v.L1, v.L2 },
		func(g *GeometricConstraints, a, b *Line) { g.AddPerpendicular(a, b) }))
	registerConstraintCodec(CollinearKind, twoLineConstraintCodec(
		func(c Constraint) (*Line, *Line) { v := c.(*CollinearConstraint); return v.L1, v.L2 },
		func(g *GeometricConstraints, a, b *Line) { g.AddCollinear(a, b) }))
	registerConstraintCodec(EqualLengthKind, twoLineConstraintCodec(
		func(c Constraint) (*Line, *Line) { v := c.(*EqualLengthConstraint); return v.L1, v.L2 },
		func(g *GeometricConstraints, a, b *Line) { g.AddEqualLength(a, b) }))
}

func registerTwoCurveConstraintCodecs() {
	registerConstraintCodec(ConcentricKind, twoCurveConstraintCodec(
		func(c Constraint) (CircularCurve, CircularCurve) { v := c.(*ConcentricConstraint); return v.C1, v.C2 },
		func(g *GeometricConstraints, a, b CircularCurve) { g.AddConcentric(a, b) }))
	registerConstraintCodec(EqualRadiusKind, twoCurveConstraintCodec(
		func(c Constraint) (CircularCurve, CircularCurve) { v := c.(*EqualRadiusConstraint); return v.C1, v.C2 },
		func(g *GeometricConstraints, a, b CircularCurve) { g.AddEqualRadius(a, b) }))
	registerConstraintCodec(CircularTangentKind, twoCurveConstraintCodec(
		func(c Constraint) (CircularCurve, CircularCurve) {
			v := c.(*CircularTangentConstraint)
			return v.C1, v.C2
		},
		func(g *GeometricConstraints, a, b CircularCurve) { g.AddCircularTangent(a, b) }))
}

func registerMixedConstraintCodecs() {
	registerConstraintCodec(PointOnLineKind, pointLineConstraintCodec(
		func(c Constraint) (*Point, *Line) { v := c.(*PointOnLineConstraint); return v.P, v.L },
		func(g *GeometricConstraints, p *Point, l *Line) { g.AddPointOnLine(p, l) }))
	registerConstraintCodec(MidpointKind, pointLineConstraintCodec(
		func(c Constraint) (*Point, *Line) { v := c.(*MidpointConstraint); return v.P, v.L },
		func(g *GeometricConstraints, p *Point, l *Line) { g.AddMidpoint(p, l) }))
	registerConstraintCodec(ArcMidpointKind, arcMidpointCodec())
	registerConstraintCodec(PointOnCircleKind, pointOnCircleCodec())
	registerConstraintCodec(TangentKind, tangentCodec())
	registerConstraintCodec(SymmetryKind, symmetryCodec())
	registerConstraintCodec(FixKind, fixCodec())
	registerConstraintCodec(SmoothKind, smoothCodec())
}

func pointOnCircleCodec() constraintCodec {
	return constraintCodec{
		encode: func(c Constraint) (ConstraintData, error) {
			v := c.(*PointOnCircleConstraint)
			return ConstraintData{Points: []int{int(v.P.id)}, Curves: []int{int(v.C.EntityID())}}, nil
		},
		decode: func(r *sketchRestorer, cd ConstraintData) error {
			p, err := r.point(cd.Points, 0)
			if err != nil {
				return err
			}
			c, err := r.curve(cd.Curves, 0)
			if err != nil {
				return err
			}
			r.s.geomCons.AddPointOnCircle(p, c)
			return nil
		},
	}
}

func tangentCodec() constraintCodec {
	return constraintCodec{
		encode: func(c Constraint) (ConstraintData, error) {
			v := c.(*TangentConstraint)
			return ConstraintData{Curves: []int{int(v.L.id), int(v.C.EntityID())}}, nil
		},
		decode: func(r *sketchRestorer, cd ConstraintData) error {
			l, err := r.line(cd.Curves, 0)
			if err != nil {
				return err
			}
			c, err := r.curve(cd.Curves, 1)
			if err != nil {
				return err
			}
			r.s.geomCons.AddTangent(l, c)
			return nil
		},
	}
}

func symmetryCodec() constraintCodec {
	return constraintCodec{
		encode: func(c Constraint) (ConstraintData, error) {
			v := c.(*SymmetryConstraint)
			return ConstraintData{Points: []int{int(v.A.id), int(v.B.id)}, Curves: []int{int(v.About.id)}}, nil
		},
		decode: func(r *sketchRestorer, cd ConstraintData) error {
			a, err := r.point(cd.Points, 0)
			if err != nil {
				return err
			}
			b, err := r.point(cd.Points, 1)
			if err != nil {
				return err
			}
			about, err := r.line(cd.Curves, 0)
			if err != nil {
				return err
			}
			r.s.geomCons.AddSymmetry(a, b, about)
			return nil
		},
	}
}

// registerEntitySymmetryConstraintCodecs covers the entity-symmetry kinds (#1870): line and
// circular symmetry both store their two operands plus the mirror line in Curves, and re-derive
// the endpoint pairing from the restored geometry (like CircularTangent's internal flag), so no
// flag needs to persist.
func registerEntitySymmetryConstraintCodecs() {
	registerConstraintCodec(LineSymmetryKind, lineSymmetryCodec())
	registerConstraintCodec(CircularSymmetryKind, circularSymmetryCodec())
}

func lineSymmetryCodec() constraintCodec {
	return constraintCodec{
		encode: func(c Constraint) (ConstraintData, error) {
			v := c.(*LineSymmetryConstraint)
			return ConstraintData{Curves: []int{int(v.L1.id), int(v.L2.id), int(v.About.id)}}, nil
		},
		decode: func(r *sketchRestorer, cd ConstraintData) error {
			return decodeThreeLines(r, cd, func(l1, l2, about *Line) {
				r.s.geomCons.AddLineSymmetry(l1, l2, about)
			})
		},
	}
}

// decodeThreeLines resolves the three line ids a line-symmetry row stores (two operands + the
// mirror axis) and applies the factory.
func decodeThreeLines(r *sketchRestorer, cd ConstraintData, add func(l1, l2, about *Line)) error {
	l1, err := r.line(cd.Curves, 0)
	if err != nil {
		return err
	}
	l2, err := r.line(cd.Curves, 1)
	if err != nil {
		return err
	}
	about, err := r.line(cd.Curves, 2)
	if err != nil {
		return err
	}
	add(l1, l2, about)
	return nil
}

func circularSymmetryCodec() constraintCodec {
	return constraintCodec{
		encode: func(c Constraint) (ConstraintData, error) {
			v := c.(*CircularSymmetryConstraint)
			return ConstraintData{Curves: []int{int(v.C1.EntityID()), int(v.C2.EntityID()), int(v.About.id)}}, nil
		},
		decode: func(r *sketchRestorer, cd ConstraintData) error {
			c1, err := r.curve(cd.Curves, 0)
			if err != nil {
				return err
			}
			c2, err := r.curve(cd.Curves, 1)
			if err != nil {
				return err
			}
			about, err := r.line(cd.Curves, 2)
			if err != nil {
				return err
			}
			r.s.geomCons.AddCircularSymmetry(c1, c2, about)
			return nil
		},
	}
}

// arcMidpointCodec stores the point + arc; decode re-seeds the point to the arc midpoint (the
// restored point is already there, so it is idempotent) and re-adds the constraint.
func arcMidpointCodec() constraintCodec {
	return constraintCodec{
		encode: func(c Constraint) (ConstraintData, error) {
			v := c.(*ArcMidpointConstraint)
			return ConstraintData{Points: []int{int(v.P.id)}, Curves: []int{int(v.A.id)}}, nil
		},
		decode: func(r *sketchRestorer, cd ConstraintData) error {
			p, err := r.point(cd.Points, 0)
			if err != nil {
				return err
			}
			a, err := r.arc(cd.Curves, 0)
			if err != nil {
				return err
			}
			r.s.geomCons.AddMidpointToArc(p, a)
			return nil
		},
	}
}

func fixCodec() constraintCodec {
	return constraintCodec{
		encode: func(c Constraint) (ConstraintData, error) {
			v := c.(*FixConstraint)
			return ConstraintData{Points: []int{int(v.P.id)}}, nil
		},
		decode: func(r *sketchRestorer, cd ConstraintData) error {
			p, err := r.point(cd.Points, 0)
			if err != nil {
				return err
			}
			r.s.geomCons.AddFix(p)
			return nil
		},
	}
}

func smoothCodec() constraintCodec {
	return constraintCodec{
		encode: func(c Constraint) (ConstraintData, error) {
			v := c.(*SmoothConstraint)
			return ConstraintData{
				Points: []int{int(v.P1.id), int(v.P2.id)},
				Curves: []int{int(v.C1.EntityID()), int(v.C2.EntityID())},
			}, nil
		},
		decode: decodeSmooth,
	}
}

func decodeSmooth(r *sketchRestorer, cd ConstraintData) error {
	c1, err := r.smooth(cd.Curves, 0)
	if err != nil {
		return err
	}
	c2, err := r.smooth(cd.Curves, 1)
	if err != nil {
		return err
	}
	p1, err := r.point(cd.Points, 0)
	if err != nil {
		return err
	}
	p2, err := r.point(cd.Points, 1)
	if err != nil {
		return err
	}
	r.s.geomCons.AddSmooth(c1, c2, p1, p2)
	return nil
}

// registerTagConstraintCodecs covers the M21 constraints (ground/offset) and
// the M06-F11 tag constraints (text-box anchor, custom).
func registerTagConstraintCodecs() {
	registerConstraintCodec(GroundKind, groundCodec())
	registerConstraintCodec(OffsetKind, offsetCodec())
	registerConstraintCodec(TextBoxAnchorKind, textBoxAnchorCodec())
	registerConstraintCodec(CustomKind, customCodec())
}

func groundCodec() constraintCodec {
	return constraintCodec{
		encode: func(c Constraint) (ConstraintData, error) {
			return ConstraintData{Points: pointIDsOf(c.(*GroundConstraint).pts)}, nil
		},
		decode: func(r *sketchRestorer, cd ConstraintData) error {
			pts, err := r.points(cd.Points, len(cd.Points))
			if err != nil {
				return err
			}
			r.s.geomCons.AddGroundPoints(pts...)
			return nil
		},
	}
}

func offsetCodec() constraintCodec {
	return constraintCodec{
		encode: func(c Constraint) (ConstraintData, error) {
			v := c.(*OffsetConstraint)
			return ConstraintData{Curves: []int{int(v.L1.id), int(v.L2.id)}, Value: v.Dist}, nil
		},
		decode: func(r *sketchRestorer, cd ConstraintData) error {
			return r.twoLines(cd, func(a, b *Line) { r.s.geomCons.AddOffset(a, b, cd.Value) })
		},
	}
}

func textBoxAnchorCodec() constraintCodec {
	return constraintCodec{
		encode: func(c Constraint) (ConstraintData, error) {
			return ConstraintData{Curves: []int{int(c.(*TextBoxAnchorConstraint).Text.id)}}, nil
		},
		// The anchor record is auto-created with its text box; restoring it
		// explicitly would duplicate it, so the persisted row is a no-op.
		decode: func(r *sketchRestorer, cd ConstraintData) error { return nil },
	}
}

func customCodec() constraintCodec {
	return constraintCodec{
		encode: func(c Constraint) (ConstraintData, error) {
			v := c.(*CustomConstraint)
			return ConstraintData{ClientID: v.ClientID, Name: v.Name, Curves: entityIDsOf(v.Entities)}, nil
		},
		decode: func(r *sketchRestorer, cd ConstraintData) error {
			ents := make([]Entity, 0, len(cd.Curves))
			for i := range cd.Curves {
				e, err := r.entity(cd.Curves, i)
				if err != nil {
					return err
				}
				ents = append(ents, e)
			}
			_, err := r.s.geomCons.AddCustom(cd.ClientID, cd.Name, ents)
			return err
		},
	}
}
