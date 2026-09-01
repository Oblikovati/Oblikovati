// SPDX-License-Identifier: GPL-2.0-only

package validate_test

import "oblikovati.org/kernel/ops/validate"

// gateQuality names a faceting a result gate is sampled at. A gate that only holds at one
// sampling is measuring the tessellation, not the geometry.
type gateQuality struct {
	name string
	q    validate.Quality
}

// gateQualities is the pair of samplings every validity gate in this package runs at.
func gateQualities() []gateQuality {
	return []gateQuality{
		{"default", validate.DefaultQuality()},
		{"property", validate.PropertyQuality()},
	}
}
