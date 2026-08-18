// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/api/wire/featureargs"
	"oblikovati.org/app"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/param"
	"oblikovati.org/model/sketch"
)

const extrudeSchema = `{
  "type": "object",
  "properties": {
    "sketchIndex": {"type": "integer", "minimum": 0, "description": "Index of the sketch to extrude (see model.tree)."},
    "profileIndex": {"type": "integer", "minimum": 0, "default": 0, "description": "Which closed profile of the sketch to extrude."},
    "distance": {"type": "string", "description": "Extrude depth with units, e.g. \"50 mm\" or \"5 cm\". Required for distance and distance-from-face extents."},
    "operation": {"type": "string", "enum": ["new", "join", "cut", "intersect", "surface"], "default": "new", "description": "Boolean against existing bodies, or \"surface\" to build an open sheet (surface) body — Inventor's kSurfaceOperation."},
    "extent": {"type": "string", "enum": ["distance", "through-all", "to-next", "to-face", "from-to", "distance-from-face"], "default": "distance", "description": "How the extrude terminates. \"from-to\" bounds the prism between fromFace and toFace; \"distance-from-face\" measures distance from toFace instead of the sketch plane."},
    "direction": {"type": "string", "enum": ["positive", "negative", "symmetric"], "default": "positive", "description": "Which side(s) of the sketch plane to grow."},
    "secondDistance": {"type": "string", "description": "Asymmetric two-direction depth on the negative side, e.g. \"10 mm\"."},
    "taper": {"type": "string", "description": "Draft angle, e.g. \"3 deg\" (positive widens away from the sketch)."},
    "toFace": {"type": "string", "description": "Termination target for the to-face / from-to (end) / distance-from-face extents: a planar face reference key (from model.referenceKeys), a work plane (\"plane/N\"), or an origin plane (\"origin/plane/xy\")."},
    "toFaceGeom": {"type": "object", "description": "The toFace target named by GEOMETRY (a planar body face's centroid + normal) instead of toFace — for an author that cannot mint a face key. The host binds the matching planar face on the current body and freezes its plane. Wins over toFace.", "properties": {"centroid": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3}, "normal": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3}}, "required": ["centroid", "normal"]},
    "fromFace": {"type": "string", "description": "Start target for the from-to extent (the prism's lower bound): a planar face reference key, a work plane (\"plane/N\"), or an origin plane (\"origin/plane/xy\")."},
    "fromFaceGeom": {"type": "object", "description": "The fromFace start target named by GEOMETRY (centroid + normal) instead of fromFace. Wins over fromFace.", "properties": {"centroid": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3}, "normal": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3}}, "required": ["centroid", "normal"]},
    "profileSeeds": {"type": "array", "description": "Select the extruded region(s) by an interior seed point [x,y] (sketch 2-D cm), one per region, instead of profileIndex — for an author that cannot predict the host's region ordering. The host resolves each seed to its containing region on the solved sketch every recompute; wins over profileIndex.", "items": {"type": "array", "items": {"type": "number"}, "minItems": 2, "maxItems": 2}}
  },
  "required": ["sketchIndex"]
}`

// extrudeDescriptor is the self-describing "extrude" operation: turn a closed sketch
// profile into a prism. It is the reference operation; further feature kinds follow
// the same shape (PBI: revolve, hole, fillet…).
func extrudeDescriptor() *OperationDescriptor {
	return &OperationDescriptor{
		Name:    featureargs.KindExtrude,
		Summary: "Extrude a closed sketch profile into a solid prism.",
		Schema:  json.RawMessage(extrudeSchema),
		Apply:   applyExtrude,
	}
}

