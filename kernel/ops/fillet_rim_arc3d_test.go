// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/blend"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// TestFilletEdgesRoutesArc3dRim is the end-to-end regression for the rim-fillet pick gate widening
// (fillet_rim.go's isClosedCircularEdge): a cylinder/cap rim stored as a closed full-sweep geom.Arc3d —
// the shape the STEP importer actually produces (kernel/exchange never emits geom.Circle) — must route
// through blend.FilletEdges to the SAME toroidal-band rim fillet as TestFilletEdgesRoutesRim's geom.Circle
// rim. Before this fix, loneRimPick/resolveRim's geom.Circle-only gate rejected this edge outright, the
// pick fell through to loneArcPick, and cylSideEdgeAt correctly declined the self-closed vertex it was
// never designed to see — the rim path was dead for every imported circular rim (OCCT blend I9 et al).
func TestFilletEdgesRoutesArc3dRim(t *testing.T) {
	t.Parallel()
	b, err := solidCylinderArc3dTopRim(math.P3(0, 0, 0), math.V3(0, 0, 1), 1.0, 2.0)
	if err != nil {
		t.Fatal(err)
	}
	res, err := blend.FilletEdges(b, [][]byte{arc3dTopRimKey(t, b, 2.0)}, 0.3)
	if err != nil {
		t.Fatalf("blend.FilletEdges on a full-sweep Arc3d rim: %v", err)
	}
	if r := ops.Validate(res); !r.Valid || !res.IsSolid() {
		t.Fatalf("Arc3d-rim-filleted cylinder not a valid solid: %+v", r)
	}
	tor := 0
	for _, f := range res.Faces() {
		if _, ok := f.Geometry().(geom.Torus); ok {
			tor++
		}
	}
	if tor != 1 {
		t.Errorf("torus faces = %d, want 1", tor)
	}
	for _, tol := range []float64{0.05, 1e-2, 1e-3, 1e-4} {
		m, _ := tessellate.TessellateBody(res, ops.Quality{ChordTolerance: tol})
		if open := meshOpenEdges(m); open != 0 {
			t.Errorf("Arc3d rim fillet at tol %g: %d open edges", tol, open)
		}
	}
	full := stdmath.Pi * 2.0
	if v := ops.BodyGeometryProperties(res, ops.Quality{ChordTolerance: 1e-3}).Volume; v >= full || v < full-0.5 {
		t.Errorf("Arc3d-rim-fillet volume = %g, want under %g (rim notch removed)", v, full)
	}
}

// solidCylinderArc3dTopRim builds the SAME topology as brep.SolidCylinder (bottom cap, side wall, top
// cap, top rim circle), but with the top rim's curve stored as a closed geom.Arc3d sweeping a full
// 2π — the exact shape kernel/exchange's STEP importer emits for a plain imported circular rim, unlike
// brep.SolidCylinder's own geom.Circle (a purely procedural convenience geom.Circle{} literal, per
// isClosedCircularEdge's doc comment on why both forms must classify identically).
func solidCylinderArc3dTopRim(base math.Point3, axis math.Vector3, radius, height float64) (*topo.Body, error) {
	bottom, err := geom.NewCircle(base, axis, radius)
	if err != nil {
		return nil, err
	}
	topCenter := base.TranslateBy(axis.Scale(math.Scalar(height)))
	top, err := geom.NewArc3d(topCenter, bottom.Normal.AsVector(), bottom.RefDir.AsVector(), radius, 0, 2*stdmath.Pi)
	if err != nil {
		return nil, err
	}
	side, err := geom.NewCylinder(base, axis, radius)
	if err != nil {
		return nil, err
	}
	capBottom, err := geom.NewPlane(base, axis.Scale(-1))
	if err != nil {
		return nil, err
	}
	capTop, err := geom.NewPlane(topCenter, axis)
	if err != nil {
		return nil, err
	}

	vbp, vtp := bottom.PointAt(0), top.PointAt(0)
	lin := func(role string, i int) topo.Lineage { return topo.NewLineage(topo.Tok("arc3drim", role, i)) }
	bld := topo.NewBuilder(true, lin("body", 0))
	vb := bld.AddVertex(vbp, lin("v", 0))
	vt := bld.AddVertex(vtp, lin("v", 1))
	eb := bld.AddEdge(bottom, vb, vb, lin("e", 0)) // closed bottom circle
	et := bld.AddEdge(top, vt, vt, lin("e", 1))    // closed top rim, Arc3d not Circle — the case under test
	es := bld.AddEdge(geom.NewLineSegment(vbp, vtp), vb, vt, lin("e", 2))

	bld.AddFace(capBottom, lin("f", 0), topo.OuterLoop(topo.Rev(eb)))
	bld.AddFace(capTop, lin("f", 1), topo.OuterLoop(topo.Fwd(et)))
	bld.AddFace(side, lin("f", 2), topo.OuterLoop(topo.Fwd(es), topo.Rev(et), topo.Rev(es), topo.Fwd(eb)))
	return bld.Build(), nil
}

// arc3dTopRimKey returns the reference key of the body's top rim edge (an Arc3d near topZ) — the
// Arc3d-typed counterpart of fillet_rim_op_test.go's topRimKey (which matches geom.Circle only).
func arc3dTopRimKey(t *testing.T, b *topo.Body, topZ float64) []byte {
	t.Helper()
	for _, e := range b.Edges() {
		if _, ok := e.Geometry().(geom.Arc3d); ok && e.RangeBox().Center().Z > topZ-1e-3 {
			return e.ReferenceKey()
		}
	}
	t.Fatal("no Arc3d top rim edge")
	return nil
}
