// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// SolidCylinderFilletedTop builds a closed cylinder whose TOP rim — the circle where the side wall
// meets the top cap — is rounded by a rolling-ball fillet of radius r, a toroidal band. It is the
// closed-rim ("no run-out") case of a curved-adjacent fillet: the rim is a FULL circle, so the
// blend closes on itself with no corner ends (unlike a partial arc, which terminates at smooth
// tangent vertices and needs variable-radius run-out). Topology mirrors SolidCylinder with one
// extra band: bottom cap, side wall (base → z=h−r), the torus blend (z=h−r → z=h), and the receded
// top cap (radius Rc−r). r must be < radius and < height.
func SolidCylinderFilletedTop(baseCenter math.Point3, axisDir math.Vector3, radius, height, r float64) (*topo.Body, error) {
	if r <= 0 || r >= radius || r >= height {
		return nil, fmt.Errorf("brep: fillet %g must be in (0, min(radius %g, height %g))", r, radius, height)
	}
	a, err := math.UnitVector3FromVector(axisDir)
	if err != nil {
		return nil, err
	}
	av := a.AsVector()
	cylTopC := baseCenter.TranslateBy(av.Scale(math.Scalar(height - r)))
	capC := baseCenter.TranslateBy(av.Scale(math.Scalar(height)))

	bottom, err := geom.NewCircle(baseCenter, axisDir, radius)
	if err != nil {
		return nil, err
	}
	// Share the bottom circle's frame so every seam sits at angle 0.
	cylTan := geom.Circle{Center: cylTopC, Normal: bottom.Normal, RefDir: bottom.RefDir, Radius: radius}
	capTan := geom.Circle{Center: capC, Normal: bottom.Normal, RefDir: bottom.RefDir, Radius: radius - r}

	wall, err := geom.NewCylinder(baseCenter, axisDir, radius)
	if err != nil {
		return nil, err
	}
	capBottom, err := geom.NewPlane(baseCenter, av.Scale(-1)) // outward −axis
	if err != nil {
		return nil, err
	}
	capTop, err := geom.NewPlane(capC, av) // outward +axis
	if err != nil {
		return nil, err
	}
	// Torus tube centre on the cyl-tangent circle: major Rc−r, minor r. v=0 → cyl-tangent (Rc, z=h−r),
	// v=π/2 → cap-tangent (Rc−r, z=h). Share the bottom frame so the torus seam lines up at angle 0.
	tor, err := geom.NewTorusWithRef(cylTopC, av, bottom.RefDir.AsVector(), radius-r, r)
	if err != nil {
		return nil, err
	}

	vbp, vcp, vtp := bottom.PointAt(0), cylTan.PointAt(0), capTan.PointAt(0)
	seamArc, err := geom.Arc3dByThreePoints(vcp, tor.PointAt(0, stdmath.Pi/4), vtp) // torus seam, v: 0→π/2
	if err != nil {
		return nil, err
	}

	bld := topo.NewBuilder(true, filLin("body", 0))
	vb := bld.AddVertex(vbp, filLin("v", 0))
	vc := bld.AddVertex(vcp, filLin("v", 1))
	vt := bld.AddVertex(vtp, filLin("v", 2))
	eb := bld.AddEdge(bottom, vb, vb, filLin("e", 0)) // bottom circle (closed)
	ec := bld.AddEdge(cylTan, vc, vc, filLin("e", 1)) // cyl-tangent circle (closed)
	et := bld.AddEdge(capTan, vt, vt, filLin("e", 2)) // cap-tangent circle (closed)
	esw := bld.AddEdge(geom.NewLineSegment(vbp, vcp), vb, vc, filLin("e", 3))
	est := bld.AddEdge(seamArc, vc, vt, filLin("e", 4))

	bld.AddFace(capBottom, filLin("f", 0), topo.OuterLoop(topo.Rev(eb)))
	bld.AddFace(capTop, filLin("f", 1), topo.OuterLoop(topo.Fwd(et)))
	// Periodic wall: seam up, cyl-tangent circle (opposite the torus), seam down, bottom circle.
	bld.AddFace(wall, filLin("f", 2), topo.OuterLoop(topo.Fwd(esw), topo.Rev(ec), topo.Rev(esw), topo.Fwd(eb)))
	// Periodic torus band: seam up, cap-tangent circle (opposite the cap), seam down, cyl-tangent circle.
	bld.AddFace(tor, filLin("f", 3), topo.OuterLoop(topo.Fwd(est), topo.Rev(et), topo.Rev(est), topo.Fwd(ec)))
	return bld.Build(), nil
}

func filLin(role string, i int) topo.Lineage { return topo.NewLineage(topo.Tok("cylfillet", role, i)) }
