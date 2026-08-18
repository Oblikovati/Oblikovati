// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/model/sketch"
)

// EmbossData is an emboss's recipe: its profile SOURCE (either closed sketch regions —
// Sketch + Profiles — OR a sketch text entity referenced by TextEntity), the depth, whether
// it engraves (cut) instead of raises (join), and a draft taper. When TextEntity is set the
// emboss stores ONLY the reference (no baked glyph geometry); its outlines are re-derived on
// load (the bug fix: embossed text must not bake geometry into the .obk).
type EmbossData struct {
	Sketch     int     `yaml:"sketch"`
	Profiles   []int   `yaml:"profiles,omitempty"`
	TextEntity uint64  `yaml:"textEntity,omitempty"` // sketch text entity id (by-reference text)
	Depth      float64 `yaml:"depth"`
	Engrave    bool    `yaml:"engrave,omitempty"`
	Taper      float64 `yaml:"taper,omitempty"`
	// Type carries the flavour only when it is NOT one of the two the `engrave` flag already
	// spells (#1893), so a document that predates the flavours round-trips byte for byte and an
	// older build still reads a raise/engrave written by this one. Wrap is the wrap face's key.
	Type string `yaml:"type,omitempty"`
	Wrap string `yaml:"wrapFace,omitempty"`
}

// embossFromPlaneName is the persisted spelling of the one flavour the legacy `engrave` flag
// cannot express.
const embossFromPlaneName = "fromPlane"

// embossTypeData splits a flavour into the legacy engrave flag and the new type key, writing the
// key only when the flag cannot carry the flavour.
func embossTypeData(t EmbossType) (engrave bool, name string) {
	switch t {
	case EngraveFromFace:
		return true, ""
	case EmbossEngraveFromPlane:
		return false, embossFromPlaneName
	default:
		return false, ""
	}
}

// embossTypeFromData reads the flavour back, the type key winning over the legacy flag. An unknown
// key is a precise error rather than a silent fall back to a raise, which would build the opposite
// of a cut.
func embossTypeFromData(d *EmbossData) (EmbossType, error) {
	switch d.Type {
	case "":
		if d.Engrave {
			return EngraveFromFace, nil
		}
		return EmbossFromFace, nil
	case embossFromPlaneName:
		return EmbossEngraveFromPlane, nil
	default:
		return 0, fmt.Errorf("emboss: unknown type %q in the recipe (want %q or omitted, the "+
			"engrave flag then choosing raise or cut)", d.Type, embossFromPlaneName)
	}
}

func serializeEmboss(def *EmbossDefinition, sk SketchIndexer) (*EmbossData, error) {
	idx, ok := sk.IndexOf(def.Sketch)
	if !ok {
		return nil, fmt.Errorf("emboss references a sketch that is not in the part")
	}
	engrave, name := embossTypeData(def.Type)
	d := &EmbossData{Sketch: idx, Depth: evalFloat(def.Depth), Engrave: engrave, Taper: def.Taper,
		Type: name, Wrap: encodeKey(def.WrapFaceKey)}
	if def.Text != nil {
		d.TextEntity = uint64(def.Text.EntityID())
		return d, nil
	}
	d.Profiles = append([]int(nil), def.ProfileIndices...)
	return d, nil
}

func restoreEmboss(fs *PartFeatures, d *EmbossData, sk SketchIndexer) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("emboss feature is missing its payload")
	}
	skt, ok := sk.At(d.Sketch)
	if !ok {
		return nil, fmt.Errorf("emboss references sketch index %d, which does not exist", d.Sketch)
	}
	typ, err := embossTypeFromData(d)
	if err != nil {
		return nil, err
	}
	wrap, err := decodeKey(d.Wrap)
	if err != nil {
		return nil, fmt.Errorf("emboss wrapFace: %w", err)
	}
	if d.TextEntity != 0 {
		return restoreTextEmboss(fs, skt, d, typ, wrap)
	}
	pf := NewEmbossFeatures(fs).Add(skt, d.Profiles, constFloat(d.Depth), typ, d.Taper)
	pf.Definition().(*EmbossFeature).Definition().WrapFaceKey = wrap
	return pf, nil
}

// restoreTextEmboss re-binds a by-reference text emboss to its sketch text entity, erroring
// if the referenced entity is missing or is not a text box.
func restoreTextEmboss(fs *PartFeatures, skt *sketch.Sketch, d *EmbossData, typ EmbossType,
	wrap []byte) (*PartFeature, error) {
	e, ok := skt.EntityByID(sketch.ID(d.TextEntity))
	if !ok {
		return nil, fmt.Errorf("emboss references text entity %d, which does not exist", d.TextEntity)
	}
	tb, ok := e.(*sketch.TextBox)
	if !ok {
		return nil, fmt.Errorf("emboss references entity %d, which is a %T, not a text box", d.TextEntity, e)
	}
	pf := NewEmbossFeatures(fs).AddText(skt, tb, constFloat(d.Depth), typ, d.Taper)
	pf.Definition().(*EmbossFeature).Definition().WrapFaceKey = wrap
	return pf, nil
}
