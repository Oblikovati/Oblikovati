// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/ops/internal/mesh"
	"oblikovati.org/kernel/ops/tessellate"
)

// Mesh and Quality are ALIASES, not new types: the value types moved down to
// kernel/ops/internal/mesh so every operation family can share them without making
// each one an ops-internal symbol (which is what blocked splitting this package).
// Aliasing keeps ops.Mesh and mesh.Mesh the same type, so the 364 files outside the
// kernel that name ops.Mesh / ops.Quality are untouched.
type (
	Mesh    = mesh.Mesh
	Quality = mesh.Quality
)

// DefaultQuality returns a reasonable display tolerance. See [mesh.DefaultQuality].
func DefaultQuality() Quality { return mesh.DefaultQuality() }

// PropertyQuality returns the tolerance for mass/geometry property readouts.
// See [mesh.PropertyQuality].
func PropertyQuality() Quality { return mesh.PropertyQuality() }

// GeometryProperties is an alias: the tessellated integrator that produces it lives with
// the tessellator (kernel/ops/tessellate), so the type does too. ops.GeometryProperties
// and tessellate.GeometryProperties are the same type, leaving every call site outside
// the kernel untouched.
type GeometryProperties = tessellate.GeometryProperties

// MeshGeometryProperties integrates a triangle mesh. See [tessellate.MeshGeometryProperties].
func MeshGeometryProperties(m *Mesh) GeometryProperties { return tessellate.MeshGeometryProperties(m) }
