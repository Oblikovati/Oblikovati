// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/math"
	"oblikovati.org/model/seq"
	"oblikovati.org/model/sketch"
)

// This file serializes a part's USER work features (planes/axes/points) into the
// recipe. The origin coordinate system is regenerated, never serialized — only its
// stable references are recorded. User features are stored in global creation order so
// a feature that references an earlier one restores after it; their definitions name
// the geometry they are built on by WorkRef (origin well-known keys, or earlier user
// features by position), which re-resolve on recompute.

// WorkFeatureData is the recipe form of one user work feature: which collection it
// belongs to, its definition kind, the references it is built on, and any parameter.
type WorkFeatureData struct {
	Collection string    `yaml:"collection"` // plane | axis | point
	Kind       string    `yaml:"kind"`
	Seq        uint64    `yaml:"seq,omitempty"` // global creation stamp; see model/seq
	Refs       []string  `yaml:"refs,omitempty"`
	Offset     float64   `yaml:"offset,omitempty"`   // plane-offset
	Angle      float64   `yaml:"angle,omitempty"`    // line-plane-angle
	Position   []float64 `yaml:"position,omitempty"` // point position / fixed-frame origin [x,y,z]
	XAxis      []float64 `yaml:"xaxis,omitempty"`    // fixed-frame X axis [x,y,z]
	YAxis      []float64 `yaml:"yaxis,omitempty"`    // fixed-frame Y axis [x,y,z]
	CloudID    string    `yaml:"cloud,omitempty"`    // point-cloud-fit: the source cloud's id (provenance, #645)
}

// MarshalWork projects the user work features into the recipe, in creation order.
func MarshalWork(g *WorkGeometry) ([]WorkFeatureData, error) {
	out := make([]WorkFeatureData, 0, len(g.userSeq))
	for i, e := range g.userSeq {
		d, err := serializeWorkFeature(g, e)
		if err != nil {
			return nil, fmt.Errorf("work feature %d (%s): %w", i, e.collection, err)
		}
		out = append(out, d)
	}
	return out, nil
}

func serializeWorkFeature(g *WorkGeometry, e userEntry) (WorkFeatureData, error) {
	d, s, err := serializeWorkDef(g, e)
	if err != nil {
		return WorkFeatureData{}, err
	}
	d.Seq = s
	return d, nil
}

// serializeWorkDef encodes the work feature's definition and returns its creation stamp,
// so MarshalWork can persist the global order shared with sketches/features (issue #132).
func serializeWorkDef(g *WorkGeometry, e userEntry) (WorkFeatureData, uint64, error) {
	switch e.collection {
	case "plane":
		w := g.planes.Item(e.index)
		d, err := serializePlaneDef(w.def)
		return d, w.seq, err
	case "axis":
		w := g.axes.Item(e.index)
		d, err := serializeAxisDef(w.def)
		return d, w.seq, err
	case "point":
		w := g.points.Item(e.index)
		d, err := serializePointDef(w.def)
		return d, w.seq, err
	default:
		return WorkFeatureData{}, 0, fmt.Errorf("unknown work collection %q", e.collection)
	}
}

func serializePlaneDef(def planeDefinition) (WorkFeatureData, error) {
	d := WorkFeatureData{Collection: "plane", Kind: def.kindName(), Refs: refStrings(def.refs())}
	switch v := def.(type) {
	case *offsetPlaneDef:
		d.Offset = v.distance() // persist the effective distance, including any browser edit
	case *fixedFramePlaneDef:
		p := v.origin()
		d.Position = []float64{float64(p.X), float64(p.Y), float64(p.Z)}
		d.XAxis, d.YAxis = unitSlice(v.x), unitSlice(v.y)
	case *pointCloudFitPlaneDef:
		// Persist the provenance link and the last good fit (the frozen frame), so the plane has
		// geometry on load even before the cloud source is re-attached (#645).
		d.CloudID = v.cloudID
		d.Position = []float64{float64(v.origin.X), float64(v.origin.Y), float64(v.origin.Z)}
		d.XAxis, d.YAxis = unitSlice(v.x), unitSlice(v.y)
	case *linePlaneAnglePlaneDef:
		d.Angle = v.angle()
	case *threePointPlaneDef, *planeAndPointPlaneDef, *twoPlanesPlaneDef, *twoLinesPlaneDef,
		*normalToCurvePlaneDef, *torusMidPlaneDef, *pointAndTangentPlaneDef,
		*planeAndTangentPlaneDef, *lineAndTangentPlaneDef:
		// references only
	default:
		return WorkFeatureData{}, fmt.Errorf("no codec for work plane definition %q", def.kindName())
	}
	return d, nil
}

