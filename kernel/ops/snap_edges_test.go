// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// radiusXY is the distance of p from the z axis — the invariant a point on a z-axis cylinder holds.
func radiusXY(p math.Point3) float64 {
	return stdmath.Hypot(float64(p.X), float64(p.Y))
}

// offSurfaceCylinder builds a closed cylinder solid whose rim circles sit at circleR while its side
// surface is a cylinder of cylR — modelling an imported edge that lies OFF its face (the SolidWorks
// gap, ADR-0030). The rims are still on the cap planes (z = 0 / height); only the cylinder side is
// missed, by |circleR − cylR|. Mirrors brep.SolidCylinder's construction.
func offSurfaceCylinder(t *testing.T, circleR, cylR, height float64) *topo.Body {
	t.Helper()
	axis, base := math.V3(0, 0, 1), math.P3(0, 0, 0)
	bottom, err := geom.NewCircle(base, axis, circleR)
	if err != nil {
		t.Fatalf("bottom circle: %v", err)
	}
	topCenter := base.TranslateBy(axis.Scale(math.Scalar(height)))
	top := geom.Circle{Center: topCenter, Normal: bottom.Normal, RefDir: bottom.RefDir, Radius: circleR}
	side, err := geom.NewCylinder(base, axis, cylR)
	if err != nil {
		t.Fatalf("cylinder: %v", err)
	}
	capBottom, err := geom.NewPlane(base, axis.Scale(-1))
	if err != nil {
		t.Fatalf("cap bottom: %v", err)
	}
	capTop, err := geom.NewPlane(topCenter, axis)
	if err != nil {
		t.Fatalf("cap top: %v", err)
	}
	vbp, vtp := bottom.PointAt(0), top.PointAt(0)
	lin := func(role string, i int) topo.Lineage { return topo.NewLineage(topo.Tok("offcyl", role, i)) }
	bld := topo.NewBuilder(true, lin("body", 0))
	vb, vt := bld.AddVertex(vbp, lin("v", 0)), bld.AddVertex(vtp, lin("v", 1))
	eb := bld.AddEdge(bottom, vb, vb, lin("e", 0))
	et := bld.AddEdge(top, vt, vt, lin("e", 1))
	es := bld.AddEdge(geom.NewLineSegment(vbp, vtp), vb, vt, lin("e", 2))
	bld.AddFace(capBottom, lin("f", 0), topo.OuterLoop(topo.Rev(eb)))
	bld.AddFace(capTop, lin("f", 1), topo.OuterLoop(topo.Fwd(et)))
	bld.AddFace(side, lin("f", 2), topo.OuterLoop(topo.Fwd(es), topo.Rev(et), topo.Rev(es), topo.Fwd(eb)))
	return bld.Build()
}

// TestSnapEdgesLeavesCleanSolidUnchanged is PBI-324's clean-fixture criterion: a natively-built solid
// already has its edges ON their surfaces, so snapping must leave every edge native (no stored snap,
// zero residual) and keep the mesh watertight. Only imported (off-surface) edges should move.
func TestSnapEdgesLeavesCleanSolidUnchanged(t *testing.T) {
	t.Parallel()
	cyl, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 10)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	SnapEdgesToSurfaces(cyl, DefaultQuality())
	for _, e := range cyl.Edges() {
		if e.SnappedCurve() != nil || e.Tolerance() != 0 {
			t.Errorf("clean edge %d was snapped (residual %g); want left native (already on its surfaces)", e.ID(), e.Tolerance())
		}
	}
	for _, gq := range gateQualities() {
		mesh, _ := tessellate.TessellateBody(cyl, gq.q)
		if free := freeEdgeCount(mesh); free != 0 {
			t.Errorf("%s quality: snapped clean cylinder tessellated with %d free edges; want 0 (still watertight)",
				gq.name, free)
		}
	}
}

