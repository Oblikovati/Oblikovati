// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/feature"
)

// planarPoints are scan points on z = 5 (a horizontal plane), shared by the provenance tests.
func planarPoints() []math.Point3 {
	return []math.Point3{
		math.P3(0, 0, 5), math.P3(2, 0, 5), math.P3(0, 2, 5),
		math.P3(2, 2, 5), math.P3(1, 3, 5), math.P3(-1, 1, 5),
	}
}

// liftZ is a pure +dz translation matrix (row-major, translation in the last column).
func liftZ(dz float64) math.Matrix4 {
	m := math.Identity4()
	c := m.Cells()
	c[11] = math.Scalar(dz)
	return math.Matrix4FromCells(c)
}

// TestPointCloudPlaneFollowsCloud: a plane fit to a cloud re-fits when the cloud moves, because it
// keeps a live link to the cloud (provenance), not a frozen frame (#645).
func TestPointCloudPlaneFollowsCloud(t *testing.T) {
	s, def := emptyPartSession(t)
	attachPlanarCloud(t, def) // points on z = 5
	wp, _, err := s.CreatePointCloudPlane("Scan")
	if err != nil {
		t.Fatalf("CreatePointCloudPlane: %v", err)
	}
	def.Recompute()
	if got := float64(wp.Plane().Origin().Z); !approxEq(got, 5) {
		t.Fatalf("initial plane Z = %v, want 5", got)
	}

	pc, _ := def.PointClouds().ByName("Scan")
	pc.SetTransform(liftZ(10)) // the cloud moves up by 10
	def.Recompute()
	if got := float64(wp.Plane().Origin().Z); !approxEq(got, 15) {
		t.Errorf("after the cloud moved, plane Z = %v, want 15 (it should follow)", got)
	}
}

// TestPointCloudPlaneProvenanceRoundTrip: the plane's cloud link survives a marshal/restore +
// relink, so a reopened document re-fits the plane to its cloud (#645).
func TestPointCloudPlaneProvenanceRoundTrip(t *testing.T) {
	s, def := emptyPartSession(t)
	attachPlanarCloud(t, def)
	if _, _, err := s.CreatePointCloudPlane("Scan"); err != nil {
		t.Fatalf("CreatePointCloudPlane: %v", err)
	}

	// Round-trip the work features into a fresh part that also carries the same-named cloud.
	data, err := feature.MarshalWork(def.WorkGeometry())
	if err != nil {
		t.Fatalf("MarshalWork: %v", err)
	}
	restored := compdef.NewPartComponentDefinition()
	rid := restored.AddResource(doc.Resource{Encoding: doc.EncodingUTF8, Value: []byte("x")})
	if _, err := restored.PointClouds().Add("Scan", "s.xyz", rid, planarPoints()); err != nil {
		t.Fatalf("attach to restored part: %v", err)
	}
	if err := feature.ApplyWork(restored.WorkGeometry(), data); err != nil {
		t.Fatalf("ApplyWork: %v", err)
	}

	// Relink the way OpenDocument does, then recompute: the plane re-fits to the restored cloud.
	s.relinkPointCloudProvenance(restored)
	restored.Recompute()
	wp := cloudFitPlaneOf(t, restored)
	if got := float64(wp.Plane().Origin().Z); !approxEq(got, 5) {
		t.Errorf("restored+relinked plane Z = %v, want 5", got)
	}

	// Moving the restored cloud now drives the relinked plane — associativity is live again.
	pc, _ := restored.PointClouds().ByName("Scan")
	pc.SetTransform(liftZ(7))
	restored.Recompute()
	if got := float64(wp.Plane().Origin().Z); !approxEq(got, 12) {
		t.Errorf("after moving the restored cloud, plane Z = %v, want 12", got)
	}
}

func cloudFitPlaneOf(t *testing.T, def *compdef.PartComponentDefinition) *feature.WorkPlane {
	t.Helper()
	planes := def.WorkPlanes()
	for i := 0; i < planes.Count(); i++ {
		if planes.Item(i).Kind() == "point-cloud-fit" {
			return planes.Item(i)
		}
	}
	t.Fatal("no point-cloud-fit plane in the restored part")
	return nil
}

func approxEq(a, b float64) bool { d := a - b; return d < 1e-9 && d > -1e-9 }

// TestCloudPlaneFitSourceEdges covers the source's no-fit branch (too few points) and a relink
// that finds no matching cloud in the part (#645).
func TestCloudPlaneFitSourceEdges(t *testing.T) {
	_, def := emptyPartSession(t)
	rid := def.AddResource(doc.Resource{Encoding: doc.EncodingUTF8, Value: []byte("x")})
	pc, err := def.PointClouds().Add("Tiny", "t.xyz", rid, []math.Point3{math.P3(0, 0, 0), math.P3(1, 1, 1)})
	if err != nil {
		t.Fatalf("attach: %v", err)
	}
	if _, _, _, ok := (cloudPlaneFitSource{pc}).FitFrame(); ok {
		t.Error("FitFrame on two points should report ok=false (no plane)")
	}
	// A point-cloud-fit plane whose cloud is absent: relink finds no match (the attach false branch).
	def.WorkPlanes().AddByPointCloudFit(cloudPlaneFitSource{pc})
	def.PointClouds().Remove("Tiny")
	s2, _ := emptyPartSession(t)
	s2.relinkPointCloudProvenance(def) // no cloud named "Tiny" now → nothing relinked, no recompute
}

