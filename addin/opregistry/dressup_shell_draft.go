// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"oblikovati.org/api/wire/featureargs"
	"oblikovati.org/app"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
)

// The two dress-ups that reshape a body's WALLS rather than its edges: shell (hollow to a wall
// thickness, on a chosen side, with per-face overrides — #1864) and draft (taper faces about a
// pull direction). Split out of dressup.go when the shell grew its per-face thicknesses.

const shellSchema = `{
  "type": "object",
  "properties": {
    "faceRefs": {"type": "array", "items": {"type": "string"}, "minItems": 1, "description": "Reference keys of the faces to remove, hollowing the body (from get_reference_keys)."},
    "thickness": {"type": "string", "description": "Remaining wall thickness with units, e.g. \"1 mm\"."},
    "facesGeom": {"type": "array", "description": "Select the removed faces by GEOMETRY instead of faceRefs, so the binding survives recompute. Give either this or faceRefs.", "items": {"type": "object", "properties": {"centroid": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3, "description": "Face centroid [x,y,z] cm."}, "normal": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3, "description": "Outward unit normal [x,y,z]."}}, "required": ["centroid", "normal"]}},
    "direction": {"type": "string", "enum": ["inside", "outside", "both"], "default": "inside", "description": "Which side the wall grows onto: \"inside\" (outer dimensions kept), \"outside\" (outer dimensions grow by thickness), or \"both\" (wall centred on the faces). Inventor's ShellDirectionEnum."},
    "faceThicknesses": {"type": "array", "description": "Per-face wall overrides on RETAINED faces (Inventor's SetFaceThickness): a thickened boss wall or a thin window in an otherwise uniform shell. A face being removed is an opening and cannot carry one.", "items": {"type": "object", "properties": {"faceRef": {"type": "string", "description": "Reference key of the retained face (get_reference_keys)."}, "thickness": {"type": "string", "description": "That face's wall thickness, e.g. \"3 mm\"."}}, "required": ["faceRef", "thickness"]}}
  },
  "required": ["thickness"]
}`

const draftSchema = `{
  "type": "object",
  "properties": {
    "faceRefs": {"type": "array", "items": {"type": "string"}, "minItems": 1, "description": "Reference keys of the faces to draft (from get_reference_keys)."},
    "angle": {"type": "string", "description": "Draft angle with units, e.g. \"3 deg\"."},
    "pullDirection": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3, "description": "Explicit pull/parting direction [dx,dy,dz] (only its orientation matters). Omit to let the host infer it from the neutral faces."},
    "facesGeom": {"type": "array", "description": "Select the drafted faces by GEOMETRY instead of faceRefs, so the binding survives recompute. Give either this or faceRefs.", "items": {"type": "object", "properties": {"centroid": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3, "description": "Face centroid [x,y,z] cm."}, "normal": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3, "description": "Outward unit normal [x,y,z]."}}, "required": ["centroid", "normal"]}},
    "neutralPlane": {"type": "string", "description": "Fixed-plane draft: a planar face key, work plane (\"plane/N\"), or origin plane (\"origin/plane/xy\"). Faces pivot on their intersection with it (dimensions in the plane preserved); pull defaults to its normal. Inventor's kFixedPlaneFaceDraftDefinitionType."}
  },
  "required": ["angle"]
}`

func shellDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: featureargs.KindShell, Summary: "Hollow a body to a wall thickness, removing the picked faces.", Schema: json.RawMessage(shellSchema), Apply: applyShell}
}

func draftDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: featureargs.KindDraft, Summary: "Taper picked faces by a draft angle.", Schema: json.RawMessage(draftSchema), Apply: applyDraft}
}

func applyShell(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeFeatureArgs[featureargs.Shell](s, raw)
	if err != nil {
		return nil, err
	}
	if len(in.FaceRefs) == 0 && len(in.FacesGeom) == 0 {
		return nil, errors.New("shell: faceRefs is empty (give faceRefs or facesGeom)")
	}
	th, err := lengthClosure(part, in.Thickness, "shell: thickness")
	if err != nil {
		return nil, err
	}
	pf := feature.NewDressUpFeatures(part.Features()).AddShell(refKeys(in.FaceRefs), th)
	if err := bindShellWallOptions(part, pf.Definition().(*feature.ShellFeature).Definition(), in); err != nil {
		return nil, err
	}
	return recomputeResult(part, pf)
}

