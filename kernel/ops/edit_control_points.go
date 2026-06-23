// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
)

// Editing a NURBS surface face's control net in place (M36-F03): find the body's freeform
// (B-spline) face, displace the listed control points, and swap the surface back in with
// ReplaceFaceSurface. Degree and knots are preserved (geom.DisplaceControlPoints), so the face's
// trim loops still evaluate onto the edited surface — no edge surgery, just a moved control net.

// EditControlPoints returns a copy of b with its first NURBS (B-spline) surface face's control
// points displaced by deltas. It errors when the body has no NURBS face or an index is out of
// range. The single-face target matches the styling workflow (one freeform quilt being shaped).
func EditControlPoints(b *topo.Body, deltas []geom.ControlPointDelta) (*topo.Body, error) {
	for _, f := range b.Faces() {
		surf, ok := f.Geometry().(geom.BSplineSurface)
		if !ok {
			continue
		}
		edited, err := surf.DisplaceControlPoints(deltas)
		if err != nil {
			return nil, fmt.Errorf("ops.EditControlPoints: %w", err)
		}
		return ReplaceFaceSurface(b, f.ReferenceKey(), edited)
	}
	return nil, fmt.Errorf("ops.EditControlPoints: body has no NURBS surface face to edit")
}