func serializeAxisDef(def axisDefinition) (WorkFeatureData, error) {
	switch v := def.(type) {
	case fixedAxisDef: // grounded "line" axis: persist its origin + direction (no references)
		p := v.origin
		return WorkFeatureData{
			Collection: "axis", Kind: def.kindName(),
			Position: []float64{float64(p.X), float64(p.Y), float64(p.Z)}, XAxis: unitSlice(v.dir),
		}, nil
	case twoPointsAxisDef, planeIntersectionAxisDef:
		return WorkFeatureData{Collection: "axis", Kind: def.kindName(), Refs: refStrings(def.refs())}, nil
	default:
		return WorkFeatureData{}, fmt.Errorf("no codec for work axis definition %q", def.kindName())
	}
}

func serializePointDef(def pointDefinition) (WorkFeatureData, error) {
	d := WorkFeatureData{Collection: "point", Kind: def.kindName(), Refs: refStrings(def.refs())}
	switch v := def.(type) {
	case positionPointDef:
		p := v.at()
		d.Position = []float64{float64(p.X), float64(p.Y), float64(p.Z)}
	case *pointCloudPointDef:
		p := v.FrozenPosition() // last good model position; the source re-derives it after relink (#645)
		d.CloudID = v.cloudID
		d.Position = []float64{float64(p.X), float64(p.Y), float64(p.Z)}
	case planeAxisPointDef:
		// references only
	default:
		return WorkFeatureData{}, fmt.Errorf("no codec for work point definition %q", def.kindName())
	}
	return d, nil
}

// ApplyWork rebuilds the user work features onto g (which already holds the origin
// frame), in order, resolving each one's references as it goes.
func ApplyWork(g *WorkGeometry, data []WorkFeatureData) error {
	for i, d := range data {
		if err := restoreWorkFeature(g, d); err != nil {
			return fmt.Errorf("work feature %d (%s/%s): %w", i, d.Collection, d.Kind, err)
		}
		restoreWorkSeq(g, d)
	}
	return nil
}

// restoreWorkSeq pins the just-restored work feature's creation stamp to its saved value
// (the Add* call above gave it a fresh one), so a reopened document keeps the original
// sketch/feature/work interleaving. The restored feature is the last in its collection.
func restoreWorkSeq(g *WorkGeometry, d WorkFeatureData) {
	switch d.Collection {
	case "plane":
		seq.Restore(&g.planes.Item(g.planes.Count()-1).seq, d.Seq)
	case "axis":
		seq.Restore(&g.axes.Item(g.axes.Count()-1).seq, d.Seq)
	case "point":
		seq.Restore(&g.points.Item(g.points.Count()-1).seq, d.Seq)
	}
}

func restoreWorkFeature(g *WorkGeometry, d WorkFeatureData) error {
	switch d.Collection {
	case "plane":
		return restorePlaneFeature(g.WorkPlanes(), d)
	case "axis":
		return restoreAxisFeature(g.WorkAxes(), d)
	case "point":
		return restorePointFeature(g.WorkPoints(), d)
	default:
		return fmt.Errorf("unknown work collection %q", d.Collection)
	}
}

