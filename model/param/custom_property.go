// SPDX-License-Identifier: GPL-2.0-only

package param

import "oblikovati.org/api/types"

// CustomPropertyFormat controls how a parameter's value is formatted when it is
// exposed as a custom document property (Parameter.ExposedAsProperty): text vs
// number, the display unit (empty means the document's display unit), the
// precision, and zero/unit-string rendering. Mirrors the reference API's
// CustomPropertyFormat object (Oblikovati#607).
type CustomPropertyFormat struct {
	PropertyType      types.CustomPropertyType
	Units             string
	Precision         types.CustomPropertyPrecision
	ShowLeadingZeros  bool
	ShowTrailingZeros bool
	ShowUnitsString   bool
}

// DefaultCustomPropertyFormat is the format every new parameter starts with:
// a text property in the document display unit, three decimal places, leading
// zeros and the unit string shown.
func DefaultCustomPropertyFormat() CustomPropertyFormat {
	return CustomPropertyFormat{
		PropertyType:     types.CustomPropertyText,
		Precision:        types.PrecisionThreeDecimalPlaces,
		ShowLeadingZeros: true,
		ShowUnitsString:  true,
	}
}
