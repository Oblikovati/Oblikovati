// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"fmt"

	"oblikovati.org/math"
	"oblikovati.org/model/occurrence"
)

// Persistence for the persistent occurrence patterns (#1976). A pattern is rebuilt from its
// arrangement and re-linked to the occurrences it drives — resolved by instance name, since the
// generated occurrences persist independently through the occurrence recipe. It restores AFTER the
// occurrences are bound (in ResolveReferences), and the same path serves the fast-undo snapshot.

// occurrencePatternRecipe persists one occurrence pattern: its name, the occurrences each element
// drives (element 0 = seed), the arrangement, and each element's suppress/reposition state.
type occurrencePatternRecipe struct {
	Name     string   `yaml:"name"`
	Kind     string   `yaml:"kind"`
	OccNames []string `yaml:"occurrences"`
	// Circular arrangement.
	Origin []float64 `yaml:"origin,omitempty"`
	Axis   []float64 `yaml:"axis,omitempty"`
	Angle  float64   `yaml:"angle,omitempty"`
	Count  int       `yaml:"count,omitempty"`
	// Rectangular arrangement.
	Dir1     []float64 `yaml:"dir1,omitempty"`
	Spacing1 float64   `yaml:"spacing1,omitempty"`
	Count1   int       `yaml:"count1,omitempty"`
	Dir2     []float64 `yaml:"dir2,omitempty"`
	Spacing2 float64   `yaml:"spacing2,omitempty"`
	Count2   int       `yaml:"count2,omitempty"`

	Elements []patternElementRecipe `yaml:"elements,omitempty"`
}

// patternElementRecipe persists one element's editable state: whether it is suppressed and, if it
// was repositioned off the grid, its 16-cell override transform.
type patternElementRecipe struct {
	Suppressed bool      `yaml:"suppressed,omitempty"`
	Override   []float64 `yaml:"override,omitempty"`
}

// patternsRecipe snapshots every persistent pattern for the assembly recipe.
func (a *AssemblyComponentDefinition) patternsRecipe() []occurrencePatternRecipe {
	var out []occurrencePatternRecipe
	for i := 0; i < a.patterns.Count(); i++ {
		out = append(out, patternRecipeOf(a.patterns.Item(i)))
	}
	return out
}

// patternRecipeOf renders one pattern to its recipe.
func patternRecipeOf(p *occurrence.OccurrencePattern) occurrencePatternRecipe {
	rec := occurrencePatternRecipe{Name: p.Name(), Kind: p.Kind()}
	for _, o := range p.Occurrences() {
		rec.OccNames = append(rec.OccNames, o.Name())
	}
	switch arr := p.Arrangement().(type) {
	case occurrence.CircularArrangement:
		rec.Origin, rec.Axis = point3Cells(arr.Origin), vec3Cells(arr.Axis.AsVector())
		rec.Angle, rec.Count = float64(arr.Step), arr.Count
	case occurrence.RectangularArrangement:
		rec.Dir1, rec.Spacing1, rec.Count1 = vec3Cells(arr.Dir1.AsVector()), float64(arr.Spacing1), arr.Count1
		rec.Dir2, rec.Spacing2, rec.Count2 = vec3Cells(arr.Dir2.AsVector()), float64(arr.Spacing2), arr.Count2
	}
	for i := 0; i < p.Count(); i++ {
		e := p.Element(i)
		er := patternElementRecipe{Suppressed: e.Suppressed()}
		if e.Repositioned() {
			cells := e.Transform().Cells()
			er.Override = cells[:]
		}
		rec.Elements = append(rec.Elements, er)
	}
	return rec
}

// restorePatterns rebuilds every pending pattern now that the occurrences are bound, re-linking each
// to its occurrences by name.
func (a *AssemblyComponentDefinition) restorePatterns() error {
	pending := a.pendingPatterns
	a.pendingPatterns = nil
	byName := make(map[string]*occurrence.Occurrence, a.occurrences.Count())
	for _, o := range a.occurrences.All() {
		byName[o.Name()] = o
	}
	for _, rec := range pending {
		if err := a.restorePattern(rec, byName); err != nil {
			return err
		}
	}
	return nil
}

