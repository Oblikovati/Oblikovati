// SPDX-License-Identifier: GPL-2.0-only

package step

import (
	"fmt"

	"oblikovati.org/kernel/exchange/step/part21"
)

// conversionUnitScale resolves mm-per-unit for a CONVERSION_BASED_UNIT (e.g. inch).
// Layout: CONVERSION_BASED_UNIT(dimensions_or_name, conversion_factor) where
// conversion_factor is a MEASURE_WITH_UNIT (value_component, unit_component). The
// value is the unit's size in its SI base (metres for length), so mm = value × the
// base SI unit's mm factor (1000 for metre).
func conversionUnitScale(g *part21.EntityGraph, ent *part21.RawEntity) (float64, error) {
	factorID, err := lastRef(ent.Params)
	if err != nil {
		return 0, fmt.Errorf("step: CONVERSION_BASED_UNIT #%d conversion_factor: %w", ent.ID, err)
	}
	value, baseID, err := measureWithUnit(g, factorID)
	if err != nil {
		return 0, err
	}
	base, err := lengthUnitScale(g, baseID)
	if err != nil {
		return 0, err
	}
	return value * base, nil
}

// measureWithUnit reads a MEASURE_WITH_UNIT into its value and its unit reference.
// The value is a typed measure (LENGTH_MEASURE(0.0254)); the unit is a reference.
func measureWithUnit(g *part21.EntityGraph, id int) (float64, int, error) {
	ent, err := g.Lookup(id)
	if err != nil {
		return 0, 0, err
	}
	if ent.Keyword != "MEASURE_WITH_UNIT" && ent.Keyword != "LENGTH_MEASURE_WITH_UNIT" {
		return 0, 0, fmt.Errorf("step: #%d is %s, want MEASURE_WITH_UNIT", id, ent.Keyword)
	}
	if len(ent.Params) < 2 {
		return 0, 0, fmt.Errorf("step: MEASURE_WITH_UNIT #%d wants 2 params, got %d", id, len(ent.Params))
	}
	value, err := measureValue(ent.Params[0])
	if err != nil {
		return 0, 0, err
	}
	unitRef, err := ent.Params[1].AsRef()
	return value, unitRef, err
}

// measureValue extracts a measure's numeric value, whether written bare (0.0254)
// or wrapped as a typed parameter (LENGTH_MEASURE(0.0254)).
func measureValue(v part21.Value) (float64, error) {
	if f, err := v.AsFloat(); err == nil {
		return f, nil
	}
	args := v.TypedArgs() // LENGTH_MEASURE(0.0254) exposes its single arg here
	if len(args) != 1 {
		return 0, fmt.Errorf("step: measure value is neither a number nor a single-arg typed measure (got %s)", v.Kind)
	}
	return args[0].AsFloat()
}

// lastRef returns the last reference-valued parameter (the conversion_factor slot,
// robust to writers that put the name first or a dimensions select before it).
func lastRef(params []part21.Value) (int, error) {
	for i := len(params) - 1; i >= 0; i-- {
		if id, err := params[i].AsRef(); err == nil {
			return id, nil
		}
	}
	return 0, fmt.Errorf("step: no reference parameter found")
}
