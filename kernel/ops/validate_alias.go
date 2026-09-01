// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/ops/validate"
	"oblikovati.org/kernel/topo"
)

// The validity family moved to kernel/ops/validate (#2211) so the kernel packages
// that must run it — and archguard, which has to assert that every public operation's
// return path passes through it — can depend on the validator without depending on
// the whole operation layer.
//
// These aliases and forwarders are the compatibility seam for the 252 files outside
// the kernel that call ops.Validate today. They add no behaviour: each one is the
// moved symbol under its old name. Migrating those call sites to validate.* is
// mechanical follow-up, and until it happens ops -> validate is the correct
// dependency direction, not a cycle.
type (
	ValidationReport = validate.ValidationReport
	ProblemEntity    = validate.ProblemEntity
	SelfIntersection = validate.SelfIntersection
	EntityCheckLevel = validate.EntityCheckLevel
)

// Validate runs the levelled validity check. See [validate.Validate].
func Validate(b *topo.Body) ValidationReport { return validate.Validate(b) }

// BoundaryEdges returns the body's boundary (non-manifold) edges.
// See [validate.BoundaryEdges].
func BoundaryEdges(b *topo.Body) []*topo.Edge { return validate.BoundaryEdges(b) }

// The entity-level check depth. See [validate.EntityCheckLevel].
const (
	CheckTopology = validate.CheckTopology
	CheckGeometry = validate.CheckGeometry
)

// ValidateBodyEntities runs the per-entity validity checks.
// See [validate.ValidateBodyEntities].
func ValidateBodyEntities(b *topo.Body, level EntityCheckLevel, q Quality) (bool, []ProblemEntity) {
	return validate.ValidateBodyEntities(b, level, q)
}

// SelfIntersections reports where a body's faces cross each other.
// See [validate.SelfIntersections].
func SelfIntersections(b *topo.Body, q Quality) []SelfIntersection {
	return validate.SelfIntersections(b, q)
}

// EdgeEntityValid reports whether an edge is individually valid.
// See [validate.EdgeEntityValid].
func EdgeEntityValid(e *topo.Edge) bool { return validate.EdgeEntityValid(e) }

// FaceEntityValid reports whether a face is individually valid.
// See [validate.FaceEntityValid].
func FaceEntityValid(f *topo.Face) bool { return validate.FaceEntityValid(f) }

// FoldEdges reports the mesh edges whose two triangles fold back on each other.
// See [validate.FoldEdges].
func FoldEdges(m *Mesh) [][2]int { return validate.FoldEdges(m) }

// FoldEdgeCount counts them. See [validate.FoldEdgeCount].
func FoldEdgeCount(m *Mesh) int { return validate.FoldEdgeCount(m) }

// MeshArea returns the mesh's total triangle area. See [validate.MeshArea].
func MeshArea(m *Mesh) float64 { return validate.MeshArea(m) }
