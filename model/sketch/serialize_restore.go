// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/math"
	"oblikovati.org/model/linetype"
	"oblikovati.org/model/seq"
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
		blockDefs: sc.blockDefs,
		pointMap:  make(map[int]*Point, len(sd.Points)),
		entityMap: make(map[int]Entity, len(sd.Entities)),
	}
	sc.restoreSketchID(r.s, sd.ID)
	restoreSketchProps(r.s, sd)
	r.restorePoints(sd.Points)
	if err := r.restoreEntities(sd.Entities); err != nil {
		return err
	}
	raiseIDSeq(r.maxID) // ids minted after load must not collide with the restored ones
	if err := r.restoreConstraints(sd.Constraints); err != nil {
		return err
	}
	if err := r.restoreDimensions(sd.Dimensions); err != nil {
		return err
	}
	return r.restoreCloudAnchors(sd.CloudAnchors)
}

// restoreCloudAnchors re-creates the scan-anchored point links over the already-restored points;
// the live cloud source is re-attached later by the host (RelinkCloudAnchors) (#645).
func (r *sketchRestorer) restoreCloudAnchors(anchors []CloudAnchorData) error {
	for _, a := range anchors {
		p, ok := r.pointMap[a.PointID]
		if !ok {
			return fmt.Errorf("cloud anchor references unknown point id %d", a.PointID)
		}
		r.s.RestoreCloudAnchor(p, a.CloudID, math.P3(math.Scalar(a.Local[0]), math.Scalar(a.Local[1]), math.Scalar(a.Local[2])))
	}
	return nil
}

// unflattenPoints rebuilds points from a [x,y,x,y,…] slice (odd tails are ignored).
func unflattenPoints(coords []float64) []math.Point2 {
	out := make([]math.Point2, 0, len(coords)/2)
	for i := 0; i+1 < len(coords); i += 2 {
		out = append(out, math.P2(math.Scalar(coords[i]), math.Scalar(coords[i+1])))
	}
	return out
}

// restoreSketchProps reapplies the persisted name and display/solve overrides onto a
// freshly-added sketch (sc.Add auto-named it; an empty persisted name keeps that).
func restoreSketchProps(s *Sketch, sd SketchData) {
	if sd.Name != "" {
		s.SetName(sd.Name)
	}
	s.SetVisible(!sd.Hidden)
	s.SetShared(sd.Shared)
	restoreSeq(s, sd.Seq)
	s.SetColor(sd.Color)
	s.SetLineType(sd.LineType)
	if sd.CustomLineName != "" {
		s.SetCustomLineType(linetype.Definition{Name: sd.CustomLineName, Pattern: sd.CustomLinePattern}, sd.CustomLineFile)
	}
	s.SetLineWeight(sd.LineWeight)
	s.SetDeferUpdates(sd.DeferUpdates)
}

// restoreSeq pins the sketch's creation stamp to its saved value (so reopened documents
// keep the original sketch/feature/work-feature interleaving) and raises the global clock
// past it. A legacy recipe with no stamp keeps the fresh one sc.Add assigned, which still
// orders sketches among themselves by load order.
func restoreSeq(s *Sketch, saved uint64) {
	seq.Restore(&s.seq, saved)
}

// sketchRestorer carries the id→object maps while rebuilding one sketch. maxID tracks the
// largest persisted local id pinned onto a point/entity, so the id clock can be raised
// past it once the sketch is rebuilt (#153).
type sketchRestorer struct {
	s         *Sketch
	blockDefs *BlockDefinitions
	pointMap  map[int]*Point
	entityMap map[int]Entity
	maxID     uint64
}

// idCarrier is the restore-only seam for pinning a sketch object's local id to its
// persisted value. Every entity satisfies it (curves/annotations via entityBase, points
// directly), so the same restore code path keeps all ids stable across save/load (#153).
type idCarrier interface{ setID(ID) }

// pin overrides obj's minted id with its persisted value and tracks the maximum seen.
func (r *sketchRestorer) pin(obj idCarrier, saved int) {
	obj.setID(ID(saved))
	if uint64(saved) > r.maxID {
		r.maxID = uint64(saved)
	}
}

func (r *sketchRestorer) restorePoints(points []PointData) {
	for _, pd := range points {
		r.pointMap[pd.ID] = r.restorePoint(pd)
	}
}

// restorePoint recreates one point (standalone SketchPoint or a curve-owned point) and
// pins its persisted local id.
func (r *sketchRestorer) restorePoint(pd PointData) *Point {
	pos := math.P2(pd.X, pd.Y)
	p := r.newPointFor(pd.Standalone, pos)
	r.pin(p, pd.ID)
	return p
}

// newPointFor creates a standalone SketchPoint or a curve-owned solver point, exactly one.
func (r *sketchRestorer) newPointFor(standalone bool, pos math.Point2) *Point {
	if standalone {
		return r.s.points.Add(pos)
	}
	return r.s.newPoint(pos)
}

