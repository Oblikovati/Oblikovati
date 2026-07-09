// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/param"
)

// A part and an assembly both host work features over this wire surface.
var (
	_ workHost = (*compdef.PartComponentDefinition)(nil)
	_ workHost = (*compdef.AssemblyComponentDefinition)(nil)
)

// workHost is the model surface the work-feature wire methods author against — a part OR an
// assembly. Both own a work-geometry collection (origin frame + user datums), a parameter DAG,
// document units, and a recompute, so workPlanes.* / workPoints.* serve an assembly exactly as a
// part. (An assembly's offset / three-point / angle planes need no body; a tangent-to-face plane
// needs a participant face and simply reports unhealthy on an assembly — it is not rejected here.)
type workHost interface {
	WorkPlanes() *feature.WorkPlanes
	WorkPoints() *feature.WorkPoints
	WorkAxes() *feature.WorkAxes
	WorkGeometry() *feature.WorkGeometry
	Units() param.UnitsOfMeasure
	Parameters() *param.Parameters
	Recompute()
}

// activeWorkHost resolves the active document's content to a work-feature host (part or assembly),
// so the work-plane/point methods no longer reject an assembly. A document whose content is neither
// (a drawing, a presentation, a reference stub) is an error.
func activeWorkHost(s *app.Session) (workHost, error) {
	d := s.ActiveDocument()
	if d == nil {
		return nil, modelaccess.ErrNoActiveDocument
	}
	host, ok := d.Content().(workHost)
	if !ok {
		return nil, fmt.Errorf("workPlanes: active document %q is not a part or assembly", d.DisplayName())
	}
	return host, nil
}

// listWorkPlanes enumerates the active model's datum planes (origin frame first, then
// user planes) with their current geometry and health.
func listWorkPlanes(_ *app.Session, host workHost) (wire.ListWorkPlanesResult, error) {
	planes := projectLiveDatums(host.WorkPlanes(), func(i int, wp *feature.WorkPlane) wire.WorkPlaneInfo {
		return workPlaneInfo(host, wp, i)
	})
	return wire.ListWorkPlanesResult{Planes: planes}, nil
}

// workPlaneInfo renders one plane as the wire DTO, including the editable inputs a redefine
// accepts: its scalars (offset/angle, in the document's preferred unit) and its re-pickable
// reference slots (each tagged plane|axis|point|face). Origin planes report neither.
func workPlaneInfo(host workHost, wp *feature.WorkPlane, index int) wire.WorkPlaneInfo {
	return wire.WorkPlaneInfo{
		Index:        index,
		Name:         wp.Name(),
		Ref:          string(wp.Key()),
		Origin:       point3Slice(wp.Plane().Origin()),
		Normal:       vector3Slice(wp.Plane().Normal().AsVector()),
		IsOrigin:     wp.IsCoordinateSystemElement(),
		Visible:      wp.Visible(),
		Construction: wp.Construction(),
		AutoResize:   wp.AutoResize(),
		Grounded:     wp.Grounded(),
		Size:         planeSizeCorners(wp),
		Healthy:      wp.Health().OK(),
		Reason:       wp.Health().Reason, // empty when healthy
		Kind:         wp.Kind(),
		Scalars:      workPlaneScalars(host, wp),
		Slots:        workPlaneSlots(wp),
	}
}

// planeSizeCorners renders the plane's displayed rectangle as its two opposite corners (#1851).
func planeSizeCorners(wp *feature.WorkPlane) [][]float64 {
	c1, c2 := wp.Size()
	return [][]float64{point3Slice(c1), point3Slice(c2)}
}

// workPlaneScalars renders the plane's editable scalars with their value in the document's
// preferred unit (mm/deg), so a client reads the current value and can set a new one.
func workPlaneScalars(host workHost, wp *feature.WorkPlane) []wire.WorkPlaneScalar {
	scalars := wp.EditableScalars()
	if len(scalars) == 0 {
		return nil
	}
	out := make([]wire.WorkPlaneScalar, len(scalars))
	for i, sc := range scalars {
		out[i] = wire.WorkPlaneScalar{
			Index: i,
			Label: sc.Label,
			Unit:  host.Units().PreferredName(sc.Unit),
			Value: host.Units().ToPreferred(param.Q(sc.Get(), sc.Unit)),
		}
	}
	return out
}

