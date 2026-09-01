// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/mesh"
	"oblikovati.org/kernel/ops/internal/tol"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The model-relative tolerance constructors moved to kernel/ops/internal/tol so every
// operation family can reach them without depending on the whole operation layer
// (ADR-0042 Phase 1 keeps the primitive itself in kernel/geom). These are the names
// kernel/ops call sites already use.

// Resolution is the model-relative coincidence scale (alias of geom.Resolution).
type Resolution = geom.Resolution

// ResolutionForSize re-exports the geom constructor. See [tol.ForSize].
func ResolutionForSize(size float64) Resolution { return tol.ForSize(size) }

// ResolutionForPoints builds a Resolution from a point set. See [tol.ForPoints].
func ResolutionForPoints(pts []math.Point3) Resolution { return tol.ForPoints(pts) }

// ResolutionForBody builds a Resolution from a body's range box. See [tol.ForBody].
func ResolutionForBody(b *topo.Body) Resolution { return tol.ForBody(b) }

// ResolutionForBodies builds a Resolution from the largest operand. See [tol.ForBodies].
func ResolutionForBodies(bodies ...*topo.Body) Resolution { return tol.ForBodies(bodies...) }

// ResolutionForTris builds a Resolution from CSG triangles. See [tol.ForTris].
func ResolutionForTris(tris []mesh.Tri) Resolution { return tol.ForTris(tris) }
