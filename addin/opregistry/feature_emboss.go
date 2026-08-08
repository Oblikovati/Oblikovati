// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"fmt"
	"strings"

	"oblikovati.org/api/wire/featureargs"
	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// The emboss: a closed profile (or a sketch text entity) raised off the part, engraved into it, or
// levelled to the sketch plane — and optionally WRAPPED onto a curved face rather than projected
// flat (#1893).

const embossSchema = `{
  "type": "object",
  "properties": {
    "sketchIndex": {"type": "integer", "minimum": 0},
    "profileIndices": {"type": "array", "items": {"type": "integer", "minimum": 0}, "description": "Profiles to emboss; omit to use profileIndex."},
    "profileIndex": {"type": "integer", "minimum": 0, "default": 0},
    "textEntity": {"type": "integer", "minimum": 1, "description": "Sketch text entity id to emboss BY REFERENCE; takes precedence over profile indices (text geometry is derived, never baked)."},
    "depth": {"type": "string", "description": "Raise (or, when cutting, engrave) depth, e.g. \"1 mm\"."},
    "engrave": {"type": "boolean", "default": false, "description": "Cut into the face instead of raising from it — the shorthand for type \"engraveFromFace\". Setting both is refused when they disagree."},
    "type": {"type": "string", "enum": ["fromFace", "engraveFromFace", "fromPlane"], "default": "fromFace", "description": "Emboss flavour. fromFace raises off the part, engraveFromFace cuts in, and fromPlane takes the region TO the sketch plane offset by depth — filling where the part falls short of it and trimming what stands above it, which is how a raised panel comes out flat on a wall that is not. fromPlane needs existing material and cannot wrap."},
    "wrapToFace": {"type": "string", "description": "Wrap the profile ONTO this curved face (a face reference key from get_reference_keys) instead of projecting it flat — text that follows a shaft. The sketch plane must be TANGENT to the face, so that sketch distances are the face's arc lengths. Cylindrical faces only; a cone is refused rather than distorted."},
    "taper": {"type": "string", "description": "Optional draft angle on the emboss walls, e.g. \"10 deg\", so a moulded raise releases from the tool."}
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

// parseEmbossType maps the flavour, reconciling it with the older `engrave` shorthand. The two
// disagreeing is refused: honouring one would silently invert the other's intent (#1893).
func parseEmbossType(spelling string, engrave bool) (feature.EmbossType, error) {
	switch strings.ToLower(strings.TrimSpace(spelling)) {
	case "":
		if engrave {
			return feature.EngraveFromFace, nil
		}
		return feature.EmbossFromFace, nil
	case "fromface":
		if engrave {
			return 0, errNoEngraveWithType("fromFace")
		}
		return feature.EmbossFromFace, nil
	case "engravefromface":
		return feature.EngraveFromFace, nil
	case "fromplane":
		if engrave {
			return 0, errNoEngraveWithType("fromPlane")
		}
		return feature.EmbossEngraveFromPlane, nil
	default:
		return 0, fmt.Errorf("emboss: unknown type %q (want fromFace, engraveFromFace or fromPlane)", spelling)
	}
}

// errNoEngraveWithType is the contradiction between an explicit type and the engrave shorthand.
func errNoEngraveWithType(name string) error {
	return fmt.Errorf("emboss: type %q does not engrave, but engrave is set; drop engrave, or ask "+
		"for type \"engraveFromFace\"", name)
}

// embossOptions resolves the flavour, the wall taper and the wrap face — the options every emboss
// carries whatever its profile source is (#1893).
func embossOptions(part *compdef.PartComponentDefinition, in featureargs.Emboss) (feature.EmbossType, float64, error) {
	typ, err := parseEmbossType(in.Type, in.Engrave)
	if err != nil {
		return 0, 0, err
	}
	taper, err := optionalAngleClosure(part, in.Taper, "emboss: taper")
	if err != nil {
		return 0, 0, err
	}
	return typ, callOrZeroF(taper), nil
}

// buildEmboss adds either a by-reference text emboss (when textEntity is set) or a
// profile-region emboss, so a text emboss never bakes glyph geometry into the document.
func buildEmboss(part *compdef.PartComponentDefinition, sk *sketch.Sketch, in featureargs.Emboss, depth func() float64) (*feature.PartFeature, error) {
	typ, taper, err := embossOptions(part, in)
	if err != nil {
		return nil, err
	}
	embs := feature.NewEmbossFeatures(part.Features())
	pf, err := addEmbossSource(embs, sk, in, depth, typ, taper)
	if err != nil {
		return nil, err
	}
	pf.Definition().(*feature.EmbossFeature).Definition().WrapFaceKey = []byte(in.WrapToFace)
	return pf, nil
}

// addEmbossSource adds the emboss from whichever profile source the caller named.
func addEmbossSource(embs *feature.EmbossFeatures, sk *sketch.Sketch, in featureargs.Emboss,
	depth func() float64, typ feature.EmbossType, taper float64) (*feature.PartFeature, error) {
	if in.TextEntity != 0 {
		tb, err := sketchTextBox(sk, in.TextEntity)
		if err != nil {
			return nil, err
		}
		return embs.AddText(sk, tb, depth, typ, taper), nil
	}
	profiles := in.ProfileIndices
	if len(profiles) == 0 {
		profiles = []int{in.ProfileIndex}
	}
	return embs.Add(sk, profiles, depth, typ, taper), nil
}

// sketchTextBox resolves a sketch text entity by id, so a text emboss references the text rather
// than baking its glyph geometry into the document.
func sketchTextBox(sk *sketch.Sketch, id uint64) (*sketch.TextBox, error) {
	e, ok := sk.EntityByID(sketch.ID(id))
	if !ok {
		return nil, fmt.Errorf("emboss: no entity with id %d", id)
	}
	tb, ok := e.(*sketch.TextBox)
	if !ok {
		return nil, fmt.Errorf("emboss: entity %d is a %T, not a text box", id, e)
	}
	return tb, nil
}
