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
	"oblikovati.org/kernel/ops"
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
    "operation": {"type": "string", "enum": ["new", "join", "cut", "intersect"], "default": "new", "description": "Boolean against existing bodies."},
    "extent": {"type": "string", "enum": ["distance", "through-all", "to-next", "to-face"], "default": "distance", "description": "How the extrude terminates."},
    "direction": {"type": "string", "enum": ["positive", "negative", "symmetric"], "default": "positive", "description": "Which side(s) of the sketch plane to grow."},
    "secondDistance": {"type": "string", "description": "Asymmetric two-direction depth on the negative side, e.g. \"10 mm\"."},
    "taper": {"type": "string", "description": "Draft angle, e.g. \"3 deg\" (positive widens away from the sketch)."},
    "toFace": {"type": "string", "description": "Termination target for the to-face extent: a planar face reference key (from model.referenceKeys), a work plane (\"plane/N\"), or an origin plane (\"origin/plane/xy\")."},
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
// the distance(s) for the distance extent or the termination plane for the to-face extent.
func buildExtent(part *compdef.PartComponentDefinition, in featureargs.Extrude) (feature.Extent, error) {
	etype, err := parseExtentType(in.Extent)
	if err != nil {
		return feature.Extent{}, err
	}
	ext := feature.Extent{Type: etype, Direction: parseExtentDirection(in.Direction)}
	switch etype {
	case feature.ToFaceExtent:
		return withToFaceTarget(part, ext, in.ToFace)
	case feature.DistanceExtent:
		return withDistance(part, ext, in)
	default:
		return ext, nil // through-all / to-next are gauged from the model, not a distance
	}
}

// withToFaceTarget resolves the to-face termination reference (a planar face key, "plane/N", or
// "origin/plane/xy") onto the extent's ToPlane.
func withToFaceTarget(part *compdef.PartComponentDefinition, ext feature.Extent, ref string) (feature.Extent, error) {
	if strings.TrimSpace(ref) == "" {
		return feature.Extent{}, errors.New(`extrude: to-face extent requires "toFace" (a planar face key, "plane/N", or "origin/plane/xy")`)
	}
	wp, err := part.WorkGeometry().PlaneTargetFromRef(ref)
	if err != nil {
		return feature.Extent{}, fmt.Errorf("extrude: to-face target %q: %w", ref, err)
	}
	ext.ToPlane = wp
	return ext, nil
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

// parseExtentType maps an extent name to its model type (empty ⇒ distance). The to-face extent
// terminates at a plane the request names via "toFace"; the remaining reference extents (from-to /
// distance-from-face) are not yet exposed over the JSON API.
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
	default:
		return 0, fmt.Errorf("extrude: unknown extent %q (want distance|through-all|to-next|to-face)", name)
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

// parseOperation maps an operation name to the boolean op (default new body).
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
	default:
		return 0, fmt.Errorf("extrude: unknown operation %q (want new|join|cut|intersect)", name)
	}
}
