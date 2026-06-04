// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"fmt"

	"github.com/Oblikovati/oblikovati/math"
)

// ApplyRecipe rebuilds the sketches from their serialized form, in order. It is the
// inverse of [Sketches.MarshalRecipe]: points are recreated first (preserving shared
// identity, so a corner shared by two lines is one point again), then curve entities
// referencing them, then geometric and dimensional constraints. Any operand that does
// not resolve, or any unknown kind, is an error — a recipe never restores partially.
func (sc *Sketches) ApplyRecipe(data []SketchData) error {
	for i, sd := range data {
		if err := restoreSketch(sc, sd); err != nil {
			return fmt.Errorf("sketch %d: %w", i, err)
		}
	}
	return nil
}

func restoreSketch(sc *Sketches, sd SketchData) error {
	plane, err := restorePlane(sd.Plane)
	if err != nil {
		return err
	}
	r := &sketchRestorer{
		s:         sc.Add(plane),
		pointMap:  make(map[int]*Point, len(sd.Points)),
		entityMap: make(map[int]Entity, len(sd.Entities)),
	}
	restoreSketchProps(r.s, sd)
	r.restorePoints(sd.Points)
	if err := r.restoreEntities(sd.Entities); err != nil {
		return err
	}
	if err := r.restoreConstraints(sd.Constraints); err != nil {
		return err
	}
	return r.restoreDimensions(sd.Dimensions)
}

// restoreSketchProps reapplies the persisted name and display/solve overrides onto a
// freshly-added sketch (sc.Add auto-named it; an empty persisted name keeps that).
func restoreSketchProps(s *Sketch, sd SketchData) {
	if sd.Name != "" {
		s.SetName(sd.Name)
	}
	s.SetVisible(!sd.Hidden)
	s.SetColor(sd.Color)
	s.SetLineType(sd.LineType)
	s.SetLineWeight(sd.LineWeight)
	s.SetDeferUpdates(sd.DeferUpdates)
}

// sketchRestorer carries the id→object maps while rebuilding one sketch.
type sketchRestorer struct {
	s         *Sketch
	pointMap  map[int]*Point
	entityMap map[int]Entity
}

func (r *sketchRestorer) restorePoints(points []PointData) {
	for _, pd := range points {
		pos := math.P2(pd.X, pd.Y)
		if pd.Standalone {
			r.pointMap[pd.ID] = r.s.points.Add(pos)
			continue
		}
		r.pointMap[pd.ID] = r.s.newPoint(pos)
	}
}

func (r *sketchRestorer) restoreEntities(entities []EntityData) error {
	for _, ed := range entities {
		e, err := r.restoreEntity(ed)
		if err != nil {
			return err
		}
		e.(interface{ SetConstruction(bool) }).SetConstruction(ed.Construction)
		r.entityMap[ed.ID] = e
	}
	return nil
}

