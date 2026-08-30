// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"oblikovati.org/api/types"
)

// Drawing-recipe — the SKETCH section (M48 #2226 split of recipe.go). The YAML shape of a sheet's
// drawing sketches (entities + hatch regions; the curves re-derive from the entities on open) and
// the snapshot/restore of that section.

// sketchRecipeItem is the YAML shape of one drawing sketch: its name, entities and hatch regions
// (the curves re-derive from them on open).
type sketchRecipeItem struct {
	Name     string               `yaml:"name"`
	Entities []sketchEntityRecipe `yaml:"entities,omitempty"`
	Hatches  []hatchRecipe        `yaml:"hatches,omitempty"`
}

// hatchRecipe is the YAML shape of one hatch region.
type hatchRecipe struct {
	X       float64 `yaml:"xmm"`
	Y       float64 `yaml:"ymm"`
	W       float64 `yaml:"widthMm"`
	H       float64 `yaml:"heightMm"`
	Pattern string  `yaml:"pattern,omitempty"`
	Spacing float64 `yaml:"spacingMm,omitempty"`
}

// sketchEntityRecipe is the YAML shape of one drawing-sketch entity.
type sketchEntityRecipe struct {
	Kind   string       `yaml:"kind"`
	Points [][2]float64 `yaml:"points,omitempty"`
	Radius float64      `yaml:"radiusMm,omitempty"`
}

// sketchRecipesOf snapshots a sheet's drawing sketches for persistence (their curves re-derive from
// the entities on open).
func sketchRecipesOf(sh *Sheet) []sketchRecipeItem {
	if sh.sketches == nil {
		return nil
	}
	out := make([]sketchRecipeItem, 0, len(sh.sketches.items))
	for _, s := range sh.sketches.items {
		rec := sketchRecipeItem{Name: s.name}
		for _, e := range s.entities {
			rec.Entities = append(rec.Entities, sketchEntityRecipe{Kind: e.kind.String(), Points: e.points, Radius: e.radius})
		}
		for _, h := range s.hatches {
			rec.Hatches = append(rec.Hatches, hatchRecipe{X: h.x, Y: h.y, W: h.w, H: h.h, Pattern: h.pattern.String(), Spacing: h.spacing})
		}
		out = append(out, rec)
	}
	return out
}

// restoreSketches rebuilds a sheet's drawing sketches from their recipe; each sketch's curves
// re-derive from its entities.
func restoreSketches(sh *Sheet, recs []sketchRecipeItem) {
	if len(recs) == 0 {
		return
	}
	ss := sh.Sketches()
	for _, sr := range recs {
		s := ss.Add(sr.Name)
		for _, er := range sr.Entities {
			kind, _ := types.ParseDrawingSketchEntityKind(er.Kind)
			s.entities = append(s.entities, DrawingSketchEntity{kind: kind, points: er.Points, radius: er.Radius})
		}
		for _, hr := range sr.Hatches {
			pattern, _ := types.ParseHatchPattern(hr.Pattern)
			s.hatches = append(s.hatches, hatchRegion{x: hr.X, y: hr.Y, w: hr.W, h: hr.H, pattern: pattern, spacing: hr.Spacing})
		}
		s.rebuild()
	}
}
