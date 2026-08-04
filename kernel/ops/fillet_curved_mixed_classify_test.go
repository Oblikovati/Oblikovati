// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// minimalRoleFace wraps a surface in a throwaway triangular face so the arm-role predicates that read
// ef.a/ef.b.Geometry() (isPlanarBandArm) have a real topo.Face to interrogate. Only Geometry() is read.
func minimalRoleFace(surf geom.Surface, tag int) *topo.Face {
	bld := topo.NewBuilder(false, topo.NewLineage(topo.Tok("mixedrole", "body", tag)))
	a := bld.AddVertex(math.P3(0, 0, 0), topo.NewLineage(topo.Tok("mixedrole", "v", tag*3)))
	b := bld.AddVertex(math.P3(10, 0, 0), topo.NewLineage(topo.Tok("mixedrole", "v", tag*3+1)))
	c := bld.AddVertex(math.P3(0, 10, 0), topo.NewLineage(topo.Tok("mixedrole", "v", tag*3+2)))
	seg := func(p, q *topo.Vertex) geom.LineSegment { return geom.NewLineSegment(p.Point(), q.Point()) }
	edge := func(p, q *topo.Vertex, i int) *topo.Edge {
		return bld.AddEdge(seg(p, q), p, q, topo.NewLineage(topo.Tok("mixedrole", "e", i)))
	}
	ab, bc, ca := edge(a, b, tag*3), edge(b, c, tag*3+1), edge(c, a, tag*3+2)
	return bld.AddFace(surf, topo.NewLineage(topo.Tok("mixedrole", "f", tag)), topo.OuterLoop(topo.Fwd(ab), topo.Fwd(bc), topo.Fwd(ca)))
}

// roleCyl / rolePlane / roleTorus wrap the shared must* helpers with the fixed frame these role-predicate
// tests use (a surface's exact placement is irrelevant to the predicates, which key only on Go type + the
// flip/concave flags).
func roleCyl(t *testing.T) geom.Cylinder {
	return mustCylinder(t, math.P3(0, 0, 0), math.V3(0, 0, 1), 5)
}
func rolePlane(t *testing.T) geom.Plane {
	return mustPlane(t, math.P3(0, 0, 0), math.V3(0, 0, 1))
}
func roleTorus(t *testing.T) geom.Torus {
	return mustTorus(t, math.P3(0, 0, 0), math.V3(0, 0, 1), 30, 5)
}

// TestIsCoveTorusArmDeclines pins the cove predicate: only a CONCAVE torus arm qualifies; a cylinder arm
// or a CONVEX torus (the N4 boss cap-rim arm) is rejected. Guards the classifier against admitting the
// wrong sense/surface as the M8 cove.
func TestIsCoveTorusArmDeclines(t *testing.T) {
	if isCoveTorusArm(edgeFillet{armSurface: roleCyl(t), armConcave: true}) {
		t.Error("isCoveTorusArm accepted a Cylinder arm — must be a Torus")
	}
	if isCoveTorusArm(edgeFillet{armSurface: roleTorus(t), armConcave: false}) {
		t.Error("isCoveTorusArm accepted a CONVEX torus — the cove is concave")
	}
	if !isCoveTorusArm(edgeFillet{armSurface: roleTorus(t), armConcave: true}) {
		t.Error("isCoveTorusArm rejected a concave torus cove arm")
	}
}

// TestIsConvexCylArmDeclines pins the convex-pivot predicate: only a Cylinder arm that is neither concave
// nor flipped qualifies. The concave cyl (N3/M4/N9), the flipped planar band, and a torus are all rejected.
func TestIsConvexCylArmDeclines(t *testing.T) {
	if isConvexCylArm(edgeFillet{armSurface: roleTorus(t)}) {
		t.Error("isConvexCylArm accepted a Torus arm")
	}
	if isConvexCylArm(edgeFillet{armSurface: roleCyl(t), armConcave: true}) {
		t.Error("isConvexCylArm accepted a CONCAVE cylinder arm")
	}
	if isConvexCylArm(edgeFillet{armSurface: roleCyl(t), flip: true}) {
		t.Error("isConvexCylArm accepted a FLIPPED (planar-band) cylinder arm")
	}
	if !isConvexCylArm(edgeFillet{armSurface: roleCyl(t)}) {
		t.Error("isConvexCylArm rejected a plain convex cylinder pivot arm")
	}
}

// TestIsPlanarBandArmDeclines pins the planar-band predicate: a flip (concave) Plane∧Plane fillet only. A
// non-flip arm, a concave-marked arm, and one whose second host is not a plane (a Plane∧Cylinder band) all
// decline. The non-flip case must short-circuit BEFORE touching a/b (they may be nil), so it is tested with
// nil faces.
func TestIsPlanarBandArmDeclines(t *testing.T) {
	if isPlanarBandArm(edgeFillet{flip: false}) {
		t.Error("isPlanarBandArm accepted a non-flip arm")
	}
	pl := rolePlane(t)
	fa := minimalRoleFace(pl, 1)
	fb := minimalRoleFace(pl, 2)
	fcyl := minimalRoleFace(roleCyl(t), 3)
	if isPlanarBandArm(edgeFillet{flip: true, armConcave: true, a: fa, b: fb}) {
		t.Error("isPlanarBandArm accepted an armConcave-marked arm (the exact concave cyl, not the planar band)")
	}
	if isPlanarBandArm(edgeFillet{flip: true, a: fa, b: fcyl}) {
		t.Error("isPlanarBandArm accepted a Plane∧Cylinder band (second host not a plane)")
	}
	if !isPlanarBandArm(edgeFillet{flip: true, a: fa, b: fb}) {
		t.Error("isPlanarBandArm rejected a flip Plane∧Plane band")
	}
}

// TestClassifyMixedRoleArmsGuards pins the M8-review hardening (Minor a): the classifier requires EXACTLY
// three arms and one arm per role — a wrong valence or a duplicated role declines to the do-no-harm floor,
// so a non-M8 valence/sense can never reach the 2r-torus solve. The happy path (1 convex + 1 cove + 1
// planar) still classifies.
func TestClassifyMixedRoleArmsGuards(t *testing.T) {
	pl := rolePlane(t)
	convex := edgeFillet{armSurface: roleCyl(t)}
	cove := edgeFillet{armSurface: roleTorus(t), armConcave: true}
	planar := edgeFillet{flip: true, a: minimalRoleFace(pl, 4), b: minimalRoleFace(pl, 5)}

	if _, ok := classifyMixedRoleArms([]edgeFillet{convex, cove}); ok {
		t.Error("classifyMixedRoleArms accepted 2 arms — needs exactly 3")
	}
	if _, ok := classifyMixedRoleArms([]edgeFillet{convex, cove, planar, planar}); ok {
		t.Error("classifyMixedRoleArms accepted 4 arms — needs exactly 3")
	}
	if _, ok := classifyMixedRoleArms([]edgeFillet{convex, cove, cove}); ok {
		t.Error("classifyMixedRoleArms accepted a duplicated cove role")
	}
	if _, ok := classifyMixedRoleArms([]edgeFillet{convex, convex, planar}); ok {
		t.Error("classifyMixedRoleArms accepted a duplicated convex role (and no cove)")
	}
	roles, ok := classifyMixedRoleArms([]edgeFillet{convex, cove, planar})
	if !ok {
		t.Fatal("classifyMixedRoleArms rejected the valid 1-convex + 1-cove + 1-planar corner")
	}
	if _, isTor := roles.cove.armSurface.(geom.Torus); !isTor {
		t.Error("classified cove arm is not the torus")
	}
}
