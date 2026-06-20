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
	s.relinkPointCloudFits(restored)
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
