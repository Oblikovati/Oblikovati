// SPDX-License-Identifier: GPL-2.0-only

package param

import (
	"fmt"

	"oblikovati.org/api/types"
)

// Document-level parameter settings and whole-model tolerance sweeps
// (M02-F07, Oblikovati#606). In the reference API the Parameters collection
// doubles as the document's parameter-settings object, so the settings live
// here rather than on a separate document options type.

// DimensionDisplayType is how dimensions tied to parameters render in
// sketches. Aliased from the contract (ADR-0018).
type DimensionDisplayType = types.DimensionDisplayType

// CollectionSettings are the document-level parameter settings: the standard
// tolerance expressions applied to dimensions without an explicit tolerance
// (when UseStandardTolerances is on), the default display precisions, and how
// dimensions render. They affect presentation and defaulting only — never an
// evaluated or model value.
type CollectionSettings struct {
	LinearStandardTolerance      string
	AngularStandardTolerance     string
	UseStandardTolerances        bool
	ExportStandardTolerances     bool
	LinearDimensionPrecision     int
	AngularDimensionPrecision    int
	DimensionDisplayType         DimensionDisplayType
	DisplayParameterAsExpression bool
}

// DefaultCollectionSettings is what every new document starts with: three
// linear / two angular display decimals, dimensions rendered as values.
func DefaultCollectionSettings() CollectionSettings {
	return CollectionSettings{
		LinearDimensionPrecision:  3,
		AngularDimensionPrecision: 2,
		DimensionDisplayType:      types.DimensionDisplayValue,
	}
}

// Settings returns the document-level parameter settings for reading and
// editing in place.
func (ps *Parameters) Settings() *CollectionSettings { return &ps.settings }

// SetAllModelValueType drives every toleranced parameter's model-value
// selection to one band bound — the reference SetAllToMax/Min/Nominal/Median
// sweeps for limit-stack studies. Parameters without an explicit tolerance
// band are left alone. It returns how many parameters it moved.
func (ps *Parameters) SetAllModelValueType(m ModelValueType) (int, error) {
	switch m {
	case Nominal, Upper, Lower, Median:
	default:
		return 0, fmt.Errorf("param: unknown sweep model value type %d; want nominal/lower/upper/median", int32(m))
	}
	affected := 0
	for _, id := range ps.order {
		p := ps.byID[id]
		if p.Tolerance() == (Tolerance{}) {
			continue
		}
		if err := p.SetModelValueType(m); err != nil {
			return affected, err
		}
		affected++
	}
	return affected, nil
}