// workPlaneSlots renders the plane's re-pickable reference slots (label + kind token).
func workPlaneSlots(wp *feature.WorkPlane) []wire.WorkPlaneRefSlot {
	slots := wp.RedefineSlots()
	if len(slots) == 0 {
		return nil
	}
	out := make([]wire.WorkPlaneRefSlot, len(slots))
	for i, sl := range slots {
		out[i] = wire.WorkPlaneRefSlot{Index: i, Label: sl.Label, Kind: sl.Kind.String()}
	}
	return out
}

// redefineWorkPlane edits a placed user work plane in place: it applies the requested scalar
// edits (offset/angle, parsed in the document's units) and reference re-picks, then recomputes
// and returns the plane's refreshed info. It fails on a bad index, an origin plane, or an
// out-of-range scalar/slot; an unsatisfiable result is reported healthy=false, not an error.
func redefineWorkPlane(_ *app.Session, host workHost, in wire.RedefineWorkPlaneArgs) (wire.RedefineWorkPlaneResult, error) {
	wp, err := userWorkPlane(host, in.Index)
	if err != nil {
		return wire.RedefineWorkPlaneResult{}, err
	}
	restore := wp.SnapshotDefinition() // an error mid-batch must not leave earlier edits applied
	if err := applyScalarEdits(host, wp, in.Scalars); err != nil {
		restore()
		return wire.RedefineWorkPlaneResult{}, err
	}
	if err := applyRepicks(wp, in.Repick); err != nil {
		restore()
		return wire.RedefineWorkPlaneResult{}, err
	}
	if err := applyDisplayEdits(wp, in); err != nil {
		restore()
		return wire.RedefineWorkPlaneResult{}, err
	}
	host.Recompute()
	return wire.RedefineWorkPlaneResult{Plane: workPlaneInfo(host, wp, in.Index)}, nil
}

// applyDisplayEdits applies the redefine's display/associativity edits — auto-resize, grounded, and
// the fixed displayed size (two corners) — to a user work plane; SetSize turns off auto-resize
// (#1851).
func applyDisplayEdits(wp *feature.WorkPlane, in wire.RedefineWorkPlaneArgs) error {
	if in.AutoResize != nil {
		wp.SetAutoResize(*in.AutoResize)
	}
	if in.Grounded != nil {
		wp.SetGrounded(*in.Grounded)
	}
	if len(in.Size) == 0 {
		return nil
	}
	if len(in.Size) != 2 {
		return fmt.Errorf("workPlanes.redefine: size needs 2 corner points, got %d", len(in.Size))
	}
	c1, err := parseCoords(in.Size[0], "workPlanes.redefine: size corner 1")
	if err != nil {
		return err
	}
	c2, err := parseCoords(in.Size[1], "workPlanes.redefine: size corner 2")
	if err != nil {
		return err
	}
	wp.SetSize(math.P3(c1[0], c1[1], c1[2]), math.P3(c2[0], c2[1], c2[2]))
	return nil
}

// flipWorkPlaneNormal reverses the normal of a placed user work plane by its index and recomputes;
// the flip is recorded on the datum and persists. It fails on a bad index or an origin plane. Like
// createWorkPlanes it self-orchestrates the work-geometry recompute (audit B1); the mutating router
// seam records the undo step / edit broadcast (#1851).
func flipWorkPlaneNormal(_ *app.Session, host workHost, in wire.FlipWorkPlaneArgs) (wire.FlipWorkPlaneResult, error) {
	wp, err := userWorkPlane(host, in.Index)
	if err != nil {
		return wire.FlipWorkPlaneResult{}, err
	}
	wp.FlipNormal()
	host.Recompute()
	return wire.FlipWorkPlaneResult{Plane: workPlaneInfo(host, wp, in.Index)}, nil
}

