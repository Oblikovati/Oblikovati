// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/math"
)

// TestAddChamferCapturesMintTimeAnchors pins ADR-0043 P6b for the NON-GUI creation path.
// Authoring a chamfer through the public builder — the wire-API / assembly / programmatic
// path, which only ever has reference keys, never resolved EdgeHandles — must still record
// each picked edge's mint-time midpoint in the definition, so a later upstream edit can fall
// back to the geometric recovery tier. Before P6b only the GUI tool captured anchors, so an
// API-authored dress-up silently degraded to ancestral-only recovery.
func TestAddChamferCapturesMintTimeAnchors(t *testing.T) {
	box, edge := box2(t)
	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(box)
	fs.Recompute() // the base is live before the edge is chamfered, as in every real authoring flow

	ch := NewDressUpFeatures(fs).AddChamfer([][]byte{edge}, func() float64 { return 0.3 })

	def := ch.Definition().(*ChamferFeature).Definition()
	got, ok := def.EdgeAnchors[string(edge)]
	if !ok {
		t.Fatalf("AddChamfer did not capture a mint-time anchor for the picked edge (EdgeAnchors=%v)", def.EdgeAnchors)
	}
	want := math.P3(2, 2, 1) // midpoint of the vertical edge (2,2,0)->(2,2,2)
	if got.DistanceTo(want) > 1e-9 {
		t.Errorf("captured anchor = %v, want the edge midpoint %v", got, want)
	}
}

// TestAddFilletCapturesMintTimeAnchors is the fillet parity of the chamfer case: the public
// edge-key fillet builder must capture each picked edge's mint-time midpoint, so an API-authored
// fillet has the geometric-recovery witness just like a GUI-authored one (ADR-0043 P6b).
func TestAddFilletCapturesMintTimeAnchors(t *testing.T) {
	box, edge := box2(t)
	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(box)
	fs.Recompute()

	fl := NewDressUpFeatures(fs).AddFillet([][]byte{edge}, func() float64 { return 0.3 })

	def := fl.Definition().(*FilletFeature).Definition()
	got, ok := def.EdgeAnchors[string(edge)]
	if !ok {
		t.Fatalf("AddFillet did not capture a mint-time anchor (EdgeAnchors=%v)", def.EdgeAnchors)
	}
	if want := math.P3(2, 2, 1); got.DistanceTo(want) > 1e-9 {
		t.Errorf("captured anchor = %v, want the edge midpoint %v", got, want)
	}
}

// TestRestoreEntryDoesNotCaptureAnchors pins the "restore never mutates the recipe" invariant:
// the low-level addChamfer entry, which the recipe restore calls, must NOT capture anchors even
// when a live body is present — reopening a document carries the persisted anchors verbatim and
// never rewrites them. Only the public authoring builders capture.
func TestRestoreEntryDoesNotCaptureAnchors(t *testing.T) {
	box, edge := box2(t)
	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(box)
	fs.Recompute() // a live tip body exists — yet the restore entry must still not capture

	ch := NewDressUpFeatures(fs).addChamfer(&ChamferDefinition{
		EdgeKeys: [][]byte{edge}, Distance: constFloat(0.3), Type: types.ChamferDistance,
	})

	def := ch.Definition().(*ChamferFeature).Definition()
	if len(def.EdgeAnchors) != 0 {
		t.Fatalf("addChamfer (the restore entry) must not capture anchors; got %v", def.EdgeAnchors)
	}
}

// TestAuthorWithoutRunningBodySkipsCapture pins the graceful degrade: when the part has not been
// recomputed (no tip body — a batch build), capture is skipped without error and recovery falls
// back to the ancestral tier, rather than panicking on a missing body.
func TestAuthorWithoutRunningBodySkipsCapture(t *testing.T) {
	_, edge := box2(t)
	fs := NewPartFeatures(nil) // never recomputed ⇒ Result() empty ⇒ no tip body

	ch := NewDressUpFeatures(fs).AddChamfer([][]byte{edge}, func() float64 { return 0.3 })

	def := ch.Definition().(*ChamferFeature).Definition()
	if len(def.EdgeAnchors) != 0 {
		t.Fatalf("with no running body capture must be skipped (ancestral-only degrade); got %v", def.EdgeAnchors)
	}
}
