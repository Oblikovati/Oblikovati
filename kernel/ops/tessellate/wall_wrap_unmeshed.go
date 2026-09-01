// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	"fmt"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// The last-resort signal for a wall the wrapping meshers all declined (Oblikovati/Oblikovati#2038).
// meshSeamCrossingFace ends at the flat best-fit-plane CDT, which is right for a sphere cap over its
// pole but cannot cover a full-wrap developable wall: it comes back open, and the body's volume and
// mass properties are quietly low. So a wall reaching it is flagged.

// CodeWallWrapUnmeshed marks a full-wrap developable side — a cylinder or cone WALL — that no
// wrapping mesher accepted and that therefore fell to the flat best-fit-plane CDT. Flattening a full
// wrap cannot cover it: the mesh comes back open and every integrated quantity (face area, body
// volume, mass properties) is under-reported, which is exactly how #2038 shipped a −77% volume on a
// Validate-clean solid. The mesh still ships — a flagged partial covering beats a missing face — but
// the degradation is recorded rather than silent.
const CodeWallWrapUnmeshed diag.Code = "tessellate.wall-wrap-unmeshed"

// recordUnmeshedWallWrap flags a developable side whose outer loop wraps the full period reaching the
// FLAT patch CDT. A sphere cap straddling its pole legitimately lands there — it is not developable —
// so the check is gated on the surface being a cylinder/cone side, where a flattened wrap is always
// wrong.
func recordUnmeshedWallWrap(m *Mesh, s geom.Surface, outer3D []math.Point3, holes int) {
	if m == nil || !isDevelopableSide(s) {
		return
	}
	if _, _, _, wraps := wrappedWallUV(s, outer3D); !wraps {
		return
	}
	m.Diagnose(diag.Diagnostic{
		Code:     CodeWallWrapUnmeshed,
		Severity: diag.Defect,
		Detail: fmt.Sprintf("a full-wrap %T wall carrying %d hole(s) fell to the flat patch CDT; "+
			"the flattened wrap covers only part of the wall, so its area and the body's volume are low", s, holes),
	})
}
