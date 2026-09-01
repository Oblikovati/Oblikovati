// SPDX-License-Identifier: GPL-2.0-only

package blend_test

import "oblikovati.org/kernel/ops/blend"

// gateQuality names a faceting a result gate is sampled at. A gate that only holds at one
// sampling is measuring the tessellation, not the geometry.
type gateQuality struct {
	name string
	q    blend.Quality
}

// gateQualities is the pair of samplings every blend result gate in this package runs at.
func gateQualities() []gateQuality {
	return []gateQuality{
		{"default", blend.DefaultQuality()},
		{"property", blend.PropertyQuality()},
	}
}