func (r *sketchRestorer) restoreEntity(ed EntityData) (Entity, error) {
	switch ed.Kind {
	case "line":
		p, err := r.points(ed.Points, 2)
		if err != nil {
			return nil, err
		}
		return r.s.lines.Add(p[0], p[1]), nil
	case "circle":
		p, err := r.points(ed.Points, 1)
		if err != nil {
			return nil, err
		}
		return r.s.circles.Add(p[0], math.Scalar(ed.Radius)), nil
	case "arc":
		p, err := r.points(ed.Points, 3)
		if err != nil {
			return nil, err
		}
		return r.s.arcs.Add(p[0], p[1], p[2], ed.CCW), nil
	case "ellipse":
		p, err := r.points(ed.Points, 1)
		if err != nil {
			return nil, err
		}
		if len(ed.MajorAxis) != 2 {
			return nil, fmt.Errorf("ellipse needs a 2-component majorAxis, got %d", len(ed.MajorAxis))
		}
		axis := math.V2(ed.MajorAxis[0], ed.MajorAxis[1])
		return r.s.ellipses.AddWithCenter(p[0], axis, math.Scalar(ed.MajorRadius), math.Scalar(ed.MinorRadius)), nil
	case "ellipticalArc":
		p, err := r.points(ed.Points, 1)
		if err != nil {
			return nil, err
		}
		if len(ed.MajorAxis) != 2 {
			return nil, fmt.Errorf("ellipticalArc needs a 2-component majorAxis, got %d", len(ed.MajorAxis))
		}
		axis := math.V2(ed.MajorAxis[0], ed.MajorAxis[1])
		return r.s.ellArcs.AddWithCenter(p[0], axis, math.Scalar(ed.MajorRadius), math.Scalar(ed.MinorRadius), math.Scalar(ed.StartAngle), math.Scalar(ed.EndAngle)), nil
	case "spline":
		p, err := r.points(ed.Points, len(ed.Points))
		if err != nil {
			return nil, err
		}
		return r.s.splines.AddWithPoints(p, ed.Closed, ed.Fit), nil
	case "image":
		if len(ed.Anchor) != 2 || len(ed.Size) != 2 {
			return nil, fmt.Errorf("image needs a 2-component anchor and size")
		}
		anchor := math.P2(math.Scalar(ed.Anchor[0]), math.Scalar(ed.Anchor[1]))
		return r.s.images.Add(ed.ImageRef, anchor, math.Scalar(ed.Size[0]), math.Scalar(ed.Size[1]), math.Scalar(ed.Rotation), ed.Opacity), nil
	case "fillRegion":
		if len(ed.Seed) != 2 {
			return nil, fmt.Errorf("fillRegion needs a 2-component seed")
		}
		return r.s.fills.Add(math.P2(math.Scalar(ed.Seed[0]), math.Scalar(ed.Seed[1])), ed.Style), nil
	case "text":
		if len(ed.Anchor) != 2 {
			return nil, fmt.Errorf("text needs a 2-component anchor")
		}
		anchor := math.P2(math.Scalar(ed.Anchor[0]), math.Scalar(ed.Anchor[1]))
		return r.s.texts.Add(anchor, ed.Text, math.Scalar(ed.TextHeight), math.Scalar(ed.Rotation), TextHJustify(ed.Justify)), nil
	default:
		return nil, fmt.Errorf("unknown entity kind %q", ed.Kind)
	}
}

func (r *sketchRestorer) restoreConstraints(constraints []ConstraintData) error {
	for _, cd := range constraints {
		if err := r.restoreConstraint(cd); err != nil {
			return fmt.Errorf("constraint %q: %w", cd.Kind, err)
		}
	}
	return nil
}

func (r *sketchRestorer) restoreConstraint(cd ConstraintData) error {
	g := r.s.geomCons
	switch cd.Kind {
	case "coincident":
		return r.twoPoints(cd, func(a, b *Point) { g.AddCoincident(a, b) })
	case "horizontal":
		return r.twoPoints(cd, func(a, b *Point) { g.AddHorizontal(a, b) })
	case "vertical":
		return r.twoPoints(cd, func(a, b *Point) { g.AddVertical(a, b) })
	case "parallel":
		return r.twoLines(cd, func(a, b *Line) { g.AddParallel(a, b) })
	case "perpendicular":
		return r.twoLines(cd, func(a, b *Line) { g.AddPerpendicular(a, b) })
	case "collinear":
		return r.twoLines(cd, func(a, b *Line) { g.AddCollinear(a, b) })
	case "equalLength":
		return r.twoLines(cd, func(a, b *Line) { g.AddEqualLength(a, b) })
	case "concentric":
		return r.twoCurves(cd, func(a, b CircularCurve) { g.AddConcentric(a, b) })
	case "equalRadius":
		return r.twoCurves(cd, func(a, b CircularCurve) { g.AddEqualRadius(a, b) })
	case "circularTangent":
		return r.twoCurves(cd, func(a, b CircularCurve) { g.AddCircularTangent(a, b) })
	case "pointOnLine":
		return r.pointAndLine(cd, func(p *Point, l *Line) { g.AddPointOnLine(p, l) })
	case "midpoint":
		return r.pointAndLine(cd, func(p *Point, l *Line) { g.AddMidpoint(p, l) })
	case "pointOnCircle":
		p, err := r.point(cd.Points, 0)
		if err != nil {
			return err
		}
		c, err := r.curve(cd.Curves, 0)
		if err != nil {
			return err
		}
		g.AddPointOnCircle(p, c)
		return nil
	case "tangent":
		l, err := r.line(cd.Curves, 0)
		if err != nil {
			return err
		}
		c, err := r.curve(cd.Curves, 1)
		if err != nil {
			return err
		}
		g.AddTangent(l, c)
		return nil
	case "symmetry":
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
		g.AddSymmetry(a, b, about)
		return nil
	case "fix":
		p, err := r.point(cd.Points, 0)
		if err != nil {
			return err
		}
		g.AddFix(p)
		return nil
	case "smooth":
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
		g.AddSmooth(c1, c2, p1, p2)
		return nil
	default:
		return fmt.Errorf("unknown constraint kind %q", cd.Kind)
	}
}

