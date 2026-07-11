// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"fmt"
	"strconv"

	"oblikovati.org/api/types"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/model/attr"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/material"
	"oblikovati.org/model/param"
	"oblikovati.org/model/sketch"
	"oblikovati.org/yamlcodec"
)

// The real part content reaches the document layer through the composition root
// (model/contentset.Default → doc.NewWorkspace), not init()-time registration
// (#1617): opening a part document reconstructs a live PartComponentDefinition
// because the workspace was CONSTRUCTED with that factory.

// var assertion: a part definition is recipe-bearing content (doc.RecipeContent), so
// the store persists and restores its model on save/open.
var _ doc.RecipeContent = (*PartComponentDefinition)(nil)

// partRecipe is the YAML shape of a part's persisted recipe (ADR-0020). It is the
// document's restorable state: display units, the end-of-part marker, and the
// parameters. Sketches and features join it in later phases. The realized B-rep is
// never stored — ApplyRecipe recomputes it.
type partRecipe struct {
	Units             map[string]string            `yaml:"units,omitempty"`
	EndOfPart         *int                         `yaml:"endOfPart,omitempty"` // nil ⇒ evaluate the whole program
	Parameters        []parameterRecipe            `yaml:"parameters,omitempty"`
	ParameterGroups   []parameterGroupRecipe       `yaml:"parameterGroups,omitempty"` // custom group records, in creation order
	ParameterSettings *parameterSettingsRecipe     `yaml:"parameterSettings,omitempty"`
	DerivedTables     []derivedTableRecipe         `yaml:"derivedParameterTables,omitempty"`
	WorkFeatures      []feature.WorkFeatureData    `yaml:"workFeatures,omitempty"`
	BlockDefinitions  []sketch.BlockDefinitionData `yaml:"blockDefinitions,omitempty"`
	Sketches          []sketch.SketchData          `yaml:"sketches,omitempty"`
	Sketches3D        []sketch.SketchData3D        `yaml:"sketches3D,omitempty"`
	Features          []feature.FeatureData        `yaml:"features,omitempty"`
	Materials         *material.RecipeData         `yaml:"materials,omitempty"`
	Properties        []propertyRecipe             `yaml:"properties,omitempty"` // document iProperties (#156)
	SheetMetal        *sheetMetalRecipe            `yaml:"sheetMetal,omitempty"` // sheet-metal rule (M13-F01); nil for ordinary parts
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
	Name       string           `yaml:"name"`
	Kind       string           `yaml:"kind"`
	ValueType  string           `yaml:"valueType,omitempty"` // "text" | "boolean"; numeric when empty
	Expression string           `yaml:"expression,omitempty"`
	Text       string           `yaml:"text,omitempty"`
	Bool       bool             `yaml:"bool,omitempty"`
	Value      float64          `yaml:"value,omitempty"`
	Unit       string           `yaml:"unit,omitempty"`
	Comment    string           `yaml:"comment,omitempty"`
	Key        bool             `yaml:"key,omitempty"`
	Export     bool             `yaml:"export,omitempty"`
	Precision  int              `yaml:"precision,omitempty"`
	Tolerance  *toleranceRecipe `yaml:"tolerance,omitempty"`
	// ModelValueType is the tolerance-band selection's wire spelling; empty
	// means nominal (Oblikovati#607).
	ModelValueType string   `yaml:"modelValueType,omitempty"`
	ExprList       []string `yaml:"expressionList,omitempty"`
	AllowCustom    bool     `yaml:"allowCustomValue,omitempty"`
	// SortedValueList is the inverse of the in-memory CustomOrder flag: a saved
	// multi-value list is authored-order by default, so only auto-sorted lists
	// persist a marker.
	SortedValueList bool `yaml:"sortedValueList,omitempty"`
	// Hidden is the inverse of Visible (parameters default to visible, and the
	// recipe omits zero values).
	Hidden bool `yaml:"hidden,omitempty"`
	// DisplayFormat is the wire spelling; empty means decimal.
	DisplayFormat  string                `yaml:"displayFormat,omitempty"`
	CustomProperty *customPropertyRecipe `yaml:"customProperty,omitempty"`
	// Renamed marks a model parameter renamed from its generated name; DisabledActions
	// is the list of restricted edit-action spellings (types.ActionType.Names()). Both
	// are sparse (omitted when default), the parameter-introspection state (#1853).
	Renamed         bool     `yaml:"renamed,omitempty"`
	DisabledActions []string `yaml:"disabledActions,omitempty"`
}