// restorePlaneFeature rebuilds one user work plane from its recipe. Reference-only kinds
// resolve through workRefs; fixed-frame/offset/angle kinds also carry scalar parameters,
// re-installed as closures so a recompute re-reads them.
func restorePlaneFeature(c *WorkPlanes, d WorkFeatureData) error {
	switch d.Kind {
	case "plane-offset":
		return restoreRefPlane(d, 1, func(r []WorkRef) {
			off := d.Offset
			c.AddByPlaneAndOffset(r[0], func() float64 { return off })
		})
	case "three-points":
		return restoreRefPlane(d, 3, func(r []WorkRef) { c.AddByThreePoints(r[0], r[1], r[2]) })
	case "fixed-frame":
		return restoreFixedFrame(c, d)
	case "point-cloud-fit":
		return restorePointCloudFit(c, d)
	case "plane-point":
		return restoreRefPlane(d, 2, func(r []WorkRef) { c.AddByPlaneAndPoint(r[0], r[1]) })
	case "two-planes":
		return restoreRefPlane(d, 2, func(r []WorkRef) { c.AddByTwoPlanes(r[0], r[1]) })
	case "line-plane-angle":
		return restoreRefPlane(d, 2, func(r []WorkRef) {
			ang := d.Angle
			c.AddByLinePlaneAndAngle(r[0], r[1], func() float64 { return ang })
		})
	case "two-lines":
		return restoreRefPlane(d, 2, func(r []WorkRef) { c.AddByTwoLines(r[0], r[1]) })
	case "normal-to-curve":
		return restoreRefPlane(d, 2, func(r []WorkRef) { c.AddByNormalToCurve(r[0], r[1]) })
	case "torus-midplane":
		return restoreRefPlane(d, 1, func(r []WorkRef) { c.AddByTorusMidPlane(r[0]) })
	case "point-tangent":
		return restoreRefPlane(d, 2, func(r []WorkRef) { c.AddByPointAndTangent(r[0], r[1]) })
	case "plane-tangent":
		return restoreRefPlane(d, 2, func(r []WorkRef) { c.AddByPlaneAndTangent(r[0], r[1]) })
	case "line-tangent":
		return restoreRefPlane(d, 2, func(r []WorkRef) { c.AddByLineAndTangent(r[0], r[1]) })
	default:
		return fmt.Errorf("no restore codec for work plane kind %q", d.Kind)
	}
}

// restoreRefPlane resolves d's n references and calls add with them, centralizing the
// arity check so each plane kind above stays a single line.
func restoreRefPlane(d WorkFeatureData, n int, add func([]WorkRef)) error {
	r, err := workRefs(d.Refs, n)
	if err != nil {
		return err
	}
	add(r)
	return nil
}

// restoreFixedFrame rebuilds an AddFixed plane from its origin and two in-plane axes.
func restoreFixedFrame(c *WorkPlanes, d WorkFeatureData) error {
	origin, err := point3From(d.Position, "fixed-frame origin")
	if err != nil {
		return err
	}
	x, err := unit3From(d.XAxis, "fixed-frame X axis")
	if err != nil {
		return err
	}
	y, err := unit3From(d.YAxis, "fixed-frame Y axis")
	if err != nil {
		return err
	}
	c.AddFixed(func() math.Point3 { return origin }, x, y)
	return nil
}

// restorePointCloudFit rebuilds an associative point-cloud-fit plane from its provenance id and
// last good fit (the frozen frame). The live cloud source is re-attached after load by the host
// (RelinkCloudFits), so until then the plane holds its frozen geometry (#645).
func restorePointCloudFit(c *WorkPlanes, d WorkFeatureData) error {
	origin, err := point3From(d.Position, "point-cloud-fit origin")
	if err != nil {
		return err
	}
	x, err := unit3From(d.XAxis, "point-cloud-fit X axis")
	if err != nil {
		return err
	}
	y, err := unit3From(d.YAxis, "point-cloud-fit Y axis")
	if err != nil {
		return err
	}
	c.addUser(&pointCloudFitPlaneDef{cloudID: d.CloudID, origin: origin, x: x, y: y, hasFit: true})
	return nil
}

func restoreAxisFeature(c *WorkAxes, d WorkFeatureData) error {
	if d.Kind == "line" { // grounded axis: rebuilt from its origin + direction, no references
		return restoreLineAxis(c, d)
	}
	r, err := workRefs(d.Refs, 2)
	if err != nil {
		return err
	}
	switch d.Kind {
	case "two-points":
		c.AddByTwoPoints(r[0], r[1])
	case "plane-intersection":
		c.AddByPlaneIntersection(r[0], r[1])
	default:
		return fmt.Errorf("no restore codec for work axis kind %q", d.Kind)
	}
	return nil
}

// restoreLineAxis rebuilds a grounded "line" axis from its persisted origin + direction.
func restoreLineAxis(c *WorkAxes, d WorkFeatureData) error {
	o, err := point3From(d.Position, "line axis origin")
	if err != nil {
		return err
	}
	dir, err := unit3From(d.XAxis, "line axis direction")
	if err != nil {
		return err
	}
	c.AddByLine(o, dir)
	return nil
}

