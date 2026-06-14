// SPDX-License-Identifier: GPL-2.0-only

package bom

import (
	"strconv"

	"oblikovati.org/model/occurrence"
)

// BOM is the bill of materials derived from an assembly's occurrence structure. It is
// a live view: each call to [BOM.Structured] or [BOM.PartsOnly] re-derives from the
// current occurrences, so edits (placements, suppression, structure changes) are
// reflected without a separate update step.
type BOM struct {
	root *occurrence.Occurrences
}

// New returns a BOM over an assembly's top-level occurrences.
func New(root *occurrence.Occurrences) *BOM { return &BOM{root: root} }

// ViewKind distinguishes the two standard BOM views.
type ViewKind int

const (
	// StructuredView is the hierarchical view: top-level rows with sub-assembly
	// children nested, quantities counted per parent.
	StructuredView ViewKind = iota
	// PartsOnlyView is the flat view: every unique counted part once, with its total
	// quantity across the whole assembly.
	PartsOnlyView
)

// ViewKinds returns the selectable view kinds in display order, so a UI chooser can enumerate
// them without hardcoding the set (it reads each label from [ViewKind.String]).
func ViewKinds() []ViewKind { return []ViewKind{StructuredView, PartsOnlyView} }

// String returns the chooser label for a view kind. An unknown value reports itself so a missing
// case is visible rather than silent.
func (k ViewKind) String() string {
	switch k {
	case StructuredView:
		return "Structured"
	case PartsOnlyView:
		return "Parts Only"
	default:
		return "ViewKind(" + strconv.Itoa(int(k)) + ")"
	}
}

// Row is one line of a BOM view: its item number, the component's part metadata, its
// BOM structure, the quantity at this level, and — in the structured view — its
// sub-assembly children.
type Row struct {
	ItemNumber  int
	PartNumber  string
	Description string
	Structure   Structure
	Quantity    int
	Children    []*Row
	Properties  map[string]string
	Definition  occurrence.Definition
}

// View is an ordered set of rows of one [ViewKind].
type View struct {
	Kind ViewKind
	Rows []*Row
}

// Structured returns the hierarchical view: components grouped per parent assembly,
// phantom sub-assemblies collapsed into their parent, purchased/inseparable
// sub-assemblies shown as a single un-expanded line. Item numbers are 1-based within
// each level.
func (b *BOM) Structured() *View {
	return &View{Kind: StructuredView, Rows: b.structuredRows(b.root)}
}

// PartsOnly returns the flat view: every unique counted part once, with its total
// quantity across the whole assembly. Normal and phantom sub-assemblies are traversed
// to their parts; purchased/inseparable sub-assemblies count as one part; reference
// components are excluded. Item numbers are 1-based in first-appearance order.
func (b *BOM) PartsOnly() *View {
	index := map[occurrence.Definition]int{}
	var rows []*Row
	b.walkParts(b.root, index, &rows)
	for i, r := range rows {
		r.ItemNumber = i + 1
	}
	return &View{Kind: PartsOnlyView, Rows: rows}
}

// structuredRows builds one level's rows: phantom-collapsed, grouped by definition,
// each non-leaf Normal row recursing into its children.
func (b *BOM) structuredRows(occs *occurrence.Occurrences) []*Row {
	groups := groupByDefinition(b.levelOccurrences(occs))
	rows := make([]*Row, 0, len(groups))
	for i, g := range groups {
		comp := componentOf(g.def)
		row := newRow(comp, g.def, len(g.occs))
		row.ItemNumber = i + 1
		if comp.BOMStructure().expands() {
			if sub := compositeOccurrences(g.def); sub != nil {
				row.Children = b.structuredRows(sub)
			}
		}
		rows = append(rows, row)
	}
	return rows
}

// walkParts accumulates flat part totals: it traverses expandable sub-assemblies and
// counts everything else (leaf parts and opaque purchased/inseparable sub-assemblies)
// once per instance, skipping suppressed and reference components.
func (b *BOM) walkParts(occs *occurrence.Occurrences, index map[occurrence.Definition]int, rows *[]*Row) {
	for _, o := range occs.All() {
		if o.Suppressed() {
			continue
		}
		comp := componentOf(o.Definition())
		structure := comp.BOMStructure()
		switch {
		case structure == Reference:
			continue
		case structure.expands() && o.SubOccurrences() != nil:
			b.walkParts(o.SubOccurrences(), index, rows)
		default:
			if i, ok := index[o.Definition()]; ok {
				(*rows)[i].Quantity++
			} else {
				index[o.Definition()] = len(*rows)
				*rows = append(*rows, newRow(comp, o.Definition(), 1))
			}
		}
	}
}

// levelOccurrences returns the occurrences forming one structured level: suppressed
// ones dropped, phantom sub-assemblies replaced inline by their children (the collapse).
func (b *BOM) levelOccurrences(occs *occurrence.Occurrences) []*occurrence.Occurrence {
	var out []*occurrence.Occurrence
	for _, o := range occs.All() {
		if o.Suppressed() {
			continue
		}
		if componentOf(o.Definition()).BOMStructure() == Phantom {
			if sub := o.SubOccurrences(); sub != nil {
				out = append(out, b.levelOccurrences(sub)...)
				continue
			}
		}
		out = append(out, o)
	}
	return out
}

// newRow builds a row from a component's metadata, its definition (identity), and a
// quantity. The item number is assigned by the caller.
func newRow(comp Component, def occurrence.Definition, quantity int) *Row {
	return &Row{
		PartNumber:  comp.PartNumber(),
		Description: comp.Description(),
		Structure:   comp.BOMStructure(),
		Quantity:    quantity,
		Properties:  comp.CustomProperties(),
		Definition:  def,
	}
}

// defGroup collects the occurrences of one shared definition at a level.
type defGroup struct {
	def  occurrence.Definition
	occs []*occurrence.Occurrence
}

// groupByDefinition groups occurrences sharing one definition (the flyweight),
// preserving first-appearance order so item numbering is deterministic.
func groupByDefinition(occs []*occurrence.Occurrence) []defGroup {
	index := map[occurrence.Definition]int{}
	var groups []defGroup
	for _, o := range occs {
		if i, ok := index[o.Definition()]; ok {
			groups[i].occs = append(groups[i].occs, o)
			continue
		}
		index[o.Definition()] = len(groups)
		groups = append(groups, defGroup{def: o.Definition(), occs: []*occurrence.Occurrence{o}})
	}
	return groups
}

// componentOf returns def's BOM metadata, or a Normal default when def does not
// implement [Component].
func componentOf(def occurrence.Definition) Component {
	if c, ok := def.(Component); ok {
		return c
	}
	return defaultComponent{}
}

// compositeOccurrences returns def's child occurrences when it is an assembly
// definition, or nil for a leaf part.
func compositeOccurrences(def occurrence.Definition) *occurrence.Occurrences {
	if c, ok := def.(occurrence.Composite); ok {
		return c.Occurrences()
	}
	return nil
}