// toleranceRecipe is the persisted form of a non-zero engineering tolerance:
// the flavor's wire spelling plus the deviation band in database units, and the
// ISO fit class strings for a fits tolerance (#1848).
type toleranceRecipe struct {
	Type           string  `yaml:"type,omitempty"`
	Upper          float64 `yaml:"upper,omitempty"`
	Lower          float64 `yaml:"lower,omitempty"`
	HoleTolerance  string  `yaml:"holeTolerance,omitempty"`
	ShaftTolerance string  `yaml:"shaftTolerance,omitempty"`
}

// parameterSettingsRecipe persists the document-level parameter settings when
// they differ from the defaults (M02-F07, Oblikovati#606). DimensionDisplayType
// carries the wire spelling.
type parameterSettingsRecipe struct {
	LinearStandardTolerance      string `yaml:"linearStandardTolerance,omitempty"`
	AngularStandardTolerance     string `yaml:"angularStandardTolerance,omitempty"`
	UseStandardTolerances        bool   `yaml:"useStandardTolerances,omitempty"`
	ExportStandardTolerances     bool   `yaml:"exportStandardTolerances,omitempty"`
	LinearDimensionPrecision     int    `yaml:"linearDimensionPrecision"`
	AngularDimensionPrecision    int    `yaml:"angularDimensionPrecision"`
	DimensionDisplayType         string `yaml:"dimensionDisplayType,omitempty"`
	DisplayParameterAsExpression bool   `yaml:"displayParameterAsExpression,omitempty"`
}

// derivedTableRecipe persists one derived parameter table: its stable id (so
// undo restores keep wire handles valid), the source document, and the linked
// names. The produced derived parameters persist in the parameters list and
// reconnect by name on restore (M02-F06, Oblikovati#605).
type derivedTableRecipe struct {
	ID             int      `yaml:"id"`
	SourceDocument string   `yaml:"sourceDocument"`
	Linked         []string `yaml:"linked,omitempty"`
	OwnedByFeature bool     `yaml:"ownedByFeature,omitempty"`
}

// parameterGroupRecipe persists one custom parameter group: the immutable
// internal name, the display name when it differs, the owning client id, and
// the member parameter names (M02-F05, Oblikovati#604).
type parameterGroupRecipe struct {
	InternalName string   `yaml:"internalName"`
	DisplayName  string   `yaml:"displayName,omitempty"`
	ClientID     string   `yaml:"clientId,omitempty"`
	Members      []string `yaml:"members,omitempty"`
}

// customPropertyRecipe persists a non-default custom-property format (the
// formatting of a parameter exposed as a document property). Enum fields carry
// wire spellings.
type customPropertyRecipe struct {
	PropertyType      string `yaml:"propertyType,omitempty"`
	Units             string `yaml:"units,omitempty"`
	Precision         string `yaml:"precision,omitempty"`
	ShowLeadingZeros  bool   `yaml:"showLeadingZeros,omitempty"`
	ShowTrailingZeros bool   `yaml:"showTrailingZeros,omitempty"`
	ShowUnitsString   bool   `yaml:"showUnitsString,omitempty"`
}

// unitCategories are the display-unit categories a document persists, in a stable
// order. The values are stable enum ids; the names come from param.Unit.String().
var unitCategories = []param.Unit{
	param.Length, param.Angle, param.Area, param.Volume, param.Mass, param.Time,
}

// MarshalRecipe renders the part's recipe as YAML bytes (doc.RecipeContent).
func (d *PartComponentDefinition) MarshalRecipe() ([]byte, error) {
	r, err := d.buildRecipe()
	if err != nil {
		return nil, err
	}
	return yamlcodec.Marshal(r)
}

