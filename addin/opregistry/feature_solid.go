// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"errors"
	"fmt"

	"oblikovati.org/api/wire/featureargs"
	"oblikovati.org/app"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// The additive sketch-profile solid features beyond extrude: revolve, rib, emboss, coil, and
// loft. Each consumes one or more closed/open sketch profiles (by sketchIndex/profileIndex)
// and follows the extrude descriptor shape, so add_feature can drive the whole additive set.

// --- revolve ---------------------------------------------------------------

const revolveSchema = `{
  "type": "object",
  "properties": {
    "sketchIndex": {"type": "integer", "minimum": 0},
    "profileIndex": {"type": "integer", "minimum": 0, "default": 0},
    "axisRef": {"type": "string", "description": "Work-axis reference to revolve about, e.g. \"origin/axis/y\" (default). See get_reference_keys / list_work_planes."},
    "angle": {"type": "string", "description": "Revolve angle with units, e.g. \"360 deg\"."},
    "angle2": {"type": "string", "description": "Optional second-direction sweep (opposite sense), e.g. \"30 deg\" — the two-directional revolve."},
    "operation": {"type": "string", "enum": ["new", "join", "cut", "intersect", "surface"], "default": "new", "description": "Boolean against existing bodies, or \"surface\" to revolve the profile into an open surface-of-revolution (sheet) body — Inventor's kSurfaceOperation."},
    "profileSeed": {"type": "array", "items": {"type": "number"}, "minItems": 2, "maxItems": 2, "description": "Select the revolved region by an interior seed point [x,y] (sketch 2-D cm) instead of profileIndex — resolved by containment on the solved sketch every recompute; wins over profileIndex."}
  },
  "required": ["sketchIndex", "angle"]
}`

func revolveDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: featureargs.KindRevolve, Summary: "Revolve a closed sketch profile about an axis into a solid.", Schema: json.RawMessage(revolveSchema), Apply: applyRevolve}
}

func applyRevolve(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeFeatureArgs[featureargs.Revolve](s, raw)
	if err != nil {
		return nil, err
	}
	sk, err := sketchAt(part, in.SketchIndex)
	if err != nil {
		return nil, err
	}
	axis, err := axisFromRef(part, in.AxisRef)
	if err != nil {
		return nil, err
	}
	angle, err := angleClosure(part, in.Angle, "revolve: angle")
	if err != nil {
		return nil, err
	}
	op, err := parseOperation(in.Operation)
	if err != nil {
		return nil, err
	}
	if in.Angle2 != "" {
		angle2, err := angleClosure(part, in.Angle2, "revolve: angle2")
		if err != nil {
			return nil, err
		}
		pf := feature.NewRevolveFeatures(part.Features()).AddTwoDirectional(sk, in.ProfileIndex, axis, angle, angle2, op)
		setRevolveProfileSeed(pf, in.ProfileSeed)
		return recomputeResult(part, pf)
	}
	pf := feature.NewRevolveFeatures(part.Features()).Add(sk, in.ProfileIndex, axis, angle, op)
	setRevolveProfileSeed(pf, in.ProfileSeed)
	return recomputeResult(part, pf)
}

// setRevolveProfileSeed records an interior seed point on the revolve so its region resolves by
// containment on the solved sketch every recompute (the stable selector for an external author).
func setRevolveProfileSeed(pf *feature.PartFeature, seed []float64) {
	if len(seed) == 0 {
		return
	}
	pf.Definition().(*feature.RevolveFeature).Definition().ProfileSeed = append([]float64(nil), seed...)
}

// --- rib -------------------------------------------------------------------

