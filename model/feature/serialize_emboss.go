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
}

func serializeEmboss(def *EmbossDefinition, sk SketchIndexer) (*EmbossData, error) {
	idx, ok := sk.IndexOf(def.Sketch)
	if !ok {
		return nil, fmt.Errorf("emboss references a sketch that is not in the part")
	}
	d := &EmbossData{Sketch: idx, Depth: evalFloat(def.Depth), Engrave: def.Engrave, Taper: def.Taper}
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
	if d.TextEntity != 0 {
		return restoreTextEmboss(fs, skt, d)
	}
	return NewEmbossFeatures(fs).Add(skt, d.Profiles, constFloat(d.Depth), d.Engrave, d.Taper), nil
}

// restoreTextEmboss re-binds a by-reference text emboss to its sketch text entity, erroring
// if the referenced entity is missing or is not a text box.
func restoreTextEmboss(fs *PartFeatures, skt *sketch.Sketch, d *EmbossData) (*PartFeature, error) {
	e, ok := skt.EntityByID(sketch.ID(d.TextEntity))
	if !ok {
		return nil, fmt.Errorf("emboss references text entity %d, which does not exist", d.TextEntity)
	}
	tb, ok := e.(*sketch.TextBox)
	if !ok {
		return nil, fmt.Errorf("emboss references entity %d, which is a %T, not a text box", d.TextEntity, e)
	}
	return NewEmbossFeatures(fs).AddText(skt, tb, constFloat(d.Depth), d.Engrave, d.Taper), nil
}
