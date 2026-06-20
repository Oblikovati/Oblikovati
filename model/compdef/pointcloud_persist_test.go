// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/doc"
)

// TestPointCloudRecordRoundTrip: a cloud's metadata + placement survive PointCloudRecords →
// SetPointCloudRecords, with its points re-decoded from the embedded resource (#645).
func TestPointCloudRecordRoundTrip(t *testing.T) {
	def := NewPartComponentDefinition()
	rid := def.AddResource(doc.Resource{Type: "PointCloudScan", Encoding: doc.EncodingUTF8, Value: []byte("0 0 0\n1 1 1\n"), Origin: "s.xyz"})
	pc, err := def.PointClouds().Add("Scan", "s.xyz", rid, nil)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	pc.SetVisible(false)
	pc.SetScale(3)
	pc.SetMaximumPointCount(1)

	records := def.PointCloudRecords()
	if len(records) != 1 || records[0].Name != "Scan" || records[0].Scale != 3 {
		t.Fatalf("records = %+v, want one Scan at scale 3", records)
	}

	target := NewPartComponentDefinition()
	target.SetResources(map[string]doc.Resource{rid: {Encoding: doc.EncodingUTF8, Value: []byte("0 0 0\n1 1 1\n")}})
	target.SetPointCloudRecords(records)

	got, ok := target.PointClouds().ByName("Scan")
	if !ok {
		t.Fatal("restored definition has no Scan cloud")
	}
	if got.Visible() || got.Scale() != 3 || got.MaximumPointCount() != 1 {
		t.Errorf("restored = visible %v scale %v max %d, want false/3/1", got.Visible(), got.Scale(), got.MaximumPointCount())
	}
	if got.TotalPointCount() != 2 { // re-decoded from the resource
		t.Errorf("restored points = %d, want 2 (re-decoded)", got.TotalPointCount())
	}
}

// TestPointCloudRecordMissingResource: a record whose resource is absent still rebuilds the cloud,
// with no points, so its metadata is not lost (#645).
func TestPointCloudRecordMissingResource(t *testing.T) {
	def := NewPartComponentDefinition()
	def.SetPointCloudRecords([]doc.PointCloudRecord{{
		Name: "Gone", Source: "s.xyz", ResourceID: "absent", Visible: true, Scale: 1,
		Transform: math.Identity4().Cells(),
	}})
	got, ok := def.PointClouds().ByName("Gone")
	if !ok {
		t.Fatal("cloud with a missing resource should still rebuild")
	}
	if got.TotalPointCount() != 0 {
		t.Errorf("missing-resource cloud points = %d, want 0", got.TotalPointCount())
	}
}