const ribSchema = `{
  "type": "object",
  "properties": {
    "sketchIndex": {"type": "integer", "minimum": 0},
    "profileIndex": {"type": "integer", "minimum": 0, "default": 0},
    "thickness": {"type": "string", "description": "Rib wall thickness, e.g. \"2 mm\"."},
    "depth": {"type": "string", "description": "How far the rib grows toward the body, e.g. \"10 mm\" (sign picks the direction; omit with toNext)."},
    "toNext": {"type": "boolean", "default": false, "description": "Extend the wall until it fully lands on the existing material (the to-next rib)."},
    "operation": {"type": "string", "enum": ["new", "join"], "default": "join"}
  },
  "required": ["sketchIndex", "thickness"]
}`

func ribDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: featureargs.KindRib, Summary: "Thicken an open sketch profile into a support rib.", Schema: json.RawMessage(ribSchema), Apply: applyRib}
}

func applyRib(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeFeatureArgs[featureargs.Rib](s, raw)
	if err != nil {
		return nil, err
	}
	sk, err := sketchAt(part, in.SketchIndex)
	if err != nil {
		return nil, err
	}
	th, err := lengthClosure(part, in.Thickness, "rib: thickness")
	if err != nil {
		return nil, err
	}
	var depth func() float64
	if in.Depth != "" {
		if depth, err = lengthClosure(part, in.Depth, "rib: depth"); err != nil {
			return nil, err
		}
	} else if !in.ToNext {
		return nil, errors.New("rib: give depth or toNext")
	}
	op, err := parseOperation(in.Operation)
	if err != nil {
		return nil, err
	}
	def := &feature.RibDefinition{
		Sketch: sk, ProfileIndex: in.ProfileIndex,
		Thickness: th, Depth: depth, ToNext: in.ToNext, Operation: op,
	}
	pf := feature.NewRibFeatures(part.Features()).AddDefinition(def)
	return recomputeResult(part, pf)
}

// --- emboss ----------------------------------------------------------------

const embossSchema = `{
  "type": "object",
  "properties": {
    "sketchIndex": {"type": "integer", "minimum": 0},
    "profileIndices": {"type": "array", "items": {"type": "integer", "minimum": 0}, "description": "Profiles to emboss; omit to use profileIndex."},
    "profileIndex": {"type": "integer", "minimum": 0, "default": 0},
    "textEntity": {"type": "integer", "minimum": 1, "description": "Sketch text entity id to emboss BY REFERENCE; takes precedence over profile indices (text geometry is derived, never baked)."},
    "depth": {"type": "string", "description": "Raise (or, with engrave, cut) depth, e.g. \"1 mm\"."},
    "engrave": {"type": "boolean", "default": false, "description": "Cut into the face instead of raising from it."}
  },
  "required": ["sketchIndex", "depth"]
}`

func embossDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: featureargs.KindEmboss, Summary: "Raise or engrave a sketch profile on a face.", Schema: json.RawMessage(embossSchema), Apply: applyEmboss}
}

func applyEmboss(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeFeatureArgs[featureargs.Emboss](s, raw)
	if err != nil {
		return nil, err
	}
	sk, err := sketchAt(part, in.SketchIndex)
	if err != nil {
		return nil, err
	}
	depth, err := lengthClosure(part, in.Depth, "emboss: depth")
	if err != nil {
		return nil, err
	}
	pf, err := buildEmboss(part, sk, in, depth)
	if err != nil {
		return nil, err
	}
	return recomputeResult(part, pf)
}

// buildEmboss adds either a by-reference text emboss (when textEntity is set) or a
// profile-region emboss, so a text emboss never bakes glyph geometry into the document.
func buildEmboss(part *compdef.PartComponentDefinition, sk *sketch.Sketch, in featureargs.Emboss, depth func() float64) (*feature.PartFeature, error) {
	embs := feature.NewEmbossFeatures(part.Features())
	if in.TextEntity != 0 {
		e, ok := sk.EntityByID(sketch.ID(in.TextEntity))
		if !ok {
			return nil, fmt.Errorf("emboss: no entity with id %d", in.TextEntity)
		}
		tb, ok := e.(*sketch.TextBox)
		if !ok {
			return nil, fmt.Errorf("emboss: entity %d is a %T, not a text box", in.TextEntity, e)
		}
		return embs.AddText(sk, tb, depth, in.Engrave, 0), nil
	}
	profiles := in.ProfileIndices
	if len(profiles) == 0 {
		profiles = []int{in.ProfileIndex}
	}
	return embs.Add(sk, profiles, depth, in.Engrave, 0), nil
}

