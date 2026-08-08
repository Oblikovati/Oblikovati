// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"oblikovati.org/api/wire/featureargs"
	"oblikovati.org/app"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
)

// The additive sketch-profile solid features beyond extrude: revolve and loft. Each consumes one
// or more closed/open sketch profiles (by sketchIndex/profileIndex) and follows the extrude
// descriptor shape, so add_feature can drive the whole additive set. The rib, coil and emboss grew
// their own option clusters (#1882/#1883/#1893) and moved to feature_rib.go / feature_coil.go /
// feature_emboss.go.

// --- revolve ---------------------------------------------------------------

const revolveSchema = `{
  "type": "object",
  "properties": {
    "sketchIndex": {"type": "integer", "minimum": 0},
    "profileIndex": {"type": "integer", "minimum": 0, "default": 0},
    "axisRef": {"type": "string", "description": "Work-axis reference to revolve about, e.g. \"origin/axis/y\" (default). See get_reference_keys / list_work_planes."},
    "aboutCenterline": {"type": "boolean", "default": false, "description": "Revolve about the sketch's single centerline (an internal, tilted axis) instead of axisRef; the sketch must have exactly one centerline."},
    "angle": {"type": "string", "description": "Revolve angle with units, e.g. \"360 deg\"."},
    "angle2": {"type": "string", "description": "Optional second-direction sweep (opposite sense), e.g. \"30 deg\" — the two-directional revolve."},
    "direction": {"type": "string", "enum": ["positive", "negative", "symmetric"], "default": "positive", "description": "Which side of the profile the angle sweeps to: positive (forward), negative (the same sweep the other way), or symmetric (half each way). Ignored when angle2 is set, and unobservable on a full revolution."},
    "operation": {"type": "string", "enum": ["new", "join", "cut", "intersect", "surface"], "default": "new", "description": "Boolean against existing bodies, or \"surface\" to revolve the profile into an open surface-of-revolution (sheet) body — Inventor's kSurfaceOperation."},
    "extent": {"type": "string", "enum": ["angle", "to-face", "from-to", "to-next"], "default": "angle", "description": "How the revolve terminates. \"angle\" sweeps the given angle; \"to-face\" sweeps until it reaches toFace; \"from-to\" bounds the wedge by fromFace and toFace; \"to-next\" stops at the next material met. The geometric extents ignore angle/angle2 and measure their own sweep."},
    "toFace": {"type": "string", "description": "Stop target for the to-face / from-to (end) extents: a planar face reference key, a work plane (\"plane/N\"), or an origin plane (\"origin/plane/xy\"). It must CONTAIN the revolve axis — only a radial face meets the sweep at one constant angle."},
    "toFaceGeom": {"type": "object", "description": "The toFace target named by GEOMETRY (a planar face's centroid + normal) instead of toFace — for an author that cannot mint a face key. Wins over toFace.", "properties": {"centroid": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3}, "normal": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3}}, "required": ["centroid", "normal"]},
    "fromFace": {"type": "string", "description": "Start target for the from-to extent, named like toFace and likewise containing the axis. The wedge runs backwards from the profile to fromFace and forwards to toFace, so it always contains the profile."},
    "fromFaceGeom": {"type": "object", "description": "The fromFace start target named by GEOMETRY (centroid + normal) instead of fromFace. Wins over fromFace.", "properties": {"centroid": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3}, "normal": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3}}, "required": ["centroid", "normal"]},
    "profileSeed": {"type": "array", "items": {"type": "number"}, "minItems": 2, "maxItems": 2, "description": "Select the revolved region by an interior seed point [x,y] (sketch 2-D cm) instead of profileIndex — resolved by containment on the solved sketch every recompute; wins over profileIndex."}
  },
  "required": ["sketchIndex"]
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
	extent, err := parseRevolveExtent(in)
	if err != nil {
		return nil, err
	}
	angle, err := revolveAngleFor(part, in, extent)
	if err != nil {
		return nil, err
	}
	op, err := parseOperation(in.Operation)
	if err != nil {
		return nil, err
	}
	if in.AboutCenterline {
		if in.Angle2 != "" {
			return nil, errors.New("revolve: aboutCenterline does not support a second-direction angle2")
		}
		pf := feature.NewRevolveFeatures(part.Features()).AddAboutCenterline(sk, in.ProfileIndex, angle, op)
		return finishRevolve(part, pf, in, extent)
	}
	axis, err := axisFromRef(part, in.AxisRef)
	if err != nil {
		return nil, err
	}
	if in.Angle2 != "" {
		angle2, err := angleClosure(part, in.Angle2, "revolve: angle2")
		if err != nil {
			return nil, err
		}
		pf := feature.NewRevolveFeatures(part.Features()).AddTwoDirectional(sk, in.ProfileIndex, axis, angle, angle2, op)
		return finishRevolve(part, pf, in, extent)
	}
	pf := feature.NewRevolveFeatures(part.Features()).Add(sk, in.ProfileIndex, axis, angle, op)
	return finishRevolve(part, pf, in, extent)
}

// parseRevolveExtent maps the revolve's extent spelling onto the model member, rejecting the
// combinations that would silently drop one of the caller's inputs (#1860). "angle" is spelled
// DistanceExtent in the model: a revolve's "distance" IS its angle (Inventor's kAngleExtent).
func parseRevolveExtent(in featureargs.Revolve) (feature.ExtentType, error) {
	extent, err := revolveExtentType(in.Extent)
	if err != nil || extent == feature.DistanceExtent {
		return extent, err
	}
	if in.Angle2 != "" {
		return 0, fmt.Errorf("revolve: extent %q measures its own sweep, so it cannot also take angle2 "+
			"(the asymmetric two-angle mode); drop one", in.Extent)
	}
	return extent, nil
}

// revolveExtentType is the extent-spelling lookup on its own.
func revolveExtentType(spelling string) (feature.ExtentType, error) {
	switch strings.TrimSpace(spelling) {
	case "", "angle":
		return feature.DistanceExtent, nil
	case "to-face":
		return feature.ToFaceExtent, nil
	case "from-to":
		return feature.FromToExtent, nil
	case "to-next":
		return feature.ToNextExtent, nil
	default:
		return 0, fmt.Errorf("revolve: unknown extent %q (want angle, to-face, from-to or to-next)", spelling)
	}
}

// revolveAngleFor resolves the sweep the angle extent reads. The geometric extents measure their
// own, so they accept a missing angle rather than demanding a number the model will ignore.
func revolveAngleFor(part *compdef.PartComponentDefinition, in featureargs.Revolve,
	extent feature.ExtentType) (func() float64, error) {
	if extent != feature.DistanceExtent {
		return optionalAngleClosure(part, in.Angle, "revolve: angle")
	}
	if strings.TrimSpace(in.Angle) == "" {
		return nil, errors.New(`revolve: the angle extent needs "angle" (e.g. "360 deg"); ` +
			`the to-face / from-to / to-next extents measure their own sweep`)
	}
	return angleClosure(part, in.Angle, "revolve: angle")
}

// finishRevolve applies the axis-independent options — the sweep direction and the region seed —
// and recomputes. Every construction path ends here so a "direction" can never be honoured on one
// axis and dropped on another (#2019).
func finishRevolve(part *compdef.PartComponentDefinition, pf *feature.PartFeature,
	in featureargs.Revolve, extent feature.ExtentType) (json.RawMessage, error) {
	def := pf.Definition().(*feature.RevolveFeature).Definition()
	def.Direction = parseExtentDirection(in.Direction)
	if err := bindRevolveExtent(part, def, in, extent); err != nil {
		return nil, err
	}
	setRevolveProfileSeed(pf, in.ProfileSeed)
	return recomputeResult(part, pf)
}

// bindRevolveExtent records the extent and resolves whichever terminator plane(s) it needs. to-next
// binds nothing: it finds its own stop against the running bodies at recompute time.
func bindRevolveExtent(part *compdef.PartComponentDefinition, def *feature.RevolveDefinition,
	in featureargs.Revolve, extent feature.ExtentType) error {
	def.Extent = extent
	var err error
	switch extent {
	case feature.FromToExtent:
		if def.FromPlane, err = extentTargetPlane(part, "revolve", "fromFace", in.FromFace, in.FromFaceGeom); err != nil {
			return err
		}
		fallthrough // from-to's END is named exactly like to-face's stop
	case feature.ToFaceExtent:
		def.ToPlane, err = extentTargetPlane(part, "revolve", "toFace", in.ToFace, in.ToFaceGeom)
	}
	return err
}

// setRevolveProfileSeed records an interior seed point on the revolve so its region resolves by
// containment on the solved sketch every recompute (the stable selector for an external author).
func setRevolveProfileSeed(pf *feature.PartFeature, seed []float64) {
	if len(seed) == 0 {
		return
	}
	pf.Definition().(*feature.RevolveFeature).Definition().ProfileSeed = append([]float64(nil), seed...)
}

// --- loft ------------------------------------------------------------------

const loftSchema = `{
  "type": "object",
  "properties": {
    "sections": {"type": "array", "minItems": 2, "items": {"type": "object", "properties": {"sketchIndex": {"type": "integer"}, "profileIndex": {"type": "integer"}, "point": {"type": "array", "items": {"type": "number"}, "minItems": 2, "maxItems": 2, "description": "[x,y] on the sketch plane → an apex (point) section so the loft tapers to a tip; only valid first or last."}, "faceRef": {"type": "string", "description": "A body-face reference key (get_reference_keys) → a face section the loft can leave Tangent/Smooth."}}, "required": ["sketchIndex"]}, "description": "Ordered cross-section profiles (>= 2) to loft through."},
    "closed": {"type": "boolean", "default": false},
    "operation": {"type": "string", "enum": ["new", "join", "cut", "surface"], "default": "new", "description": "Boolean against existing bodies, or \"surface\" to skin the sections into an open loft-surface (sheet) body — Inventor's kSurfaceOperation. A multi-bore section is not supported as a surface."},
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
