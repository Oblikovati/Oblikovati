// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire/featureargs"
	"oblikovati.org/app"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
)

// The surfacing features: build surface bodies (boundary patch, ruled surface), modify them
// (surface offset, extend), and convert between surfaces and solids (thicken — see
// feature_modify.go, midSurface, stitch, sculpt). Profile-based surfaces take a sketch
// profile; the rest act on the part's existing surface/solid bodies.

// --- boundary patch --------------------------------------------------------

const boundaryPatchSchema = `{
  "type": "object",
  "properties": {
    "sketchIndex": {"type": "integer", "minimum": 0},
    "profileIndex": {"type": "integer", "minimum": 0, "default": 0},
    "condition": {"type": "string", "enum": ["free", "tangent"], "default": "free", "description": "Edge continuity to neighbouring faces."}
  },
  "required": ["sketchIndex"]
}`

func boundaryPatchDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: featureargs.KindBoundaryPatch, Summary: "Fill a closed sketch loop with a surface patch.", Schema: json.RawMessage(boundaryPatchSchema), Apply: applyBoundaryPatch}
}

func applyBoundaryPatch(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeFeatureArgs[featureargs.BoundaryPatch](s, raw)
	if err != nil {
		return nil, err
	}
	sk, err := sketchAt(part, in.SketchIndex)
	if err != nil {
		return nil, err
	}
	cond := feature.PatchFree
	if strings.EqualFold(in.Condition, "tangent") {
		cond = feature.PatchTangent
	}
	pf := feature.NewBoundaryPatchFeatures(part.Features()).Add(sk, in.ProfileIndex, cond)
	return recomputeResult(part, pf)
}

// --- ruled surface ---------------------------------------------------------

const ruledSchema = `{
  "type": "object",
  "properties": {
    "sketchIndex": {"type": "integer", "minimum": 0},
    "profileIndex": {"type": "integer", "minimum": 0, "default": 0},
    "type": {"type": "string", "enum": ["normal", "tangent", "perpendicular"], "default": "normal", "description": "Direction of the straight rulings."},
    "distance": {"type": "string", "description": "Ruling length, e.g. \"10 mm\"."}
  },
  "required": ["sketchIndex", "distance"]
}`

func ruledSurfaceDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: featureargs.KindRuledSurface, Summary: "Sweep straight rulings off a profile into a surface.", Schema: json.RawMessage(ruledSchema), Apply: applyRuledSurface}
}

func applyRuledSurface(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeFeatureArgs[featureargs.RuledSurface](s, raw)
	if err != nil {
		return nil, err
	}
	sk, err := sketchAt(part, in.SketchIndex)
	if err != nil {
		return nil, err
	}
	dist, err := lengthClosure(part, in.Distance, "ruledSurface: distance")
	if err != nil {
		return nil, err
	}
	pf := feature.NewRuledSurfaceFeatures(part.Features()).AddByDistance(sk, in.ProfileIndex, ruledType(in.Type), dist)
	return recomputeResult(part, pf)
}

func ruledType(name string) feature.RuledSurfaceType {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "tangent":
		return feature.RuledTangent
	case "perpendicular":
		return feature.RuledPerpendicular
	default:
		return feature.RuledNormal
	}
}

// --- surface offset / extend / midSurface / stitch / sculpt ----------------

const surfaceOffsetSchema = `{
  "type": "object",
  "properties": {"distance": {"type": "string", "description": "Offset distance for the surface bodies, e.g. \"2 mm\"."}},
  "required": ["distance"]
}`

const extendSchema = `{
  "type": "object",
  "properties": {
    "edgeRefs": {"type": "array", "items": {"type": "string"}, "description": "Reference keys of the surface boundary edges to extend (get_reference_keys)."},
    "edgeRef": {"type": "string", "description": "Legacy single edge (use edgeRefs)."},
    "extentType": {"type": "string", "enum": ["distance", "toPlane", "toObject"], "default": "distance", "description": "distance grows by distance; toPlane/toObject grows each edge until it reaches targetRef."},
    "distance": {"type": "string", "description": "Extend distance for extentType distance, e.g. \"5 mm\"."},
    "targetRef": {"type": "string", "description": "Target plane/face for extentType toPlane (a planar face key, \"plane/N\", or \"origin/plane/xy\")."},
    "extensionType": {"type": "string", "enum": ["stretched", "natural"], "default": "stretched", "description": "Continuity mode; coincides with stretched for planar faces."}
  }
}`