func applyExtrude(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	in, sk, extent, op, taper, err := resolveExtrude(part, raw)
	if err != nil {
		return nil, err
	}
	pf := feature.NewExtrudeFeatures(part.Features()).
		AddExtrude(sk, []int{in.ProfileIndex}, op, extent, taper)
	// Interior seed points select the region(s) by containment on the solved sketch every
	// recompute — the stable selector for an author that cannot predict the host's region order.
	if len(in.ProfileSeeds) > 0 {
		pf.Definition().(*feature.ExtrudeFeature).Definition().ProfileSeeds = in.ProfileSeeds
	}
	// Uniform feature result (feature/kind/bodies/healthy/reason), shared with every other
	// operation so callers read one shape — and so an unhealthy extrude is reported, not hidden.
	return recomputeResult(part, pf)
}

// resolveExtrude decodes and validates the extrude args against the part: the target
// sketch, the boolean operation, the full extent (type/direction/distances), and the
// taper, in database units.
func resolveExtrude(part *compdef.PartComponentDefinition, raw json.RawMessage) (featureargs.Extrude, *sketch.Sketch, feature.Extent, ops.PartFeatureOperation, float64, error) {
	var in featureargs.Extrude
	if err := json.Unmarshal(raw, &in); err != nil {
		return in, nil, feature.Extent{}, 0, 0, fmt.Errorf("extrude: invalid args: %w", err)
	}
	sk, err := sketchAt(part, in.SketchIndex)
	if err != nil {
		return in, nil, feature.Extent{}, 0, 0, err
	}
	op, err := parseOperation(in.Operation)
	if err != nil {
		return in, nil, feature.Extent{}, 0, 0, err
	}
	extent, err := buildExtent(part, in)
	if err != nil {
		return in, nil, feature.Extent{}, 0, 0, err
	}
	taper, err := parseTaperAngle(part, in.Taper)
	if err != nil {
		return in, nil, feature.Extent{}, 0, 0, err
	}
	return in, sk, extent, op, taper, nil
}

// buildExtent assembles the model extent from the request: the extent type and direction, plus
// the distance(s) for the distance extent, or the termination plane(s) for the to-face / from-to /
// distance-from-face extents.
func buildExtent(part *compdef.PartComponentDefinition, in featureargs.Extrude) (feature.Extent, error) {
	etype, err := parseExtentType(in.Extent)
	if err != nil {
		return feature.Extent{}, err
	}
	ext := feature.Extent{Type: etype, Direction: parseExtentDirection(in.Direction)}
	switch etype {
	case feature.ToFaceExtent:
		ext.ToPlane, err = resolveEndPlane(part, in)
		return ext, err
	case feature.FromToExtent:
		return withFromTo(part, ext, in)
	case feature.DistanceFromFaceExtent:
		// The distance-from-face base is the "to"/reference plane; the distance then offsets from it.
		if ext.ToPlane, err = resolveEndPlane(part, in); err != nil {
			return feature.Extent{}, err
		}
		return withDistance(part, ext, in)
	case feature.DistanceExtent:
		return withDistance(part, ext, in)
	default:
		return ext, nil // through-all / to-next are gauged from the model, not a distance
	}
}

// withFromTo resolves both terminators of a from-to extent: FromPlane (the start, from
// fromFace/fromFaceGeom) and ToPlane (the end, from toFace/toFaceGeom). The sketch plane supplies
// only the profile; the prism is bounded below by FromPlane and above by ToPlane.
func withFromTo(part *compdef.PartComponentDefinition, ext feature.Extent, in featureargs.Extrude) (feature.Extent, error) {
	from, err := resolveStartPlane(part, in)
	if err != nil {
		return feature.Extent{}, err
	}
	to, err := resolveEndPlane(part, in)
	if err != nil {
		return feature.Extent{}, err
	}
	ext.FromPlane, ext.ToPlane = from, to
	return ext, nil
}

// resolveEndPlane resolves the extent's "to"/reference plane from the request — a geometric target
// (toFaceGeom, centroid+normal) wins over a toFace key/plane-ref: an external author who cannot mint
// the face key names the stop face by geometry (see ToFaceGeom). A geometric target that matches no
// face resolves nil, so the extent recomputes unhealthy rather than erroring the apply.
func resolveEndPlane(part *compdef.PartComponentDefinition, in featureargs.Extrude) (*feature.WorkPlane, error) {
	return extentTargetPlane(part, "extrude", "toFace", in.ToFace, in.ToFaceGeom)
}

