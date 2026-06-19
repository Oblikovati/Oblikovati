// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"
	"strings"

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
func listWorkPlanes(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	host, err := activeWorkHost(s)
	if err != nil {
		return nil, err
	}
	planes := host.WorkPlanes()
	out := make([]wire.WorkPlaneInfo, planes.Count())
	for i := 0; i < planes.Count(); i++ {
		out[i] = workPlaneInfo(host, planes.Item(i), i)
	}
	return json.Marshal(wire.ListWorkPlanesResult{Planes: out})
}

// workPlaneInfo renders one plane as the wire DTO, including the editable inputs a redefine
// accepts: its scalars (offset/angle, in the document's preferred unit) and its re-pickable
// reference slots (each tagged plane|axis|point|face). Origin planes report neither.
func workPlaneInfo(host workHost, wp *feature.WorkPlane, index int) wire.WorkPlaneInfo {
	return wire.WorkPlaneInfo{
		Index:    index,
		Name:     wp.Name(),
		Ref:      string(wp.Key()),
		Origin:   point3Slice(wp.Plane().Origin()),
		Normal:   vector3Slice(wp.Plane().Normal().AsVector()),
		IsOrigin: wp.IsCoordinateSystemElement(),
		Healthy:  wp.Health().OK(),
		Reason:   wp.Health().Reason, // empty when healthy
		Kind:     wp.Kind(),
		Scalars:  workPlaneScalars(host, wp),
		Slots:    workPlaneSlots(wp),
	}
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
func redefineWorkPlane(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	host, err := activeWorkHost(s)
	if err != nil {
		return nil, err
	}
	var in wire.RedefineWorkPlaneArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	wp, err := userWorkPlane(host, in.Index)
	if err != nil {
		return nil, err
	}
	restore := wp.SnapshotDefinition() // an error mid-batch must not leave earlier edits applied
	if err := applyScalarEdits(host, wp, in.Scalars); err != nil {
		restore()
		return nil, err
	}
	if err := applyRepicks(wp, in.Repick); err != nil {
		restore()
		return nil, err
	}
	host.Recompute()
	return json.Marshal(wire.RedefineWorkPlaneResult{Plane: workPlaneInfo(host, wp, in.Index)})
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
func createWorkPlanes(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	host, err := activeWorkHost(s)
	if err != nil {
		return nil, err
	}
	var in wire.CreateWorkPlaneArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	wp, err := buildWorkPlane(host, in)
	if err != nil {
		return nil, err
	}
	host.Recompute()
	return json.Marshal(wire.CreateWorkPlaneResult{
		Index:   host.WorkPlanes().Count() - 1,
		Ref:     string(wp.Key()),
		Name:    wp.Name(),
		Healthy: wp.Health().OK(),
	})
}

// createWorkPoint adds a datum point fixed at the requested position to the active model and
// recomputes, returning its index and reference (usable as a point input to a work plane or a
// redefine re-pick).
func createWorkPoint(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	host, err := activeWorkHost(s)
	if err != nil {
		return nil, err
	}
	var in wire.CreateWorkPointArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	at, err := parseCoords(in.At, "workPoints.create: at")
	if err != nil {
		return nil, err
	}
	p := math.P3(at[0], at[1], at[2])
	wp := host.WorkPoints().AddByPosition(func() math.Point3 { return p })
	host.Recompute()
	return json.Marshal(wire.CreateWorkPointResult{
		Index: host.WorkPoints().Count() - 1,
		Ref:   string(wp.Key()),
		Name:  wp.Name(),
	})
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
// wire (e.g. a three-point plane through three created points, or a redefine re-pick).
func toWorkRef(r string) feature.WorkRef {
	if isWorkFeatureRef(r) {
		return feature.WorkRef(r)
	}
	return feature.FaceRef([]byte(r))
}

// workFeatureRefPrefixes are the reference-string prefixes that name a work feature (an origin
// element, a user plane/axis/point/coordinate-system, or an encoded vertex) rather than a raw
// B-rep face key.
var workFeatureRefPrefixes = []string{"origin/", "plane/", "axis/", "point/", "ucs/", "vertex/"}

// isWorkFeatureRef reports whether r names a work feature (so it is passed through as a plane/
// axis/point ref) rather than a raw face key.
func isWorkFeatureRef(r string) bool {
	for _, p := range workFeatureRefPrefixes {
		if strings.HasPrefix(r, p) {
			return true
		}
	}
	return false
}

// modelLengthClosure turns a distance argument into a live, parameter-aware value: a
// plain literal is constant; an expression ("h", "h/2") is backed by an auto model
// parameter, so editing the parameter and recomputing re-reads it (a work plane is
// re-evaluated by the host's Recompute). Mirrors opregistry's lengthClosure for the router
// args that feed func() float64 (work-plane offset).
func modelLengthClosure(host workHost, expr string) (func() float64, error) {
	if q, err := resolveQuantity(host, expr, param.Length); err == nil {
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