// TestSnapEdgesLandsOffSurfaceRimOnCylinder is the end-to-end criterion: an imported rim circle that
// sits off its cylinder side is snapped ONTO the cylinder (the grid-meshed neighbour, preferred over
// the verbatim cap plane), with the gap recorded as the edge tolerance.
func TestSnapEdgesLandsOffSurfaceRimOnCylinder(t *testing.T) {
	t.Parallel()
	const circleR, cylR, height = 5.0, 4.8, 10.0
	body := offSurfaceCylinder(t, circleR, cylR, height)
	SnapEdgesToSurfaces(body, DefaultQuality())
	rims := 0
	for _, e := range body.Edges() {
		if _, isCircle := e.Geometry().(geom.Circle); !isCircle {
			continue // the seam line, handled separately
		}
		snapped := e.SnappedCurve()
		if snapped == nil {
			t.Errorf("off-surface rim edge %d not snapped; gap %g exceeds the floor", e.ID(), circleR-cylR)
			continue
		}
		for i, p := range snapped {
			if r := radiusXY(p); stdmath.Abs(r-cylR) > 1e-6 {
				t.Errorf("rim edge %d point %d at radius %g; want the cylinder radius %g", e.ID(), i, r, cylR)
			}
		}
		if tol := e.Tolerance(); stdmath.Abs(tol-(circleR-cylR)) > 1e-3 {
			t.Errorf("rim edge %d residual %g; want ~the %g off-surface gap", e.ID(), tol, circleR-cylR)
		}
		rims++
	}
	if rims != 2 {
		t.Fatalf("expected 2 rim circles; exercised %d", rims)
	}
}

// TestSnapEdgesMakesOffSurfaceWatertight is the watertightness payoff: a body whose rims sit off the
// cylinder side tessellates with free edges (each grid-meshed wall diverges from its neighbour), and
// snapping the edges onto the surfaces closes them into a watertight shell.
func TestSnapEdgesMakesOffSurfaceWatertight(t *testing.T) {
	t.Parallel()
	body := offSurfaceCylinder(t, 5.0, 4.8, 10.0)
	before, _ := tessellate.TessellateBody(body, DefaultQuality())
	if freeEdgeCount(before) == 0 {
		t.Fatal("off-surface body was already watertight; the fixture must leak to exercise the snap")
	}
	SnapEdgesToSurfaces(body, DefaultQuality())
	for _, gq := range gateQualities() {
		after, _ := tessellate.TessellateBody(body, gq.q)
		if free := freeEdgeCount(after); free != 0 {
			t.Errorf("%s quality: snapped off-surface body has %d free edges; want 0 (watertight)", gq.name, free)
		}
	}
}

// TestSnapEdgesSharesIdenticalBoundary pins that after snapping, both faces of an edge mesh the SAME
// boundary — discretizeEdge returns the one stored polyline, which loopBoundary feeds to every face.
func TestSnapEdgesSharesIdenticalBoundary(t *testing.T) {
	t.Parallel()
	body := offSurfaceCylinder(t, 5.0, 4.8, 10.0)
	SnapEdgesToSurfaces(body, DefaultQuality())
	manifoldCurved := 0
	for _, e := range body.Edges() {
		snapped := e.SnappedCurve()
		if snapped == nil {
			continue
		}
		got := tessellate.DiscretizeEdge(e, DefaultQuality())
		if len(got) != len(snapped) {
			t.Fatalf("edge %d: discretizeEdge len %d != snapped len %d", e.ID(), len(got), len(snapped))
		}
		for i := range got {
			if got[i] != snapped[i] {
				t.Errorf("edge %d point %d: discretizeEdge %v != snapped %v (faces would diverge)", e.ID(), i, got[i], snapped[i])
			}
		}
		if len(e.Faces()) == 2 && len(snapped) > 2 {
			manifoldCurved++ // a rim circle — the formerly-leaking case
		}
	}
	if manifoldCurved == 0 {
		t.Fatal("no manifold curved edge exercised; the cylinder rim circles should qualify")
	}
}

// TestOnSurfacePolylineLandsOnCylinder pins that an off-surface polyline projects back exactly onto a
// cylinder (radius restored) — the core of closing the ~mm import gap.
func TestOnSurfacePolylineLandsOnCylinder(t *testing.T) {
	t.Parallel()
	cyl, err := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 5)
	if err != nil {
		t.Fatalf("NewCylinder: %v", err)
	}
	raw := []math.Point3{math.P3(5.5, 0, 0), math.P3(0, 5.5, 4), math.P3(-5.5, 0, 9)} // radius 5.5, off-surface
	for i, p := range onSurfacePolyline(cyl, raw) {
		if r := radiusXY(p); stdmath.Abs(r-5) > 1e-9 {
			t.Errorf("point %d snapped to radius %g; want 5 (on the cylinder)", i, r)
		}
	}
}

