// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/api/types"
	"oblikovati.org/app"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
)

// The pattern features replicate one or more existing features (referenced by name, as shown
// in model.tree) — rectangular grid, circular array, and mirror. The source features are
// resolved to their ids; the layout is given numerically (counts, steps, axis, plane normal).

type patternArgs struct {
	SourceFeatures    []string     `json:"sourceFeatures"`
	CountX            int          `json:"countX,omitempty"`
	CountY            int          `json:"countY,omitempty"`
	CountXExpr        string       `json:"countXExpr,omitempty"`
	CountYExpr        string       `json:"countYExpr,omitempty"`
	StepX             []float64    `json:"stepX,omitempty"`
	StepY             []float64    `json:"stepY,omitempty"`
	Count             int          `json:"count,omitempty"`
	CountExpr         string       `json:"countExpr,omitempty"`
	Angle             string       `json:"angle,omitempty"`
	AxisPoint         []float64    `json:"axisPoint,omitempty"`
	AxisDir           []float64    `json:"axisDir,omitempty"`
	Normal            []float64    `json:"normal,omitempty"`
	Origin            []float64    `json:"origin,omitempty"`
	SpacingType       string       `json:"spacingType,omitempty"`
	ComputeType       string       `json:"computeType,omitempty"`
	Orientation       string       `json:"orientation,omitempty"`
	PositioningMethod string       `json:"positioningMethod,omitempty"`
	Boundary          *boundaryArg `json:"boundary,omitempty"`
}

// boundaryArg is the optional pattern-clipping boundary (M20-F18): a closed loop of 3D
// points (cm) projected into the plane through planeOrigin with planeNormal; an occurrence
// is dropped when its centre falls outside the loop.
type boundaryArg struct {
	PlaneOrigin []float64   `json:"planeOrigin,omitempty"`
	PlaneNormal []float64   `json:"planeNormal"`
	Polygon     [][]float64 `json:"polygon"`
	Inclusion   string      `json:"inclusion,omitempty"`
}

const rectPatternSchema = `{
  "type": "object",
  "properties": {
    "sourceFeatures": {"type": "array", "items": {"type": "string"}, "minItems": 1, "description": "Names of the features to replicate (see model.tree), e.g. [\"Extrusion1\"]."},
    "countX": {"type": "integer", "minimum": 1, "default": 2},
    "countY": {"type": "integer", "minimum": 1, "default": 1},
    "countXExpr": {"type": "string", "description": "Parameter-expression form of countX; when set it supersedes countX."},
    "countYExpr": {"type": "string", "description": "Parameter-expression form of countY; when set it supersedes countY."},
    "stepX": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3, "description": "Spacing vector for the first direction [x,y,z] in cm."},
    "stepY": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3, "description": "Spacing vector for the second direction [x,y,z] in cm."},
    "spacingType": {"type": "string", "enum": ["spacing", "fitted", "fitToPathLength"], "description": "How a step is read: 'spacing' = gap between occurrences (default); 'fitted' = total span divided across the count."},
    "computeType": {"type": "string", "enum": ["identical", "adjustToModel", "optimized"], "description": "How each occurrence is computed."},
    "orientation": {"type": "string", "enum": ["identical", "direction1", "direction2"], "description": "How each occurrence is oriented."},
    "positioningMethod": {"type": "string", "enum": ["fitted", "incremental"], "description": "How occurrences are positioned along a direction."},
    "boundary": {"type": "object", "description": "Clip the pattern: drop occurrences whose centre falls outside this loop.", "properties": {
      "planeOrigin": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3, "description": "Boundary plane origin [x,y,z] (default origin)."},
      "planeNormal": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3, "description": "Boundary plane normal [x,y,z]."},
      "polygon": {"type": "array", "items": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3}, "description": "Closed loop of 3D points (cm) projected into the plane."},
      "inclusion": {"type": "string", "enum": ["enclosed", "centroid", "basePoint"], "description": "Which occurrence point the boundary tests (default centroid)."}
    }, "required": ["planeNormal", "polygon"]}
  },
  "required": ["sourceFeatures", "stepX"]
}`