func restorePointFeature(c *WorkPoints, d WorkFeatureData) error {
	switch d.Kind {
	case "position":
		pos, err := point3From(d.Position, "position point")
		if err != nil {
			return err
		}
		c.AddByPosition(func() math.Point3 { return pos })
		return nil
	case "point-cloud-point":
		pos, err := point3From(d.Position, "point-cloud point")
		if err != nil {
			return err
		}
		// The live cloud source is re-attached after load (RelinkCloudPoints); until then the point
		// holds its frozen position (#645).
		c.addUser(&pointCloudPointDef{cloudID: d.CloudID, frozen: pos, hasPos: true})
		return nil
	case "plane-axis-intersection":
		r, err := workRefs(d.Refs, 2)
		if err != nil {
			return err
		}
		c.AddByPlaneAndAxisIntersection(r[0], r[1])
		return nil
	default:
		return fmt.Errorf("no restore codec for work point kind %q", d.Kind)
	}
}

// RevolveData is a revolve's recipe: the sketch profile, the revolution axis (a
// WorkRef, typically an origin axis or a user work axis), the swept angle, and the
// boolean operation. Generation is still deferred, but the definition round-trips.
type RevolveData struct {
	Sketch     int     `yaml:"sketch"`
	Profile    int     `yaml:"profile"`
	Axis       string  `yaml:"axis,omitempty"`       // a work-axis WorkRef; empty ⇒ centerline mode
	AxisSketch int     `yaml:"axisSketch,omitempty"` // 1-based sketch index of a specific centerline (0 = none)
	AxisLine   int     `yaml:"axisLine,omitempty"`   // that centerline's line index
	Angle      float64 `yaml:"angle,omitempty"`      // 0 ⇒ full revolution
	Angle2     float64 `yaml:"angle2,omitempty"`     // second-direction sweep (#313)
	Operation  string  `yaml:"operation"`
	// ProfilePoint selects the revolved region by an interior seed point (sketch 2-D cm) rather
	// than by index, since an external author cannot predict the reader's region ordering. When
	// present it wins over Profile; absent, the index is used unchanged.
	ProfilePoint []float64 `yaml:"profilePoint,omitempty"`
}

func serializeRevolve(def *RevolveDefinition, sk SketchIndexer) (*RevolveData, error) {
	idx, ok := sk.IndexOf(def.Sketch)
	if !ok {
		return nil, fmt.Errorf("revolve references a sketch that is not in the part")
	}
	op, err := operationName(def.Operation)
	if err != nil {
		return nil, err
	}
	d := &RevolveData{Sketch: idx, Profile: def.ProfileIndex, Angle: evalFloat(def.Angle), Angle2: evalFloat(def.Angle2), Operation: op}
	switch {
	case def.Axis != nil:
		d.Axis = string(def.Axis.Key())
	case def.AxisCenterline != nil: // a specific centerline (1-based so 0 means "none")
		asi, ok := sk.IndexOf(def.AxisCenterlineSketch)
		if !ok {
			return nil, fmt.Errorf("revolve axis centerline references a sketch not in the part")
		}
		d.AxisSketch, d.AxisLine = asi+1, lineIndexOf(def.AxisCenterlineSketch, def.AxisCenterline)
	}
	return d, nil // both empty ⇒ revolve about the profile sketch's own centerline
}

// lineIndexOf returns the position of line within the sketch's line collection (-1 if absent).
func lineIndexOf(sk *sketch.Sketch, line *sketch.Line) int {
	for i := 0; i < sk.Lines().Count(); i++ {
		if sk.Lines().Item(i) == line {
			return i
		}
	}
	return -1
}