func (r *sketchRestorer) restoreEntities(entities []EntityData) error {
	for _, ed := range entities {
		e, err := r.restoreEntity(ed)
		if err != nil {
			return err
		}
		// Projected reference entities (#1268) carry no construction flag and pin their own ids,
		// so each post-step is optional rather than a hard type-assert.
		if sc, ok := e.(interface{ SetConstruction(bool) }); ok {
			sc.SetConstruction(ed.Construction)
		}
		if ed.Centerline {
			if cl, ok := e.(interface{ SetCenterline(bool) }); ok {
				cl.SetCenterline(true)
			}
		}
		if ic, ok := e.(idCarrier); ok {
			r.pin(ic, ed.ID)
		}
		r.entityMap[ed.ID] = e
	}
	return nil
}

// restoreEntity rebuilds one entity through its kind's registered codec — the
// same pairing its serializeEntity encode came from, so the two can never
// drift (#1624). An unknown kind is a hard error: a recipe never restores
// partially.
func (r *sketchRestorer) restoreEntity(ed EntityData) (Entity, error) {
	c, ok := entityCodecs2D[EntityKind(ed.Kind)]
	if !ok {
		return nil, fmt.Errorf("unknown entity kind %q", ed.Kind)
	}
	return c.decode(r, ed)
}

// restoreProjectedPoint rebuilds a frozen projected point and registers its anchor in the point
// map so constraints referencing the anchor restore, pinning the anchor id (#1268).
func (r *sketchRestorer) restoreProjectedPoint(ed EntityData) (Entity, error) {
	if len(ed.Points) < 1 || len(ed.Anchor) < 2 {
		return nil, fmt.Errorf("projectedPoint needs an anchor id and a 2-component anchor")
	}
	pos := math.P2(math.Scalar(ed.Anchor[0]), math.Scalar(ed.Anchor[1]))
	pp := r.s.RestoreProjectedPoint(ID(ed.Points[0]), pos, ed.SourceKind, ed.Source)
	r.pointMap[ed.Points[0]] = pp.Anchor()
	r.note(ed.Points[0])
	r.note(ed.ID)
	return pp, nil
}

// note raises the restorer's max-id watermark for an id pinned outside pin() (the
// self-id'd projected entities), so later minted ids never collide with restored ones.
func (r *sketchRestorer) note(id int) {
	if uint64(id) > r.maxID {
		r.maxID = uint64(id)
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

// restoreConstraint decodes through the paired codec registry (#1625) — an
// unknown kind is a corrupt-recipe error, named honestly.
func (r *sketchRestorer) restoreConstraint(cd ConstraintData) error {
	codec, ok := constraintCodecs2D[ConstraintKind(cd.Kind)]
	if !ok {
		return fmt.Errorf("unknown constraint kind %q", cd.Kind)
	}
	return codec.decode(r, cd)
}

// restoreSplineExtras rebuilds a spline's fit method and active tangency
// handles (M06-F11, #626).
func restoreSplineExtras(s *Sketch, sp *Spline, ed EntityData) error {
	if ed.FitMethod != "" {
		m, ok := types.ParseSplineFitMethod(ed.FitMethod)
		if !ok {
			return fmt.Errorf("unknown spline fit method %q (want smooth|sweet|chord)", ed.FitMethod)
		}
		sp.FitMethod = m
	}
	for _, hd := range ed.Handles {
		h, err := s.splineHandles.Activate(sp, hd.FitIndex)
		if err != nil {
			return err
		}
		h.End.SetPosition(math.P2(math.Scalar(hd.EndX), math.Scalar(hd.EndY)))
	}
	return nil
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
		return r.restoreAdvancedDimension(dd)
	}
}

// restoreAdvancedDimension rebuilds the M21 dimension kinds (offset/three-point-angle/
// ellipse-radius); split out of restoreDimension to keep that switch small.
func (r *sketchRestorer) restoreAdvancedDimension(dd DimensionData) (*DimensionConstraint, error) {
	dc := r.s.dimCons
	switch dd.Kind {
	case "offsetDim":
		p, err := r.point(dd.Points, 0)
		if err != nil {
			return nil, err
		}
		l, err := r.line(dd.Curves, 0)
		if err != nil {
			return nil, err
		}
		return dc.AddOffsetDim(p, l, dd.Expression)
	case "threePointAngle":
		pts, err := r.points(dd.Points, 3)
		if err != nil {
			return nil, err
		}
		return dc.AddThreePointAngle(pts[0], pts[1], pts[2], dd.Expression)
	case "ellipseRadius":
		e, err := r.ellipse(dd.Curves, 0)
		if err != nil {
			return nil, err
		}
		return dc.AddEllipseRadius(e, dd.Expression)
	case "tangentDistance":
		l, err := r.line(dd.Curves, 0)
		if err != nil {
			return nil, err
		}
		c, err := r.curve(dd.Curves, 1)
		if err != nil {
			return nil, err
		}
		return dc.AddTangentDistance(l, c, dd.FarSide, dd.Expression)
	default:
		return nil, fmt.Errorf("unknown dimension kind %q", dd.Kind)
	}
}

// ellipse resolves the i-th id to a restored *Ellipse.
func (r *sketchRestorer) ellipse(ids []int, i int) (*Ellipse, error) {
	if i >= len(ids) {
		return nil, fmt.Errorf("ellipse operand %d missing", i)
	}
	e, ok := r.entityMap[ids[i]].(*Ellipse)
	if !ok {
		return nil, fmt.Errorf("entity %d is not an ellipse", ids[i])
	}
	return e, nil
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