const circPatternSchema = `{
  "type": "object",
  "properties": {
    "sourceFeatures": {"type": "array", "items": {"type": "string"}, "minItems": 1, "description": "Names of the features to replicate (see model.tree)."},
    "count": {"type": "integer", "minimum": 1, "default": 4},
    "countExpr": {"type": "string", "description": "Parameter-expression form of count (e.g. \"slots\", \"poles/2\"); when set it supersedes count so the occurrence count tracks the parameter."},
    "angle": {"type": "string", "description": "Total sweep angle, e.g. \"360 deg\"."},
    "axisPoint": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3, "description": "A point on the rotation axis [x,y,z] (default origin)."},
    "axisDir": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3, "description": "Rotation axis direction [x,y,z] (default +Z)."},
    "spacingType": {"type": "string", "enum": ["spacing", "fitted", "fitToPathLength"], "description": "How 'angle' is read: 'spacing' = per-occurrence increment; 'fitted' = spread count across the angle inclusive; default = angle divided by count (full sweep)."},
    "computeType": {"type": "string", "enum": ["identical", "adjustToModel", "optimized"], "description": "How each occurrence is computed."},
    "orientation": {"type": "string", "enum": ["identical", "direction1", "direction2"], "description": "How each occurrence is oriented."},
    "positioningMethod": {"type": "string", "enum": ["fitted", "incremental"], "description": "How occurrences are positioned."},
    "boundary": {"type": "object", "description": "Clip the pattern: drop occurrences whose centre falls outside this loop.", "properties": {
      "planeOrigin": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3},
      "planeNormal": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3},
      "polygon": {"type": "array", "items": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3}},
      "inclusion": {"type": "string", "enum": ["enclosed", "centroid", "basePoint"]}
    }, "required": ["planeNormal", "polygon"]}
  },
  "required": ["sourceFeatures", "count", "angle"]
}`

const mirrorSchema = `{
  "type": "object",
  "properties": {
    "sourceFeatures": {"type": "array", "items": {"type": "string"}, "minItems": 1, "description": "Names of the features to mirror (see model.tree)."},
    "origin": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3, "description": "A point on the mirror plane [x,y,z] (default origin)."},
    "normal": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3, "description": "Mirror-plane normal [x,y,z], e.g. [1,0,0] for the YZ plane."}
  },
  "required": ["sourceFeatures", "normal"]
}`

func rectPatternDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: "patternRectangular", Summary: "Replicate features on a rectangular grid.", Schema: json.RawMessage(rectPatternSchema), Apply: applyRectPattern}
}

func circPatternDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: "patternCircular", Summary: "Replicate features in a circular array about an axis.", Schema: json.RawMessage(circPatternSchema), Apply: applyCircPattern}
}

func mirrorDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: "mirror", Summary: "Mirror features across a plane.", Schema: json.RawMessage(mirrorSchema), Apply: applyMirror}
}

// decodePattern resolves the active part, decoded args, and the source-feature ids common to
// every pattern operation.
func decodePattern(s *app.Session, raw json.RawMessage) (*compdef.PartComponentDefinition, patternArgs, []feature.ID, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, patternArgs{}, nil, err
	}
	var in patternArgs
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, patternArgs{}, nil, err
	}
	ids, err := featureIDsByName(part, in.SourceFeatures)
	if err != nil {
		return nil, patternArgs{}, nil, err
	}
	return part, in, ids, nil
}

func applyRectPattern(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, ids, err := decodePattern(s, raw)
	if err != nil {
		return nil, err
	}
	stepX, err := vec3(in.StepX, "patternRectangular: stepX")
	if err != nil {
		return nil, err
	}
	stepY := math.Vector3{}
	if len(in.StepY) == 3 {
		if stepY, err = vec3(in.StepY, "patternRectangular: stepY"); err != nil {
			return nil, err
		}
	}
	countX, countY, err := rectCounts(part, in)
	if err != nil {
		return nil, err
	}
	opts, err := patternOptions(in)
	if err != nil {
		return nil, err
	}
	f := feature.NewPatternFeatures(part.Features()).AddRectangular(ids, countX, countY, stepX, stepY)
	f.Definition().Options = opts
	return lastFeatureResult(part)
}

// rectCounts resolves the two grid counts as live closures, preferring the parameter-expression
// forms (countXExpr/countYExpr) over the numeric counts (#189).
func rectCounts(part *compdef.PartComponentDefinition, in patternArgs) (countX, countY func() int, err error) {
	countX, err = countClosure(part, in.CountXExpr, "patternRectangular: countXExpr", defaultInt(in.CountX, 2))
	if err != nil {
		return nil, nil, err
	}
	countY, err = countClosure(part, in.CountYExpr, "patternRectangular: countYExpr", defaultInt(in.CountY, 1))
	if err != nil {
		return nil, nil, err
	}
	return countX, countY, nil
}

func applyCircPattern(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, ids, err := decodePattern(s, raw)
	if err != nil {
		return nil, err
	}
	angle, err := angleClosure(part, in.Angle, "patternCircular: angle")
	if err != nil {
		return nil, err
	}
	count, err := countClosure(part, in.CountExpr, "patternCircular: countExpr", defaultInt(in.Count, 4))
	if err != nil {
		return nil, err
	}
	opts, err := patternOptions(in)
	if err != nil {
		return nil, err
	}
	f := feature.NewPatternFeatures(part.Features()).AddCircular(ids, count, angle, originOr(in.AxisPoint), zAxisOr(in.AxisDir))
	f.Definition().Options = opts
	return lastFeatureResult(part)
}