// userWorkPlane resolves a redefine index to a user (non-origin) work plane.
func userWorkPlane(host workHost, index int) (*feature.WorkPlane, error) {
	planes := host.WorkPlanes()
	if index < 0 || index >= planes.Count() {
		return nil, fmt.Errorf("workPlanes.redefine: index %d out of range (%d planes)", index, planes.Count())
	}
	wp := planes.Item(index)
	if wp.IsCoordinateSystemElement() {
		return nil, fmt.Errorf("workPlanes.redefine: plane %d is an origin plane and cannot be redefined", index)
	}
	return wp, nil
}

// applyScalarEdits sets the plane's scalars from unit-bearing strings, parsed by each scalar's
// quantity kind (length or angle) into the database units its Set expects.
func applyScalarEdits(host workHost, wp *feature.WorkPlane, edits []wire.ScalarEdit) error {
	scalars := wp.EditableScalars()
	for _, e := range edits {
		if e.Index < 0 || e.Index >= len(scalars) {
			return fmt.Errorf("workPlanes.redefine: scalar index %d out of range (%d scalars)", e.Index, len(scalars))
		}
		q, err := resolveQuantity(host, e.Value, scalars[e.Index].Unit)
		if err != nil {
			return fmt.Errorf("workPlanes.redefine: scalar %d value %q: %w", e.Index, e.Value, err)
		}
		scalars[e.Index].Set(q.Value)
	}
	return nil
}

// applyRepicks re-points the plane's reference slots at the requested references (origin
// constants / list refs / face keys, via toWorkRef). A reference the model refuses — one that
// would create a reference cycle, or names a work feature that does not exist — fails the call.
func applyRepicks(wp *feature.WorkPlane, repicks []wire.SlotRepick) error {
	slots := wp.RedefineSlots()
	for _, rp := range repicks {
		if rp.Slot < 0 || rp.Slot >= len(slots) {
			return fmt.Errorf("workPlanes.redefine: slot %d out of range (%d slots)", rp.Slot, len(slots))
		}
		if err := slots[rp.Slot].Set(toWorkRef(rp.Ref)); err != nil {
			return fmt.Errorf("workPlanes.redefine: slot %d: %w", rp.Slot, err)
		}
	}
	return nil
}

// createWorkPlanes adds a datum plane of the requested kind to the active model and
// recomputes. An unsatisfiable definition still creates the plane (reported healthy=false)
// — the call only fails on bad arguments (unknown kind, wrong reference count, …).
func createWorkPlanes(_ *app.Session, host workHost, in wire.CreateWorkPlaneArgs) (wire.CreateWorkPlaneResult, error) {
	wp, err := buildWorkPlane(host, in)
	if err != nil {
		return wire.CreateWorkPlaneResult{}, err
	}
	if in.Visible != nil {
		wp.SetVisible(*in.Visible) // viewport display, orthogonal to Construction
	}
	wp.SetConstruction(in.Construction) // hidden, consumer-tied datum (#1849)
	host.Recompute()
	return wire.CreateWorkPlaneResult{
		Index:   host.WorkPlanes().Count() - 1,
		Ref:     string(wp.Key()),
		Name:    wp.Name(),
		Healthy: wp.Health().OK(),
	}, nil
}

// createWorkPoint adds a datum point of the requested kind (fixed position, or the point where an
// axis pierces a plane — built by buildWorkPoint) to the active model and recomputes, returning its
// index, reference (usable as a point input to a work plane or a redefine re-pick), name, and
// health. A well-formed but unsatisfiable definition (an axis parallel to the plane) still creates
// the point, reported healthy=false; the call only fails on bad arguments.
func createWorkPoint(_ *app.Session, host workHost, in wire.CreateWorkPointArgs) (wire.CreateWorkPointResult, error) {
	wp, err := buildWorkPoint(host, in)
	if err != nil {
		return wire.CreateWorkPointResult{}, err
	}
	wp.SetConstruction(in.Construction) // hidden, consumer-tied datum (#1849)
	host.Recompute()
	return wire.CreateWorkPointResult{
		Index:   host.WorkPoints().Count() - 1,
		Ref:     string(wp.Key()),
		Name:    wp.Name(),
		Healthy: wp.Health().OK(),
		Reason:  wp.Health().Reason, // empty when healthy
	}, nil
}

