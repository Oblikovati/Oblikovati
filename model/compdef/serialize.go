// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"fmt"

	"github.com/Oblikovati/oblikovati/kernel/topo"
	"github.com/Oblikovati/oblikovati/model/doc"
	"github.com/Oblikovati/oblikovati/model/feature"
	"github.com/Oblikovati/oblikovati/model/identity"
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
	Units           map[string]string         `yaml:"units,omitempty"`
	EndOfPart       *int                      `yaml:"endOfPart,omitempty"` // nil ⇒ evaluate the whole program
	Parameters      []parameterRecipe         `yaml:"parameters,omitempty"`
	ParameterGroups []string                  `yaml:"parameterGroups,omitempty"` // custom group names, in order
	WorkFeatures    []feature.WorkFeatureData `yaml:"workFeatures,omitempty"`
	Sketches        []sketch.SketchData       `yaml:"sketches,omitempty"`
	Sketches3D      []sketch.SketchData3D     `yaml:"sketches3D,omitempty"`
	Features        []feature.FeatureData     `yaml:"features,omitempty"`
	Materials       *material.RecipeData      `yaml:"materials,omitempty"`
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

// parameterRecipe is one parameter. A numeric editable parameter carries an Expression; a
// text/boolean parameter carries Text/Bool (ValueType names the flavor); a read-only
// parameter (reference/derived) carries a measured Value + Unit. The remaining fields are
// the shared presentation/behavior state (comment, key, export, precision, tolerance,
// multi-value list, group membership).
type parameterRecipe struct {
	Name        string           `yaml:"name"`
	Kind        string           `yaml:"kind"`
	ValueType   string           `yaml:"valueType,omitempty"` // "text" | "boolean"; numeric when empty
	Expression  string           `yaml:"expression,omitempty"`
	Text        string           `yaml:"text,omitempty"`
	Bool        bool             `yaml:"bool,omitempty"`
	Value       float64          `yaml:"value,omitempty"`
	Unit        string           `yaml:"unit,omitempty"`
	Comment     string           `yaml:"comment,omitempty"`
	Key         bool             `yaml:"key,omitempty"`
	Export      bool             `yaml:"export,omitempty"`
	Precision   int              `yaml:"precision,omitempty"`
	Tolerance   *toleranceRecipe `yaml:"tolerance,omitempty"`
	ExprList    []string         `yaml:"expressionList,omitempty"`
	AllowCustom bool             `yaml:"allowCustomValue,omitempty"`
	Group       string           `yaml:"group,omitempty"`
}

// toleranceRecipe is the persisted form of a non-zero engineering tolerance.
type toleranceRecipe struct {
	Upper float64 `yaml:"upper,omitempty"`
	Lower float64 `yaml:"lower,omitempty"`
	Type  uint8   `yaml:"type,omitempty"`
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
	sketches3D, err := d.sketches3D.MarshalRecipe3D()
	if err != nil {
		return nil, fmt.Errorf("compdef: marshal 3D sketches: %w", err)
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
		Units:           d.unitsRecipe(),
		Parameters:      d.parametersRecipe(),
		ParameterGroups: d.params.Groups(),
		WorkFeatures:    work,
		Sketches:        sketches,
		Sketches3D:      sketches3D,
		Features:        features,
		Materials:       d.materialsRecipe(),
	}
	if d.eop != endOfPartAtEnd {
		eop := d.eop
		r.EndOfPart = &eop
	}
	return yamlcodec.Marshal(r)
}

// RestoreRecipe replaces the part's entire recipe with the snapshot in model and
// recomputes — the undo/redo restore path. Unlike [ApplyRecipe] (which loads onto a
// fresh, empty definition and so merges additively), RestoreRecipe first resets the
// definition to empty in place, so applying a snapshot to an already-populated part
// yields exactly that snapshot rather than a union. The definition pointer is
// preserved, so the document's Content and any held reference to it stay valid.
func (d *PartComponentDefinition) RestoreRecipe(model []byte) error {
	d.resetRecipe()
	return d.ApplyRecipe(model)
}

// resetRecipe returns the definition's recipe-bearing state to the empty configuration
// a freshly constructed part has (see [NewPartComponentDefinition]), reusing the
// definition object so external references to it remain valid. Geometry is cleared too;
// ApplyRecipe's recompute rebuilds it.
func (d *PartComponentDefinition) resetRecipe() {
	d.params = param.NewParameters()
	d.keys = identity.NewKeyManager()
	d.sketches = sketch.NewSketches()
	d.sketches3D = sketch.NewSketches3D()
	d.features = feature.NewPartFeatures(d.params, d.keys)
	d.work = feature.NewWorkGeometry()
	d.units = param.DefaultUnitsOfMeasure()
	d.eop = endOfPartAtEnd
	d.assignments = material.NewAssignmentStore()
	d.assets = material.NewAssetSet()
	d.bodies = topo.NewSurfaceBodies()
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
	if err := d.applyParameters(r.ParameterGroups, r.Parameters); err != nil {
		return err
	}
	if err := feature.ApplyWork(d.work, r.WorkFeatures); err != nil {
		return fmt.Errorf("compdef: restore work features: %w", err)
	}
	if err := d.sketches.ApplyRecipe(r.Sketches); err != nil {
		return fmt.Errorf("compdef: restore sketches: %w", err)
	}
	if err := d.sketches3D.ApplyRecipe3D(r.Sketches3D); err != nil {
		return fmt.Errorf("compdef: restore 3D sketches: %w", err)
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
		out = append(out, d.parameterRecipeOf(p))
	}
	return out
}

