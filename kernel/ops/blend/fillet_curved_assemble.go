// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	"fmt"
)

// M5 Slice A assembly gate (m5-curved-arm-derivation.md §D5, task-5-brief.md Step 0). A convex
// axis-aligned Plane∧Cylinder pick builds a curved-arm edgeFillet whose surface is an exact torus
// (axis ⊥ plane) or cylinder (axis ∥ plane) carried in ef.armSurface, with the planar cyl/c0/c1 left
// ZERO. Every planar-assembly consumer (applyRunoutSetback → edgeHasFanEnd, filletBlendFaces,
// classifyEndCorners, transformFace) reads cyl/c0/c1, so an unassembled curved arm nil-derefs there.
// Until the curved weld is built, curvedArmFils detects such arms and filletResolvedEdges honest-rejects
// the whole op — restoring the never-panic invariant across the axis-aligned [Cylinder,Plane,Plane]
// family (B3/N1/O1 + siblings). This is the do-no-harm floor: an unassembled curved arm errors cleanly.

// curvedArmFils reports whether any solved fillet is an unassembled curved-arm edgeFillet — one carrying
// an exact analytic arm surface (ef.armSurface != nil) that the planar setback/emit path cannot consume.
func curvedArmFils(fils []edgeFillet) bool {
	for i := range fils {
		if fils[i].armSurface != nil {
			return true
		}
	}
	return false
}

// curvedArmUnweldedError names the first curved-arm edge whose watertight weld into the solid declined,
// as an honest, actionable reject (do-no-harm): rounding a curved Plane∧Cylinder edge is classified and
// its exact surface built, but this corner could not be welded, so the whole op declines rather than
// emitting a partial body. reason (T5.1-review requirement) carries WHY — a station gap, a host
// non-tangency, a Gauss–Bonnet closure failure, a host-retrim decline, or a non-solid weld — so a real
// reject is diagnosable at the point it reaches the user (repo rule: messages carry the offending shape).
func curvedArmUnweldedError(fils []edgeFillet, reason string) error {
	for i := range fils {
		if fils[i].armSurface != nil {
			return fmt.Errorf("fillet: curved (%s-arm) Plane∧Cylinder edge %d could not be welded into the solid: %s",
				surfaceKind(fils[i].armSurface), fils[i].edge.ID(), reason)
		}
	}
	return nil
}