// TestWorkPointFollowsCloud: a work point anchored on a scan point follows the cloud when it moves,
// because it keeps a live link (provenance) rather than a frozen position (#645).
func TestWorkPointFollowsCloud(t *testing.T) {
	s, def := emptyPartSession(t)
	rid := def.AddResource(doc.Resource{Encoding: doc.EncodingUTF8, Value: []byte("x")})
	pc, err := def.PointClouds().Add("Scan", "s.xyz", rid, []math.Point3{math.P3(3, 4, 5)})
	if err != nil {
		t.Fatalf("attach: %v", err)
	}
	s.Select(PointCloudPointHandle{Cloud: pc, Point: math.P3(3, 4, 5)})
	wp, err := s.CreateWorkPointAtSelectedCloudPoint()
	if err != nil {
		t.Fatalf("CreateWorkPointAtSelectedCloudPoint: %v", err)
	}
	if wp.Point() != math.P3(3, 4, 5) {
		t.Fatalf("initial work point = %v, want (3,4,5)", wp.Point())
	}

	pc.SetTransform(liftZ(10)) // the cloud moves up
	def.Recompute()
	if wp.Point() != math.P3(3, 4, 15) {
		t.Errorf("after the cloud moved, work point = %v, want (3,4,15) (it should follow)", wp.Point())
	}
}

// TestWorkPointProvenanceRoundTrip: the work point's cloud link survives marshal/restore + relink,
// so a reopened document re-anchors it and it follows the restored cloud (#645).
func TestWorkPointProvenanceRoundTrip(t *testing.T) {
	s, def := emptyPartSession(t)
	rid := def.AddResource(doc.Resource{Encoding: doc.EncodingUTF8, Value: []byte("x")})
	pc, _ := def.PointClouds().Add("Scan", "s.xyz", rid, []math.Point3{math.P3(1, 2, 3)})
	s.Select(PointCloudPointHandle{Cloud: pc, Point: math.P3(1, 2, 3)})
	if _, err := s.CreateWorkPointAtSelectedCloudPoint(); err != nil {
		t.Fatalf("create: %v", err)
	}

	data, err := feature.MarshalWork(def.WorkGeometry())
	if err != nil {
		t.Fatalf("MarshalWork: %v", err)
	}
	restored := compdef.NewPartComponentDefinition()
	rid2 := restored.AddResource(doc.Resource{Encoding: doc.EncodingUTF8, Value: []byte("x")})
	rpc, _ := restored.PointClouds().Add("Scan", "s.xyz", rid2, []math.Point3{math.P3(1, 2, 3)})
	if err := feature.ApplyWork(restored.WorkGeometry(), data); err != nil {
		t.Fatalf("ApplyWork: %v", err)
	}
	s.relinkPointCloudProvenance(restored)
	restored.Recompute()

	wp := restoredCloudPoint(t, restored)
	if wp.Point() != math.P3(1, 2, 3) {
		t.Errorf("restored+relinked work point = %v, want (1,2,3)", wp.Point())
	}
	rpc.SetTransform(liftZ(8)) // moving the restored cloud drives the relinked point
	restored.Recompute()
	if wp.Point() != math.P3(1, 2, 11) {
		t.Errorf("after moving the restored cloud, work point = %v, want (1,2,11)", wp.Point())
	}
}

func restoredCloudPoint(t *testing.T, def *compdef.PartComponentDefinition) *feature.WorkPoint {
	t.Helper()
	pts := def.WorkPoints()
	for i := 0; i < pts.Count(); i++ {
		// the position kind is "position"; the anchored one round-trips as a distinct frozen point
		if pts.Item(i).Point() == math.P3(1, 2, 3) {
			return pts.Item(i)
		}
	}
	t.Fatal("no restored cloud work point")
	return nil
}

// TestWorkPointProvenanceEdges covers the degenerate-placement anchor fallback and a work-point
// relink that finds no matching cloud (#645).
func TestWorkPointProvenanceEdges(t *testing.T) {
	s, def := emptyPartSession(t)
	rid := def.AddResource(doc.Resource{Encoding: doc.EncodingUTF8, Value: []byte("x")})
	pc, err := def.PointClouds().Add("Scan", "s.xyz", rid, []math.Point3{math.P3(1, 1, 1)})
	if err != nil {
		t.Fatalf("attach: %v", err)
	}
	// A singular placement makes FromModelSpace fail → newCloudPointSource anchors at the raw point.
	pc.SetTransform(math.Matrix4FromCells([16]math.Scalar{}))
	src := newCloudPointSource(pc, math.P3(2, 2, 2))
	if src.local != math.P3(2, 2, 2) {
		t.Errorf("degenerate anchor local = %v, want the raw point (2,2,2)", src.local)
	}

	// A point-cloud work point whose cloud is gone: relink finds no match (the attach false branch).
	def.WorkPoints().AddByCloudPoint(newCloudPointSource(pc, math.P3(1, 1, 1)))
	def.PointClouds().Remove("Scan")
	s.relinkPointCloudProvenance(def) // no cloud "Scan" now → nothing relinked
}
