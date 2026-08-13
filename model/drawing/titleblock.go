// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"oblikovati.org/api/types"
	"oblikovati.org/model/attr"
)

// titleBlockField is one field of a title block: a label, and either a binding to a
// referenced-model iProperty (set+prop) or a static literal (set==""). The value is
// resolved on demand against the drawing's referenced model, so editing the model's
// iProperties updates the title block.
type titleBlockField struct {
	name   string // the field label / key
	set    string // iProperty set name; "" ⇒ static
	prop   string // iProperty name within the set
	static string // literal value when set==""
}

// source returns the field's wire source token ("Set:Prop"), or "" for a static field.
func (f titleBlockField) source() string {
	if f.set == "" {
		return ""
	}
	return f.set + ":" + f.prop
}

// TitleBlockDefinition is a reusable title-block template: an ordered set of fields.
// V1 ships one default whose fields bind to the standard iProperty sets; custom
// definitions are a follow-up.
type TitleBlockDefinition struct {
	name   string
	fields []titleBlockField
}

// DefaultTitleBlockDefinition is the standard title block: the common drafting fields,
// each bound to the model iProperty that drives it.
func DefaultTitleBlockDefinition() *TitleBlockDefinition {
	return &TitleBlockDefinition{
		name: "Default",
		fields: []titleBlockField{
			{name: "Title", set: attr.SummaryInformation, prop: "Title"},
			{name: "Part Number", set: attr.DesignTracking, prop: "Part Number"},
			{name: "Material", set: attr.DesignTracking, prop: "Material"},
			{name: "Drawn By", set: attr.SummaryInformation, prop: "Author"},
			{name: "Revision", set: attr.DesignTracking, prop: "Revision Number"},
			{name: "Company", set: attr.DocumentSummary, prop: "Company"},
		},
	}
}

// Name returns the title-block definition's name.
func (d *TitleBlockDefinition) Name() string { return d.name }

// ResolvedField is a title-block field paired with its resolved value and the source it
// resolved from (a "Set:Prop" iProperty token, or "" for static text).
type ResolvedField struct {
	Name   string
	Value  string
	Source string
}

// TitleBlock is a sheet's title block — an instance of a [TitleBlockDefinition] that
// resolves its fields against the drawing's referenced model via the lookup hook.
type TitleBlock struct {
	def      *TitleBlockDefinition
	lookup   propertyLookup
	location types.TitleBlockLocation // sheet corner it sits in (#1989); zero ⇒ bottom-right
}

func newTitleBlock(def *TitleBlockDefinition, lookup propertyLookup) *TitleBlock {
	return &TitleBlock{def: def, lookup: lookup}
}

// DefinitionName returns the name of the title-block definition this block instantiates
// (contract.DrawingTitleBlock).
func (t *TitleBlock) DefinitionName() string { return t.def.name }

// Location returns the sheet corner the title block sits in (#1989).
func (t *TitleBlock) Location() types.TitleBlockLocation { return t.location }

// SetLocation moves the title block to a sheet corner (#1989).
func (t *TitleBlock) SetLocation(l types.TitleBlockLocation) { t.location = l }

// FieldValue resolves the named field against the referenced model, returning the value
// and whether the field exists (contract.DrawingTitleBlock).
func (t *TitleBlock) FieldValue(name string) (string, bool) {
	f, ok := t.field(name)
	if !ok {
		return "", false
	}
	return t.resolve(f), true
}

// Fields returns every field resolved against the current referenced model — the title
// block as the renderer and export draw it, in definition order.
func (t *TitleBlock) Fields() []ResolvedField {
	out := make([]ResolvedField, 0, len(t.def.fields))
	for _, f := range t.def.fields {
		out = append(out, ResolvedField{Name: f.name, Value: t.resolve(f), Source: f.source()})
	}
	return out
}

func (t *TitleBlock) field(name string) (titleBlockField, bool) {
	for _, f := range t.def.fields {
		if f.name == name {
			return f, true
		}
	}
	return titleBlockField{}, false
}

// resolve returns a field's value: static text directly, or the referenced model's
// iProperty via the lookup hook (empty when no model is referenced or the property is
// absent).
func (t *TitleBlock) resolve(f titleBlockField) string {
	if f.set == "" {
		return f.static
	}
	if t.lookup == nil {
		return ""
	}
	v, _ := t.lookup(f.set, f.prop)
	return v
}