// resolveStartPlane resolves the from-to extent's start ("from") plane, mirroring resolveEndPlane
// over fromFace/fromFaceGeom.
func resolveStartPlane(part *compdef.PartComponentDefinition, in featureargs.Extrude) (*feature.WorkPlane, error) {
	return extentTargetPlane(part, "extrude", "fromFace", in.FromFace, in.FromFaceGeom)
}

// extentTargetPlane resolves ONE terminator of a geometric extent, for whichever feature names it —
// extrude terminates on planes parallel to its sketch, revolve on planes containing its axis, but
// both name the target the same way. A geometric selector (centroid + normal) wins over a reference
// key, as it does everywhere else; feat prefixes the errors so each feature reports as itself.
func extentTargetPlane(part *compdef.PartComponentDefinition, feat, field, ref string,
	sel *featureargs.GeomFaceSel) (*feature.WorkPlane, error) {
	if sel != nil {
		return planeFromGeom(part, *sel)
	}
	return planeFromRef(part, feat, field, ref)
}

// planeFromRef resolves a termination reference (a planar face key, "plane/N", or
// "origin/plane/xy") to a work plane; an empty reference is a caller error naming the missing field.
func planeFromRef(part *compdef.PartComponentDefinition, feat, field, ref string) (*feature.WorkPlane, error) {
	if strings.TrimSpace(ref) == "" {
		return nil, fmt.Errorf("%s: this extent requires %q (a planar face key, \"plane/N\", or \"origin/plane/xy\")", feat, field)
	}
	wp, err := part.WorkGeometry().PlaneTargetFromRef(ref)
	if err != nil {
		return nil, fmt.Errorf("%s: %s target %q: %w", feat, field, ref, err)
	}
	return wp, nil
}

// planeFromGeom resolves a termination target named by GEOMETRY (a planar face's centroid+normal)
// to a frozen work plane; a malformed selector errors, an unmatched one resolves nil (the extent
// then recomputes unhealthy, matching to-face's graceful degradation — see toFaceGeomPlane).
func planeFromGeom(part *compdef.PartComponentDefinition, sel featureargs.GeomFaceSel) (*feature.WorkPlane, error) {
	ref, err := geomFaceRef(sel)
	if err != nil {
		return nil, err
	}
	wp, _ := toFaceGeomPlane(part, ref)
	return wp, nil
}

// toFaceGeomBindTol is the model-space distance a to-face geometric target's centroid may sit from
// a candidate face's centroid and still bind (see [topo.Body.FindFaceByGeometry]); it matches the
// hole/dress-up geometric-selector tolerance.
const toFaceGeomBindTol = 1e-3

// toFaceGeomPlane finds the planar body face matching the geometric descriptor and freezes its
// plane. It tries an exact centroid+normal match first, then a plane-through-centroid match (a
// large/annular stop face whose centroid sits off the recorded point still binds by its plane).
// ok is false when no planar face matches — the caller leaves the extent unresolved so it degrades
// to an unhealthy recompute (the hole's lost-placement-face pattern) rather than a hard error.
func toFaceGeomPlane(part *compdef.PartComponentDefinition, ref topo.GeometricFaceRef) (*feature.WorkPlane, bool) {
	for _, b := range part.SurfaceBodies().All() {
		if f, ok := b.FindFaceByGeometry(ref, toFaceGeomBindTol); ok {
			if wp, err := fixedPlaneOfFace(f); err == nil {
				return wp, true
			}
		}
	}
	for _, b := range part.SurfaceBodies().All() {
		if f, ok := b.FindPlanarFaceThrough(ref.Centroid, ref.Normal, toFaceGeomBindTol*10); ok {
			if wp, err := fixedPlaneOfFace(f); err == nil {
				return wp, true
			}
		}
	}
	return nil, false
}

