// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"

	"oblikovati/math"
)

func TestEllipticalCylinderSurfaceMethods(t *testing.T) {
	c, err := NewEllipticalCylinder(math.P3(1, 2, 3), math.V3(0, 0, 1), math.V3(1, 0, 0), 2, 1)
	if err != nil {
		t.Fatalf("NewEllipticalCylinder: %v", err)
	}
	p := c.PointAt(stdmath.Pi/2, 4)
	if !near3(p, math.P3(1, 3, 7), 1e-12) {
		t.Fatalf("PointAt(pi/2, 4) = %v, want (1,3,7)", p)
	}
	du, dv := c.DerivativesAt(0, 4)
	if !nearVec3(du, math.V3(0, 1, 0), 1e-12) || !nearVec3(dv, math.V3(0, 0, 1), 1e-12) {
		t.Fatalf("DerivativesAt(0,4) = %v %v", du, dv)
	}
	if n := c.NormalAt(0, 4); !nearVec3(n, math.V3(1, 0, 0), 1e-12) {
		t.Fatalf("NormalAt(0,4) = %v, want +X", n)
	}
	lo, hi := c.UDomain()
	if lo != 0 || hi != twoPi {
		t.Fatalf("UDomain = [%v,%v], want [0,2pi]", lo, hi)
	}
	lo, hi = c.VDomain()
	if !stdmath.IsInf(lo, -1) || !stdmath.IsInf(hi, 1) {
		t.Fatalf("VDomain = [%v,%v], want unbounded", lo, hi)
	}
	u, v := c.ParamAt(p)
	if stdmath.Abs(u-stdmath.Pi/2) > 1e-12 || stdmath.Abs(v-4) > 1e-12 {
		t.Fatalf("ParamAt(PointAt(pi/2,4)) = %v,%v", u, v)
	}
}

func TestEllipticalConeSurfaceMethods(t *testing.T) {
	c, err := NewEllipticalCone(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), stdmath.Atan(0.5), stdmath.Atan(0.25))
	if err != nil {
		t.Fatalf("NewEllipticalCone: %v", err)
	}
	if _, err := NewEllipticalCone(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), 0, stdmath.Atan(0.25)); err == nil {
		t.Fatal("NewEllipticalCone accepted zero half-angle")
	}
	if _, err := NewEllipticalCone(math.P3(0, 0, 0), math.V3(0, 0, 0), math.V3(1, 0, 0), stdmath.Atan(0.5), stdmath.Atan(0.25)); err == nil {
		t.Fatal("NewEllipticalCone accepted zero axis")
	}
	p := c.PointAt(0, 4)
	if !near3(p, math.P3(2, 0, 4), 1e-12) {
		t.Fatalf("PointAt(0,4) = %v, want (2,0,4)", p)
	}
	du, dv := c.DerivativesAt(0, 4)
	if !nearVec3(du, math.V3(0, 1, 0), 1e-12) || !nearVec3(dv, math.V3(0.5, 0, 1), 1e-12) {
		t.Fatalf("DerivativesAt(0,4) = %v %v", du, dv)
	}
	if n := c.NormalAt(stdmath.Pi/2, 4); stdmath.Abs(n.Length()-1) > 1e-12 {
		t.Fatalf("NormalAt(pi/2,4) length = %v, want unit", n.Length())
	}
	lo, hi := c.UDomain()
	if lo != 0 || hi != twoPi {
		t.Fatalf("UDomain = [%v,%v], want [0,2pi]", lo, hi)
	}
	lo, hi = c.VDomain()
	if lo != 0 || !stdmath.IsInf(hi, 1) {
		t.Fatalf("VDomain = [%v,%v], want [0,+Inf]", lo, hi)
	}
	u, v := c.ParamAt(p)
	if stdmath.Abs(u) > 1e-12 || stdmath.Abs(v-4) > 1e-12 {
		t.Fatalf("ParamAt(PointAt(0,4)) = %v,%v", u, v)
	}
}

