// SPDX-License-Identifier: GPL-2.0-only

package boolean

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/mesh"
	"oblikovati.org/kernel/ops/internal/tol"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/ops/validate"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The mesh types the CSG and mesh-arrangement paths carry. They are declared here rather than
// imported through the leaf's own name because most files in this package hold a local variable
// called `mesh`, which would shadow the package at every use.
type (
	Mesh       = mesh.Mesh
	Quality    = mesh.Quality
	Tri        = mesh.Tri
	Resolution = geom.Resolution
)

// DefaultQuality returns the display tolerance. See [mesh.DefaultQuality].
func DefaultQuality() Quality { return mesh.DefaultQuality() }

// PropertyQuality returns the tolerance for mass-property readouts. See [mesh.PropertyQuality].
func PropertyQuality() Quality { return mesh.PropertyQuality() }

// ResolutionForBodies builds a Resolution from the largest operand. See [tol.ForBodies].
func ResolutionForBodies(bodies ...*topo.Body) Resolution { return tol.ForBodies(bodies...) }

// ResolutionForBody builds a Resolution from a body's range box. See [tol.ForBody].
func ResolutionForBody(b *topo.Body) Resolution { return tol.ForBody(b) }

// ResolutionForSize re-exports the geom constructor. See [tol.ForSize].
func ResolutionForSize(size float64) Resolution { return tol.ForSize(size) }

// ResolutionForPoints builds a Resolution from a point set. See [tol.ForPoints].
func ResolutionForPoints(pts []math.Point3) Resolution { return tol.ForPoints(pts) }

// ResolutionForTris builds a Resolution from CSG triangles. See [tol.ForTris].
func ResolutionForTris(tris []Tri) Resolution { return tol.ForTris(tris) }

// TessellateBody meshes a body at the given quality, returning the mesh and the wire polylines.
// See [tessellate.TessellateBody].
func TessellateBody(b *topo.Body, q Quality) (*Mesh, [][]math.Point3) {
	return tessellate.TessellateBody(b, q)
}

// ValidationReport is a body validity verdict. See [validate.ValidationReport].
type ValidationReport = validate.ValidationReport

// Validate is the post-condition every operation in this package runs on its result.
// See [validate.Validate].
func Validate(b *topo.Body) ValidationReport { return validate.Validate(b) }
