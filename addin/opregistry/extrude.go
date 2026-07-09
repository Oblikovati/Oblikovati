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
    "extent": {"type": "string", "enum": ["distance", "through-all", "to-next", "to-face"], "default": "distance", "description": "How the extrude terminates."},
    "direction": {"type": "string", "enum": ["positive", "negative", "symmetric"], "default": "positive", "description": "Which side(s) of the sketch plane to grow."},
    "secondDistance": {"type": "string", "description": "Asymmetric two-direction depth on the negative side, e.g. \"10 mm\"."},
    "taper": {"type": "string", "description": "Draft angle, e.g. \"3 deg\" (positive widens away from the sketch)."},
    "toFace": {"type": "string", "description": "Termination target for the to-face extent: a planar face reference key (from model.referenceKeys), a work plane (\"plane/N\"), or an origin plane (\"origin/plane/xy\")."},
    "toFaceGeom": {"type": "object", "description": "Termination target for the to-face extent named by GEOMETRY (a planar body face's centroid + normal) instead of toFace — for an author that cannot mint a face key. The host binds the matching planar face on the current body and freezes its plane. Wins over toFace.", "properties": {"centroid": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3}, "normal": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3}}, "required": ["centroid", "normal"]},
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
		// A geometric target (centroid+normal) wins over a ToFace key/plane-ref: an external
		// author cannot mint the face key, so it names the stop face by geometry (see ToFaceGeom).
		if in.ToFaceGeom != nil {
			return withToFaceGeom(part, ext, *in.ToFaceGeom)
		}
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

// toFaceGeomBindTol is the model-space distance a to-face geometric target's centroid may sit from
// a candidate face's centroid and still bind (see [topo.Body.FindFaceByGeometry]); it matches the
// hole/dress-up geometric-selector tolerance.
const toFaceGeomBindTol = 1e-3

// withToFaceGeom resolves a to-face termination target named by GEOMETRY (a planar face's
// centroid + normal) rather than a reference key — the extent counterpart of the hole's
// PlacementFaceGeom. It finds the matching planar face on the already-built body and freezes its
// plane, the same *WorkPlane a face-key toFace yields via NewFixedWorkPlane. Feature build order
// guarantees the target face exists when the extrude applies, so a one-time resolve is stable.
func withToFaceGeom(part *compdef.PartComponentDefinition, ext feature.Extent, sel featureargs.GeomFaceSel) (feature.Extent, error) {
	ref, err := geomFaceRef(sel)
	if err != nil {
		return feature.Extent{}, err // a malformed selector is a caller error, not a resolution miss
	}
	// A target that matches no face leaves ToPlane nil; the to-face extent then recomputes UNHEALTHY
	// (see toFaceGeomPlane) rather than erroring the whole apply, so a batch author (the exporter,
	// reading an under-built base) flags the feature and keeps going instead of aborting the part.
	ext.ToPlane, _ = toFaceGeomPlane(part, ref)
	return ext, nil
}

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
