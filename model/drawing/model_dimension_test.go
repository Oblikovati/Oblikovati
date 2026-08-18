// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"testing"

	gmath "oblikovati.org/math"
)

// fakeModelDims is a named fake (CLAUDE.md: no inline stubs) resolving a fixed set of model
// dimensions — the referenced part's parametric dimensions a retrieve pulls onto a view.
type fakeModelDims struct{ dims []ModelDimension }

func (f fakeModelDims) ModelDimensions(string) ([]ModelDimension, bool) { return f.dims, true }

// TestListRetrievableDimensions: the referenced model's parametric dimensions list for a base view
// with their names and values (#1991).
func TestListRetrievableDimensions(t *testing.T) {
	c := drawingWithCylinder(t, 2)
	c.SetModelDimensionResolver(fakeModelDims{dims: []ModelDimension{
		{Name: "width", Value: 2, A: gmath.P3(0, 0, 0), B: gmath.P3(2, 0, 0)},
	}})
	topBase(t, c.Sheets().Active().Views())

	list, err := c.Sheets().Active().Dimensions().ListRetrievable("TOP")
	if err != nil {
		t.Fatalf("ListRetrievable: %v", err)
	}
	if len(list) != 1 || list[0].Name != "width" || list[0].ValueMM != 20 {
		t.Fatalf("retrievable = %+v, want one 'width' at 20 mm", list)
	}
}

// TestRetrieveModelDimensions: retrieving a model dimension creates a drawing dimension flagged
// Retrieved with a RetrievedFrom back-reference, measuring the parameter's value, and re-measures when
// the model changes; editing its value is rejected with a clear error (#1991).
func TestRetrieveModelDimensions(t *testing.T) {
	c := drawingWithCylinder(t, 2)
	c.SetModelDimensionResolver(fakeModelDims{dims: []ModelDimension{
		{Name: "width", Value: 2, A: gmath.P3(0, 0, 0), B: gmath.P3(2, 0, 0)},
	}})
	topBase(t, c.Sheets().Active().Views())
	ds := c.Sheets().Active().Dimensions()

	got, err := ds.Retrieve("TOP", []string{"width"}, 10)
	if err != nil {
		t.Fatalf("Retrieve: %v", err)
	}
	d := got[0]
	if !d.Retrieved() || d.RetrievedFrom() != "width" || d.ValueMM() != 20 || d.CurveCount() == 0 {
		t.Fatalf("retrieved dim = (retrieved %v, from %q, %g mm, %d curves), want retrieved/width/20/glyph",
			d.Retrieved(), d.RetrievedFrom(), d.ValueMM(), d.CurveCount())
	}

	// Editing a retrieved dimension's value is rejected this increment (AC3).
	if err := ds.SetRetrievedValue(d.Name(), 5); err == nil {
		t.Error("editing a retrieved dimension's value should be rejected with a clear error")
	}

	// The parameter grows to 3 cm; on recompute the retrieved dimension re-measures (AC4).
	c.SetModelDimensionResolver(fakeModelDims{dims: []ModelDimension{
		{Name: "width", Value: 3, A: gmath.P3(0, 0, 0), B: gmath.P3(3, 0, 0)},
	}})
	c.RecomputeViews()
	if d.ValueMM() != 30 {
		t.Errorf("after the model grew, retrieved dim = %g mm, want 30 (re-measured)", d.ValueMM())
	}
}

// TestRetrieveUnknownDimensionRejected: retrieving a name the model does not offer is a clean error.
func TestRetrieveUnknownDimensionRejected(t *testing.T) {
	c := drawingWithCylinder(t, 2)
	c.SetModelDimensionResolver(fakeModelDims{dims: []ModelDimension{
		{Name: "width", Value: 2, A: gmath.P3(0, 0, 0), B: gmath.P3(2, 0, 0)},
	}})
	topBase(t, c.Sheets().Active().Views())
	if _, err := c.Sheets().Active().Dimensions().Retrieve("TOP", []string{"ghost"}, 0); err == nil {
		t.Error("retrieving an unknown model dimension should fail")
	}
}