func (r *sketchRestorer) restoreDimensions(dims []DimensionData) error {
	for _, dd := range dims {
		d, err := r.restoreDimension(dd)
		if err != nil {
			return fmt.Errorf("dimension %q: %w", dd.Kind, err)
		}
		if dd.Driven {
			d.SetDriven(true)
		}
		if dd.Limits != nil {
			d.SetLimits(dd.Limits.Min, dd.Limits.Max)
		}
	}
	return nil
}

func (r *sketchRestorer) restoreDimension(dd DimensionData) (*DimensionConstraint, error) {
	dc := r.s.dimCons
	switch dd.Kind {
	case "distance":
		a, err := r.point(dd.Points, 0)
		if err != nil {
			return nil, err
		}
		b, err := r.point(dd.Points, 1)
		if err != nil {
			return nil, err
		}
		return dc.AddDistance(a, b, dd.Expression)
	case "radius":
		c, err := r.circle(dd.Curves, 0)
		if err != nil {
			return nil, err
		}
		return dc.AddRadius(c, dd.Expression)
	case "diameter":
		c, err := r.circle(dd.Curves, 0)
		if err != nil {
			return nil, err
		}
		return dc.AddDiameter(c, dd.Expression)
	case "angle":
		l1, err := r.line(dd.Curves, 0)
		if err != nil {
			return nil, err
		}
		l2, err := r.line(dd.Curves, 1)
		if err != nil {
			return nil, err
		}
		return dc.AddAngle(l1, l2, dd.Expression)
	case "arcLength":
		a, err := r.arc(dd.Curves, 0)
		if err != nil {
			return nil, err
		}
		return dc.AddArcLength(a, dd.Expression)
	default:
		return nil, fmt.Errorf("unknown dimension kind %q", dd.Kind)
	}
}

// --- operand resolution -------------------------------------------------------------

// twoPoints/twoLines/twoCurves/pointAndLine resolve the common operand shapes and
// invoke the constraint factory, keeping restoreConstraint flat.
func (r *sketchRestorer) twoPoints(cd ConstraintData, add func(a, b *Point)) error {
	p, err := r.points(cd.Points, 2)
	if err != nil {
		return err
	}
	add(p[0], p[1])
	return nil
}

func (r *sketchRestorer) twoLines(cd ConstraintData, add func(a, b *Line)) error {
	a, err := r.line(cd.Curves, 0)
	if err != nil {
		return err
	}
	b, err := r.line(cd.Curves, 1)
	if err != nil {
		return err
	}
	add(a, b)
	return nil
}

func (r *sketchRestorer) twoCurves(cd ConstraintData, add func(a, b CircularCurve)) error {
	a, err := r.curve(cd.Curves, 0)
	if err != nil {
		return err
	}
	b, err := r.curve(cd.Curves, 1)
	if err != nil {
		return err
	}
	add(a, b)
	return nil
}

