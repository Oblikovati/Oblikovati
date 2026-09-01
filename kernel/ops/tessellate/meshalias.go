// SPDX-License-Identifier: GPL-2.0-only

// Package tessellate turns the analytic B-rep into triangle meshes: the face and edge
// tessellators, the constrained Delaunay and earcut triangulators, the trimmed-surface
// grid, the periodic and seamed special cases, and the conformance repair that makes
// adjacent face meshes agree.
//
// It is a DERIVED view of the B-rep (kernel ground rules): nothing here decides topology,
// and nothing in the modelling layers may read what it produces to make one.
package tessellate

import "oblikovati.org/kernel/ops/internal/mesh"

// Mesh and Quality are aliases of the shared value types, declared here as well as in
// kernel/ops so this package's 53 files keep naming them unqualified. That is not
// cosmetic: 35 of those files have a local variable called `mesh`, which would shadow the
// leaf package at every use — the aliases sidestep the collision entirely instead of
// renaming a hundred locals.
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