const midSurfaceSchema = `{
  "type": "object",
  "properties": {
    "maxThickness": {"type": "string", "description": "Max wall thickness to auto-pair into a mid-surface, e.g. \"3 mm\"."},
    "minThickness": {"type": "string", "description": "Min wall thickness for auto-pairing (default 0)."},
    "bodyIndices": {"type": "array", "items": {"type": "integer", "minimum": 0}, "description": "Input body indices (model.tree order); default the last body."},
    "facePairs": {"type": "array", "items": {"type": "object", "properties": {"a": {"type": "string"}, "b": {"type": "string"}}, "required": ["a", "b"]}, "description": "Manual face-key pairs (get_reference_keys) — pairs these directly instead of auto-pairing."}
  }
}`

const stitchSchema = `{
  "type": "object",
  "properties": {
    "tolerance": {"type": "string", "description": "Stitch gap tolerance, e.g. \"0.1 mm\".", "default": "0.1 mm"},
    "maintainAsSurface": {"type": "boolean", "default": false, "description": "Keep a surface body instead of solidifying a closed quilt."}
  }
}`

const sculptSchema = `{
  "type": "object",
  "properties": {
    "operation": {"type": "string", "enum": ["new", "join", "cut"], "default": "new"},
    "tolerance": {"type": "string", "description": "Sculpt tolerance, e.g. \"0.1 mm\".", "default": "0.1 mm"}
  }
}`

func surfaceOffsetDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: featureargs.KindSurfaceOffset, Summary: "Offset the part's surface bodies by a distance.", Schema: json.RawMessage(surfaceOffsetSchema), Apply: applySurfaceOffset}
}

func extendDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: featureargs.KindExtend, Summary: "Extend a surface body past a picked edge.", Schema: json.RawMessage(extendSchema), Apply: applyExtend}
}

func midSurfaceDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: featureargs.KindMidSurface, Summary: "Build the mid-surface between thin-wall face pairs.", Schema: json.RawMessage(midSurfaceSchema), Apply: applyMidSurface}
}

func stitchDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: featureargs.KindStitch, Summary: "Stitch surface bodies into a quilt (or solid).", Schema: json.RawMessage(stitchSchema), Apply: applyStitch}
}

func sculptDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: featureargs.KindSculpt, Summary: "Combine surfaces and solids into a sculpted body.", Schema: json.RawMessage(sculptSchema), Apply: applySculpt}
}

const fillSurfaceSchema = `{
  "type": "object",
  "properties": {
    "continuity": {"type": "string", "enum": ["g0", "g1", "g2"], "default": "g2", "description": "Continuity to the bounding surfaces (api/types SurfaceContinuity): g0 (position), g1 (tangent), g2 (curvature). NURBS neighbours are matched to the chosen continuity; planar/merged/split sides fill position-only."},
    "sides": {"type": "integer", "default": 4, "description": "Number of bounding surface bodies forming the opening (the LAST N surface bodies). 4 = classic four-sided fill; 3, 5, 6… = N-sided fill mapped onto four logical sides."}
  }
}`

func fillSurfaceDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: featureargs.KindFillSurface, Summary: "Close an N-sided opening bounded by the last N surface bodies with a single NURBS (G0/G1/G2).", Schema: json.RawMessage(fillSurfaceSchema), Apply: applyFillSurface}
}