func (r *sketchRestorer) pointAndLine(cd ConstraintData, add func(p *Point, l *Line)) error {
	p, err := r.point(cd.Points, 0)
	if err != nil {
		return err
	}
	l, err := r.line(cd.Curves, 0)
	if err != nil {
		return err
	}
	add(p, l)
	return nil
}

// points resolves exactly n point operands.
func (r *sketchRestorer) points(ids []int, n int) ([]*Point, error) {
	if len(ids) != n {
		return nil, fmt.Errorf("expected %d point operands, got %d", n, len(ids))
	}
	out := make([]*Point, n)
	for i := range ids {
		p, err := r.point(ids, i)
		if err != nil {
			return nil, err
		}
		out[i] = p
	}
	return out, nil
}

func (r *sketchRestorer) point(ids []int, i int) (*Point, error) {
	id, err := at(ids, i, "point")
	if err != nil {
		return nil, err
	}
	p, ok := r.pointMap[id]
	if !ok {
		return nil, fmt.Errorf("unresolved point id %d", id)
	}
	return p, nil
}

func (r *sketchRestorer) entity(ids []int, i int) (Entity, error) {
	id, err := at(ids, i, "entity")
	if err != nil {
		return nil, err
	}
	e, ok := r.entityMap[id]
	if !ok {
		return nil, fmt.Errorf("unresolved entity id %d", id)
	}
	return e, nil
}

func (r *sketchRestorer) line(ids []int, i int) (*Line, error) {
	e, err := r.entity(ids, i)
	if err != nil {
		return nil, err
	}
	l, ok := e.(*Line)
	if !ok {
		return nil, fmt.Errorf("entity %d is %T, want a line", ids[i], e)
	}
	return l, nil
}

func (r *sketchRestorer) circle(ids []int, i int) (*Circle, error) {
	e, err := r.entity(ids, i)
	if err != nil {
		return nil, err
	}
	c, ok := e.(*Circle)
	if !ok {
		return nil, fmt.Errorf("entity %d is %T, want a circle", ids[i], e)
	}
	return c, nil
}

func (r *sketchRestorer) arc(ids []int, i int) (*Arc, error) {
	e, err := r.entity(ids, i)
	if err != nil {
		return nil, err
	}
	a, ok := e.(*Arc)
	if !ok {
		return nil, fmt.Errorf("entity %d is %T, want an arc", ids[i], e)
	}
	return a, nil
}

func (r *sketchRestorer) curve(ids []int, i int) (CircularCurve, error) {
	e, err := r.entity(ids, i)
	if err != nil {
		return nil, err
	}
	c, ok := e.(CircularCurve)
	if !ok {
		return nil, fmt.Errorf("entity %d is %T, want a circular curve", ids[i], e)
	}
	return c, nil
}

func (r *sketchRestorer) smooth(ids []int, i int) (SmoothCurve, error) {
	e, err := r.entity(ids, i)
	if err != nil {
		return nil, err
	}
	c, ok := e.(SmoothCurve)
	if !ok {
		return nil, fmt.Errorf("entity %d is %T, want a smooth curve", ids[i], e)
	}
	return c, nil
}

// at returns ids[i] or a descriptive error when the operand is missing.
func at(ids []int, i int, what string) (int, error) {
	if i >= len(ids) {
		return 0, fmt.Errorf("missing %s operand %d (have %d)", what, i, len(ids))
	}
	return ids[i], nil
}

// restorePlane rebuilds a sketch plane from its serialized origin and axes.
func restorePlane(pd PlaneData) (Plane, error) {
	x, err := math.UnitVector3FromVector(vector3(pd.XAxis))
	if err != nil {
		return Plane{}, fmt.Errorf("plane x-axis: %w", err)
	}
	y, err := math.UnitVector3FromVector(vector3(pd.YAxis))
	if err != nil {
		return Plane{}, fmt.Errorf("plane y-axis: %w", err)
	}
	return NewPlane(math.P3(pd.Origin[0], pd.Origin[1], pd.Origin[2]), x, y)
}
