// SPDX-License-Identifier: GPL-2.0-only

// Package heal repairs a B-rep so it is a valid, watertight solid: sewing and stitching
// separate sheets, snapping edges onto their surfaces, rebuilding the parameter-space trim,
// closing openings and holes, dropping faces, and classifying and removing void shells.
//
// Healing is a separate, explicit operation on a copy (kernel ground rules), so nothing here
// is called from inside another operation's pipeline to rescue it.
package heal

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/mesh"
	"oblikovati.org/kernel/ops/internal/tol"
	"oblikovati.org/kernel/ops/validate"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The mesh and tolerance types this package's operations carry. They are declared here rather
// than imported through the leaf's own name because several files hold a local variable called
// `mesh`, which would shadow the package at every use.
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

// ResolutionForBodies builds a Resolution from the largest operand. See [tol.ForBodies].
func ResolutionForBodies(bodies ...*topo.Body) Resolution { return tol.ForBodies(bodies...) }

// ResolutionForPoints builds a Resolution from a point set. See [tol.ForPoints].
func ResolutionForPoints(pts []math.Point3) Resolution { return tol.ForPoints(pts) }

// ResolutionForSize re-exports the geom constructor. See [tol.ForSize].
func ResolutionForSize(size float64) Resolution { return tol.ForSize(size) }

// Validate is the post-condition every operation in this package runs on its result.
// See [validate.Validate].
func Validate(b *topo.Body) ValidationReport { return validate.Validate(b) }

// BoundaryEdges returns the body's unpaired edges — what a heal has to close.
// See [validate.BoundaryEdges].
func BoundaryEdges(b *topo.Body) []*topo.Edge { return validate.BoundaryEdges(b) }
