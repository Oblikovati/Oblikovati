// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/math"
)

// fakePlaneFitSource is a named fake cloud source: it fits a horizontal plane at a settable height,
// and can report "cannot fit" to exercise the freeze path.
type fakePlaneFitSource struct {
	id     string
	height float64
	ok     bool
}

func (f *fakePlaneFitSource) SourceID() string { return f.id }
func (f *fakePlaneFitSource) FitFrame() (math.Point3, math.UnitVector3, math.UnitVector3, bool) {
	x, _ := math.NewUnitVector3(1, 0, 0)
	y, _ := math.NewUnitVector3(0, 1, 0)
	return math.P3(0, 0, math.Scalar(f.height)), x, y, f.ok
}

func newCloudFitPlanes(t *testing.T) *WorkPlanes {
	t.Helper()
	return NewWorkGeometry().WorkPlanes()
}

// TestCloudFitPlaneRefitsOnRecompute: the plane follows the source as it moves, and freezes the
// last good fit when the source can no longer fit (#645).
func TestCloudFitPlaneRefitsOnRecompute(t *testing.T) {
	t.Parallel()
	src := &fakePlaneFitSource{id: "Scan", height: 5, ok: true}
	planes := newCloudFitPlanes(t)
	wp := planes.AddByPointCloudFit(src)

	wp.recompute(nil)
	if got := float64(wp.Plane().Origin().Z); got != 5 {
		t.Errorf("initial fit Z = %v, want 5", got)
	}
	src.height = 9 // the cloud moves up
	wp.recompute(nil)
	if got := float64(wp.Plane().Origin().Z); got != 9 {
		t.Errorf("re-fit Z = %v, want 9 (plane should follow the cloud)", got)
	}
	src.ok = false // source can no longer fit → freeze at the last good fit (9)
	wp.recompute(nil)
	if got := float64(wp.Plane().Origin().Z); got != 9 {
		t.Errorf("frozen Z = %v, want 9 (last good fit)", got)
	}
}

// TestCloudFitPlaneSerializeRoundTrip: the provenance id and frozen frame round-trip, and the
// restored plane is unlinked until relinked (#645).
func TestCloudFitPlaneSerializeRoundTrip(t *testing.T) {
	t.Parallel()
	src := &fakePlaneFitSource{id: "CapstanScan", height: 3, ok: true}
	g := NewWorkGeometry()
	g.WorkPlanes().AddByPointCloudFit(src)

	data, err := MarshalWork(g)
	if err != nil {
		t.Fatalf("MarshalWork: %v", err)
	}
	var fit *WorkFeatureData
	for i := range data {
		if data[i].Kind == "point-cloud-fit" {
			fit = &data[i]
		}
	}
	if fit == nil {
		t.Fatal("no point-cloud-fit feature serialized")
	}
	if fit.CloudID != "CapstanScan" {
		t.Errorf("serialized cloud id = %q, want CapstanScan", fit.CloudID)
	}
	if len(fit.Position) != 3 || fit.Position[2] != 3 {
		t.Errorf("serialized frozen origin = %v, want z=3", fit.Position)
	}

	g2 := NewWorkGeometry()
	if err := ApplyWork(g2, data); err != nil {
		t.Fatalf("UnmarshalWork: %v", err)
	}
	planes := g2.WorkPlanes()
	var wp *WorkPlane
	for i := 0; i < planes.Count(); i++ { // the origin planes are restored too; find the user one
		if _, ok := planes.Item(i).def.(*pointCloudFitPlaneDef); ok {
			wp = planes.Item(i)
		}
	}
	if wp == nil {
		t.Fatal("no restored point-cloud-fit plane")
	}
	wp.recompute(nil) // no live source yet → frozen frame
	if got := float64(wp.Plane().Origin().Z); got != 3 {
		t.Errorf("restored frozen Z = %v, want 3", got)
	}

	// Relink a live source whose id matches → associativity restored.
	moved := &fakePlaneFitSource{id: "CapstanScan", height: 12, ok: true}
	if n := planes.RelinkCloudFits(func(id string) (PlaneFitSource, bool) {
		if id == "CapstanScan" {
			return moved, true
		}
		return nil, false
	}); n != 1 {
		t.Fatalf("RelinkCloudFits relinked %d, want 1", n)
	}
	wp.recompute(nil)
	if got := float64(wp.Plane().Origin().Z); got != 12 {
		t.Errorf("after relink Z = %v, want 12 (follows the re-attached cloud)", got)
	}
}

// TestCloudFitPlaneEvalAndRelinkEdges covers the error/edge branches: eval with neither a source
// nor a prior fit, the CloudID accessor, and a relink whose id does not match (#645).
func TestCloudFitPlaneEvalAndRelinkEdges(t *testing.T) {
	t.Parallel()
	d := &pointCloudFitPlaneDef{cloudID: "X"} // no source, no frozen fit
	if _, err := d.eval(nil); err == nil {
		t.Error("eval with no source and no prior fit should error")
	}
	if d.CloudID() != "X" {
		t.Errorf("CloudID = %q, want X", d.CloudID())
	}
	if d.relink(&fakePlaneFitSource{id: "other"}) {
		t.Error("relink with a mismatched id should report false")
	}
}