func applyFillSurface(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeFeatureArgs[featureargs.FillSurface](s, raw)
	if err != nil {
		return nil, err
	}
	cont, ok := types.ParseSurfaceContinuity(in.Continuity, types.ContinuityG2)
	if !ok {
		return nil, fmt.Errorf("fillSurface: unknown continuity %q (want g0, g1, or g2)", in.Continuity)
	}
	sides := in.Sides
	if sides <= 0 {
		sides = feature.DefaultFillSides
	}
	pf := feature.NewFillFeatures(part.Features()).AddSides(cont.Order(), sides)
	return recomputeResult(part, pf)
}

const bridgeSurfaceSchema = `{
  "type": "object",
  "properties": {
    "continuityA": {"type": "string", "enum": ["g0", "g1", "g2"], "default": "g2", "description": "Continuity to the FIRST of the last two surface bodies (api/types SurfaceContinuity): g0/g1/g2."},
    "continuityB": {"type": "string", "enum": ["g0", "g1", "g2"], "default": "g2", "description": "Continuity to the SECOND of the last two surface bodies."}
  }
}`

func bridgeSurfaceDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: featureargs.KindBridgeSurface, Summary: "Connect the last two surface bodies with a clean NURBS transition (G0/G1/G2 per side).", Schema: json.RawMessage(bridgeSurfaceSchema), Apply: applyBridgeSurface}
}

func applyBridgeSurface(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeFeatureArgs[featureargs.BridgeSurface](s, raw)
	if err != nil {
		return nil, err
	}
	ca, ok := types.ParseSurfaceContinuity(in.ContinuityA, types.ContinuityG2)
	if !ok {
		return nil, fmt.Errorf("bridgeSurface: unknown continuityA %q (want g0, g1, or g2)", in.ContinuityA)
	}
	cb, ok := types.ParseSurfaceContinuity(in.ContinuityB, types.ContinuityG2)
	if !ok {
		return nil, fmt.Errorf("bridgeSurface: unknown continuityB %q (want g0, g1, or g2)", in.ContinuityB)
	}
	pf := feature.NewBridgeFeatures(part.Features()).Add(ca.Order(), cb.Order())
	return recomputeResult(part, pf)
}

const networkSurfaceSchema = `{
  "type": "object",
  "properties": {
    "uCurves": {"type": "array", "minItems": 2, "description": "U-direction curves, each a list of [x,y,z] points (model units).", "items": {"type": "array", "items": {"type": "array", "items": {"type": "number"}}}},
    "vCurves": {"type": "array", "minItems": 2, "description": "V-direction curves, each a list of [x,y,z] points; must approximately intersect the U-curves at a grid.", "items": {"type": "array", "items": {"type": "array", "items": {"type": "number"}}}}
  },
  "required": ["uCurves", "vCurves"]
}`

func networkSurfaceDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: featureargs.KindNetworkSurface, Summary: "Interpolate a grid of intersecting U/V curves with a single NURBS (Gordon network surface).", Schema: json.RawMessage(networkSurfaceSchema), Apply: applyNetworkSurface}
}

func applyNetworkSurface(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeFeatureArgs[featureargs.NetworkSurface](s, raw)
	if err != nil {
		return nil, err
	}
	pf := feature.NewNetworkFeatures(part.Features()).Add(networkPolylines(in.UCurves), networkPolylines(in.VCurves))
	return recomputeResult(part, pf)
}

const fairSurfaceSchema = `{
  "type": "object",
  "properties": {
    "continuity": {"type": "string", "enum": ["g0", "g1", "g2"], "default": "g2", "description": "Boundary continuity to hold while fairing the running surface (api/types SurfaceContinuity)."},
    "strength": {"type": "number", "default": 0.5, "description": "Per-iteration relaxation (0<s<=1)."},
    "iterations": {"type": "integer", "default": 20, "description": "Number of fairing iterations."}
  }
}`

func fairSurfaceDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: featureargs.KindFairSurface, Summary: "Smooth curvature wrinkles out of the running surface, holding its boundary continuity (G0/G1/G2).", Schema: json.RawMessage(fairSurfaceSchema), Apply: applyFairSurface}
}