// patternOptions resolves the M20-F18 option fields (spacing/compute/orientation/
// positioning + boundary) from the request, defaulting to the legacy zero value.
func patternOptions(in patternArgs) (feature.PatternOptions, error) {
	o, err := patternEnumOptions(in)
	if err != nil {
		return o, err
	}
	if in.Boundary != nil {
		o.Boundary, err = buildPatternBoundary(in.Boundary)
	}
	return o, err
}

// patternEnumOptions parses the four enum option fields, rejecting a non-empty unknown.
func patternEnumOptions(in patternArgs) (feature.PatternOptions, error) {
	var o feature.PatternOptions
	var ok bool
	if in.SpacingType != "" {
		if o.Spacing, ok = types.ParsePatternSpacingType(in.SpacingType); !ok {
			return o, fmt.Errorf("pattern: unknown spacingType %q", in.SpacingType)
		}
	}
	if in.ComputeType != "" {
		if o.Compute, ok = types.ParsePatternComputeType(in.ComputeType); !ok {
			return o, fmt.Errorf("pattern: unknown computeType %q", in.ComputeType)
		}
	}
	if in.Orientation != "" {
		if o.Orientation, ok = types.ParsePatternOrientation(in.Orientation); !ok {
			return o, fmt.Errorf("pattern: unknown orientation %q", in.Orientation)
		}
	}
	if in.PositioningMethod != "" {
		if o.Positioning, ok = types.ParsePatternPositioningMethod(in.PositioningMethod); !ok {
			return o, fmt.Errorf("pattern: unknown positioningMethod %q", in.PositioningMethod)
		}
	}
	return o, nil
}

// buildPatternBoundary turns the boundary request into a model clipping boundary.
func buildPatternBoundary(b *boundaryArg) (*feature.PatternBoundary, error) {
	if len(b.PlaneNormal) != 3 {
		return nil, errors.New("pattern boundary: planeNormal needs 3 components [x,y,z]")
	}
	normal, _ := vec3(b.PlaneNormal, "pattern boundary normal")
	poly := make([]math.Point3, len(b.Polygon))
	for i, p := range b.Polygon {
		q, err := point3(p, "pattern boundary polygon vertex")
		if err != nil {
			return nil, err
		}
		poly[i] = q
	}
	incl := types.IncludeByCentroid
	if b.Inclusion != "" {
		v, ok := types.ParsePatternBoundaryInclusion(b.Inclusion)
		if !ok {
			return nil, fmt.Errorf("pattern boundary: unknown inclusion %q", b.Inclusion)
		}
		incl = v
	}
	return feature.NewPatternBoundary(originOr(b.PlaneOrigin), normal, poly, incl)
}

func applyMirror(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, ids, err := decodePattern(s, raw)
	if err != nil {
		return nil, err
	}
	if len(in.Normal) != 3 {
		return nil, errors.New("mirror: normal needs 3 components [x,y,z]")
	}
	normal, _ := vec3(in.Normal, "mirror: normal")
	feature.NewPatternFeatures(part.Features()).AddMirror(ids, nil, originOr(in.Origin), normal)
	return lastFeatureResult(part)
}

func defaultInt(v, def int) int {
	if v <= 0 {
		return def
	}
	return v
}

// countClosure turns a pattern count into a live func() int (Oblikovati.API#189): when expr is a
// parameter expression ("slots", "poles/2") it is evaluated through the part's parameter engine on
// every recompute, so the occurrence count tracks the parameter; otherwise the numeric fallback (a
// literal count, defaulted) is used as a constant. Non-positive results clamp to 1 (a pattern has
// at least the seed).
func countClosure(part *compdef.PartComponentDefinition, expr, field string, fallback int) (func() int, error) {
	if strings.TrimSpace(expr) == "" {
		return constIntFn(fallback), nil
	}
	f, err := numberClosure(part, expr, field)
	if err != nil {
		return nil, err
	}
	return func() int {
		n := int(f() + 0.5) // counts are positive — round to nearest
		if n < 1 {
			return 1
		}
		return n
	}, nil
}

func originOr(a []float64) math.Point3 {
	if len(a) == 3 {
		p, _ := point3(a, "point")
		return p
	}
	return math.P3(0, 0, 0)
}

func zAxisOr(a []float64) math.Vector3 {
	if len(a) == 3 {
		v, _ := vec3(a, "axis")
		return v
	}
	return math.V3(0, 0, 1)
}
