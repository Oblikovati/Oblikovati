// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	stdmath "math"
	"testing"

	"github.com/Oblikovati/oblikovati/kernel/ops"
	"github.com/Oblikovati/oblikovati/kernel/subd"
	"github.com/Oblikovati/oblikovati/math"
)

// offsetBoxBody builds a 2×2×2 box translated to corner (dx,dy,dz) by offsetting the
// cage vertices (no TransformBody), isolating the tessellation winding.
func offsetBoxBody(dx, dy, dz float64) *ops.Mesh {
	m := subd.Box(2, 2, 2)
	for i := range m.Verts {
		m.Verts[i] = m.Verts[i].TranslateBy(math.V3(dx, dy, dz))
	}
	mesh, _ := ops.TessellateBody(subd.ToBody(m, "s"), ops.DefaultQuality())
	return mesh
}

// meshSignedVolume is the divergence-theorem volume of a triangle mesh; it equals the
// true volume only when every triangle is wound consistently outward.
func meshSignedVolume(m *ops.Mesh) float64 {
	v := 0.0
	for i := 0; i+2 < len(m.Indices); i += 3 {
		a := m.Positions[m.Indices[i]].AsVector()
		b := m.Positions[m.Indices[i+1]].AsVector()
		c := m.Positions[m.Indices[i+2]].AsVector()
		v += a.Dot(b.Cross(c))
	}
	return v / 6
}

// TestTessellationWindingIsOutwardAndTranslationInvariant guards the planeProjector
// orientation fix: a planar-faced solid must tessellate with consistently outward
// winding, so its divergence-theorem volume equals the analytic volume regardless of
// where it sits (a divergence sum is translation-invariant only for a watertight,
// coherently-wound mesh). Before the fix, negative-axis-normal faces were wound inward
// and an off-origin box reported 13.33 instead of 8.
func TestTessellationWindingIsOutwardAndTranslationInvariant(t *testing.T) {
	for _, off := range [][3]float64{{0, 0, 0}, {1, 0.5, 0.5}, {10, 20, 30}, {-5, -7, -9}} {
		got := meshSignedVolume(offsetBoxBody(off[0], off[1], off[2]))
		if stdmath.Abs(got-8) > 1e-9 {
			t.Errorf("box at %v: tessellated volume = %g, want 8 (inconsistent winding?)", off, got)
		}
	}
}