// TestReconcileTwoGridSurfacesLandsOnBoth pins that an edge off BOTH grid-meshed surfaces (a cylinder
// and a sphere) reconciles onto their intersection — on both within tolerance — and records the gap.
func TestReconcileTwoGridSurfacesLandsOnBoth(t *testing.T) {
	t.Parallel()
	cyl, err := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 5)
	if err != nil {
		t.Fatalf("NewCylinder: %v", err)
	}
	rSphere := stdmath.Sqrt(125) // sphere ∩ coaxial r=5 cylinder is the circle z=±10, radius 5
	sphere, err := geom.NewSphere(math.P3(0, 0, 0), rSphere)
	if err != nil {
		t.Fatalf("NewSphere: %v", err)
	}
	raw := []math.Point3{math.P3(5.3, 0, 10.2)} // off both, near the (5,0,10) intersection point
	snapped, residual := reconcileOntoSurfaces(raw, []geom.Surface{cyl, sphere})
	p := snapped[0]
	if r := radiusXY(p); stdmath.Abs(r-5) > 1e-4 {
		t.Errorf("snapped radius %g; want 5 (on the cylinder)", r)
	}
	if d := float64(p.DistanceTo(math.P3(0, 0, 0))); stdmath.Abs(d-rSphere) > 1e-4 {
		t.Errorf("snapped |p|=%g; want %g (on the sphere)", d, rSphere)
	}
	if residual <= 0 {
		t.Errorf("residual %g; want the recorded off-surface gap > 0", residual)
	}
}

// TestMergeProjectionsPrefersGridMeshed pins the watertightness-critical policy: against a verbatim
// neighbour (a plane meshes its 3D boundary directly) the boundary is taken ON the grid-meshed
// cylinder, because the cylinder's grid mesher only reproduces an on-surface boundary.
func TestMergeProjectionsPrefersGridMeshed(t *testing.T) {
	t.Parallel()
	cyl, err := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 5)
	if err != nil {
		t.Fatalf("NewCylinder: %v", err)
	}
	plane, err := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	if err != nil {
		t.Fatalf("NewPlane: %v", err)
	}
	onCyl := []math.Point3{math.P3(1, 2, 3)}
	onPlane := []math.Point3{math.P3(4, 5, 6)}
	if got := mergeProjections(cyl, plane, onCyl, onPlane); got[0] != onCyl[0] {
		t.Errorf("grid-first merge = %v; want the cylinder (grid) projection %v", got[0], onCyl[0])
	}
	if got := mergeProjections(plane, cyl, onPlane, onCyl); got[0] != onCyl[0] {
		t.Errorf("verbatim-first merge = %v; want the cylinder (grid) projection %v", got[0], onCyl[0])
	}
}

// TestHealImportedBodySnapsAndAttachesPcurves pins the heal-pass composition: snapping runs (the
// off-surface rims land on the cylinder) AND every edge-use comes out with a pcurve, in one call.
func TestHealImportedBodySnapsAndAttachesPcurves(t *testing.T) {
	t.Parallel()
	body := offSurfaceCylinder(t, 5.0, 4.8, 10.0)
	HealImportedBody(body, DefaultQuality())
	snappedRims := 0
	for _, e := range body.Edges() {
		if _, isCircle := e.Geometry().(geom.Circle); isCircle && e.SnappedCurve() != nil {
			snappedRims++
		}
	}
	if snappedRims != 2 {
		t.Errorf("heal snapped %d rim circles; want 2", snappedRims)
	}
	for _, f := range body.Faces() {
		for _, l := range f.Loops() {
			for _, u := range l.EdgeUses() {
				if u.Pcurve() == nil {
					t.Errorf("edge-use on face %d has no pcurve after heal", f.ID())
				}
			}
		}
	}
}

// TestSnapEdgesIdempotent pins that re-healing reproduces the same result — snapEdge re-samples the
// raw curve, not the prior snap, so it never drifts.
func TestSnapEdgesIdempotent(t *testing.T) {
	t.Parallel()
	body := offSurfaceCylinder(t, 5.0, 4.8, 10.0)
	SnapEdgesToSurfaces(body, DefaultQuality())
	first := map[uint64][]math.Point3{}
	for _, e := range body.Edges() {
		first[e.ID()] = append([]math.Point3(nil), e.SnappedCurve()...)
	}
	SnapEdgesToSurfaces(body, DefaultQuality())
	for _, e := range body.Edges() {
		got, want := e.SnappedCurve(), first[e.ID()]
		if len(got) != len(want) {
			t.Fatalf("edge %d: second snap len %d != first %d", e.ID(), len(got), len(want))
		}
		for i := range got {
			if got[i] != want[i] {
				t.Errorf("edge %d point %d: second snap %v != first %v (not idempotent)", e.ID(), i, got[i], want[i])
			}
		}
	}
}
