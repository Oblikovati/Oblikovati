// SPDX-License-Identifier: GPL-2.0-only

package app

import "oblikovati.org/api/types"

// The #331 approximation request shared by the Offset Face and Thicken tools: which
// bound the offset may approximate to. The kernel computes the exact offset (which
// satisfies every bound), so the choice is carried and recorded, not geometry-changing.

// featureApproximations are the choices in display order (none is the zero default).
var featureApproximations = []types.FeatureApproximationType{
	types.NoApproximation, types.MeanApproximation,
	types.NeverTooThickApproximation, types.NeverTooThinApproximation,
}

// ApproximationOptions lists the display labels, index-aligned with featureApproximations.
func ApproximationOptions() []string {
	return []string{"None (exact)", "Mean", "Never too thick", "Never too thin"}
}

// approximationAt resolves a choice index to its frozen type, clamped to the valid range.
func approximationAt(i int) types.FeatureApproximationType {
	return featureApproximations[clampRange(i, len(featureApproximations))]
}
