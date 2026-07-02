// SPDX-License-Identifier: GPL-2.0-only

package sketch

import "oblikovati.org/math"

// RadiusDOF exposes the solver variable behind a circular 3D entity's radius —
// the DOF a radius dimension or drag drives. Stated on the entity so consumers
// need no type switch over the radius-bearing kinds (#1624, audit I1).
func (c *Circle3D) RadiusDOF() *math.Scalar { return &c.Radius }

// RadiusDOF is the helix's start radius: the variable its radius dimension drives.
func (h *HelicalCurve3D) RadiusDOF() *math.Scalar { return &h.StartRadius }