// bindShellWallOptions resolves the options that decide where the WALL lands: which side it grows
// onto, its per-face thicknesses (#1864), and the geometric binding of the removed faces.
func bindShellWallOptions(part *compdef.PartComponentDefinition, def *feature.ShellDefinition,
	in featureargs.Shell) error {
	dir, ok := feature.ParseShellDirection(strings.ToLower(strings.TrimSpace(in.Direction)))
	if !ok {
		return fmt.Errorf("shell: unknown direction %q (want inside|outside|both)", in.Direction)
	}
	fts, err := shellFaceThicknesses(part, in.FaceThicknesses)
	if err != nil {
		return err
	}
	def.Direction, def.FaceThicknesses = dir, fts
	if len(in.FacesGeom) == 0 {
		return nil
	}
	// Bind the removed faces by geometry when authored geometrically (survives recompute).
	refs, err := geomFaceRefs(in.FacesGeom)
	if err != nil {
		return err
	}
	def.GeomFaces = refs
	return nil
}

// shellFaceThicknesses resolves the per-face wall overrides (#1864). Each thickness is a driven
// expression like the shell's own, so a parameter change moves the thick wall with the rest.
func shellFaceThicknesses(part *compdef.PartComponentDefinition,
	in []featureargs.ShellFaceThickness) ([]feature.ShellFaceThickness, error) {
	out := make([]feature.ShellFaceThickness, 0, len(in))
	for _, ft := range in {
		if ft.FaceRef == "" {
			return nil, errors.New("shell: faceThicknesses entry has an empty faceRef")
		}
		th, err := lengthClosure(part, ft.Thickness, "shell: face thickness")
		if err != nil {
			return nil, err
		}
		out = append(out, feature.ShellFaceThickness{FaceKey: []byte(ft.FaceRef), Thickness: th})
	}
	return out, nil
}

func applyDraft(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeFeatureArgs[featureargs.Draft](s, raw)
	if err != nil {
		return nil, err
	}
	if len(in.FaceRefs) == 0 && len(in.FacesGeom) == 0 {
		return nil, errors.New("draft: faceRefs is empty (give faceRefs or facesGeom)")
	}
	a, err := angleClosure(part, in.Angle, "draft: angle")
	if err != nil {
		return nil, err
	}
	pf, err := buildDraft(part, in, a)
	if err != nil {
		return nil, err
	}
	if len(in.FacesGeom) > 0 {
		// Bind the drafted faces by geometry when authored geometrically (survives recompute).
		refs, err := geomFaceRefs(in.FacesGeom)
		if err != nil {
			return nil, err
		}
		pf.Definition().(*feature.FaceDraftFeature).Definition().GeomFaces = refs
	}
	return recomputeResult(part, pf)
}

// buildDraft adds the draft feature: a fixed-plane (neutral) draft when neutralPlane is given,
// else an explicit-pull draft (AddDraftPull) when pullDirection is given, else the host's inferred
// pull (AddDraft, default +Z).
func buildDraft(part *compdef.PartComponentDefinition, in featureargs.Draft, angle func() float64) (*feature.PartFeature, error) {
	du := feature.NewDressUpFeatures(part.Features())
	keys := refKeys(in.FaceRefs)
	if strings.TrimSpace(in.NeutralPlane) != "" {
		return buildNeutralDraft(part, du, in, keys, angle)
	}
	if len(in.PullDirection) == 0 {
		return du.AddDraft(keys, angle), nil
	}
	pull, err := vec3(in.PullDirection, "draft: pullDirection")
	if err != nil {
		return nil, err
	}
	return du.AddDraftPull(keys, pull, angle), nil
}

// buildNeutralDraft resolves the fixed neutral plane (a planar face key, "plane/N", or origin
// plane) and drafts the faces pivoting on their intersection with it — Inventor's fixed-plane
// draft. The pull defaults to the plane normal unless pullDirection overrides it. #1866.
func buildNeutralDraft(part *compdef.PartComponentDefinition, du *feature.DressUpFeatures, in featureargs.Draft, keys [][]byte, angle func() float64) (*feature.PartFeature, error) {
	wp, err := part.WorkGeometry().PlaneTargetFromRef(in.NeutralPlane)
	if err != nil {
		return nil, fmt.Errorf("draft: neutralPlane %q: %w", in.NeutralPlane, err)
	}
	pl := wp.Plane()
	neutral, err := geom.NewPlane(pl.Origin(), pl.Normal().AsVector())
	if err != nil {
		return nil, fmt.Errorf("draft: neutralPlane %q: %w", in.NeutralPlane, err)
	}
	pull := pl.Normal().AsVector()
	if len(in.PullDirection) > 0 {
		if pull, err = vec3(in.PullDirection, "draft: pullDirection"); err != nil {
			return nil, err
		}
	}
	return du.AddDraftPullNeutral(keys, pull, &neutral, angle), nil
}