// --- coil ------------------------------------------------------------------

const coilSchema = `{
  "type": "object",
  "properties": {
    "sketchIndex": {"type": "integer", "minimum": 0},
    "profileIndex": {"type": "integer", "minimum": 0, "default": 0},
    "axisRef": {"type": "string", "description": "Work-axis reference to coil about, e.g. \"origin/axis/z\" (default)."},
    "pitch": {"type": "string", "description": "Axial rise per revolution, e.g. \"5 mm\". Give exactly two of pitch/revolutions/height."},
    "revolutions": {"type": "string", "description": "Number of turns, e.g. \"4\"."},
    "height": {"type": "string", "description": "Total axial rise, e.g. \"30 mm\" — combines with pitch OR revolutions."},
    "taper": {"type": "string", "description": "Optional taper angle, e.g. \"5 deg\" — the helix radius grows with height."},
    "operation": {"type": "string", "enum": ["new", "join", "cut"], "default": "new"},
    "startTransitionAngle": {"type": "string", "description": "Spring start-end transition sweep (pitch winds down to zero), e.g. \"90 deg\". Grounds/flattens the coil start."},
    "startFlatAngle": {"type": "string", "description": "Spring start-end flat sweep (zero pitch) after the transition, e.g. \"180 deg\"."},
    "endTransitionAngle": {"type": "string", "description": "Spring end transition sweep (pitch winds down to zero), e.g. \"90 deg\"."},
    "endFlatAngle": {"type": "string", "description": "Spring end flat sweep (zero pitch) after the transition, e.g. \"180 deg\"."}
  },
  "required": ["sketchIndex"]
}`

func coilDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: featureargs.KindCoil, Summary: "Sweep a profile along a helix into a spring/thread.", Schema: json.RawMessage(coilSchema), Apply: applyCoil}
}

func applyCoil(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeFeatureArgs[featureargs.Coil](s, raw)
	if err != nil {
		return nil, err
	}
	sk, err := sketchAt(part, in.SketchIndex)
	if err != nil {
		return nil, err
	}
	axis, err := coilAxis(part, in.AxisRef)
	if err != nil {
		return nil, err
	}
	pitch, revs, height, err := coilShapeArgs(part, in)
	if err != nil {
		return nil, err
	}
	taper, err := optionalAngleClosure(part, in.Taper, "coil: taper")
	if err != nil {
		return nil, err
	}
	op, err := parseOperation(in.Operation)
	if err != nil {
		return nil, err
	}
	startEnd, err := coilEndCondition(part, in.StartTransitionAngle, in.StartFlatAngle, "coil: start")
	if err != nil {
		return nil, err
	}
	endEnd, err := coilEndCondition(part, in.EndTransitionAngle, in.EndFlatAngle, "coil: end")
	if err != nil {
		return nil, err
	}
	def := &feature.CoilDefinition{
		Sketch: sk, ProfileIndex: in.ProfileIndex, Axis: axis,
		Pitch: pitch, Revolutions: revs, Height: height,
		Taper: callOrZeroF(taper), Operation: op,
		StartEnd: startEnd, EndEnd: endEnd,
	}
	pf := feature.NewCoilFeatures(part.Features()).AddDefinition(def)
	return recomputeResult(part, pf)
}