// buildWorkPoint dispatches a create request to the matching model constructor. An empty kind is
// the position constructor, so the original position-only request (just "at") is unchanged.
func buildWorkPoint(host workHost, in wire.CreateWorkPointArgs) (*feature.WorkPoint, error) {
	pts := host.WorkPoints()
	switch types.WorkPointKind(in.Kind) {
	case "", types.WorkPointPosition:
		return addPositionPoint(host, in)
	case types.WorkPointPlaneAxisIntersection:
		return addPlaneAxisPoint(host, in)
	case types.WorkPointOnPoint:
		return refPoint(in, 1, func(r []feature.WorkRef) *feature.WorkPoint { return pts.AddByPoint(r[0]) })
	case types.WorkPointTwoLines:
		return refPoint(in, 2, func(r []feature.WorkRef) *feature.WorkPoint { return pts.AddByTwoLines(r[0], r[1]) })
	case types.WorkPointThreePlanes:
		return refPoint(in, 3, func(r []feature.WorkRef) *feature.WorkPoint { return pts.AddByThreePlanes(r[0], r[1], r[2]) })
	default:
		return nil, fmt.Errorf("workPoints.create: unknown kind %q (see api/types WorkPoint*)", in.Kind)
	}
}

// refPoint builds a reference-model work point from exactly n references, erroring on the wrong
// count — the shared arity guard for the on-point / two-lines / three-planes kinds.
func refPoint(in wire.CreateWorkPointArgs, n int, build func([]feature.WorkRef) *feature.WorkPoint) (*feature.WorkPoint, error) {
	refs := toWorkRefs(in.Refs)
	if len(refs) != n {
		return nil, fmt.Errorf("workPoints.create: %q needs %d references, got %d", in.Kind, n, len(refs))
	}
	return build(refs), nil
}

// addPositionPoint builds the fixed-position point from its [x, y, z] coordinate.
func addPositionPoint(host workHost, in wire.CreateWorkPointArgs) (*feature.WorkPoint, error) {
	at, err := parseCoords(in.At, "workPoints.create: at")
	if err != nil {
		return nil, err
	}
	p := math.P3(at[0], at[1], at[2])
	return host.WorkPoints().AddByPosition(func() math.Point3 { return p }), nil
}

// addPlaneAxisPoint builds the plane∩axis point from exactly two references [plane, axis].
func addPlaneAxisPoint(host workHost, in wire.CreateWorkPointArgs) (*feature.WorkPoint, error) {
	refs := toWorkRefs(in.Refs)
	if len(refs) != 2 {
		return nil, fmt.Errorf("workPoints.create: plane-axis-intersection needs 2 references [plane, axis], got %d", len(refs))
	}
	return host.WorkPoints().AddByPlaneAndAxisIntersection(refs[0], refs[1]), nil
}

// createWorkAxis adds a datum axis of the requested kind (line / two-points / plane-intersection,
// built by buildWorkAxis in work_axes.go) to the active model and recomputes, returning its index,
// reference (usable to build further datums or as a revolve axis), name, and health. It lives here
// beside createWorkPoint because it self-orchestrates the work-geometry recompute (audit B1).
func createWorkAxis(_ *app.Session, host workHost, in wire.CreateWorkAxisArgs) (wire.CreateWorkAxisResult, error) {
	wa, err := buildWorkAxis(host, in)
	if err != nil {
		return wire.CreateWorkAxisResult{}, err
	}
	wa.SetConstruction(in.Construction) // hidden, consumer-tied datum (#1849)
	host.Recompute()
	return wire.CreateWorkAxisResult{
		Index:   host.WorkAxes().Count() - 1,
		Ref:     string(wa.Key()),
		Name:    wa.Name(),
		Healthy: wa.Health().OK(),
		Reason:  wa.Health().Reason,
	}, nil
}

