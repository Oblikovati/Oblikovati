// SPDX-License-Identifier: GPL-2.0-only

package ops

import "oblikovati.org/kernel/ops/internal/mesh"

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