// buildRecipe captures the part's full parametric state as a [partRecipe] value, the shared
// step behind both the YAML save ([MarshalRecipe]) and the fast undo snapshot
// ([MarshalSnapshot]) — so a snapshot can re-use a faster codec without re-deriving the recipe.
func (d *PartComponentDefinition) buildRecipe() (partRecipe, error) {
	var r partRecipe
	// The fallible sub-marshals share one error wrap (each names its section), so a sub-marshal
	// failure does not need five near-identical branches.
	for _, step := range []struct {
		what string
		run  func() error
	}{
		{"sketches", func() (err error) { r.Sketches, err = d.sketches.MarshalRecipe(); return }},
		{"block definitions", func() (err error) { r.BlockDefinitions, err = d.sketches.MarshalBlockDefinitions(); return }},
		{"3D sketches", func() (err error) { r.Sketches3D, err = d.sketches3D.MarshalRecipe3D(); return }},
		{"features", func() (err error) { r.Features, err = d.features.MarshalRecipe(sketchIndex{d.sketches}); return }},
		{"work features", func() (err error) { r.WorkFeatures, err = feature.MarshalWork(d.work); return }},
	} {
		if err := step.run(); err != nil {
			return partRecipe{}, fmt.Errorf("compdef: marshal %s: %w", step.what, err)
		}
	}
	r.Units = d.unitsRecipe()
	r.Parameters = parametersRecipeOf(d.params)
	r.ParameterGroups = parameterGroupsRecipeOf(d.params)
	r.ParameterSettings = parameterSettingsRecipeOf(d.params)
	r.DerivedTables = derivedTablesRecipeOf(d.params)
	r.Materials = d.materialsRecipe()
	r.Properties = propertiesRecipeOf(d.props)
	r.SheetMetal = d.sheetMetalRecipeOf()
	if d.eop != endOfPartAtEnd {
		eop := d.eop
		r.EndOfPart = &eop
	}
	return r, nil
}

// resetRecipe returns the definition's recipe-bearing state to the empty configuration
// a freshly constructed part has (see [NewPartComponentDefinition]), reusing the
// definition object so external references to it remain valid. Geometry is cleared too;
// ApplyRecipe's recompute rebuilds it.
func (d *PartComponentDefinition) resetRecipe() {
	d.params = param.NewParameters()
	d.sketches = sketch.NewSketches()
	d.sketches3D = sketch.NewSketches3D()
	d.features = feature.NewPartFeatures(d.params)
	d.features.SetResourceStore(d) // re-wire after the engine is recreated; resources survive the reset
	d.features.SetFontResolver(d)  // re-wire the document font resolver too
	d.work = feature.NewWorkGeometry()
	d.units = param.DefaultUnitsOfMeasure()
	d.eop = endOfPartAtEnd
	d.assignments = material.NewAssignmentStore()
	d.assets = material.NewAssetSet()
	d.bodies = topo.NewSurfaceBodies()
	d.props = attr.NewPropertySets()
	d.sheetMetal = nil // re-derived from the recipe's sheetMetal section on restore
}

// ApplyRecipe restores the part from recipe YAML and recomputes (doc.RecipeContent).
func (d *PartComponentDefinition) ApplyRecipe(model []byte) error {
	var r partRecipe
	if err := yamlcodec.Unmarshal(model, &r); err != nil {
		return fmt.Errorf("compdef: parse part recipe: %w", err)
	}
	return d.applyRecipeStruct(r)
}