func restoreRevolve(fs *PartFeatures, d *RevolveData, sk SketchIndexer, work *WorkGeometry) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("revolve feature is missing its payload")
	}
	skt, ok := sk.At(d.Sketch)
	if !ok {
		return nil, fmt.Errorf("revolve references sketch index %d, which does not exist", d.Sketch)
	}
	op, err := parseOperation(d.Operation)
	if err != nil {
		return nil, err
	}
	angle := d.Angle
	// resolveSeed gives the fallback index; the seed is ALSO kept on the definition
	// (withProfileSeed) so it re-resolves against the current regions every recompute — the
	// sketch re-solves between load and recompute and reorders its regions (#region-seed).
	profile := resolveSeed(skt, d.ProfilePoint, d.Profile)
	if d.AxisSketch > 0 { // a specific centerline (1-based index)
		axisSk, ok := sk.At(d.AxisSketch - 1)
		if !ok {
			return nil, fmt.Errorf("revolve axis centerline references sketch index %d, which does not exist", d.AxisSketch-1)
		}
		if d.AxisLine < 0 || d.AxisLine >= axisSk.Lines().Count() {
			return nil, fmt.Errorf("revolve axis centerline references line %d out of range", d.AxisLine)
		}
		pf := NewRevolveFeatures(fs).AddAboutCenterlineLine(skt, profile, axisSk, axisSk.Lines().Item(d.AxisLine), func() float64 { return angle }, op)
		return withProfileSeed(restoreSecondAngle(pf, d.Angle2), d.ProfilePoint), nil
	}
	if d.Axis == "" { // revolve about the profile sketch's own (single) centerline
		pf := NewRevolveFeatures(fs).AddAboutCenterline(skt, profile, func() float64 { return angle }, op)
		return withProfileSeed(restoreSecondAngle(pf, d.Angle2), d.ProfilePoint), nil
	}
	if work == nil {
		return nil, fmt.Errorf("revolve needs the part's work geometry to resolve its axis")
	}
	axis, err := work.axis(WorkRef(d.Axis))
	if err != nil {
		return nil, fmt.Errorf("revolve axis: %w", err)
	}
	if d.Angle2 > 0 {
		angle2 := d.Angle2
		return withProfileSeed(NewRevolveFeatures(fs).AddTwoDirectional(skt, profile, axis,
			func() float64 { return angle }, func() float64 { return angle2 }, op), d.ProfilePoint), nil
	}
	return withProfileSeed(NewRevolveFeatures(fs).Add(skt, profile, axis, func() float64 { return angle }, op), d.ProfilePoint), nil
}

// withProfileSeed records the interior seed point on a restored revolve so its region is
// re-resolved by containment each recompute (survives the sketch re-solving — #region-seed).
func withProfileSeed(pf *PartFeature, seed []float64) *PartFeature {
	if len(seed) > 0 {
		if rf, ok := pf.feature.(*RevolveFeature); ok {
			rf.def.ProfileSeed = append([]float64(nil), seed...)
		}
	}
	return pf
}

// restoreSecondAngle re-applies a persisted second-direction sweep onto a
// freshly restored revolve (the centerline Add paths take one angle).
func restoreSecondAngle(pf *PartFeature, angle2 float64) *PartFeature {
	if angle2 <= 0 {
		return pf
	}
	if rf, ok := pf.feature.(*RevolveFeature); ok {
		rf.def.Angle2 = func() float64 { return angle2 }
	}
	return pf
}

// CoilData is a coil's recipe: the sketch profile, the helix axis (a WorkRef), the
// pitch (per revolution), the number of revolutions, the taper, and the operation.
type CoilData struct {
	Sketch      int     `yaml:"sketch"`
	Profile     int     `yaml:"profile"`
	Axis        string  `yaml:"axis"`
	Pitch       float64 `yaml:"pitch,omitempty"`
	Revolutions float64 `yaml:"revolutions,omitempty"`
	Height      float64 `yaml:"height,omitempty"` // two-of-three shape spec (#316)
	Taper       float64 `yaml:"taper,omitempty"`
	Operation   string  `yaml:"operation"`
	// Variable-pitch rail + end conditions (M06-F09, #624).
	PitchRows []CoilPitchRowData `yaml:"pitchRows,omitempty"`
	StartEnd  *CoilEndData       `yaml:"startEnd,omitempty"`
	EndEnd    *CoilEndData       `yaml:"endEnd,omitempty"`
}

// CoilPitchRowData is one persisted pitch station.
type CoilPitchRowData struct {
	Pitch      float64 `yaml:"pitch"`
	Revolution float64 `yaml:"revolution"`
}

// CoilEndData is one persisted flat-end condition (radians).
type CoilEndData struct {
	TransitionAngle float64 `yaml:"transitionAngle,omitempty"`
	FlatAngle       float64 `yaml:"flatAngle,omitempty"`
}

