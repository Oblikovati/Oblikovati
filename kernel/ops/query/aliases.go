// SPDX-License-Identifier: GPL-2.0-only

// Package query answers read-only questions about a body: picking, mass and geometry
// properties (analytic and tessellated), inertia, oriented and precise range boxes,
// visible edges, boundary crossing, shell and body containment, and the identical-bodies
// signature.
//
// Every function here is a pure function of its input — the package changes nothing. That
// is what lets archguard eventually assert it is mutation-free, and it is why the mass
// properties belong here rather than in the operation layer (#2207).
package query

import (
	"oblikovati.org/kernel/ops/internal/mesh"
	"oblikovati.org/kernel/ops/tessellate"
)

// Mesh and Quality are aliases of the shared value types, declared here as well as in
// kernel/ops so this package's files keep naming them unqualified — several have a local
// variable called `mesh`, which would shadow the leaf package at every use.
type (
	Mesh               = mesh.Mesh
	Quality            = mesh.Quality
	GeometryProperties = tessellate.GeometryProperties
)

// DefaultQuality returns the display tolerance. See [mesh.DefaultQuality].
func DefaultQuality() Quality { return mesh.DefaultQuality() }

// PropertyQuality returns the tolerance for mass-property readouts.
// See [mesh.PropertyQuality].
func PropertyQuality() Quality { return mesh.PropertyQuality() }

// MeshGeometryProperties integrates a triangle mesh — the tessellated fallback the
// analytic path takes when it declines. See [tessellate.MeshGeometryProperties].
func MeshGeometryProperties(m *Mesh) GeometryProperties {
	return tessellate.MeshGeometryProperties(m)
}