func TestThreadedCylinderSurfaceMethods(t *testing.T) {
	base, err := NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 2)
	if err != nil {
		t.Fatalf("NewCylinder: %v", err)
	}
	external := ThreadedCylinder{Cylinder: base, Pitch: 1, Depth: 0.2, RightHanded: true, VMin: 0, VMax: 4}
	if got := external.radiusAt(0, 0); stdmath.Abs(got-2) > 1e-12 {
		t.Fatalf("external radius at runout start = %v, want base radius", got)
	}
	if got := external.radiusAt(0, 2.5); stdmath.Abs(got-1.8) > 1e-12 {
		t.Fatalf("external radius at groove root = %v, want 1.8", got)
	}
	internal := external
	internal.Internal = true
	internal.RightHanded = false
	if got := internal.radiusAt(0, 2.5); stdmath.Abs(got-2.2) > 1e-12 {
		t.Fatalf("internal radius at groove root = %v, want 2.2", got)
	}
	p := external.PointAt(0, 2.5)
	wantP := base.Origin.TranslateBy(base.AxisDir.AsVector().Scale(2.5)).TranslateBy(base.radial(0).Scale(1.8))
	if !near3(p, wantP, 1e-12) {
		t.Fatalf("PointAt(0,2.5) = %v", p)
	}
	du, dv := external.DerivativesAt(0.4, 2.5)
	if du.Length() == 0 || dv.Length() == 0 {
		t.Fatalf("DerivativesAt returned degenerate partials: %v %v", du, dv)
	}
	if n := external.NormalAt(0.4, 2.5); stdmath.Abs(n.Length()-1) > 1e-9 {
		t.Fatalf("NormalAt length = %v, want unit", n.Length())
	}
	lo, hi := external.UDomain()
	if lo != 0 || hi != twoPi {
		t.Fatalf("UDomain = [%v,%v]", lo, hi)
	}
	lo, hi = external.VDomain()
	if lo != 0 || hi != 4 {
		t.Fatalf("VDomain = [%v,%v]", lo, hi)
	}
	u, v := external.ParamAt(p)
	if stdmath.Abs(u) > 1e-12 || stdmath.Abs(v-2.5) > 1e-12 {
		t.Fatalf("ParamAt(PointAt(0,2.5)) = %v,%v", u, v)
	}
	noRunout := external
	noRunout.Pitch = 0
	if got := noRunout.runout(0); got != 1 {
		t.Fatalf("runout with pitch 0 = %v, want 1", got)
	}
}

func TestPolylineFromCurve2AndBSplineCurve2dDomain(t *testing.T) {
	line, err := NewLine2d(math.P2(0, 0), math.V2(2, 0))
	if err != nil {
		t.Fatalf("NewLine2d: %v", err)
	}
	seg := NewLineSegment2d(math.P2(2, 0), math.P2(6, 0))
	pl, err := PolylineFromCurve2(seg, 2)
	if err != nil {
		t.Fatalf("PolylineFromCurve2: %v", err)
	}
	if len(pl.Vertices) != 3 || pl.Vertices[0] != math.P2(2, 0) || pl.Vertices[2] != math.P2(6, 0) {
		t.Fatalf("PolylineFromCurve2 vertices = %v", pl.Vertices)
	}
	if _, err := PolylineFromCurve2(line, 1); err == nil {
		t.Fatal("PolylineFromCurve2 accepted an unbounded curve")
	}
	c, err := NewBSplineCurve2dUniformWeights(1, []math.Point2{math.P2(0, 0), math.P2(1, 0)}, []float64{0, 0, 2, 2})
	if err != nil {
		t.Fatalf("NewBSplineCurve2dUniformWeights: %v", err)
	}
	lo, hi := c.Domain()
	if lo != 0 || hi != 2 {
		t.Fatalf("BSplineCurve2d domain = [%v,%v], want [0,2]", lo, hi)
	}
}

func near3(a, b math.Point3, tol float64) bool {
	return a.DistanceTo(b) <= tol
}

func nearVec3(a, b math.Vector3, tol float64) bool {
	return a.Sub(b).Length() <= tol
}
