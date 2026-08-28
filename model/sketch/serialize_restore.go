// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"fmt"

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
	r.applyPendingOffsetDrives()
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
	s.SetHostWorkRef(sd.HostPlaneRef) // the host re-wires the live plane host; this keeps the consumer link (#1849)
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
	// pendingOffsetDrives are driven offset constraints whose driving dimension parameter is not yet
	// created when constraints restore (dimensions restore afterwards); applyPendingOffsetDrives
	// re-binds them by parameter name once the dimensions exist.
	pendingOffsetDrives []pendingOffsetDrive
}

// pendingOffsetDrive is a driven offset constraint held back until its driving dimension parameter
// exists (see sketchRestorer.pendingOffsetDrives).
type pendingOffsetDrive struct {
	l1, l2 *Line
	name   string
	sign   float64
	dist   float64 // frozen fallback when the named parameter cannot be resolved
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

// note raises the restorer's max-id watermark for an id pinned outside pin() (the
// self-id'd projected entities), so later minted ids never collide with restored ones.
func (r *sketchRestorer) note(id int) {
	if uint64(id) > r.maxID {
		r.maxID = uint64(id)
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

func (r *sketchRestorer) offsetSpline(ids []int, i int) (*OffsetSpline, error) {
	if i >= len(ids) {
		return nil, fmt.Errorf("offset-spline operand %d missing", i)
	}
	o, ok := r.entityMap[ids[i]].(*OffsetSpline)
	if !ok {
		return nil, fmt.Errorf("entity %d is not an offset spline", ids[i])
	}
	return o, nil
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

// applyPendingOffsetDrives re-binds each deferred driven offset constraint to its driving dimension's
// parameter (now that dimensions have been restored), or falls back to a frozen offset at the persisted
// distance when the parameter no longer resolves.
func (r *sketchRestorer) applyPendingOffsetDrives() {
	for _, p := range r.pendingOffsetDrives {
		if driver, ok := r.s.params.ByName(p.name); ok {
			r.s.geomCons.AddOffsetDriven(p.l1, p.l2, driver, p.sign)
			continue
		}
		r.s.geomCons.AddOffset(p.l1, p.l2, p.dist)
	}
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
	if e, ok := r.entityMap[id]; ok {
		return e, nil
	}
	// A *Point is itself an Entity (a custom tag constraint may anchor to a
	// bare point), but points persist in the Points table, not the entity
	// rows — fall back there. Surfaced by the #1625 constraintseam live test.
	if p, ok := r.pointMap[id]; ok {
		return p, nil
	}
	return nil, fmt.Errorf("unresolved entity id %d", id)
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
