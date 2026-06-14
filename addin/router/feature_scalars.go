// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"

	"oblikovati.org/api/wire"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/param"
)

// Shared scalar render/validate logic for the two Edit Feature surfaces — the part
// features.edit and the assembly assemblyFeatures.edit (Oblikovati/Oblikovati#725) —
// so both expose and apply scalar edits identically against any feature.Editable
// definition.

// editableScalars renders f's editable scalars in units's preferred unit, or nil when
// the definition exposes nothing editable. The index is the scalar's slot for an edit.
func editableScalars(units param.UnitsOfMeasure, f feature.Feature) []wire.FeatureScalar {
	ed, ok := f.(feature.Editable)
	if !ok {
		return nil
	}
	params := ed.EditableParams()
	out := make([]wire.FeatureScalar, len(params))
	for i, p := range params {
		out[i] = wire.FeatureScalar{
			Index:   i,
			Label:   p.Label,
			Unit:    units.PreferredName(p.Unit),
			Value:   units.ToPreferred(param.Q(p.Get(), p.Unit)),
			Integer: p.Integer,
		}
	}
	return out
}

// planScalarEdits validates a whole batch against ed — non-empty, each index in range,
// each value parseable in its scalar's unit — and returns one closure applying every Set
// (Set itself cannot fail), so a bad value mid-batch never leaves a definition
// half-edited. method names the wire method for errors.
func planScalarEdits(units param.UnitsOfMeasure, ed feature.Editable, edits []wire.ScalarEdit, method string) (func(), error) {
	if len(edits) == 0 {
		return nil, fmt.Errorf("%s: scalars is empty; expected at least one {index,value} edit", method)
	}
	params := ed.EditableParams()
	sets := make([]func(), len(edits))
	for i, e := range edits {
		if e.Index < 0 || e.Index >= len(params) {
			return nil, fmt.Errorf("%s: scalar index %d out of range (%d scalars)", method, e.Index, len(params))
		}
		p := params[e.Index]
		q, err := units.Parse(e.Value, p.Unit)
		if err != nil {
			return nil, fmt.Errorf("%s: scalar %d value %q: %w", method, e.Index, e.Value, err)
		}
		set, v := p.Set, q.Value
		sets[i] = func() { set(v) }
	}
	return func() {
		for _, set := range sets {
			set()
		}
	}, nil
}