// coilEndCondition parses one spring end's transition + flat sweep angles into a CoilEndCondition
// (radians). It is active (Flat) only when at least one angle is given; the transition sweeps the
// pitch down to zero and the flat sweep then holds zero pitch (a ground spring end). #1883.
func coilEndCondition(part *compdef.PartComponentDefinition, transition, flat, ctx string) (feature.CoilEndCondition, error) {
	if transition == "" && flat == "" {
		return feature.CoilEndCondition{}, nil
	}
	tf, err := optionalAngleClosure(part, transition, ctx+": transitionAngle")
	if err != nil {
		return feature.CoilEndCondition{}, err
	}
	ff, err := optionalAngleClosure(part, flat, ctx+": flatAngle")
	if err != nil {
		return feature.CoilEndCondition{}, err
	}
	return feature.CoilEndCondition{Flat: true, TransitionAngle: callOrZeroF(tf), FlatAngle: callOrZeroF(ff)}, nil
}

// callOrZeroF evaluates an optional closure (nil ⇒ 0).
func callOrZeroF(f func() float64) float64 {
	if f == nil {
		return 0
	}
	return f()
}

// coilAxis resolves the coil's work axis, defaulting to the Z origin axis when no ref is given
// (a coil's natural axis), unlike the revolve default of Y.
func coilAxis(part *compdef.PartComponentDefinition, ref string) (*feature.WorkAxis, error) {
	if ref == "" {
		ref = string(feature.OriginZAxis)
	}
	return axisFromRef(part, ref)
}

// coilShapeArgs resolves the two-of-three coil shape spec (#316): pitch and
// height are unit-bearing lengths, revolutions a plain number; absent fields
// stay nil (the model validates the combination).
func coilShapeArgs(part *compdef.PartComponentDefinition, in featureargs.Coil) (pitch, revs, height func() float64, err error) {
	if in.Pitch != "" {
		if pitch, err = lengthClosure(part, in.Pitch, "coil: pitch"); err != nil {
			return nil, nil, nil, err
		}
	}
	if in.Revolutions != "" {
		if revs, err = numberClosure(part, in.Revolutions, "coil: revolutions"); err != nil {
			return nil, nil, nil, err
		}
	}
	if in.Height != "" {
		if height, err = lengthClosure(part, in.Height, "coil: height"); err != nil {
			return nil, nil, nil, err
		}
	}
	return pitch, revs, height, nil
}

// --- loft ------------------------------------------------------------------

const loftSchema = `{
  "type": "object",
  "properties": {
    "sections": {"type": "array", "minItems": 2, "items": {"type": "object", "properties": {"sketchIndex": {"type": "integer"}, "profileIndex": {"type": "integer"}, "point": {"type": "array", "items": {"type": "number"}, "minItems": 2, "maxItems": 2, "description": "[x,y] on the sketch plane → an apex (point) section so the loft tapers to a tip; only valid first or last."}, "faceRef": {"type": "string", "description": "A body-face reference key (get_reference_keys) → a face section the loft can leave Tangent/Smooth."}}, "required": ["sketchIndex"]}, "description": "Ordered cross-section profiles (>= 2) to loft through."},
    "closed": {"type": "boolean", "default": false},
    "operation": {"type": "string", "enum": ["new", "join", "cut"], "default": "new"},
    "rails": {"type": "array", "items": {"type": "object", "properties": {"pathSketchIndex": {"type": "integer"}, "pathIndex": {"type": "integer", "default": 0}}, "required": ["pathSketchIndex"]}, "description": "Optional guide rails: open paths the loft surface follows between sections (each touches the end sections)."},
    "centerline": {"type": "object", "properties": {"pathSketchIndex": {"type": "integer"}, "pathIndex": {"type": "integer", "default": 0}}, "required": ["pathSketchIndex"], "description": "Optional spine path the section centroids follow (the loft bends along it); mutually exclusive with rails."},
    "areaGraph": {"type": "array", "items": {"type": "object", "properties": {"t": {"type": "number"}, "scale": {"type": "number"}}, "required": ["t", "scale"]}, "description": "Optional cross-section area graph: stops {t (0..1 along the loft), scale} the section areas follow (ends pinned to 1)."},
    "mapCurves": {"type": "array", "items": {"type": "object", "properties": {"pathSketchIndex": {"type": "integer"}, "pathIndex": {"type": "integer", "default": 0}}, "required": ["pathSketchIndex"]}, "description": "Optional explicit point correspondence (MapPointCurves): each path gives one anchor point per section, overriding the automatic min-twist alignment."},
    "first": {"type": "object", "description": "Start-section condition (curves the loft).", "properties": {"condition": {"type": "string", "enum": ["free", "angle", "direction", "sharp", "tangent-to-plane", "tangent", "smooth", "g3"], "default": "free"}, "angle": {"type": "string", "description": "Takeoff angle to the section plane (angle/direction), e.g. \"45 deg\"."}, "impact": {"type": "number", "description": "Takeoff weight (default 1); larger curves more."}, "reversed": {"type": "boolean", "description": "Flip the takeoff through the section plane (undercut/dish)."}}},
    "last": {"type": "object", "description": "End-section condition (curves the loft).", "properties": {"condition": {"type": "string", "enum": ["free", "angle", "direction", "sharp", "tangent-to-plane", "tangent", "smooth", "g3"], "default": "free"}, "angle": {"type": "string", "description": "Takeoff angle to the section plane (angle/direction), e.g. \"45 deg\"."}, "impact": {"type": "number", "description": "Takeoff weight (default 1); larger curves more."}, "reversed": {"type": "boolean", "description": "Flip the takeoff through the section plane (undercut/dish)."}}}
  },
  "required": ["sections"]
}`

func loftDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: featureargs.KindLoft, Summary: "Blend two or more profiles into a solid (loft).", Schema: json.RawMessage(loftSchema), Apply: applyLoft}
}

func applyLoft(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeFeatureArgs[featureargs.Loft](s, raw)
	if err != nil {
		return nil, err
	}
	if len(in.Sections) < 2 {
		return nil, errors.New("loft: needs >= 2 sections")
	}
	sections := make([]feature.LoftSection, len(in.Sections))
	for i, r := range in.Sections {
		if r.FaceRef != "" { // a body-face section (loft tangent to that face) needs no sketch
			sections[i] = feature.LoftSection{FaceKey: []byte(r.FaceRef)}
			continue
		}
		sk, serr := sketchAt(part, r.SketchIndex)
		if serr != nil {
			return nil, serr
		}
		ls := feature.LoftSection{Sketch: sk, ProfileIndex: r.ProfileIndex}
		if len(r.Point) == 2 { // an apex section: [x,y] on the sketch plane
			mp := sk.Plane().ToModel(math.P2(r.Point[0], r.Point[1]))
			ls.Point = &mp
		}
		sections[i] = ls
	}
	op, err := parseOperation(in.Operation)
	if err != nil {
		return nil, err
	}
	first, err := loftEndProvider(part, in.First, "first")
	if err != nil {
		return nil, err
	}
	last, err := loftEndProvider(part, in.Last, "last")
	if err != nil {
		return nil, err
	}
	liveEnds := func() (feature.LoftEnd, feature.LoftEnd) { return first(), last() }
	rails, err := loftRailProviders(part, in.Rails)
	if err != nil {
		return nil, err
	}
	centerline, err := loftCenterlineProvider(part, in.Centerline)
	if err != nil {
		return nil, err
	}
	mapCurves, err := loftRailProviders(part, in.MapCurves)
	if err != nil {
		return nil, err
	}
	guides := feature.LoftGuideSet{Rails: rails, Centerline: centerline, AreaGraph: areaStops(in.AreaGraph), MapCurves: mapCurves}
	pf := feature.NewLoftFeatures(part.Features()).AddConditionedLiveGuided(sections, in.Closed, op, liveEnds, guides)
	return recomputeResult(part, pf)
}

