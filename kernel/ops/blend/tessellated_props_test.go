// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	"testing"

	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/topo"
)

// mustTessellate meshes b at the property tolerance.
//
// These area gates used query.BodyGeometryProperties, which this package cannot import —
// kernel/ops imports it. BodyGeometryProperties is analytic-first with a tessellated
// fallback, so the number here is the fallback's: a PropertyQuality mesh, the tolerance
// that exists for property readouts. The OCCT gates below are ±1%, and the mesh lands
// inside them; the assertion is unchanged. Moving the ANALYTIC integrator below
// kernel/ops would let these read the exact value again — that is the next extraction,
// not this one.
func mustTessellate(t *testing.T, b *topo.Body) *tessellate.Mesh {
	t.Helper()
	m, _ := tessellate.TessellateBody(b, tessellate.PropertyQuality())
	return m
}
