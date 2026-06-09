// SPDX-License-Identifier: GPL-2.0-only

// Package step is the AP203/214/242 reader/writer facade over the part21, schema,
// geommap and topomap layers. units.go resolves a file's length unit into the
// scale factor that converts file lengths into the kernel's database unit (mm).
package step

import (
	"fmt"
	"strings"

	"oblikovati.org/kernel/exchange/step/part21"
)

// mmPerLengthUnit returns the scale that converts a file length into millimeters.
// It finds the length unit declared by the GLOBAL_UNIT_ASSIGNED_CONTEXT (directly,
// or via the (… GEOMETRIC_REPRESENTATION_CONTEXT …) complex instance) and reads it.
// When no unit is found it defaults to 1.0 (assume mm) and signals so via the bool,
// letting the caller warn.
func mmPerLengthUnit(g *part21.EntityGraph) (float64, bool, error) {
	unitID, ok := findLengthUnit(g)
	if !ok {
		return 1.0, false, nil
	}
	scale, err := lengthUnitScale(g, unitID)
	return scale, true, err
}

// findLengthUnit returns the id of the length-bearing unit entity (an SI_UNIT or
// CONVERSION_BASED_UNIT carrying LENGTH_UNIT) referenced by any unit-assigning
// context. Returns ok=false when none is present.
func findLengthUnit(g *part21.EntityGraph) (int, bool) {
	for _, id := range g.IDs() {
		ent, _ := g.Lookup(id)
		for _, ref := range unitsOf(ent) {
			if isLengthUnit(g, ref) {
				return ref, true
			}
		}
	}
	return 0, false
}

// unitsOf returns the unit references a context entity assigns, scanning both the
// simple GLOBAL_UNIT_ASSIGNED_CONTEXT and the complex representation-context form.
func unitsOf(ent *part21.RawEntity) []int {
	if ent.Keyword == "GLOBAL_UNIT_ASSIGNED_CONTEXT" {
		return refsInFirstList(ent.Params)
	}
	for _, c := range ent.Components {
		if c.Keyword == "GLOBAL_UNIT_ASSIGNED_CONTEXT" {
			return refsInFirstList(c.Params)
		}
	}
	return nil
}

// refsInFirstList collects the entity references inside the first list parameter
// (the units set of a unit-assigned context).
func refsInFirstList(params []part21.Value) []int {
	if len(params) == 0 {
		return nil
	}
	items, err := params[0].AsList()
	if err != nil {
		return nil
	}
	var out []int
	for _, item := range items {
		if id, err := item.AsRef(); err == nil {
			out = append(out, id)
		}
	}
	return out
}

// isLengthUnit reports whether the referenced unit is a length unit (an SI_UNIT
// with name METRE, or a CONVERSION_BASED_UNIT whose dimensions are length).
func isLengthUnit(g *part21.EntityGraph, id int) bool {
	ent, err := g.Lookup(id)
	if err != nil {
		return false
	}
	switch ent.Keyword {
	case "SI_UNIT":
		return hasComponentNamed(ent, "METRE") || enumContains(ent.Params, "METRE")
	case "CONVERSION_BASED_UNIT":
		return true // its target is the SI metre; the reader trusts the conversion factor
	default:
		return hasLengthComponent(ent)
	}
}

// hasLengthComponent reports a complex unit instance carrying an SI metre component.
func hasLengthComponent(ent *part21.RawEntity) bool {
	for _, c := range ent.Components {
		if c.Keyword == "SI_UNIT" && enumContains(c.Params, "METRE") {
			return true
		}
		if c.Keyword == "LENGTH_UNIT" {
			return true
		}
	}
	return false
}

// hasComponentNamed reports whether a complex instance has the named component.
func hasComponentNamed(ent *part21.RawEntity, keyword string) bool {
	for _, c := range ent.Components {
		if c.Keyword == keyword {
			return true
		}
	}
	return false
}

// enumContains reports whether any parameter is the named enumeration.
func enumContains(params []part21.Value, name string) bool {
	for _, p := range params {
		if e, err := p.AsEnum(); err == nil && strings.EqualFold(e, name) {
			return true
		}
	}
	return false
}

// lengthUnitScale resolves the mm-per-unit factor for a length unit entity.
func lengthUnitScale(g *part21.EntityGraph, id int) (float64, error) {
	ent, err := g.Lookup(id)
	if err != nil {
		return 0, err
	}
	switch ent.Keyword {
	case "SI_UNIT":
		return siUnitScale(ent.Params)
	case "CONVERSION_BASED_UNIT":
		return conversionUnitScale(g, ent)
	default:
		return complexUnitScale(g, ent)
	}
}

// complexUnitScale resolves a complex unit instance: a CONVERSION_BASED_UNIT
// component (e.g. inch) takes precedence; otherwise the SI_UNIT metre component.
func complexUnitScale(g *part21.EntityGraph, ent *part21.RawEntity) (float64, error) {
	for _, c := range ent.Components {
		if c.Keyword == "CONVERSION_BASED_UNIT" {
			return conversionUnitScale(g, &part21.RawEntity{ID: ent.ID, Keyword: c.Keyword, Params: c.Params})
		}
	}
	return complexSiScale(ent)
}

// siUnitScale computes mm per SI metre unit, honoring a prefix (.MILLI., .CENTI.).
func siUnitScale(params []part21.Value) (float64, error) {
	prefix := ""
	if len(params) > 0 && !params[0].IsNull() {
		if e, err := params[0].AsEnum(); err == nil {
			prefix = e
		}
	}
	return siPrefixFactor(prefix) * 1000.0, nil // metre → mm is ×1000; prefix scales the metre
}

// complexSiScale finds the SI_UNIT component inside a complex length-unit instance.
func complexSiScale(ent *part21.RawEntity) (float64, error) {
	for _, c := range ent.Components {
		if c.Keyword == "SI_UNIT" {
			return siUnitScale(c.Params)
		}
	}
	return 0, fmt.Errorf("step: unit #%d has no SI_UNIT length component", ent.ID)
}

// siPrefixFactor returns the metre fraction a Part 21 SI prefix denotes.
func siPrefixFactor(prefix string) float64 {
	switch strings.ToUpper(prefix) {
	case "MILLI":
		return 1e-3
	case "CENTI":
		return 1e-2
	case "DECI":
		return 1e-1
	case "KILO":
		return 1e3
	case "MICRO":
		return 1e-6
	default: // no prefix ⇒ a plain metre
		return 1.0
	}
}