// parameterRecipeOf captures one parameter's persisted form: its value (by flavor) plus
// the shared comment/key/export/precision/tolerance/multi-value/group state.
func (d *PartComponentDefinition) parameterRecipeOf(p *param.Parameter) parameterRecipe {
	pr := parameterRecipe{
		Name: p.Name(), Kind: p.Kind().String(),
		Comment: p.Comment, Key: p.IsKey, Export: p.ExposedAsProperty, Precision: p.Precision,
	}
	switch {
	case p.IsText():
		pr.ValueType, pr.Text = "text", p.Text()
	case p.IsBoolean():
		pr.ValueType, pr.Bool = "boolean", p.Bool()
	case p.Kind().Editable():
		pr.Expression = p.Expression()
	default:
		pr.Value, pr.Unit = p.Value().Value, p.Value().Unit.String()
	}
	if t := p.Tolerance(); t != (param.Tolerance{}) {
		pr.Tolerance = &toleranceRecipe{Upper: t.Upper, Lower: t.Lower, Type: uint8(t.Type)}
	}
	if p.IsMultiValue() {
		pr.ExprList, pr.AllowCustom = p.ExpressionList(), p.AllowsCustomValue()
	}
	if g, ok := d.params.GroupOf(p.ID()); ok {
		pr.Group = g
	}
	return pr
}

// applyParameters re-creates the custom groups (in their saved order) then re-adds each
// parameter in recipe order. A parse error (bad expression, duplicate name, unknown
// kind/unit) aborts the load rather than dropping the parameter silently.
func (d *PartComponentDefinition) applyParameters(groups []string, params []parameterRecipe) error {
	for _, g := range groups {
		if err := d.params.AddGroup(g); err != nil {
			return fmt.Errorf("compdef: restore parameter group %q: %w", g, err)
		}
	}
	for _, pr := range params {
		p, err := d.addParameter(pr)
		if err != nil {
			return fmt.Errorf("compdef: restore parameter %q: %w", pr.Name, err)
		}
		if err := d.applyParameterState(p, pr); err != nil {
			return fmt.Errorf("compdef: restore parameter %q: %w", pr.Name, err)
		}
	}
	return nil
}

// addParameter re-creates one parameter's value from its recipe entry, returning it so the
// shared state can be applied. Read-only parameters return nil (no editable state to set).
func (d *PartComponentDefinition) addParameter(pr parameterRecipe) (*param.Parameter, error) {
	switch pr.ValueType {
	case "text":
		return d.params.AddTextUserParameter(pr.Name, pr.Text)
	case "boolean":
		return d.params.AddBooleanUserParameter(pr.Name, pr.Bool)
	}
	switch pr.Kind {
	case param.UserParam.String():
		return d.params.AddUserParameter(pr.Name, pr.Expression)
	case param.ModelParam.String():
		return d.params.AddModelParameter(pr.Name, pr.Expression)
	case param.TableParam.String():
		return d.params.AddTableParameter(pr.Name, pr.Expression)
	case param.ReferenceParam.String():
		return d.addReadOnlyParameter(pr, d.params.AddReferenceParameter)
	case param.DerivedParam.String():
		return d.addReadOnlyParameter(pr, d.params.AddDerivedParameter)
	default:
		return nil, fmt.Errorf("unknown parameter kind %q (want user|model|table|reference|derived)", pr.Kind)
	}
}

// applyParameterState restores the shared presentation/behavior fields onto a freshly
// re-added parameter: comment, key, export, precision, tolerance, multi-value list, group.
func (d *PartComponentDefinition) applyParameterState(p *param.Parameter, pr parameterRecipe) error {
	p.Comment, p.IsKey, p.ExposedAsProperty, p.Precision = pr.Comment, pr.Key, pr.Export, pr.Precision
	if pr.Tolerance != nil {
		p.SetTolerance(param.Tolerance{Upper: pr.Tolerance.Upper, Lower: pr.Tolerance.Lower, Type: param.ModelValueType(pr.Tolerance.Type)})
	}
	if len(pr.ExprList) > 0 {
		if err := p.SetExpressionList(pr.ExprList, pr.AllowCustom); err != nil {
			return err
		}
	}
	if pr.Group != "" {
		return d.params.AddToGroup(p.ID(), pr.Group)
	}
	return nil
}

// addReadOnlyParameter rebuilds a read-only parameter's measured quantity from its
// value + unit and adds it through the given collection method.
func (d *PartComponentDefinition) addReadOnlyParameter(pr parameterRecipe, add func(string, param.Quantity) (*param.Parameter, error)) (*param.Parameter, error) {
	unit, ok := unitCategoryByName(pr.Unit)
	if !ok {
		return nil, fmt.Errorf("unknown unit %q", pr.Unit)
	}
	return add(pr.Name, param.Q(pr.Value, unit))
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