func applyFairSurface(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeFeatureArgs[featureargs.FairSurface](s, raw)
	if err != nil {
		return nil, err
	}
	cont, ok := types.ParseSurfaceContinuity(in.Continuity, types.ContinuityG2)
	if !ok {
		return nil, fmt.Errorf("fairSurface: unknown continuity %q (want g0, g1, or g2)", in.Continuity)
	}
	strength, iters := in.Strength, in.Iterations
	if strength <= 0 {
		strength = 0.5
	}
	if iters <= 0 {
		iters = 20
	}
	pf := feature.NewFairFeatures(part.Features()).Add(cont.Order(), strength, iters)
	return recomputeResult(part, pf)
}

const fitSurfaceSchema = `{
  "type": "object",
  "required": ["cloud"],
  "properties": {
    "cloud": {"type": "string", "description": "Name of the point cloud whose cropped region is fitted. Crop the cloud to the region first."},
    "degree": {"type": "integer", "default": 3, "description": "Surface degree each way (3 = bicubic, the Class-A default)."},
    "nu": {"type": "integer", "default": 6, "description": "Control-point (span) count in U; must exceed the degree."},
    "nv": {"type": "integer", "default": 6, "description": "Control-point (span) count in V; must exceed the degree."}
  }
}`

func fitSurfaceDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: featureargs.KindFitSurface, Summary: "Fit a clean Class-A NURBS surface to a scanned point-cloud region (degree + U/V spans).", Schema: json.RawMessage(fitSurfaceSchema), Apply: applyFitSurface}
}

func applyFitSurface(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeFeatureArgs[featureargs.FitSurface](s, raw)
	if err != nil {
		return nil, err
	}
	pc, ok := part.PointClouds().ByName(in.Cloud)
	if !ok {
		return nil, fmt.Errorf("fitSurface: no point cloud named %q", in.Cloud)
	}
	degree, nu, nv := fitSurfaceDefaults(in)
	pf := feature.NewFitFeatures(part.Features()).Add(pc.CroppedModelPoints(), degree, nu, nv)
	return recomputeResult(part, pf)
}

// fitSurfaceDefaults fills the bicubic 6×6 defaults for any unset fit parameter.
func fitSurfaceDefaults(in featureargs.FitSurface) (degree, nu, nv int) {
	degree, nu, nv = in.Degree, in.NU, in.NV
	if degree <= 0 {
		degree = feature.DefaultFitDegree
	}
	if nu <= 0 {
		nu = feature.DefaultFitSpans
	}
	if nv <= 0 {
		nv = feature.DefaultFitSpans
	}
	return degree, nu, nv
}

// networkPolylines converts wire [x,y,z] curve point lists to model points (skipping malformed points).
func networkPolylines(curves [][][]float64) [][]math.Point3 {
	out := make([][]math.Point3, len(curves))
	for i, c := range curves {
		out[i] = make([]math.Point3, 0, len(c))
		for _, p := range c {
			if len(p) == 3 {
				out[i] = append(out[i], math.P3(math.Scalar(p[0]), math.Scalar(p[1]), math.Scalar(p[2])))
			}
		}
	}
	return out
}

func applySurfaceOffset(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeFeatureArgs[featureargs.SurfaceOffset](s, raw)
	if err != nil {
		return nil, err
	}
	d, err := lengthClosure(part, in.Distance, "surfaceOffset: distance")
	if err != nil {
		return nil, err
	}
	pf := feature.NewSurfaceOffsetFeatures(part.Features()).AddByDistance(d)
	return recomputeResult(part, pf)
}

func applyExtend(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeFeatureArgs[featureargs.Extend](s, raw)
	if err != nil {
		return nil, err
	}
	edges := extendEdgeKeys(in)
	if len(edges) == 0 {
		return nil, errors.New("extend: edgeRefs is empty")
	}
	def := &feature.ExtendDefinition{EdgeKeys: edges, Natural: in.ExtensionType == "natural"}
	if err := resolveExtendExtent(part, in, def); err != nil {
		return nil, err
	}
	pf := feature.NewExtendFeatures(part.Features()).AddExtend(def)
	return recomputeResult(part, pf)
}