// deleteWorkFeature tombstones the user datum work plane, axis, or point named by the request's ref
// and recomputes, so any retained dependent re-derives and goes unhealthy. It fails on an origin
// datum, an unknown ref, or an already-deleted datum (feature.DeleteWork enforces this), and
// returns every ref removed — the named datum plus, when RetainDependents is false, its cascaded
// dependents (#1855). Like createWorkPoint it self-orchestrates the work-geometry recompute (audit
// B1); the mutating router seam records the undo step / edit broadcast.
func deleteWorkFeature(_ *app.Session, host workHost, in wire.DeleteWorkFeatureArgs) (wire.DeleteWorkFeatureResult, error) {
	removed, err := host.WorkGeometry().DeleteWork(toWorkRef(in.Ref), in.RetainDependents)
	if err != nil {
		return wire.DeleteWorkFeatureResult{}, err
	}
	host.Recompute()
	return wire.DeleteWorkFeatureResult{Deleted: workRefStrings(removed)}, nil
}

// workRefStrings renders a slice of work references as their wire string form.
func workRefStrings(refs []feature.WorkRef) []string {
	out := make([]string, len(refs))
	for i, r := range refs {
		out[i] = string(r)
	}
	return out
}

// refPlaneCtor builds a work plane from exactly its references (no scalar parameters);
// refPlaneCtors is the table of the kinds that need only references, keeping
// buildWorkPlane's body to the kinds with extra inputs (offset, angle, fixed frame).
type refPlaneCtor struct {
	arity int
	build func(*feature.WorkPlanes, []feature.WorkRef) *feature.WorkPlane
}

var refPlaneCtors = map[types.WorkPlaneKind]refPlaneCtor{
	types.WorkPlaneThreePoints: {3, func(p *feature.WorkPlanes, r []feature.WorkRef) *feature.WorkPlane {
		return p.AddByThreePoints(r[0], r[1], r[2])
	}},
	types.WorkPlanePlaneAndPoint: {2, func(p *feature.WorkPlanes, r []feature.WorkRef) *feature.WorkPlane {
		return p.AddByPlaneAndPoint(r[0], r[1])
	}},
	types.WorkPlaneTwoPlanes: {2, func(p *feature.WorkPlanes, r []feature.WorkRef) *feature.WorkPlane {
		return p.AddByTwoPlanes(r[0], r[1])
	}},
	types.WorkPlaneTwoLines: {2, func(p *feature.WorkPlanes, r []feature.WorkRef) *feature.WorkPlane {
		return p.AddByTwoLines(r[0], r[1])
	}},
	types.WorkPlaneLineAndPoint: {2, func(p *feature.WorkPlanes, r []feature.WorkRef) *feature.WorkPlane {
		return p.AddByLineAndPoint(r[0], r[1])
	}},
	types.WorkPlaneNormalToCurve: {2, func(p *feature.WorkPlanes, r []feature.WorkRef) *feature.WorkPlane {
		return p.AddByNormalToCurve(r[0], r[1])
	}},
	types.WorkPlaneTorusMidPlane: {1, func(p *feature.WorkPlanes, r []feature.WorkRef) *feature.WorkPlane { return p.AddByTorusMidPlane(r[0]) }},
	types.WorkPlanePointAndTangent: {2, func(p *feature.WorkPlanes, r []feature.WorkRef) *feature.WorkPlane {
		return p.AddByPointAndTangent(r[0], r[1])
	}},
	types.WorkPlanePlaneAndTangent: {2, func(p *feature.WorkPlanes, r []feature.WorkRef) *feature.WorkPlane {
		return p.AddByPlaneAndTangent(r[0], r[1])
	}},
	types.WorkPlaneLineAndTangent: {2, func(p *feature.WorkPlanes, r []feature.WorkRef) *feature.WorkPlane {
		return p.AddByLineAndTangent(r[0], r[1])
	}},
}