// areaStops converts the area-graph args into model area stops.
func areaStops(args []featureargs.LoftAreaStop) []feature.LoftAreaStop {
	if len(args) == 0 {
		return nil
	}
	out := make([]feature.LoftAreaStop, len(args))
	for i, a := range args {
		out[i] = feature.LoftAreaStop{T: a.T, Scale: a.Scale}
	}
	return out
}

// loftCenterlineProvider validates the centerline path up front (nil when absent) and returns a
// live polyline provider, like the rails.
func loftCenterlineProvider(part *compdef.PartComponentDefinition, ref *featureargs.LoftRail) (func() []math.Point3, error) {
	if ref == nil {
		return nil, nil
	}
	prov, err := loftGuideProvider(part, *ref, "loft centerline")
	if err != nil {
		return nil, err
	}
	return prov, nil
}

// loftGuideProvider turns one guide ref into a live polyline provider: explicit Points when given,
// otherwise the sketch path (validated up front).
func loftGuideProvider(part *compdef.PartComponentDefinition, r featureargs.LoftRail, what string) (func() []math.Point3, error) {
	if len(r.Points) > 0 {
		pts := make([]math.Point3, 0, len(r.Points))
		for _, p := range r.Points {
			if len(p) != 3 {
				return nil, fmt.Errorf("%s: each point needs [x,y,z], got %v", what, p)
			}
			pts = append(pts, math.P3(p[0], p[1], p[2]))
		}
		return func() []math.Point3 { return pts }, nil
	}
	if _, err := pathFromSketch(part, r.PathSketchIndex, r.PathIndex); err != nil {
		return nil, fmt.Errorf("%s: %w", what, err)
	}
	return func() []math.Point3 {
		p, err := pathFromSketch(part, r.PathSketchIndex, r.PathIndex)
		if err != nil || p == nil {
			return nil
		}
		return p.Points()
	}, nil
}

// loftRailProviders returns a live polyline provider per ref (paths or explicit points).
func loftRailProviders(part *compdef.PartComponentDefinition, refs []featureargs.LoftRail) ([]func() []math.Point3, error) {
	var rails []func() []math.Point3
	for _, rr := range refs {
		prov, err := loftGuideProvider(part, rr, "loft rail")
		if err != nil {
			return nil, err
		}
		rails = append(rails, prov)
	}
	return rails, nil
}

// loftEndProvider turns an end-condition arg into a live LoftEnd provider (re-read each recompute
// so a parameter-driven angle reshapes the loft). A nil/free condition yields a Free end; the
// face/point conditions are not yet supported and are rejected with the offending value.
func loftEndProvider(part *compdef.PartComponentDefinition, a *featureargs.LoftEnd, which string) (func() feature.LoftEnd, error) {
	if a == nil || feature.LoftCondition(a.Condition).IsFree() {
		return func() feature.LoftEnd { return feature.LoftEnd{} }, nil
	}
	cond := feature.LoftCondition(a.Condition)
	impact, reversed := a.Impact, a.Reversed
	switch {
	case cond.CurvesViaAngle(): // angle/direction takeoff on a profile section
		angle, err := optionalAngleClosure(part, a.Angle, "loft: "+which+" angle")
		if err != nil {
			return nil, err
		}
		return func() feature.LoftEnd {
			return feature.LoftEnd{Condition: cond, Angle: angle(), Impact: impact, Reversed: reversed}
		}, nil
	// Apex (sharp / tangent-to-plane) and face-continuity (tangent / smooth) conditions both
	// carry only impact + reversed — no angle — so they build the same end-condition closure.
	case cond.IsPointCondition(), cond.IsFaceContinuity():
		return func() feature.LoftEnd {
			return feature.LoftEnd{Condition: cond, Impact: impact, Reversed: reversed}
		}, nil
	default:
		return nil, fmt.Errorf("loft: %s condition %q is not recognized", which, a.Condition)
	}
}