// resolveExtendExtent fills the extend definition's extent: a target plane for extentType
// toPlane/toObject (resolved from a work plane / planar face), else the distance closure (#1878).
func resolveExtendExtent(part *compdef.PartComponentDefinition, in featureargs.Extend, def *feature.ExtendDefinition) error {
	switch in.ExtentType {
	case "toPlane", "toObject":
		wp, err := part.WorkGeometry().PlaneTargetFromRef(in.TargetRef)
		if err != nil {
			return fmt.Errorf("extend: target %q: %w", in.TargetRef, err)
		}
		pl, err := geom.NewPlane(wp.Plane().Origin(), wp.Plane().Normal().AsVector())
		if err != nil {
			return fmt.Errorf("extend: target %q: %w", in.TargetRef, err)
		}
		def.TargetPlane = &pl
		return nil
	default:
		d, err := lengthClosure(part, in.Distance, "extend: distance")
		if err != nil {
			return err
		}
		def.Distance = d
		return nil
	}
}

// extendEdgeKeys returns the extend edges: the multi-edge EdgeRefs, else the legacy single EdgeRef.
func extendEdgeKeys(in featureargs.Extend) [][]byte {
	if len(in.EdgeRefs) > 0 {
		return refKeys(in.EdgeRefs)
	}
	if in.EdgeRef != "" {
		return [][]byte{[]byte(in.EdgeRef)}
	}
	return nil
}

func applyMidSurface(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeFeatureArgs[featureargs.MidSurface](s, raw)
	if err != nil {
		return nil, err
	}
	def := &feature.MidSurfaceDefinition{BodyIndices: in.BodyIndices, Pairs: midFacePairs(in.FacePairs)}
	if len(def.Pairs) == 0 {
		if def.MaxThickness, err = lengthValue(part, in.MaxThickness, "midSurface: maxThickness"); err != nil {
			return nil, err
		}
		if in.MinThickness != "" {
			if def.MinThickness, err = lengthValue(part, in.MinThickness, "midSurface: minThickness"); err != nil {
				return nil, err
			}
		}
	}
	pf := feature.NewMidSurfaceFeatures(part.Features()).AddMidSurface(def)
	return recomputeResult(part, pf)
}

// midFacePairs converts the wire face pairs to reference-key pairs.
func midFacePairs(pairs []featureargs.FacePair) [][2][]byte {
	if len(pairs) == 0 {
		return nil
	}
	out := make([][2][]byte, len(pairs))
	for i, pr := range pairs {
		out[i] = [2][]byte{[]byte(pr.A), []byte(pr.B)}
	}
	return out
}

func applyStitch(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeFeatureArgs[featureargs.Stitch](s, raw)
	if err != nil {
		return nil, err
	}
	tol, err := toleranceValue(part, in.Tolerance)
	if err != nil {
		return nil, err
	}
	pf := feature.NewStitchFeatures(part.Features()).Add(tol, in.MaintainAsSurface)
	return recomputeResult(part, pf)
}

func applySculpt(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeFeatureArgs[featureargs.Sculpt](s, raw)
	if err != nil {
		return nil, err
	}
	tol, err := toleranceValue(part, in.Tolerance)
	if err != nil {
		return nil, err
	}
	op, err := parseOperation(in.Operation)
	if err != nil {
		return nil, err
	}
	pf := feature.NewSculptFeatures(part.Features()).Add(op, tol)
	return recomputeResult(part, pf)
}

// toleranceValue parses a stitch/sculpt tolerance ("" ⇒ 0.1 mm = 0.01 cm).
func toleranceValue(part *compdef.PartComponentDefinition, expr string) (float64, error) {
	if strings.TrimSpace(expr) == "" {
		return 0.01, nil
	}
	v, err := lengthValue(part, expr, "tolerance")
	if err != nil {
		return 0, fmt.Errorf("invalid tolerance: %w", err)
	}
	return v, nil
}
