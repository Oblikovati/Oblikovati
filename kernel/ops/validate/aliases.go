// SPDX-License-Identifier: GPL-2.0-only

package validate

import "oblikovati.org/kernel/mesh"

// The mesh types the fold and self-intersection checks read, named here so a caller of this
// package spells them the same way it spells them everywhere else in kernel/ops.
type (
	Mesh    = mesh.Mesh
	Quality = mesh.Quality
)

// DefaultQuality returns the display tolerance. See [mesh.DefaultQuality].
func DefaultQuality() Quality { return mesh.DefaultQuality() }

// PropertyQuality returns the tolerance for mass-property readouts. See [mesh.PropertyQuality].
func PropertyQuality() Quality { return mesh.PropertyQuality() }