// restorePattern rebuilds one pattern: resolve its seed + generated occurrences by name, rebuild the
// arrangement, record it, then re-apply each element's suppress/reposition state.
func (a *AssemblyComponentDefinition) restorePattern(rec occurrencePatternRecipe, byName map[string]*occurrence.Occurrence) error {
	if len(rec.OccNames) == 0 {
		return fmt.Errorf("compdef: occurrence pattern %q has no occurrences to restore", rec.Name)
	}
	seed, ok := byName[rec.OccNames[0]]
	if !ok {
		return fmt.Errorf("compdef: occurrence pattern %q seed %q was not restored", rec.Name, rec.OccNames[0])
	}
	arr, err := arrangementFromRecipe(rec)
	if err != nil {
		return err
	}
	generated := make([]*occurrence.Occurrence, 0, len(rec.OccNames)-1)
	for _, name := range rec.OccNames[1:] {
		o, ok := byName[name]
		if !ok {
			return fmt.Errorf("compdef: occurrence pattern %q element %q was not restored", rec.Name, name)
		}
		generated = append(generated, o)
	}
	pat := occurrence.NewOccurrencePattern(seed.Definition(), seed.Transform(), arr)
	a.patterns.Add(pat, rec.Name, seed, generated)
	return applyPatternElementStates(pat, rec.Elements)
}

// applyPatternElementStates re-applies the per-element suppress/reposition state onto a restored
// pattern. Its occurrences already carry their own restored state, so these calls are idempotent.
func applyPatternElementStates(pat *occurrence.OccurrencePattern, elems []patternElementRecipe) error {
	for i, er := range elems {
		if er.Suppressed {
			if err := pat.SetElementSuppressed(i, true); err != nil {
				return err
			}
		}
		if len(er.Override) == 16 {
			var cells [16]float64
			copy(cells[:], er.Override)
			if err := pat.RepositionElement(i, math.Matrix4FromCells(cells)); err != nil {
				return err
			}
		}
	}
	return nil
}

// arrangementFromRecipe rebuilds a pattern arrangement from its persisted parameters.
func arrangementFromRecipe(rec occurrencePatternRecipe) (occurrence.Arrangement, error) {
	switch rec.Kind {
	case "circular":
		axis, err := unitFromCells(rec.Axis)
		if err != nil {
			return nil, fmt.Errorf("compdef: pattern %q axis: %w", rec.Name, err)
		}
		return occurrence.CircularArrangement{Origin: pointFromCells(rec.Origin), Axis: axis, Step: math.Scalar(rec.Angle), Count: rec.Count}, nil
	case "rectangular":
		d1, err := unitFromCells(rec.Dir1)
		if err != nil {
			return nil, fmt.Errorf("compdef: pattern %q dir1: %w", rec.Name, err)
		}
		d2, err := unitFromCells(rec.Dir2)
		if err != nil {
			return nil, fmt.Errorf("compdef: pattern %q dir2: %w", rec.Name, err)
		}
		return occurrence.RectangularArrangement{Dir1: d1, Spacing1: math.Scalar(rec.Spacing1), Count1: rec.Count1, Dir2: d2, Spacing2: math.Scalar(rec.Spacing2), Count2: rec.Count2}, nil
	}
	return nil, fmt.Errorf("compdef: pattern %q has unknown arrangement kind %q", rec.Name, rec.Kind)
}

// point3Cells / vec3Cells flatten a point/vector to [x,y,z]; pointFromCells / unitFromCells rebuild.
func point3Cells(p math.Point3) []float64 { return []float64{float64(p.X), float64(p.Y), float64(p.Z)} }
func vec3Cells(v math.Vector3) []float64  { return []float64{float64(v.X), float64(v.Y), float64(v.Z)} }
func pointFromCells(c []float64) math.Point3 {
	if len(c) != 3 {
		return math.P3(0, 0, 0)
	}
	return math.P3(c[0], c[1], c[2])
}

// unitFromCells rebuilds a unit direction from [x,y,z], erroring on a zero/short vector.
func unitFromCells(c []float64) (math.UnitVector3, error) {
	if len(c) != 3 {
		return math.UnitVector3{}, fmt.Errorf("direction has %d cells, want 3", len(c))
	}
	return math.NewUnitVector3(c[0], c[1], c[2])
}