// buildWorkPlane dispatches a create request to the matching model constructor.
func buildWorkPlane(host workHost, in wire.CreateWorkPlaneArgs) (*feature.WorkPlane, error) {
	planes := host.WorkPlanes()
	refs := toWorkRefs(in.Refs)
	kind := types.WorkPlaneKind(in.Kind)
	if wp, handled, err := addSolutionPlane(planes, kind, refs, in); handled {
		return wp, err
	}
	if c, ok := refPlaneCtors[kind]; ok {
		if len(refs) != c.arity {
			return nil, fmt.Errorf("workPlanes.create: %s needs %d references, got %d", in.Kind, c.arity, len(refs))
		}
		return c.build(planes, refs), nil
	}
	switch kind {
	case types.WorkPlaneOffset:
		return addOffsetPlane(host, refs, in)
	case types.WorkPlaneLinePlaneAngle:
		return addAnglePlane(host, refs, in)
	case types.WorkPlaneFixed:
		return addFixedWorkPlane(host.WorkPlanes(), in)
	default:
		return nil, fmt.Errorf("workPlanes.create: unknown kind %q (see api/types WorkPlane*)", in.Kind)
	}
}

// addSolutionPlane handles the tangent/bisector kinds when a proximity (tangent) or quadrant
// (two-planes) point is supplied, threading it to the ...Toward constructor so the chosen side is
// recorded. It returns handled=false — so buildWorkPlane falls through to the default constructor —
// for any other kind or when no solution point is given (#1844).
func addSolutionPlane(planes *feature.WorkPlanes, kind types.WorkPlaneKind, refs []feature.WorkRef, in wire.CreateWorkPlaneArgs) (*feature.WorkPlane, bool, error) {
	var raw []float64
	switch kind {
	case types.WorkPlaneTwoPlanes:
		raw = in.Quadrant
	case types.WorkPlanePlaneAndTangent, types.WorkPlaneLineAndTangent:
		raw = in.Proximity
	default:
		return nil, false, nil
	}
	if len(raw) == 0 {
		return nil, false, nil // no solution point → default constructor
	}
	coords, err := parseCoords(raw, fmt.Sprintf("workPlanes.create: %s solution point", kind))
	if err != nil {
		return nil, true, err
	}
	if len(refs) != 2 {
		return nil, true, fmt.Errorf("workPlanes.create: %s needs 2 references, got %d", kind, len(refs))
	}
	p := math.P3(coords[0], coords[1], coords[2])
	switch kind {
	case types.WorkPlaneTwoPlanes:
		return planes.AddByTwoPlanesToward(refs[0], refs[1], p), true, nil
	case types.WorkPlanePlaneAndTangent:
		return planes.AddByPlaneAndTangentToward(refs[0], refs[1], p), true, nil
	default: // WorkPlaneLineAndTangent
		return planes.AddByLineAndTangentToward(refs[0], refs[1], p), true, nil
	}
}

// addOffsetPlane builds the plane-offset kind: one base plane reference + a distance.
func addOffsetPlane(host workHost, refs []feature.WorkRef, in wire.CreateWorkPlaneArgs) (*feature.WorkPlane, error) {
	if len(refs) != 1 {
		return nil, fmt.Errorf("workPlanes.create: %s needs 1 reference, got %d", in.Kind, len(refs))
	}
	off, err := modelLengthClosure(host, in.Offset)
	if err != nil {
		return nil, err
	}
	return host.WorkPlanes().AddByPlaneAndOffset(refs[0], off), nil
}

// addAnglePlane builds the line-plane-angle kind: a line + a plane reference + an angle.
func addAnglePlane(host workHost, refs []feature.WorkRef, in wire.CreateWorkPlaneArgs) (*feature.WorkPlane, error) {
	if len(refs) != 2 {
		return nil, fmt.Errorf("workPlanes.create: %s needs 2 references, got %d", in.Kind, len(refs))
	}
	a, err := modelAngle(host, in.Angle)
	if err != nil {
		return nil, err
	}
	return host.WorkPlanes().AddByLinePlaneAndAngle(refs[0], refs[1], func() float64 { return a }), nil
}

