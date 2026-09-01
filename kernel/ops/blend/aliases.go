// SPDX-License-Identifier: GPL-2.0-only

// Package blend is the fillet/chamfer/draft operation family: edge fillets and their
// arms, corner blends, setbacks, runouts, miters, obstacle handling, and the corner-weld
// assembly that stitches the result back into a solid.
//
// The kernel ground rules treat fillet, chamfer and draft as ONE blend engine
// (ADR-0050/0051). This package is that engine's OPERATION layer — it works on a
// topo.Body. The lower-level blend primitives (spine, section functional, marcher, law)
// live in kernel/blend, which this package uses; #2200 tracks resolving the two.
package blend

import "oblikovati.org/kernel/ops/internal/mesh"

// Mesh and Quality are aliases of the shared value types, declared here as well as in
// kernel/ops so this package's files keep naming them unqualified — the same reason
// kernel/ops/tessellate carries its own: several files here have a local variable called
// `mesh`, which would shadow the leaf package at every use.
type (
	Mesh    = mesh.Mesh
	Quality = mesh.Quality
	Tri     = mesh.Tri
)

// DefaultQuality returns the display tolerance. See [mesh.DefaultQuality].
func DefaultQuality() Quality { return mesh.DefaultQuality() }

// PropertyQuality returns the tolerance for mass-property readouts.
// See [mesh.PropertyQuality].
func PropertyQuality() Quality { return mesh.PropertyQuality() }
