// SPDX-License-Identifier: GPL-2.0-only

package ops

// Fixture builders restated from kernel/ops/tessellate's test package. Go cannot share a
// _test.go helper across packages, so a test that stayed here while the tessellator moved
// needs its own copy — the test scaffolding sonar.cpd.exclusions already accounts for.

import (
	"oblikovati.org/math"
)

// bodyVolume sums the divergence-theorem volume of a merged mesh (positive = outward-oriented closed).
func bodyVolume(m *Mesh) float64 {
	v := 0.0
	for t := 0; t+2 < len(m.Indices); t += 3 {
		a := math.Point3{}.VectorTo(m.Positions[m.Indices[t]])
		b := math.Point3{}.VectorTo(m.Positions[m.Indices[t+1]])
		c := math.Point3{}.VectorTo(m.Positions[m.Indices[t+2]])
		v += float64(a.Dot(b.Cross(c))) / 6.0
	}
	return v
}