// addFixedWorkPlane builds an AddFixed plane from its origin point and two axis vectors.
func addFixedWorkPlane(planes *feature.WorkPlanes, in wire.CreateWorkPlaneArgs) (*feature.WorkPlane, error) {
	origin, err := parseCoords(in.Origin, "workPlanes.create: origin")
	if err != nil {
		return nil, err
	}
	x, err := parseAxisVector(in.XAxis, "workPlanes.create: xaxis")
	if err != nil {
		return nil, err
	}
	y, err := parseAxisVector(in.YAxis, "workPlanes.create: yaxis")
	if err != nil {
		return nil, err
	}
	o := math.P3(origin[0], origin[1], origin[2])
	return planes.AddFixed(func() math.Point3 { return o }, x, y), nil
}

// toWorkRefs converts the request's reference strings to model work references.
func toWorkRefs(refs []string) []feature.WorkRef {
	out := make([]feature.WorkRef, len(refs))
	for i, r := range refs {
		out[i] = toWorkRef(r)
	}
	return out
}

// toWorkRef wraps one reference string as a work ref. A work-feature reference — an origin
// constant ("origin/plane/xy", "origin/axis/z", …), a user plane/axis/point/ucs ref
// ("plane/3", "point/0" — the Ref strings create/list return), or an encoded vertex ref
// ("vertex/…") — passes through verbatim; any other string is a B-rep topology reference key
// (from model.referenceKeys) and is tagged as a FaceRef so the resolver builds the plane on
// that body face. This lets a user work point/plane/axis feed another work feature over the
// wire (e.g. a three-point plane through three created points, or a redefine re-pick). The
// classification lives in feature.ParseWorkRef so the router and the op registry share it.
func toWorkRef(r string) feature.WorkRef { return feature.ParseWorkRef(r) }

// modelLengthClosure turns a distance argument into a live, parameter-aware value. A plain
// literal ("5 mm") has no parameter dependency, so its value is baked. ANY expression that
// references parameters ("h", "h/2", "L - 2*m") must stay LIVE — it is backed by an auto model
// parameter and re-read on every recompute, so a later edit to L/h re-offsets the work plane or
// re-spaces the sketch pattern that uses it (issue #1230). The discriminator is a pure-literal
// parse: only Units().Parse (which never consults the parameter table) may bake; the previous
// resolveQuantity fast path baked parameter expressions too, freezing them.
func modelLengthClosure(host workHost, expr string) (func() float64, error) {
	if q, ok := literalLength(host, expr, param.Length); ok {
		v := q.Value
		return func() float64 { return v }, nil
	}
	p, err := host.Parameters().AddAutoModelParameter(expr)
	if err != nil {
		return nil, fmt.Errorf("workPlanes.create: offset %q: %w", expr, err)
	}
	if h := p.Health(); !h.OK() {
		return nil, fmt.Errorf("workPlanes.create: offset %q: %s", expr, h.Reason)
	}
	return func() float64 { return p.ModelValue() }, nil
}

// modelAngle parses a unit-bearing angle ("45 deg") into a model value (radians).
func modelAngle(host workHost, expr string) (float64, error) {
	q, err := resolveQuantity(host, expr, param.Angle)
	if err != nil {
		return 0, fmt.Errorf("workPlanes.create: angle %q: %w", expr, err)
	}
	return q.Value, nil
}

// parseCoords requires a 3-component coordinate slice, naming what (including the wire
// method, e.g. "workPlanes.create: origin") for errors.
func parseCoords(s []float64, what string) ([]float64, error) {
	if len(s) != 3 {
		return nil, fmt.Errorf("%s needs 3 components, got %d", what, len(s))
	}
	return s, nil
}

// parseAxisVector requires a 3-component direction and normalizes it to a unit vector.
func parseAxisVector(s []float64, what string) (math.UnitVector3, error) {
	if _, err := parseCoords(s, what); err != nil {
		return math.UnitVector3{}, err
	}
	return math.NewUnitVector3(s[0], s[1], s[2])
}

// point3Slice / vector3Slice render geometry as [x,y,z] for the wire DTOs.
func point3Slice(p math.Point3) []float64 {
	return []float64{float64(p.X), float64(p.Y), float64(p.Z)}
}

func vector3Slice(v math.Vector3) []float64 {
	return []float64{float64(v.X), float64(v.Y), float64(v.Z)}
}