func serializeCoil(def *CoilDefinition, sk SketchIndexer) (*CoilData, error) {
	idx, ok := sk.IndexOf(def.Sketch)
	if !ok {
		return nil, fmt.Errorf("coil references a sketch that is not in the part")
	}
	if def.Axis == nil {
		return nil, fmt.Errorf("coil has no axis")
	}
	op, err := operationName(def.Operation)
	if err != nil {
		return nil, err
	}
	d := &CoilData{
		Sketch: idx, Profile: def.ProfileIndex, Axis: string(def.Axis.Key()),
		Pitch: evalFloat(def.Pitch), Revolutions: evalFloat(def.Revolutions),
		Height: evalFloat(def.Height), Taper: def.Taper, Operation: op,
	}
	for _, r := range def.PitchRows {
		d.PitchRows = append(d.PitchRows, CoilPitchRowData(r))
	}
	d.StartEnd = coilEndData(def.StartEnd)
	d.EndEnd = coilEndData(def.EndEnd)
	return d, nil
}

// coilEndData persists a flat end (nil for natural).
func coilEndData(c CoilEndCondition) *CoilEndData {
	if !c.Flat {
		return nil
	}
	return &CoilEndData{TransitionAngle: c.TransitionAngle, FlatAngle: c.FlatAngle}
}

func restoreCoil(fs *PartFeatures, d *CoilData, sk SketchIndexer, work *WorkGeometry) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("coil feature is missing its payload")
	}
	skt, ok := sk.At(d.Sketch)
	if !ok {
		return nil, fmt.Errorf("coil references sketch index %d, which does not exist", d.Sketch)
	}
	if work == nil {
		return nil, fmt.Errorf("coil needs the part's work geometry to resolve its axis")
	}
	axis, err := work.axis(WorkRef(d.Axis))
	if err != nil {
		return nil, fmt.Errorf("coil axis: %w", err)
	}
	op, err := parseOperation(d.Operation)
	if err != nil {
		return nil, err
	}
	def := &CoilDefinition{
		Sketch: skt, ProfileIndex: d.Profile, Axis: axis,
		Pitch: constFloat(d.Pitch), Revolutions: constFloat(d.Revolutions),
		Height: constFloat(d.Height), Taper: d.Taper, Operation: op,
	}
	pf := NewCoilFeatures(fs).AddDefinition(def)
	for _, r := range d.PitchRows {
		def.PitchRows = append(def.PitchRows, CoilPitchRow(r))
	}
	def.StartEnd = coilEndFromData(d.StartEnd)
	def.EndEnd = coilEndFromData(d.EndEnd)
	return pf, nil
}

// coilEndFromData rebuilds a persisted flat end (nil stays natural).
func coilEndFromData(d *CoilEndData) CoilEndCondition {
	if d == nil {
		return CoilEndCondition{}
	}
	return CoilEndCondition{Flat: true, TransitionAngle: d.TransitionAngle, FlatAngle: d.FlatAngle}
}

// refStrings renders work references as their string form for YAML.
func refStrings(refs []WorkRef) []string {
	if len(refs) == 0 {
		return nil
	}
	out := make([]string, len(refs))
	for i, r := range refs {
		out[i] = string(r)
	}
	return out
}

// workRefs requires exactly n references, converting them from their string form.
func workRefs(refs []string, n int) ([]WorkRef, error) {
	if len(refs) != n {
		return nil, fmt.Errorf("expected %d references, got %d", n, len(refs))
	}
	out := make([]WorkRef, n)
	for i, r := range refs {
		out[i] = WorkRef(r)
	}
	return out, nil
}

// unitSlice renders a unit vector as its [x,y,z] components for YAML.
func unitSlice(u math.UnitVector3) []float64 {
	v := u.AsVector()
	return []float64{float64(v.X), float64(v.Y), float64(v.Z)}
}

// point3From reads a 3-component coordinate slice into a point, naming what for errors.
func point3From(s []float64, what string) (math.Point3, error) {
	if len(s) != 3 {
		return math.Point3{}, fmt.Errorf("%s needs 3 coordinates, got %d", what, len(s))
	}
	return math.P3(s[0], s[1], s[2]), nil
}

// unit3From reads a 3-component slice into a unit vector (erroring on a zero vector).
func unit3From(s []float64, what string) (math.UnitVector3, error) {
	if len(s) != 3 {
		return math.UnitVector3{}, fmt.Errorf("%s needs 3 components, got %d", what, len(s))
	}
	return math.NewUnitVector3(s[0], s[1], s[2])
}
