// SPDX-License-Identifier: GPL-2.0-only

// Package surface holds the operations that act on a body's FACES rather than on its material:
// extending, offsetting, replacing, rebuilding, fairing and fitting a face's surface, untrimming
// it to its full domain, building ruled and network surfaces between edges, sculpting a control
// net, reconstructing parameter-space curves, and offsetting a planar wire.
//
// None of them is a boolean: they rebuild a face or a sheet and leave the classification alone.
package surface

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/mesh"
	"oblikovati.org/kernel/ops/internal/tol"
	"oblikovati.org/kernel/ops/validate"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The mesh and tolerance types these operations carry, declared here rather than reached through
// the leaf's own name because several files hold a local variable called `mesh`.
type (
	Mesh       = mesh.Mesh
	Quality    = mesh.Quality
	Resolution = geom.Resolution

	// ValidationReport is the verdict Validate returns. See [validate.ValidationReport].
	ValidationReport = validate.ValidationReport
)

// DefaultQuality returns the display tolerance. See [mesh.DefaultQuality].
func DefaultQuality() Quality { return mesh.DefaultQuality() }

// PropertyQuality returns the tolerance for mass-property readouts. See [mesh.PropertyQuality].
func PropertyQuality() Quality { return mesh.PropertyQuality() }

// ResolutionForBody builds a Resolution from a body's range box. See [tol.ForBody].
func ResolutionForBody(b *topo.Body) Resolution { return tol.ForBody(b) }

// ResolutionForPoints builds a Resolution from a point set. See [tol.ForPoints].
func ResolutionForPoints(pts []math.Point3) Resolution { return tol.ForPoints(pts) }

// ResolutionForSize re-exports the geom constructor. See [tol.ForSize].
func ResolutionForSize(size float64) Resolution { return tol.ForSize(size) }

// Validate is the post-condition every operation in this package runs on its result.
// See [validate.Validate].
func Validate(b *topo.Body) ValidationReport { return validate.Validate(b) }
