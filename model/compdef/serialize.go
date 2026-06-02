// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"fmt"

	"github.com/Oblikovati/oblikovati/model/doc"
	"github.com/Oblikovati/oblikovati/model/feature"
	"github.com/Oblikovati/oblikovati/model/material"
	"github.com/Oblikovati/oblikovati/model/param"
	"github.com/Oblikovati/oblikovati/model/sketch"
	"github.com/Oblikovati/oblikovati/persistence/yamlcodec"
)

// init registers the real part content with the document layer so opening a part
// document reconstructs a live PartComponentDefinition (with its recipe machinery),
// not the identity-only stub (see doc.RegisterContentFactory).
func init() {
	doc.RegisterContentFactory(doc.Part, func() doc.Content { return NewPartComponentDefinition() })
}

// var assertion: a part definition is recipe-bearing content (doc.RecipeContent), so
// the store persists and restores its model on save/open.
var _ doc.RecipeContent = (*PartComponentDefinition)(nil)

// partRecipe is the YAML shape of a part's persisted recipe (ADR-0020). It is the
// document's restorable state: display units, the end-of-part marker, and the
// parameters. Sketches and features join it in later phases. The realized B-rep is
// never stored — ApplyRecipe recomputes it.
type partRecipe struct {
	Units        map[string]string         `yaml:"units,omitempty"`
	EndOfPart    *int                      `yaml:"endOfPart,omitempty"` // nil ⇒ evaluate the whole program
	Parameters   []parameterRecipe         `yaml:"parameters,omitempty"`
	WorkFeatures []feature.WorkFeatureData `yaml:"workFeatures,omitempty"`
	Sketches     []sketch.SketchData       `yaml:"sketches,omitempty"`
	Features     []feature.FeatureData     `yaml:"features,omitempty"`
	Materials    *material.RecipeData      `yaml:"materials,omitempty"`
}

// sketchIndex adapts a part's sketch collection to feature.SketchIndexer so features
// can record and re-bind their input sketch by index.
type sketchIndex struct{ sketches *sketch.Sketches }

func (si sketchIndex) IndexOf(s *sketch.Sketch) (int, bool) {
	for i := 0; i < si.sketches.Count(); i++ {
		if si.sketches.Item(i) == s {
			return i, true
		}
	}
	return 0, false
}

func (si sketchIndex) At(i int) (*sketch.Sketch, bool) {
	if i < 0 || i >= si.sketches.Count() {
		return nil, false
	}
	return si.sketches.Item(i), true
}

// parameterRecipe is one parameter: an editable parameter carries an Expression; a
// read-only parameter (reference/derived) carries a measured Value + Unit.
type parameterRecipe struct {
	Name       string  `yaml:"name"`
	Kind       string  `yaml:"kind"`
	Expression string  `yaml:"expression,omitempty"`
	Value      float64 `yaml:"value,omitempty"`
	Unit       string  `yaml:"unit,omitempty"`
}

// unitCategories are the display-unit categories a document persists, in a stable
// order. The values are stable enum ids; the names come from param.Unit.String().
var unitCategories = []param.Unit{
	param.Length, param.Angle, param.Area, param.Volume, param.Mass, param.Time,
}

// MarshalRecipe renders the part's recipe as YAML bytes (doc.RecipeContent).
func (d *PartComponentDefinition) MarshalRecipe() ([]byte, error) {
	sketches, err := d.sketches.MarshalRecipe()
	if err != nil {
		return nil, fmt.Errorf("compdef: marshal sketches: %w", err)
	}
	features, err := d.features.MarshalRecipe(sketchIndex{d.sketches})
	if err != nil {
		return nil, fmt.Errorf("compdef: marshal features: %w", err)
	}
	work, err := feature.MarshalWork(d.work)
	if err != nil {
		return nil, fmt.Errorf("compdef: marshal work features: %w", err)
	}
	r := partRecipe{
		Units:        d.unitsRecipe(),
		Parameters:   d.parametersRecipe(),
		WorkFeatures: work,
		Sketches:     sketches,
		Features:     features,
		Materials:    d.materialsRecipe(),
	}
	if d.eop != endOfPartAtEnd {
		eop := d.eop
		r.EndOfPart = &eop
	}
	return yamlcodec.Marshal(r)
}

