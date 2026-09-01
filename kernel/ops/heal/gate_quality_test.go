// SPDX-License-Identifier: GPL-2.0-only

package heal

import "oblikovati.org/kernel/ops/tessellate"

// gateQuality names a faceting a result gate is sampled at. A gate that only holds at one
// sampling is measuring the tessellation, not the geometry.
type gateQuality struct {
	name string
	q    Quality
}

// gateQualities is the pair of samplings every heal result gate in this package runs at.
func gateQualities() []gateQuality {
	return []gateQuality{
		{"default", DefaultQuality()},
		{"property", PropertyQuality()},
	}
}

// freeEdgeCount counts a mesh's unpaired edges — a watertight heal leaves none.
// See [tessellate.FreeEdgeCount].
func freeEdgeCount(m *Mesh) int { return tessellate.FreeEdgeCount(m) }