// fixedPlaneOfFace freezes a planar face's surface as a transient work plane usable as an extent
// target (mirrors WorkGeometry.facePlane's face→plane step for a face found by geometry).
func fixedPlaneOfFace(f *topo.Face) (*feature.WorkPlane, error) {
	pl, ok := f.Geometry().(geom.Plane)
	if !ok {
		return nil, errors.New("extrude: to-face geometric target is not a planar face")
	}
	sp, err := sketch.NewPlane(pl.Origin, pl.UAxis, pl.VAxis)
	if err != nil {
		return nil, err
	}
	return feature.NewFixedWorkPlane(sp), nil
}

// withDistance fills the distance extent's primary (and optional asymmetric) depth.
func withDistance(part *compdef.PartComponentDefinition, ext feature.Extent, in featureargs.Extrude) (feature.Extent, error) {
	dist, err := lengthClosure(part, in.Distance, "extrude: distance")
	if err != nil {
		return feature.Extent{}, err
	}
	if dist() == 0 {
		return feature.Extent{}, errors.New("extrude: distance is zero")
	}
	ext.Distance = dist
	if in.SecondDistance != "" {
		d2, err := lengthClosure(part, in.SecondDistance, "extrude: secondDistance")
		if err != nil {
			return feature.Extent{}, err
		}
		ext.Distance2 = d2
	}
	return ext, nil
}

// parseTaperAngle parses an optional draft angle ("" ⇒ no taper).
func parseTaperAngle(part *compdef.PartComponentDefinition, expr string) (float64, error) {
	if strings.TrimSpace(expr) == "" {
		return 0, nil
	}
	a, err := part.Units().Parse(expr, param.Angle)
	if err != nil {
		return 0, fmt.Errorf("extrude: taper %q: %w", expr, err)
	}
	return a.Value, nil
}

// parseExtentType maps an extent name to its model type (empty ⇒ distance). The reference extents
// terminate on plane(s) the request names: to-face / distance-from-face via "toFace", from-to via
// both "fromFace" (start) and "toFace" (end).
func parseExtentType(name string) (feature.ExtentType, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "distance":
		return feature.DistanceExtent, nil
	case "through-all", "throughall":
		return feature.ThroughAllExtent, nil
	case "to-next", "tonext":
		return feature.ToNextExtent, nil
	case "to-face", "toface":
		return feature.ToFaceExtent, nil
	case "from-to", "fromto":
		return feature.FromToExtent, nil
	case "distance-from-face", "distancefromface":
		return feature.DistanceFromFaceExtent, nil
	default:
		return 0, fmt.Errorf("extrude: unknown extent %q (want distance|through-all|to-next|to-face|from-to|distance-from-face)", name)
	}
}

// parseExtentDirection maps a direction name to its model value (empty/unknown ⇒ positive).
func parseExtentDirection(name string) feature.ExtentDirection {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "negative":
		return feature.NegativeDir
	case "symmetric":
		return feature.SymmetricDir
	default:
		return feature.PositiveDir
	}
}

// sketchAt returns the part's sketch at index i, bounds-checked.
func sketchAt(part *compdef.PartComponentDefinition, i int) (*sketch.Sketch, error) {
	sks := part.Sketches()
	if i < 0 || i >= sks.Count() {
		return nil, fmt.Errorf("extrude: sketch index %d out of range (part has %d sketches)", i, sks.Count())
	}
	return sks.Item(i), nil
}

// parseOperation maps an operation name to the feature operation (default new body). "surface"
// is Inventor's kSurfaceOperation — an open sheet body rather than a boolean (#1858).
func parseOperation(name string) (ops.PartFeatureOperation, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "new", "newbody", "new-body":
		return ops.NewBody, nil
	case "join":
		return ops.Join, nil
	case "cut":
		return ops.Cut, nil
	case "intersect":
		return ops.Intersect, nil
	case "surface":
		return ops.Surface, nil
	default:
		return 0, fmt.Errorf("extrude: unknown operation %q (want new|join|cut|intersect|surface)", name)
	}
}
