// SPDX-License-Identifier: GPL-2.0-only

package app

import "oblikovati.org/api/types"

// The GD&T characteristic combo for the model-tolerance panel (#2049). The order is the ASME
// Y14.5 grouping — form, profile, orientation, location, runout — rather than the enum's bit
// order, so the list reads the way the symbols are taught.

// geometricCharacteristics maps the combo index to the characteristic.
var geometricCharacteristics = []types.GeometricCharacteristic{
	types.CharacteristicStraightness,
	types.CharacteristicFlatness,
	types.CharacteristicCircularity,
	types.CharacteristicCylindricity,
	types.CharacteristicProfileOfAnyLine,
	types.CharacteristicProfileOfAnySurface,
	types.CharacteristicProfileOfASection,
	types.CharacteristicParallelProfile,
	types.CharacteristicAngularity,
	types.CharacteristicPerpendicularity,
	types.CharacteristicParallelism,
	types.CharacteristicPosition,
	types.CharacteristicConcentricity,
	types.CharacteristicSymmetry,
	types.CharacteristicAxiality,
	types.CharacteristicAxisIntersection,
	types.CharacteristicCircularRunout,
	types.CharacteristicTotalRunout,
	types.CharacteristicCircularRunoutFilled,
	types.CharacteristicTotalRunoutFilled,
}

// GeometricCharacteristicOptions labels the characteristic combo, in index order.
func GeometricCharacteristicOptions() []string {
	out := make([]string, len(geometricCharacteristics))
	for i, c := range geometricCharacteristics {
		out[i] = c.String()
	}
	return out
}