// applyRecipeStruct restores the part from an already-decoded [partRecipe] and recomputes —
// the shared tail of [ApplyRecipe] (YAML) and [RestoreSnapshot] (the fast undo codec), so both
// decode formats converge on one apply path.
func (d *PartComponentDefinition) applyRecipeStruct(r partRecipe) error {
	if err := d.applyUnits(r.Units); err != nil {
		return err
	}
	applyPropertiesRecipe(d.props, r.Properties)
	if err := applyParametersTo(d.params, r.ParameterGroups, r.Parameters); err != nil {
		return err
	}
	if err := applyParameterSettingsTo(d.params, r.ParameterSettings); err != nil {
		return err
	}
	if err := applyDerivedTablesTo(d.params, r.DerivedTables); err != nil {
		return err
	}
	if err := feature.ApplyWork(d.work, r.WorkFeatures); err != nil {
		return fmt.Errorf("compdef: restore work features: %w", err)
	}
	// Definitions restore before sketches: instances re-bind by name.
	if err := d.sketches.ApplyBlockDefinitions(r.BlockDefinitions); err != nil {
		return fmt.Errorf("compdef: restore block definitions: %w", err)
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
	if err := d.applySheetMetalRecipe(r.SheetMetal); err != nil {
		return fmt.Errorf("compdef: restore sheet-metal rule: %w", err)
	}
	if r.EndOfPart != nil {
		d.SetEndOfPart(*r.EndOfPart)
	}
	d.rebindSketchProjections()   // re-attach live sources to restored projections before recompute (#1268)
	d.rebindSketch3DConstraints() // and to restored surface-bound 3D constraints (onFace, #1839)
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
	return unitsRecipeFor(d.units)
}

// applyUnits restores the preferred display unit for each named category.
func (d *PartComponentDefinition) applyUnits(units map[string]string) error {
	return applyUnitsTo(&d.units, units)
}

// Reserved units-recipe keys carry the display precision/format alongside the
// per-category unit names in the same map. They are NOT category names, so old
// recipes that lack them simply restore the defaults.
const (
	keyLengthPrecision = "lengthPrecision"
	keyAnglePrecision  = "anglePrecision"
	keyLengthFormat    = "lengthFormat"
	keyAngleFormat     = "angleFormat"
	keyWorkingScale    = "workingScale" // ADR-0042 Phase 2: cm size of one stored working length unit
)

// unitsRecipeFor captures the preferred display-unit name for each category plus
// the display precision/format — shared by the part and assembly recipes, which
// persist units identically.
func unitsRecipeFor(u param.UnitsOfMeasure) map[string]string {
	out := make(map[string]string, len(unitCategories)+4)
	for _, cat := range unitCategories {
		out[cat.String()] = u.PreferredName(cat)
	}
	out[keyLengthPrecision] = strconv.Itoa(u.LengthPrecision())
	out[keyAnglePrecision] = strconv.Itoa(u.AnglePrecision())
	out[keyLengthFormat] = u.LengthFormat().String()
	out[keyAngleFormat] = angleFormatName(u.AngleFormat())
	// Persist the working scale only when it is not the centimetre default (ADR-0042
	// Phase 2). Omitting it keeps every existing cm document's .obk byte-identical and
	// makes the migration automatic: a recipe with no key restores the cm default — which
	// is exactly what a pre-Phase-2 (cm-stored) document is.
	if ws := u.WorkingScale(); ws != 1 {
		out[keyWorkingScale] = strconv.FormatFloat(ws, 'g', -1, 64)
	}
	return out
}

// applyUnitsTo restores the preferred display unit for each named category plus
// the precision/format reserved keys onto u. An unknown category name or an
// invalid value is a corrupt-recipe error (no silent loss).
func applyUnitsTo(u *param.UnitsOfMeasure, units map[string]string) error {
	for name, val := range units {
		handled, err := applyReservedUnitKey(u, name, val)
		if err != nil {
			return err
		}
		if handled {
			continue
		}
		cat, ok := unitCategoryByName(name)
		if !ok {
			return fmt.Errorf("compdef: unknown unit category %q in recipe", name)
		}
		if err := u.SetPreferred(cat, val); err != nil {
			return fmt.Errorf("compdef: restore units: %w", err)
		}
	}
	return nil
}

// applyReservedUnitKey applies a precision/format reserved key, reporting whether
// the key was one (so the category path is skipped) and any parse/validation error.
func applyReservedUnitKey(u *param.UnitsOfMeasure, name, val string) (bool, error) {
	switch name {
	case keyLengthPrecision, keyAnglePrecision:
		n, err := strconv.Atoi(val)
		if err != nil {
			return true, fmt.Errorf("compdef: %s %q is not an integer: %w", name, val, err)
		}
		set := u.SetLengthPrecision
		if name == keyAnglePrecision {
			set = u.SetAnglePrecision
		}
		return true, set(n)
	case keyLengthFormat:
		f, ok := types.ParseParameterDisplayFormat(val)
		if !ok {
			return true, fmt.Errorf("compdef: unknown length display format %q in recipe", val)
		}
		u.SetLengthFormat(f)
		return true, nil
	case keyAngleFormat:
		f, ok := parseAngleFormat(val)
		if !ok {
			return true, fmt.Errorf("compdef: unknown angle display format %q in recipe", val)
		}
		u.SetAngleFormat(f)
		return true, nil
	case keyWorkingScale:
		ws, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return true, fmt.Errorf("compdef: working scale %q is not a number: %w", val, err)
		}
		return true, u.SetWorkingScale(ws)
	}
	return false, nil
}

// angleFormatName / parseAngleFormat are the recipe spellings of param.AngleFormat.
func angleFormatName(f param.AngleFormat) string {
	if f == param.AngleDMS {
		return "dms"
	}
	return "decimal"
}

func parseAngleFormat(s string) (param.AngleFormat, bool) {
	switch s {
	case "decimal":
		return param.AngleDecimal, true
	case "dms":
		return param.AngleDMS, true
	}
	return param.AngleDecimal, false
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