// ApplyRecipe restores the part from recipe YAML and recomputes (doc.RecipeContent).
func (d *PartComponentDefinition) ApplyRecipe(model []byte) error {
	var r partRecipe
	if err := yamlcodec.Unmarshal(model, &r); err != nil {
		return fmt.Errorf("compdef: parse part recipe: %w", err)
	}
	if err := d.applyUnits(r.Units); err != nil {
		return err
	}
	if err := d.applyParameters(r.Parameters); err != nil {
		return err
	}
	if err := feature.ApplyWork(d.work, r.WorkFeatures); err != nil {
		return fmt.Errorf("compdef: restore work features: %w", err)
	}
	if err := d.sketches.ApplyRecipe(r.Sketches); err != nil {
		return fmt.Errorf("compdef: restore sketches: %w", err)
	}
	if err := d.features.ApplyRecipe(r.Features, sketchIndex{d.sketches}, d.work); err != nil {
		return fmt.Errorf("compdef: restore features: %w", err)
	}
	if r.Materials != nil {
		if err := material.ApplyRecipe(*r.Materials, d.assets, d.assignments); err != nil {
			return fmt.Errorf("compdef: restore materials: %w", err)
		}
	}
	if r.EndOfPart != nil {
		d.SetEndOfPart(*r.EndOfPart)
	}
	d.Recompute()
	return nil
}

// materialsRecipe captures the document's embedded assets and assignments, or nil when
// the part has neither (keeps an un-styled part's recipe minimal).
func (d *PartComponentDefinition) materialsRecipe() *material.RecipeData {
	data := material.MarshalRecipe(d.assets, d.assignments)
	if len(data.Appearances) == 0 && len(data.Materials) == 0 && data.Assignments == nil {
		return nil
	}
	return &data
}

// unitsRecipe captures the preferred display-unit name for each category.
func (d *PartComponentDefinition) unitsRecipe() map[string]string {
	out := make(map[string]string, len(unitCategories))
	for _, cat := range unitCategories {
		out[cat.String()] = d.units.PreferredName(cat)
	}
	return out
}

// applyUnits restores the preferred display unit for each named category. An unknown
// category name or an invalid unit is a corrupt-recipe error (no silent loss).
func (d *PartComponentDefinition) applyUnits(units map[string]string) error {
	for name, unitName := range units {
		cat, ok := unitCategoryByName(name)
		if !ok {
			return fmt.Errorf("compdef: unknown unit category %q in recipe", name)
		}
		if err := d.units.SetPreferred(cat, unitName); err != nil {
			return fmt.Errorf("compdef: restore units: %w", err)
		}
	}
	return nil
}

// parametersRecipe captures every parameter in creation order (a valid order to
// re-add: an expression can only reference parameters created before it).
func (d *PartComponentDefinition) parametersRecipe() []parameterRecipe {
	all := d.params.All()
	if len(all) == 0 {
		return nil
	}
	out := make([]parameterRecipe, 0, len(all))
	for _, p := range all {
		pr := parameterRecipe{Name: p.Name(), Kind: p.Kind().String()}
		if p.Kind().Editable() {
			pr.Expression = p.Expression()
		} else {
			pr.Value, pr.Unit = p.Value().Value, p.Value().Unit.String()
		}
		out = append(out, pr)
	}
	return out
}

// applyParameters re-adds each parameter in recipe order. A parse error (bad
// expression, duplicate name, unknown kind/unit) aborts the load rather than dropping
// the parameter silently.
func (d *PartComponentDefinition) applyParameters(params []parameterRecipe) error {
	for _, pr := range params {
		if err := d.addParameter(pr); err != nil {
			return fmt.Errorf("compdef: restore parameter %q: %w", pr.Name, err)
		}
	}
	return nil
}

// addParameter re-creates one parameter from its recipe entry.
func (d *PartComponentDefinition) addParameter(pr parameterRecipe) error {
	switch pr.Kind {
	case param.UserParam.String():
		_, err := d.params.AddUserParameter(pr.Name, pr.Expression)
		return err
	case param.ModelParam.String():
		_, err := d.params.AddModelParameter(pr.Name, pr.Expression)
		return err
	case param.TableParam.String():
		_, err := d.params.AddTableParameter(pr.Name, pr.Expression)
		return err
	case param.ReferenceParam.String():
		return d.addReadOnlyParameter(pr, d.params.AddReferenceParameter)
	case param.DerivedParam.String():
		return d.addReadOnlyParameter(pr, d.params.AddDerivedParameter)
	default:
		return fmt.Errorf("unknown parameter kind %q (want user|model|table|reference|derived)", pr.Kind)
	}
}

// addReadOnlyParameter rebuilds a read-only parameter's measured quantity from its
// value + unit and adds it through the given collection method.
func (d *PartComponentDefinition) addReadOnlyParameter(pr parameterRecipe, add func(string, param.Quantity) (*param.Parameter, error)) error {
	unit, ok := unitCategoryByName(pr.Unit)
	if !ok {
		return fmt.Errorf("unknown unit %q", pr.Unit)
	}
	_, err := add(pr.Name, param.Q(pr.Value, unit))
	return err
}

// unitCategoryByName maps a category name (param.Unit.String()) back to its Unit.
func unitCategoryByName(name string) (param.Unit, bool) {
	for _, cat := range unitCategories {
		if cat.String() == name {
			return cat, true
		}
	}
	return 0, false
}
